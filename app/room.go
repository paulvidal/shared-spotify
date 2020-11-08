package app

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/appmodels"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"net/http"
)

const defaultRoomName = "Room #%s"

var failedToCreateRoom = errors.New("Failed to create room")
var failedToGetRoom = errors.New("Failed to get room")
var failedToGetRooms = errors.New("Failed to get rooms")
var roomDoesNotExistError = errors.New("Room does not exists")
var roomIsNotAccessibleError = errors.New("Room is not accessible to user")
var authenticationError = errors.New("Failed to authenticate user")
var roomLockedError = errors.New("Room is locked and not accepting new members. Create a new one to share music")
var processingInProgressError = errors.New("Processing of music is already in progress")
var processingNotStartedError = errors.New("Processing of music has not been done, cannot get playlists")
var processingFailedError = errors.New("Processing of music failed, cannot get playlists")
var failedToCreatePlaylistError = errors.New("An error occurred while creating the playlist")

// we store in memory the rooms not processed so that if the server crashes, we do not need to manage recovery of
// ongoing processing - it has the pitfall though that we won't preserve state for not processed room
var roomNotProcessed = make(map[string]*appmodels.Room)

func addRoomNotProcessed(room *appmodels.Room) {
	roomNotProcessed[room.Id] = room
}

func removeRoomNotProcessed(room *appmodels.Room) {
	err := mongoclient.InsertRoom(room)

	if err != nil {
		// if we fail to insert the result in mongo, we declare processing as failed
		success := false
		roomNotProcessed[room.Id].MusicLibrary.SetProcessingSuccess(&success)

	} else {
		// otherwise we delete the room from the rooms being processed
		delete(roomNotProcessed, room.Id)
	}
}

func getRoom(roomId string) (*appmodels.Room, error) {
	// we check if a room not processed exists first, and we use it if it exists
	if roomNotProcessed, ok := roomNotProcessed[roomId]; ok {
		return roomNotProcessed, nil
	}

	room, err := mongoclient.GetRoom(roomId)

	if err == mongoclient.NotFound {
		return nil, roomDoesNotExistError
	}

	if err != nil {
		return nil, failedToGetRoom
	}

	return room, nil
}

func getRoomAndCheckUser(roomId string, r *http.Request) (*appmodels.Room, *spotifyclient.User, error) {
	room, err := getRoom(roomId)

	if err != nil {
		return nil, nil, err
	}

	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		return nil, nil, authenticationError
	}

	if !room.IsUserInRoom(user) {
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

	} else if err == appmodels.ErrorPlaylistTypeNotFound {
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

	rooms, err := mongoclient.GetRoomsForUser(user)

	if err != nil {
		handleError(failedToGetRooms, w, r, user)
		return
	}

	// we add to these rooms the not processed yet rooms
	for _, room := range roomNotProcessed {
		if room.IsUserInRoom(user) {
			rooms = append(rooms, room)
		}
	}

	httputils.SendJson(w, &rooms)
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

	room := appmodels.CreateRoom(roomId, roomName, user)

	// Add the room in memory (it will be added to mongo once processed)
	addRoomNotProcessed(room)

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

	roomWithOwnerInfo := appmodels.RoomWithOwnerInfo{
		Room:    room,
		IsOwner: room.IsOwner(user),
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

	if room.IsUserInRoom(user) {
		// if user is already in room, just send ok
		httputils.SendOk(w)
	}

	if *room.Locked {
		handleError(roomLockedError, w, r, user)
		return
	}

	room.AddUser(user)

	httputils.SendOk(w)
}
