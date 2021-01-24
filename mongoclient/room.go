package mongoclient

import (
	"context"
	"errors"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	spotifyclient "github.com/shared-spotify/musicclient/spotify"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const roomCollection = "rooms"

var NotFound = errors.New("Not found")

type MongoRoom struct {
	*app.Room `bson:"inline"`
	Playlists map[string]*MongoPlaylist `bson:"playlists"`
}

type MongoPlaylist struct {
	app.PlaylistMetadata   `bson:"inline"`
	TrackIdsPerSharedCount map[int][]string              `bson:"track_ids_per_shared_count"`
	UserIdsPerSharedTracks map[string][]string           `bson:"user_ids_per_shared_tracks"`
	Users                  map[string]*clientcommon.User `bson:"users"`
}

func InsertRoom(room *app.Room) error {
	playlists := room.GetPlaylists()

	// we insert the users
	err := InsertUsers(room.Users)

	newUserCount := len(room.Users)
	datadog.Increment(newUserCount, datadog.RoomUsers,
		datadog.RoomIdTag.Tag(room.Id),
		datadog.RoomNameTag.Tag(room.Name),
	)

	if err != nil {
		return err
	}

	// we insert the tracks
	tracks := getAllTracksForPlaylists(playlists)
	err = InsertTracks(tracks)

	if err != nil {
		return err
	}

	// we insert the room
	mongoPlaylists := convertPlaylistsToMongoPlaylists(playlists, room)

	mongoRoom := MongoRoom{
		room,
		mongoPlaylists,
	}

	insertResult, err := getDatabase().Collection(roomCollection).InsertOne(context.TODO(), mongoRoom)

	if err != nil {
		logger.Logger.Error("Failed to insert room in mongo ", err)
		return err
	}

	logger.Logger.Info("Room was inserted successfully in mongo ", insertResult.InsertedID)

	return nil
}

func GetRoom(roomId string) (*app.Room, error) {
	var mongoRoom MongoRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := getDatabase().Collection(roomCollection).FindOne(context.TODO(), filter).Decode(&mongoRoom)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, NotFound
		}

		logger.Logger.Error("Failed to find room in mongo ", err)
	}

	room := mongoRoom.Room

	// we then form back the playlists and recreate the room
	playlists, err := convertMongoPlaylistsToPlaylists(mongoRoom.Playlists)

	if err != nil {
		return nil, err
	}

	room.SetPlaylists(playlists)

	return room, err
}

func GetRoomsForUser(user *clientcommon.User) ([]*app.Room, error) {
	mongoRooms := make([]*MongoRoom, 0)
	rooms := make([]*app.Room, 0)

	filter := bson.D{{
		"users._id",
		user.GetId(),
	}}

	cursor, err := getDatabase().Collection(roomCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find rooms for user in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &mongoRooms)

	if err != nil {
		logger.Logger.Error("Failed to find rooms for user in mongo ", err)
		return nil, err
	}

	for _, mongoRoom := range mongoRooms {
		rooms = append(rooms, mongoRoom.Room)
	}

	return rooms, nil
}

func DeleteRoomForUser(room *app.Room, user *clientcommon.User) error {
	filter := bson.D{{
		"_id",
		room.Id,
	}}

	update := bson.D{{
		"$pull",
		bson.D{{
			"users",
			bson.D{{
				"_id",
				user.GetId(),
			}},
		}},
	}}

	_, err := getDatabase().Collection(roomCollection).UpdateOne(context.TODO(), filter, update)

	if err != nil {
		logger.Logger.Error("Failed to delete room for user in mongo ", err)
		return err
	}

	logger.Logger.Info("Room was delete successfully in mongo for user ", user)

	return nil
}

func convertPlaylistsToMongoPlaylists(playlists map[string]*app.Playlist, room *app.Room) map[string]*MongoPlaylist {
	mongoPlaylists := make(map[string]*MongoPlaylist)

	for playlistId, playlist := range playlists {
		trackIdsPerSharedCount := make(map[int][]string)
		totalTracks := 0

		for sharedCount, tracks := range playlist.TracksPerSharedCount {
			trackIdsPerSharedCount[sharedCount] = getTrackIds(tracks)
			totalTracks += len(tracks)
		}

		mongoPlaylist := MongoPlaylist{
			playlist.PlaylistMetadata,
			trackIdsPerSharedCount,
			playlist.UserIdsPerSharedTracks,
			playlist.Users,
		}

		datadog.Increment(totalTracks, datadog.TrackForRoom,
			datadog.RoomIdTag.Tag(room.Id),
			datadog.RoomNameTag.Tag(room.Name),
			datadog.PlaylistTypeTag.Tag(playlist.Type),
		)

		mongoPlaylists[playlistId] = &mongoPlaylist
	}

	return mongoPlaylists
}

func convertMongoPlaylistsToPlaylists(mongoPlaylists map[string]*MongoPlaylist) (map[string]*app.Playlist, error) {
	playlists := make(map[string]*app.Playlist)

	// we get the tracks
	allTrackIds := make([]string, 0)
	for _, mongoPlaylist := range mongoPlaylists {
		for _, trackIds := range mongoPlaylist.TrackIdsPerSharedCount {
			allTrackIds = append(allTrackIds, trackIds...)
		}
	}

	trackPerId, err := GetTracks(allTrackIds)

	if err != nil {
		logger.Logger.Error("Failed to get tracks when converting mongo playlist to playlists ", err)
		return nil, err
	}

	for playlistId, mongoPlaylist := range mongoPlaylists {
		tracksPerSharedCount := make(map[int][]*spotify.FullTrack)

		for sharedCount, trackIds := range mongoPlaylist.TrackIdsPerSharedCount {

			tracks := make([]*spotify.FullTrack, 0)
			for _, trackId := range trackIds {
				track := trackPerId[trackId]
				tracks = append(tracks, track)
			}

			tracksPerSharedCount[sharedCount] = tracks
		}

		playlists[playlistId] = &app.Playlist{
			PlaylistMetadata:       mongoPlaylist.PlaylistMetadata,
			TracksPerSharedCount:   tracksPerSharedCount,
			UserIdsPerSharedTracks: mongoPlaylist.UserIdsPerSharedTracks,
			Users:                  mongoPlaylist.Users,
		}
	}

	return playlists, nil
}

func getTrackIds(tracks []*spotify.FullTrack) []string {
	trackIds := make([]string, 0)

	for _, track := range tracks {
		isrc, _ := spotifyclient.GetTrackISRC(track)
		trackIds = append(trackIds, isrc)
	}

	return trackIds
}

func getAllTracksForPlaylists(playlists map[string]*app.Playlist) []*spotify.FullTrack {
	allTracks := make([]*spotify.FullTrack, 0)

	for _, playlist := range playlists {
		tracks := playlist.GetAllTracks()
		allTracks = append(allTracks, tracks...)
	}

	return allTracks
}
