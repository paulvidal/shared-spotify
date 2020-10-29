package app

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"net/http"
)

var roomDoesNotExistError = errors.New("room does not exists")
var roomIsNotAccessibleError = errors.New("room is not accessible to user")
var authenticationError = errors.New("failed to authenticate user")
var roomLockedError = errors.New("room is locked and not accepting new members")
var processingInProgressError = errors.New("processing of music is already in progress")
var processingNotStartedError = errors.New("processing of music has not been done, cannot get playlists")
var processingFailedError = errors.New("processing of music failed, cannot get playlists")
var failedToCreatePlaylistError = errors.New("an error occurred while creating the playlist")


// An in memory representation of all the rooms, would be better if it was persistent but for now this is fine
var allRooms = RoomCollection{make(map[string]*Room)}

type RoomCollection struct {
	Rooms map[string]*Room `json:"rooms"`
}

type Room struct {
	Id            string                `json:"id"`
	Users         []*spotifyclient.User `json:"users"`
	Locked        *bool                 `json:"locked"`
	MusicLibrary  *SharedMusicLibrary   `json:"shared_music_library"`

}

func createRoom() *Room {
	randomId := utils.GenerateStrongHash()
	locked := false
	room := &Room{randomId, make([]*spotifyclient.User, 0), &locked, nil}

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

func (room *Room) getUserIds() []string {
	userNames := make([]string, 0)
	for _, user := range room.Users {
		userNames = append(userNames, user.Infos.Id)
	}
	return userNames
}

func getRoom(roomId string) (*Room, error) {
	room, ok := allRooms.Rooms[roomId]

	if !ok {
		return nil, roomDoesNotExistError
	}

	return room, nil
}

func getRoomAndCheckUser(roomId string, r *http.Request) (*Room, *spotifyclient.User, error) {
	room, err := getRoom(roomId)

	if err != nil {
		return nil, nil, err
	}

	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		return nil, nil, authenticationError
	}

	if !room.isUserInRoom(user) {
		return nil, user, roomIsNotAccessibleError
	}

	return room, user, nil
}

func handleError(err error, w http.ResponseWriter, r *http.Request, user *spotifyclient.User) {
	userId := "unknown"

	if user != nil {
		userId = user.GetUserId()
	}

	logger.WithUser(userId).Error("Handling error: ", err)

	if err == roomDoesNotExistError {
		http.Error(w, err.Error(), http.StatusBadRequest)

	} else if err == roomIsNotAccessibleError {
		http.Error(w, err.Error(), http.StatusUnauthorized)

	} else if err == authenticationError {
		httputils.AuthenticationError(w, r)

	} else if err == roomLockedError {
		http.Error(w, err.Error(), http.StatusBadRequest)

	} else if err == errorPlaylistTypeNotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)

	} else if err == processingInProgressError || err == processingFailedError || err == processingNotStartedError {
		http.Error(w, err.Error(), http.StatusBadRequest)

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
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		handleError(authenticationError, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to get rooms", user.GetUserId())

	rooms := make(map[string]*Room)

	for roomId, room := range allRooms.Rooms {
		if room.isUserInRoom(user) {
			rooms[roomId] = room
		}
	}

	roomCollection := RoomCollection{rooms}

	httputils.SendJson(w, &roomCollection)
}

type NewRoom struct {
	RoomId string `json:"room_id"`
}

func CreateRoom(w http.ResponseWriter, r *http.Request)  {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		handleError(authenticationError, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to create a room", user.GetUserId())

	room := createRoom()
	room.addUser(user)

	httputils.SendJson(w, NewRoom{room.Id})
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

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to get room %s", user.GetUserId(), roomId)

	httputils.SendJson(w, room)
}

/*
  Room users handler
*/

func RoomUsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		AddRoomUser(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func AddRoomUser(w http.ResponseWriter, r *http.Request) {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		handleError(authenticationError, w, r, user)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoom(roomId)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to be added to room %s", user.GetUserId(), roomId)

	if room.isUserInRoom(user) {
		// if user is already in room, just send ok
		httputils.SendOk(w)
	}

	if *room.Locked {
		handleError(roomLockedError, w, r, user)
		return
	}

	room.addUser(user)

	httputils.SendOk(w)
}