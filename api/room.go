package api

import (
	"context"
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
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
)

const defaultRoomName = "Room #%s"

var failedToGetRoom = errors.New("Failed to get room")
var failedToDeleteRoom = errors.New("Failed to delete room")
var failedToGetRooms = errors.New("Failed to get rooms")
var roomDoesNotExistError = errors.New("Room does not exists")
var roomIsNotAccessibleError = errors.New("Room is not accessible to user")
var failedToCreateRoom = errors.New("Failed to create room")
var failedToAddUserToRoom = errors.New("Failed to add user to room")
var authenticationError = errors.New("Failed to authenticate user")
var roomLockedError = errors.New("Room is locked and not accepting new members. Create a new one to share music")
var processingLaunchError = errors.New("Failed to launch processing")
var processingInProgressError = errors.New("Processing of music is already in progress")
var processingNotStartedError = errors.New("Processing of music has not been done, cannot get playlists")
var processingFailedError = errors.New("Processing of music failed, cannot get playlists")
var roomExpiredError = errors.New("Room has expired because some users are no longer connected to their music " +
	"provider, create a new room to retry")
var failedToCreatePlaylistError = errors.New("An error occurred while creating the playlist")

func addRoomNotProcessed(room *app.Room) error {
	datadog.Increment(1, datadog.RoomCount,
		datadog.RoomIdTag.Tag(room.Id),
		datadog.RoomNameTag.Tag(room.Name),
	)

	return mongoclientapp.UpdateUnprocessedRoom(room, nil)
}

// this function should run in a go routine only, so it should be fine to make it panic
func updateRoomNotProcessed(room *app.Room, success bool, ctx context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.unprocessed.update.callback")
	defer span.Finish()

	// we set processing result
	room.MusicLibrary.SetProcessingSuccess(&success)

	// send time taken to process the room
	datadog.Distribution(room.MusicLibrary.GetProcessingTime(), datadog.RoomProcessedTime,
		datadog.RoomIdTag.Tag(room.Id),
		datadog.RoomNameTag.Tag(room.Name),
	)

	if !success {
		datadog.Increment(1, datadog.RoomProcessedFailed,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)
		err := updateRoomWithCtx(room, ctx)

		if err != nil {
			span.Finish(tracer.WithError(err))
			logger.Logger.Errorf("Could not update mongo for finished processed room! this is bad, " +
				"we need to make sure we recover properly for the room %v %v", err, span)
		}

		return
	}

	// we insert the room result in mongo
	err := mongoclientapp.InsertRoom(room, ctx)

	if err != nil {
		// if we fail to insert the result in mongo, we declare processing as failed
		success := false
		room.MusicLibrary.SetProcessingSuccess(&success)
		datadog.Increment(1, datadog.RoomProcessedFailed,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)
		err := updateRoomWithCtx(room, ctx)

		if err != nil {
			span.Finish(tracer.WithError(err))
			logger.Logger.Errorf("Could not update mongo for finished processed room! this is bad, " +
				"we need to make sure we recover properly for the room %v %v", err, span)
		}

	} else {
		// TODO: handle the case where we fail to delete the mongo room
		// otherwise we delete the room from the rooms being processed
		_ = deleteRoomNotProcessed(room, ctx)

		datadog.Increment(1, datadog.RoomProcessedCount,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
		)
	}
}

func updateRoom(room *app.Room) error {
	return updateRoomWithCtx(room, nil)
}

func updateRoomWithCtx(room *app.Room, ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.unprocessed.update")
	defer span.Finish()

	err := mongoclientapp.UpdateUnprocessedRoom(room, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to set processing status for room %s %v %v", room.Id, err, span)
	}

	return err
}

func deleteRoomNotProcessed(room *app.Room, ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.unprocessed.delete.unprocessed")
	defer span.Finish()

	err := mongoclientapp.DeleteUnprocessedRoom(room.Id, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to delete unprocessed room %s %v %v", room.Id, err, span)
	}

	return err
}

