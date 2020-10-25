package app

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"math/rand"
	"net/http"
	"strconv"
)

const maxNumberRooms = 100

var roomDoesNotExistError = errors.New("room does not exists")
var roomIsNotAccessibleError = errors.New("room is not accessible to user")
var authenticationError = errors.New("failed to authenticate user")
var roomLockedError = errors.New("room is locked and not accepting new members")
var processingInProgressError = errors.New("processing of music is already in progress")
var processingNotStartedError = errors.New("processing of music has not been done, cannot get playlists")
var processingFailedError = errors.New("processing of music failed, cannot get playlists")
var failedToCreatePlaylistError = errors.New("an error occurred while creating the playlist")


// An in memory representation of all the rooms, would be better if it was persistent but for now this is fine
var allRooms = AllRooms{make(map[string]*Room, 0)}

type AllRooms struct {
	Rooms map[string]*Room `json:"rooms"`
}

type Room struct {
	Id            string                 `json:"id"`
	Users         []*spotifyclient.User  `json:"users"`
	Locked        *bool                  `json:"locked"`
	MusicLibrary  *SharedMusicLibrary    `json:"shared_music_library"`

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

	} else if err == roomLockedError {
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

	case http.MethodPost:
		AddRoomUser(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
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

	if room.isUserInRoom(user) {
		// if user is already in room, just send ok
		httputils.SendOk(w)
	}

	if *room.Locked {
		handleError(roomLockedError, w)
		return
	}

	room.addUser(user)

	httputils.SendOk(w)
}

/*
  Room playlist handler
*/

func RoomPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		GetPlaylistForRoom(w, r)
	case http.MethodPost:
		FindPlaylistsForRoom(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func GetPlaylistForRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w)
		return
	}

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.hasProcessingFinished() {
		handleError(processingInProgressError, w)
		return
	}

	if musicLibrary.hasProcessingFailed() {
		handleError(processingFailedError, w)
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
		handleError(err, w)
		return
	}

	if room.MusicLibrary != nil && !room.MusicLibrary.hasProcessingFailed() {
		handleError(processingInProgressError, w)
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
  Room ADD playlist handler
*/

func RoomAddPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		AddPlaylistsForUser(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func AddPlaylistsForUser(w http.ResponseWriter, r *http.Request)  {
	user, err := spotifyclient.CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w)
		return
	}

	vars := mux.Vars(r)
	roomId := vars["roomId"]

	room, err := getRoomAndCheckUser(roomId, r)

	if err != nil {
		handleError(err, w)
		return
	}

	musicLibrary := room.MusicLibrary

	if musicLibrary == nil {
		handleError(processingNotStartedError, w)
		return
	}

	// check the processing is over and it did not fail
	if !musicLibrary.hasProcessingFinished() {
		handleError(processingInProgressError, w)
		return
	}

	if musicLibrary.hasProcessingFailed() {
		handleError(processingFailedError, w)
		return
	}

	// we add the playlist
	newPlaylist := CreateNewPlaylist(roomId)
	tracks := room.MusicLibrary.GetPlaylist()

	spotifyUrl, err := user.CreatePlaylist(newPlaylist.Name, tracks)

	if spotifyUrl != nil {
		newPlaylist.SpotifyUrl = *spotifyUrl
	}

	if err != nil {
		handleError(failedToCreatePlaylistError, w)
		return
	}

	httputils.SendJson(w, newPlaylist)
}

type NewPlaylist struct {
	Name        string `json:"name"`
	SpotifyUrl  string `json:"spotify_url"`
}

func CreateNewPlaylist(roomId string) *NewPlaylist {
	playlistName := fmt.Sprintf("Shared Spotify - Room #%s", roomId)
	return &NewPlaylist{playlistName, ""}
}