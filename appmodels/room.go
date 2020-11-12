package appmodels

import (
	"github.com/shared-spotify/spotifyclient"
	"time"
)

type Room struct {
	Id           string                `json:"id" bson:"_id"`
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

func CreateRoom(roomId string, roomName string, owner *spotifyclient.User) *Room {
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
	room.AddUser(owner)

	return room
}

func (room *Room) AddUser(user *spotifyclient.User) {
	// If the user is already in the room, do not add it
	if room.IsUserInRoom(user) {
		return
	}

	room.Users = append(room.Users, user)
}

func (room *Room) IsUserInRoom(user *spotifyclient.User) bool {
	for _, roomUser := range room.Users {
		if roomUser.IsEqual(user) {
			return true
		}
	}

	return false
}

func (room *Room) GetUserIds() []string {
	userNames := make([]string, 0)
	for _, user := range room.Users {
		userNames = append(userNames, user.Id)
	}
	return userNames
}

func (room *Room) IsOwner(user *spotifyclient.User) bool {
	return room.Owner.IsEqual(user)
}

func (room *Room) GetPlaylists() map[string]*Playlist {
	return room.MusicLibrary.CommonPlaylists.Playlists
}

func (room *Room) SetPlaylists(playlists map[string]*Playlist) {
	room.MusicLibrary.CommonPlaylists = &CommonPlaylists{Playlists: playlists}
}