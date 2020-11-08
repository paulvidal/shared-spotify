package app

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"net/http"
	"time"
)

const defaultRoomName = "Room #%s"

var roomDoesNotExistError = errors.New("Room does not exists")
var roomIsNotAccessibleError = errors.New("Room is not accessible to user")
var authenticationError = errors.New("Failed to authenticate user")
var roomLockedError = errors.New("Room is locked and not accepting new members. Create a new one to share music")
var processingInProgressError = errors.New("Processing of music is already in progress")
var processingNotStartedError = errors.New("Processing of music has not been done, cannot get playlists")
var processingFailedError = errors.New("Processing of music failed, cannot get playlists")
var failedToCreatePlaylistError = errors.New("An error occurred while creating the playlist")

// An in memory representation of all the rooms, would be better if it was persistent but for now this is fine
var allRooms = RoomCollection{make(map[string]*Room)}

type RoomCollection struct {
	Rooms map[string]*Room `json:"rooms"`
}

type Room struct {
	Id           string                `json:"id"`
	Name         string                `json:"name"`
	Owner        *spotifyclient.User   `json:"owner"`
	Users        []*spotifyclient.User `json:"users"`
	CreationTime time.Time             `json:"creation_time"`
	Locked       *bool                 `json:"locked"`
	MusicLibrary *SharedMusicLibrary   `json:"shared_music_library"`
}

type RoomWithOwnerInfo struct {
	*Room
	IsOwner bool `json:"is_owner"`
}

func createRoom(roomId string, roomName string, owner *spotifyclient.User) *Room {
	locked := false
	room := &Room{
		roomId,
		roomName,
		owner,
		make([]*spotifyclient.User, 0),
		time.Now(),
		&locked,
		nil,
	}

	// Add the owner to the room
	room.addUser(owner)

	// Add to the room list
	allRooms.Rooms[roomId] = room

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

func (room *Room) isOwner(user *spotifyclient.User) bool {
	return room.Owner.IsEqual(user)
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

func GetRooms(w http.ResponseWriter, r *http.Request) {
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

type CreatedRoom struct {
	RoomId string `json:"room_id"`
}

type NewRoom struct {
	RoomName string `json:"room_name"`
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		handleError(authenticationError, w, r, user)
		return
	}

	var newRoom NewRoom
	err = httputils.DeserialiseBody(r, &newRoom)

	if err != nil {
		logger.Logger.Error("Failed to decode json body for add playlist for user")
		handleError(err, w, r, user)
		return
	}

	roomId := utils.GenerateStrongHash()
	roomName := newRoom.RoomName

	// In case no room name was given, we use the room id
	if roomName == "" {
		roomName = fmt.Sprintf(defaultRoomName, roomId)
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to create room with name=%s roomId=%s",
		user.GetUserId(), roomName, roomId)

	room := createRoom(roomId, roomName, user)

	httputils.SendJson(w, CreatedRoom{room.Id})
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

	roomWithOwnerInfo := RoomWithOwnerInfo{
		room,
		room.isOwner(user),
	}

	httputils.SendJson(w, roomWithOwnerInfo)
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
