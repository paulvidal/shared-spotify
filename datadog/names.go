package datadog

import "fmt"

type Tag struct {
	Key string
}

func (t Tag) Tag(value string) string {
	return fmt.Sprintf("%s:%s", t.Key, value)
}

// For users
const UsersNewCount = "users.new.count"

var UserIdTag = Tag{"user_id"}

// For rooms
const RoomCount = "rooms.new.count"
const RoomProcessedCount = "rooms.processed.count"
const RoomProcessedFailed = "rooms.processed.failed"
const TrackForRoom = "rooms.tracks.common.count"
const RoomUsers = "rooms.users.count"

var RoomIdTag = Tag{"room_id"}
var PlaylistTypeTag = Tag{"playlist_type"}