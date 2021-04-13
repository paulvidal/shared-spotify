package api

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/thoas/go-funk"
	"github.com/zmb3/spotify"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
	"time"
)

// perform all shutdown operations here
func Shutdown()  {
	// Cancel all processing here
	app.CancelAll()
}

/*
  Room playlists handler
*/

func RoomPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetPlaylistsForRoom(w, r)
	case http.MethodPost:
		FindPlaylistsForRoom(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetPlaylistsForRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested playlist for room %s", user.GetUserId(), roomId)

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w, r, user)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.HasProcessingFinished() {
		handleError(processingInProgressError, w, r, user)
		return
	}

	if musicLibrary.HasProcessingFailed() {
		handleError(processingFailedError, w, r, user)
		return
	}

	datadog.Increment(1, datadog.RoomPlaylistAllRequest,
		datadog.UserIdTag.Tag(user.GetId()),
		datadog.RoomIdTag.Tag(roomId),
		datadog.RoomNameTag.Tag(room.Name),
	)

	playlists := room.MusicLibrary.CommonPlaylists.GetPlaylistsMetadata()
	httputils.SendJson(w, playlists)
}

// Here, we launch the process of finding the musics for the users in the room
func FindPlaylistsForRoom(w http.ResponseWriter, r *http.Request) {
	span, ctx := tracer.StartSpanFromContext(r.Context(), "playlist.find.for.room")
	defer span.Finish()

	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, user, err := getRoomAndCheckUserWithCtx(roomId, r, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(err, w, r, user)
		return
	}

	if room.IsExpired(ctx) {
		datadog.Increment(1, datadog.RoomExpired,
			datadog.UserIdTag.Tag(user.GetId()),
			datadog.RoomIdTag.Tag(roomId),
			datadog.RoomNameTag.Tag(room.Name),
		)
		logger.WithUser(user.GetUserId()).Errorf("Room %s declared as expired %v %v", roomId, err, span)
		span.Finish(tracer.WithError(roomExpiredError))
		handleError(roomExpiredError, w, r, user)
		return
	}

	// we re-create all the clients for the room
	err = room.RecreateClients(ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.WithUser(user.GetUserId()).Errorf("Failed to recreate clients when fetching common musics %v %v", err, span)
		handleError(roomExpiredError, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to find the playlists for room %s %v",
		user.GetUserId(), roomId, span)

	if room.MusicLibrary != nil && !room.MusicLibrary.HasProcessingFailed() {
		span.Finish(tracer.WithError(processingInProgressError))
		handleError(processingInProgressError, w, r, user)
		return
	}

	// we lock the room, so no one should be able to enter it now
	*room.Locked = true

	// we create the music library
	room.MusicLibrary = app.CreateSharedMusicLibrary(len(room.Users))

	err = updateRoomWithCtx(room, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.WithUser(user.GetUserId()).Errorf("Failed to update room %s %v %v", roomId, err, span)
		handleError(processingLaunchError, w, r, user)
		return
	}

	// we now process the library of the users (all this is done async)
	logger.Logger.Infof("Starting processing of room %s for users %s %v", roomId, room.GetUserIds(), span)

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), app.TimeoutRoomProcessing)
	processingSpan, ctxWithTimeout := tracer.StartSpanFromContext(ctxWithTimeout, "room.process")

	err = room.MusicLibrary.Process(room, func(success bool, ctx context.Context) {
		// callback function
		updateRoomNotProcessed(room, success, ctx)

		// remove the cancel as room is done processing
		app.RemoveCancel(roomId)

		// close the span
		processingSpan.Finish()

	}, func(ctx context.Context) error {
		// we update the last time checkpoint
		room.MusicLibrary.ProcessingStatus.CheckpointTime = time.Now()

		return updateRoomWithCtx(room, ctx)
	}, ctxWithTimeout)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.WithUser(user.GetUserId()).Errorf("Failed to launch processing %s %v", roomId, err)
		handleError(processingLaunchError, w, r, user)
		return
	}

	// add the cancel function for the room in case we need to shutdown processing
	app.AddCancel(roomId, cancel)

	httputils.SendOk(w)
}

/*
  Room playlist handler
*/

func RoomPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetPlaylist(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]
	playlistId := vars["playlistId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested playlist %s for room %s", user.GetUserId(),
		playlistId, roomId)

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w, r, user)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.HasProcessingFinished() {
		handleError(processingInProgressError, w, r, user)
		return
	}

	if musicLibrary.HasProcessingFailed() {
		handleError(processingFailedError, w, r, user)
		return
	}

	playlist, err := room.MusicLibrary.GetPlaylist(playlistId)

	if err != nil {
		logger.Logger.Error("Playlist %s was not found for room %s, user is %s",
			playlistId, roomId, user.GetUserId())
		handleError(app.ErrorPlaylistTypeNotFound, w, r, user)
		return
	}

	datadog.Increment(1, datadog.RoomPlaylistRequest,
		datadog.UserIdTag.Tag(user.GetId()),
		datadog.RoomIdTag.Tag(roomId),
		datadog.RoomNameTag.Tag(room.Name),
		datadog.PlaylistTypeTag.Tag(playlist.Type),
	)

	httputils.SendJson(w, playlist)
}

/*
  Room ADD playlist handler
*/

func RoomAddPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		AddPlaylistForUser(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

type AddPlaylistRequestBody struct {
	SharedUserCount []int `json:"shared_user_count"`
}

func AddPlaylistForUser(w http.ResponseWriter, r *http.Request) {
	span, ctx := tracer.StartSpanFromContext(r.Context(), "playlist.add.for.user")
	defer span.Finish()

	vars := mux.Vars(r)
	roomId := vars["roomId"]
	playlistId := vars["playlistId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		span.Finish(tracer.WithError(err))
		handleError(err, w, r, user)
		return
	}

	var addPlaylistRequestBody AddPlaylistRequestBody
	err = httputils.DeserialiseBody(r, &addPlaylistRequestBody)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.
			WithUserAndRoom(user.GetUserId(), roomId).
			WithError(err).
			Errorf("Failed to decode json body for add playlist for user %v", span)
		handleError(err, w, r, user)
		return
	}

	logger.
		WithUserAndRoom(user.GetUserId(), roomId).
		Infof("User requested to create playlist %s for room with user count songs %v %v",
			playlistId, addPlaylistRequestBody.SharedUserCount, span)

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		span.Finish(tracer.WithError(processingNotStartedError))
		handleError(processingNotStartedError, w, r, user)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.HasProcessingFinished() {
		span.Finish(tracer.WithError(processingInProgressError))
		handleError(processingInProgressError, w, r, user)
		return
	}

	if musicLibrary.HasProcessingFailed() {
		span.Finish(tracer.WithError(processingFailedError))
		handleError(processingFailedError, w, r, user)
		return
	}

	playlist, err := room.MusicLibrary.GetPlaylist(playlistId)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.
			WithUserAndRoom(user.GetUserId(), roomId).
			WithError(err).
			Errorf("Playlist %s was not found for room %v", playlistId, span)
		handleError(app.ErrorPlaylistTypeNotFound, w, r, user)
		return
	}

	// we create in spotify the playlist
	newPlaylist := CreateNewPlaylist(room.Name, playlist.Name)

	// we get the songs that are above the min shared count limit requested by the user
	tracks := make([]*spotify.FullTrack, 0)

	for sharedCount, sharedTracks := range playlist.TracksPerSharedCount {
		if funk.ContainsInt(addPlaylistRequestBody.SharedUserCount, sharedCount) {
			tracks = append(tracks, sharedTracks...)
		}
	}

	playlistUrl, err := musicclient.CreatePlaylist(user, newPlaylist.Name, tracks, ctx)

	if playlistUrl != nil {
		// TODO: change field name
		newPlaylist.SpotifyUrl = *playlistUrl
	}

	tags := []string{
		datadog.UserIdTag.Tag(user.GetId()),
		datadog.RoomIdTag.Tag(roomId),
		datadog.RoomNameTag.Tag(room.Name),
		datadog.PlaylistTypeTag.Tag(playlist.Type),
	}

	if err != nil {
		span.Finish(tracer.WithError(failedToCreatePlaylistError))
		tags = append(tags, datadog.Success.TagBool(false))
		datadog.Increment(1, datadog.RoomPlaylistAdd, tags...)
		handleError(failedToCreatePlaylistError, w, r, user)
		return
	}

	tags = append(tags, datadog.Success.TagBool(true))
	datadog.Increment(1, datadog.RoomPlaylistAdd, tags...)

	logger.
		WithUserAndRoom(user.GetUserId(), roomId).
		Infof("User created successfully his playlist %s for room", playlistId)

	httputils.SendJson(w, newPlaylist)
}

type NewPlaylist struct {
	Name       string `json:"name"`
	SpotifyUrl string `json:"spotify_url"`
}

func CreateNewPlaylist(roomName string, playlistName string) *NewPlaylist {
	spotifyPlaylistName := fmt.Sprintf("%s - %s %s", roomName, playlistName, clientcommon.NameCredits)
	return &NewPlaylist{spotifyPlaylistName, ""}
}
