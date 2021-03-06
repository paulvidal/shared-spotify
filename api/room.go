package api

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	mongoclientapp "github.com/shared-spotify/mongoclient/app"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/utils"
	"net/http"
	"sync"
)

const defaultRoomName = "Room #%s"

var failedToGetRoom = errors.New("Failed to get room")
var failedToDeleteRoom = errors.New("Failed to delete room")
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
var roomNotProcessed = sync.Map{} // map[string]*app.Room

func addRoomNotProcessed(room *app.Room) {
	datadog.Increment(1, datadog.RoomCount,
		datadog.RoomIdTag.Tag(room.Id),
		datadog.RoomNameTag.Tag(room.Name),
	)

	roomNotProcessed.Store(room.Id, room)
}

func updateRoomNotProcessed(room *app.Room, success bool) {
	// we set processing result
	roomNotProcessed, _ := roomNotProcessed.Load(room.Id)
	roomNotProcessed.(*app.Room).MusicLibrary.SetProcessingSuccess(&success)

	if !success {
		datadog.Increment(1, datadog.RoomProcessedFailed,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)
		return
	}

	// we insert the room result in mongo
	err := mongoclientapp.InsertRoom(room)

	if err != nil {
		// if we fail to insert the result in mongo, we declare processing as failed
		success := false
		roomNotProcessed.(*app.Room).MusicLibrary.SetProcessingSuccess(&success)
		datadog.Increment(1, datadog.RoomProcessedFailed,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)

	} else {
		// otherwise we delete the room from the rooms being processed
		deleteRoomNotProcessed(room)
		datadog.Increment(1, datadog.RoomProcessedCount,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)
	}
}

func deleteRoomNotProcessed(room *app.Room) {
	roomNotProcessed.Delete(room.Id)
}

func getRoom(roomId string) (*app.Room, error) {
	// we check if a room not processed exists first, and we use it if it exists
	if roomNotProcessed, ok := roomNotProcessed.Load(roomId); ok {
		return roomNotProcessed.(*app.Room), nil
	}

	room, err := mongoclientapp.GetRoom(roomId)

	if err == mongoclientapp.NotFound {
		return nil, roomDoesNotExistError
	}

	if err != nil {
		return nil, failedToGetRoom
	}

	return room, nil
}

func getRoomAndCheckUser(roomId string, r *http.Request) (*app.Room, *clientcommon.User, error) {
	room, err := getRoom(roomId)

	if err != nil {
		return nil, nil, err
	}

	user, err := musicclient.CreateUserFromRequest(r)

	if err != nil {
		return nil, nil, authenticationError
	}

	if !room.IsUserInRoom(user) {
		return nil, user, roomIsNotAccessibleError
	}

	return room, user, nil
}

func handleError(err error, w http.ResponseWriter, r *http.Request, user *clientcommon.User) {
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

	} else if err == app.ErrorPlaylistTypeNotFound {
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
	user, err := musicclient.CreateUserFromRequest(r)

	if err != nil {
		handleError(authenticationError, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to get rooms", user.GetUserId())

	rooms, err := mongoclientapp.GetRoomsForUser(user)

	if err != nil {
		handleError(failedToGetRooms, w, r, user)
		return
	}

	// we add to these rooms the not processed yet rooms
	roomNotProcessed.Range(func(_, value interface{}) bool {
		room := value.(*app.Room)

		if room.IsUserInRoom(user) {
			rooms = append(rooms, room)
		}

		return true
	})

	httputils.SendJson(w, &rooms)
}

type CreatedRoom struct {
	RoomId string `json:"room_id"`
}

type NewRoom struct {
	RoomName string `json:"room_name"`
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	user, err := musicclient.CreateUserFromRequest(r)

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

	room := app.CreateRoom(roomId, roomName, user)

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
	case http.MethodDelete:
		DeleteRoom(w, r)
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

	roomWithOwnerInfo := app.RoomWithOwnerInfo{
		Room:    room,
		IsOwner: room.IsOwner(user),
	}

	httputils.SendJson(w, roomWithOwnerInfo)
}

func DeleteRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to delete room %s", user.GetUserId(), roomId)

	// If room has been processed, we delete it in mongo
	if room.HasRoomBeenProcessed() {
		err = mongoclientapp.DeleteRoomForUser(room, user)

		if err != nil {
			handleError(failedToDeleteRoom, w, r, user)
			return
		}
	}

	deleteRoomNotProcessed(room)

	httputils.SendOk(w)
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
	user, err := musicclient.CreateUserFromRequest(r)

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
		return
	}

	if *room.Locked {
		handleError(roomLockedError, w, r, user)
		return
	}

	room.AddUser(user)

	httputils.SendOk(w)
}
