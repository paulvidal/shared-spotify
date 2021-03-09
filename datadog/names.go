package datadog

import "fmt"

type Tag struct {
	Key string
}

func (t Tag) Tag(value string) string {
	return fmt.Sprintf("%s:%s", t.Key, value)
}

func (t Tag) TagBool(value bool) string {
	return fmt.Sprintf("%s:%t", t.Key, value)
}

// For users
const UserLoginStarted = "user.login.started"
const UserLoginSuccess = "user.login.success"
const UserLogout = "user.logout"
const UsersNewCount = "users.new.count"

var UserIdTag = Tag{"user_id"}

// For rooms
const RoomCount = "rooms.new.count"
const RoomProcessedCount = "rooms.processed.count"
const RoomProcessedFailed = "rooms.processed.failed"
const TrackForRoom = "rooms.tracks.common.count"
const RoomUsers = "rooms.users.count"
const RoomPlaylistAllRequest = "rooms.playlist.all.request"
const RoomPlaylistRequest = "rooms.playlist.request"
const RoomPlaylistAdd = "rooms.playlist.add"

var RoomIdTag = Tag{"room_id"}
var RoomNameTag = Tag{"room_name"}
var PlaylistTypeTag = Tag{"playlist_type"}

// For api requests
const ApiRequests = "api.requests"

const SpotifyProvider = "spotify"
const AppleMusicProvider = "applemusic"

var Provider = Tag{"provider"}
var RequestType = Tag{"request_type"}
var Authenticated = Tag{"authenticated"}
var Success = Tag{"success"}

const RequestTypeAuth = "auth"
const RequestTypeUserInfo = "user_info"
const RequestTypeSavedSongs = "saved_songs"
const RequestTypePlaylistSongs = "playlist_songs"
const RequestTypePlaylists = "playlists"
const RequestTypeSongs = "songs"
const RequestTypeArtists = "artists"
const RequestTypeAlbums = "albums"
const RequestTypeAudioFeatures = "audio_features"
const RequestTypeSearch = "search"
const RequestTypePlaylistCreated = "playlist_created"
const RequestTypePlaylistSongsAdded = "playlist_songs_added"