func getRoom(roomId string, ctx context.Context) (*app.Room, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.get")
	defer span.Finish()

	unprocessedRoom, unprocessedRoomErr := mongoclientapp.GetUnprocessedRoom(roomId, ctx)
	room, roomErr := mongoclientapp.GetRoom(roomId, ctx)

	// we make a check to see if do not have the same room unprocessed and processed
	if unprocessedRoom != nil && room != nil {
		logger.WithRoom(roomId).Warningf("Found unprocessed room with processed room, deleting unprocessed one %v", span)

		// remove unprocessed room that was not removed before
		err := deleteRoomNotProcessed(unprocessedRoom, ctx)

		if err != nil {
			logger.WithRoom(roomId).Errorf("Failed to delete duplicate rooms unprocessed and processed %v %v", err, span)
			span.Finish(tracer.WithError(err))
			return nil, failedToGetRoom
		}

		logger.WithRoom(roomId).Warningf("Successfully deleted unprocessed room that had processed room %v", span)

		return room, nil
	}

	if unprocessedRoom != nil {
		return unprocessedRoom, nil
	}

	if room != nil {
		return room, nil
	}

	if unprocessedRoomErr == mongoclientapp.NotFound && roomErr == mongoclientapp.NotFound {
		return nil, roomDoesNotExistError
	}

	if unprocessedRoomErr != nil {
		span.Finish(tracer.WithError(unprocessedRoomErr))
		logger.WithRoom(roomId).Errorf("Failed to query unprocessed rooms %v %v", unprocessedRoomErr, span)
	}

	if roomErr != nil {
		span.Finish(tracer.WithError(roomErr))
		logger.WithRoom(roomId).Errorf("Failed to query rooms %v %v", roomErr, span)
	}

	return nil, failedToGetRoom
}

func getRoomAndCheckUser(roomId string, r *http.Request) (*app.Room, *clientcommon.User, error) {
	return getRoomAndCheckUserWithCtx(roomId, r, r.Context())
}

func getRoomAndCheckUserWithCtx(roomId string, r *http.Request, ctx context.Context) (*app.Room, *clientcommon.User, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.get.check_user")
	defer span.Finish()

	room, err := getRoom(roomId, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		return nil, nil, err
	}

	user, err := musicclient.CreateUserFromRequestWithCtx(r, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		return nil, nil, authenticationError
	}

	if !room.IsUserInRoom(user) {
		span.Finish(tracer.WithError(roomIsNotAccessibleError))
		return nil, user, roomIsNotAccessibleError
	}

	return room, user, nil
}

func handleError(err error, w http.ResponseWriter, r *http.Request, user *clientcommon.User) {
	userId := "unknown"

	if user != nil {
		userId = user.GetUserId()
	}

	logger.
		WithUser(userId).
		WithError(err).
		Errorf("Handling error")

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
	span, ctx := tracer.StartSpanFromContext(r.Context(), "rooms.get")
	user, err := musicclient.CreateUserFromRequestWithCtx(r, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(authenticationError, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to get rooms", user.GetUserId())

	rooms, err := mongoclientapp.GetRoomsForUser(user, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(failedToGetRooms, w, r, user)
		return
	}

	// we add to these rooms the not processed yet rooms
	unprocessedRooms, err := mongoclientapp.GetUnprocessedRoomsForUser(user, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(failedToGetRooms, w, r, user)
		return
	}

	rooms = append(rooms, unprocessedRooms...)

	httputils.SendJsonWithCtx(w, &rooms, ctx)
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

	err = addRoomNotProcessed(room)

	if err != nil {
		logger.Logger.Errorf("Failed to create room %s %v", roomId, err)
		handleError(failedToCreateRoom, w, r, user)
		return
	}

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

	// check if room has not been processing without result for too long
	if room.HasProcessingTimedOut() {
		logger.WithUser(user.GetUserId()).Warningf("Processing timed out for room %s, reset the room", roomId)

		// if so, reset the library and update it in mongo, so we can trigger a new processing
		room.ResetMusicLibrary()
		err = updateRoom(room)

		if err != nil {
			handleError(failedToGetRoom, w, r, user)
			return
		}
	}

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

	err = nil

	if room.HasRoomBeenProcessed() {
		err = mongoclientapp.DeleteRoomForUser(room, user)

	} else {
		// TODO: not ideal, if room is not processed and deleted, it is deleted for ALL users
		err = deleteRoomNotProcessed(room, nil)
	}

	if err != nil {
		handleError(failedToDeleteRoom, w, r, user)
		return
	}

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
	span, ctx := tracer.StartSpanFromContext(r.Context(), "room.add.user")
	defer span.Finish()

	user, err := musicclient.CreateUserFromRequestWithCtx(r, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(authenticationError, w, r, user)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoom(roomId, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
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
		span.Finish(tracer.WithError(roomLockedError))
		handleError(roomLockedError, w, r, user)
		return
	}

	room.AddUser(user)

	err = updateRoom(room)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(failedToAddUserToRoom, w, r, user)
		return
	}

	httputils.SendOk(w)
}
