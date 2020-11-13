package app

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/appmodels"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
	"net/http"
)

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

	playlists := room.MusicLibrary.CommonPlaylists.GetPlaylistsMetadata()
	httputils.SendJson(w, playlists)
}

// Here, we launch the process of finding the musics for the users in the room
func FindPlaylistsForRoom(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to find the playlists for room %s",
		user.GetUserId(), roomId)

	if room.MusicLibrary != nil && !room.MusicLibrary.HasProcessingFailed() {
		handleError(processingInProgressError, w, r, user)
		return
	}

	// we lock the room, so no one should be able to enter it now
	*room.Locked = true

	// we create the music library
	room.MusicLibrary = appmodels.CreateSharedMusicLibrary(len(room.Users))

	// we now process the library of the users (all this is done async)
	logger.Logger.Infof("Starting processing of room %s for users %s", roomId, room.GetUserIds())
	room.MusicLibrary.Process(room.Users, func(success bool) {
		updateRoomNotProcessed(room, success) // callback function
	})

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

func GetPlaylist(w http.ResponseWriter, r *http.Request)  {
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

	playlistType, err := room.MusicLibrary.GetPlaylist(playlistId)

	if err != nil {
		logger.Logger.Error("PlaylistMetadata %s was not found for room %s, user is %s",
			playlistId, roomId, user.GetUserId())
		handleError(appmodels.ErrorPlaylistTypeNotFound, w, r, user)
		return
	}

	httputils.SendJson(w, playlistType)
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
	MinSharedCount int `json:"min_shared_count"`
}

func AddPlaylistForUser(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	roomId := vars["roomId"]
	playlistId := vars["playlistId"]

	room, user, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r, user)
		return
	}

	var addPlaylistRequestBody AddPlaylistRequestBody
	err = httputils.DeserialiseBody(r, &addPlaylistRequestBody)

	if err != nil {
		logger.Logger.Error("Failed to decode json body for add playlist for user")
		handleError(err, w, r, user)
		return
	}

	logger.WithUser(user.GetUserId()).Infof("User %s requested to create playlist %s for room %s",
		user.GetUserId(), playlistId, roomId)

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
		logger.Logger.Error("PlaylistMetadata %s was not found for room %s, user is %s",
			playlistId, roomId, user.GetUserId())
		handleError(appmodels.ErrorPlaylistTypeNotFound, w, r, user)
		return
	}

	// we create in spotify the playlist
	newPlaylist := CreateNewPlaylist(room.Name, playlist.Name)

	// we get the songs that are above the min shared count limit requested by the user
	tracks := make([]*spotify.FullTrack, 0)

	for sharedCount, sharedTracks := range playlist.TracksPerSharedCount {
		if sharedCount >= addPlaylistRequestBody.MinSharedCount {
			tracks = append(tracks, sharedTracks...)
		}
	}

	spotifyUrl, err := user.CreatePlaylist(newPlaylist.Name, tracks)

	if spotifyUrl != nil {
		newPlaylist.SpotifyUrl = *spotifyUrl
	}

	if err != nil {
		handleError(failedToCreatePlaylistError, w, r, user)
		return
	}

	httputils.SendJson(w, newPlaylist)
}

type NewPlaylist struct {
	Name        string `json:"name"`
	SpotifyUrl  string `json:"spotify_url"`
}

func CreateNewPlaylist(roomName string, playlistName string) *NewPlaylist {
	spotifyPlaylistName := fmt.Sprintf("%s - %s by Shared Spotify", roomName, playlistName)
	return &NewPlaylist{spotifyPlaylistName, ""}
}