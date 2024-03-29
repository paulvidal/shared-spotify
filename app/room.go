package app

import (
	"context"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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

func (room *Room) HasRoomBeenProcessedSuccessfully() bool {
	return room.MusicLibrary != nil && room.MusicLibrary.HasProcessingSucceeded()
}

func (room *Room) HasProcessingTimedOut() bool {
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

func (room *Room) RecreateClients(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "clients.recreate")
	defer span.Finish()

	owner, err := recreateUserWithClient(room.Owner)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to recreate client for owner %v %v", err, span)
		return err
	}

	room.Owner = owner

	usersWithClients := make([]*clientcommon.User, 0)
	users := room.Users

	for _, user := range users {
		newUser, err := recreateUserWithClient(user)

		if err != nil {
			span.Finish(tracer.WithError(err))
			logger.Logger.Errorf("Failed to recreate client for user %v %v", err, span)
			return err
		}

		usersWithClients = append(usersWithClients, newUser)
	}

	room.Users = usersWithClients

	return nil
}

func recreateUserWithClient(user *clientcommon.User) (*clientcommon.User, error) {
	loginType := user.LoginType
	token := user.Token

	return musicclient.CreateUserFromToken(token, loginType, nil)
}

// checks if a room can still be processed, by checking if every user in the room can have a client created for them
// if a client cannot be created, it means the user must have revoqued its token
func (room *Room) IsExpired(ctx context.Context) bool {
	span, ctx := tracer.StartSpanFromContext(ctx, "room.is_expired")
	defer span.Finish()

	if room.HasRoomBeenProcessedSuccessfully() {
		span.SetTag("expired", false)
		return false
	}

	for _, user := range room.Users {
		_, err := musicclient.CreateUserFromToken(user.Token, user.LoginType, nil)

		if err != nil {
			span.SetTag("expired", true)
			span.Finish(tracer.WithError(err))
			return true
		}
	}

	span.SetTag("expired", false)
	return false
}

/**
  Room processing
 */

var cancels = make(map[string]context.CancelFunc)

func AddCancel(roomId string, cancel context.CancelFunc) {
	cancels[roomId] = cancel
}

func RemoveCancel(roomId string) {
	delete(cancels, roomId)
}

func CancelAll() {
	for roomId, cancel := range cancels {
		logger.WithRoom(roomId).Warning("Cancelling processing for room")
		cancel()
	}

	// reinitialise the map to be sure we never call cancel twice
	cancels = make(map[string]context.CancelFunc)
}