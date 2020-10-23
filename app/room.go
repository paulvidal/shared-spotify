package app

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/spotifyclient"
	"math/rand"
	"net/http"
	"strconv"
)

const maxNumberRooms = 100

var roomDoesNotExistError = errors.New("room does not exists")
var roomIsNotAccessibleError = errors.New("room is not accessible to user")
var authenticationError = errors.New("failed to authenticate user")

// An in memory representation of all the rooms, would be better if it was persistent but for now this is fine
var allRooms = AllRooms{make(map[string]*Room, 0)}

type AllRooms struct {
	Rooms map[string]*Room `json:"rooms"`
}

type Room struct {
	Id       string                 `json:"id"`
	Users    []*spotifyclient.User  `json:"users"`
	Locked   bool                   `json:"locked"`
}

func createRoom() *Room {
	var randomId string

	for {
		randomId = strconv.Itoa(rand.Intn(maxNumberRooms))

		// Find a room id that is not already taken
		if _, exists := allRooms.Rooms[randomId]; !exists {
			break
		}
	}

	room := &Room{randomId, make([]*spotifyclient.User, 0), false}

	// Add the rooms
	allRooms.Rooms[randomId] = room

	return room
}

func (room *Room) addUser(user *spotifyclient.User) {
	// If the user is already in the room, do not add it
	if room.isUserInRoom(user) {
		return
	}

	room.Users = append(room.Users, user)
}

func (room *Room) isUserInRoom(user *spotifyclient.User) bool {
	for _, roomUser := range room.Users {
		if roomUser.IsEqual(user) {
			return true
		}
	}

	return false
}

func getRoom(roomId string) (*Room, error) {
	room, ok := allRooms.Rooms[roomId]

	if !ok {
		return nil, roomDoesNotExistError
	}

	return room, nil
}

func getRoomAndCheckUser(roomId string, r *http.Request) (*Room, error) {
	room, err := getRoom(roomId)

	if err != nil {
		return nil, err
	}

	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		return nil, authenticationError
	}

	if !room.isUserInRoom(user) {
		return nil, roomIsNotAccessibleError
	}

	return room, nil
}

func handleError(err error, w http.ResponseWriter) {
	if err == roomDoesNotExistError {
		http.Error(w, err.Error(), http.StatusBadRequest)

	} else if err == roomIsNotAccessibleError {
		http.Error(w, err.Error(), http.StatusUnauthorized)

	} else if err == authenticationError {
		httputils.AuthenticationError(w)

	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
  Rooms handler
 */
func RoomsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetRooms(w, r)
	case http.MethodPost:
		CreateRoom(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetRooms(w http.ResponseWriter, r *http.Request)  {
	httputils.SendJson(w, allRooms)
}

func CreateRoom(w http.ResponseWriter, r *http.Request)  {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w)
		return
	}

	room := createRoom()
	room.addUser(user)

	httputils.SendOk(w)
}

/*
  Room handler
*/

func RoomHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetRoom(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w)
		return
	}

	httputils.SendJson(w, room)
}

/*
  Room users handler
*/

func RoomUsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetRoomUsers(w, r)
	case http.MethodPost:
		AddRoomUser(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetRoomUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w)
		return
	}

	httputils.SendJson(w, room.Users)
}

func AddRoomUser(w http.ResponseWriter, r *http.Request) {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoom(roomId)

	if err != nil {
		handleError(err, w)
		return
	}

	room.addUser(user)

	httputils.SendOk(w)
}

/*
  Room music handler
*/

func RoomMusicHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetMusicsForRoom(w, r)
	case http.MethodPost:
		FindMusicsForRoom(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetMusicsForRoom(w http.ResponseWriter, r *http.Request) {

}

// Here, we launch the process of finding the musics for the users in the room
func FindMusicsForRoom(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	_, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w)
		return
	}

	handleError(err, w)
}