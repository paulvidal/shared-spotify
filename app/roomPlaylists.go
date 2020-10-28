package app

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
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

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r)
		return
	}

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w, r)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.hasProcessingFinished() {
		handleError(processingInProgressError, w, r)
		return
	}

	if musicLibrary.hasProcessingFailed() {
		handleError(processingFailedError, w, r)
		return
	}

	httputils.SendJson(w, room.MusicLibrary.CommonPlaylists)
}

// Here, we launch the process of finding the musics for the users in the room
func FindPlaylistsForRoom(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r)
		return
	}

	if room.MusicLibrary != nil && !room.MusicLibrary.hasProcessingFailed() {
		handleError(processingInProgressError, w, r)
		return
	}

	// we lock the room, so no one should be able to enter it now
	*room.Locked = true

	// we create the music library
	room.MusicLibrary = CreateSharedMusicLibrary(len(room.Users))

	// we now process the library of the users (all this is done async)
	logger.Logger.Infof("Starting processing of room %s for users %s", roomId, room.getUserIds())
	room.MusicLibrary.Process(room.Users)

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
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w, r)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]
	playlistId := vars["playlistId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r)
		return
	}

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w, r)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.hasProcessingFinished() {
		handleError(processingInProgressError, w, r)
		return
	}

	if musicLibrary.hasProcessingFailed() {
		handleError(processingFailedError, w, r)
		return
	}

	playlist, err := room.MusicLibrary.GetPlaylist(playlistId)

	if err != nil {
		logger.Logger.Error("Playlist %s was not found for room %s, user is %s",
			playlistId, roomId, user.GetUserId())
		handleError(errorPlaylistNotFound, w, r)
		return
	}

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

func AddPlaylistForUser(w http.ResponseWriter, r *http.Request)  {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w, r)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]
	playlistId := vars["playlistId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w, r)
		return
	}

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w, r)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.hasProcessingFinished() {
		handleError(processingInProgressError, w, r)
		return
	}

	if musicLibrary.hasProcessingFailed() {
		handleError(processingFailedError, w, r)
		return
	}

	playlist, err := room.MusicLibrary.GetPlaylist(playlistId)

	if err != nil {
		logger.Logger.Error("Playlist %s was not found for room %s, user is %s",
			playlistId, roomId, user.GetUserId())
		handleError(errorPlaylistNotFound, w, r)
		return
	}

	// we create in spotify the playlist
	newPlaylist := CreateNewPlaylist(roomId, playlist.Name)
	spotifyUrl, err := user.CreatePlaylist(newPlaylist.Name, playlist.Tracks)

	if spotifyUrl != nil {
		newPlaylist.SpotifyUrl = *spotifyUrl
	}

	if err != nil {
		handleError(failedToCreatePlaylistError, w, r)
		return
	}

	httputils.SendJson(w, newPlaylist)
}

type NewPlaylist struct {
	Name        string `json:"name"`
	SpotifyUrl  string `json:"spotify_url"`
}

func CreateNewPlaylist(roomId string, playlistName string) *NewPlaylist {
	spotifyPlaylistName := fmt.Sprintf("Shared Spotify - Room #%s - %s", roomId, playlistName)
	return &NewPlaylist{spotifyPlaylistName, ""}
}