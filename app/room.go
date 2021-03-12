package app

import (
	"github.com/shared-spotify/musicclient/clientcommon"
	"time"
)

type Room struct {
	Id           string               `json:"id" bson:"_id"`
	Name         string               `json:"name"`
	Owner        *clientcommon.User   `json:"owner"`
	Users        []*clientcommon.User `json:"users"`
	CreationTime time.Time            `json:"creation_time"`
	Locked       *bool                `json:"locked"`
	MusicLibrary *SharedMusicLibrary  `json:"shared_music_library"`
}

type RoomWithOwnerInfo struct {
	*Room
	IsOwner bool `json:"is_owner"`
}

func CreateRoom(roomId string, roomName string, owner *clientcommon.User) *Room {
	locked := false
	room := &Room{
		roomId,
		roomName,
		owner,
		make([]*clientcommon.User, 0),
		time.Now(),
		&locked,
		nil,
	}

	// Add the owner to the room
	room.AddUser(owner)

	return room
}

func (room *Room) AddUser(user *clientcommon.User) {
	// If the user is already in the room, do not add it
	if room.IsUserInRoom(user) {
		return
	}

	room.Users = append(room.Users, user)
}

func (room *Room) IsUserInRoom(user *clientcommon.User) bool {
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

func (room *Room) IsOwner(user *clientcommon.User) bool {
	return room.Owner.IsEqual(user)
}

func (room *Room) HasRoomBeenProcessed() bool {
	return room.MusicLibrary != nil && room.MusicLibrary.HasProcessingFinished()
}

func (room *Room) HasProcessingTimedOut() bool{
	return room.MusicLibrary != nil && room.MusicLibrary.HasTimedOut()
}

func (room *Room) GetPlaylists() map[string]*Playlist {
	return room.MusicLibrary.CommonPlaylists.Playlists
}

func (room *Room) SetPlaylists(playlists map[string]*Playlist) {
	room.MusicLibrary.CommonPlaylists = &CommonPlaylists{Playlists: playlists}
}

func (room *Room) ResetMusicLibrary() {
	room.MusicLibrary = nil
}