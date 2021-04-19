package app

import (
	"context"
	"errors"
	"github.com/jinzhu/copier"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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

func InsertRoom(room *app.Room, ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "mongo.room.insert")
	defer span.Finish()

	playlists := room.GetPlaylists()

	// we insert the users
	err := mongoclient.InsertUsersWithCtx(room.Users, ctx)

	newUserCount := len(room.Users)
	datadog.Increment(newUserCount, datadog.RoomUsers,
		datadog.RoomIdTag.Tag(room.Id),
		datadog.RoomNameTag.Tag(room.Name),
	)

	if err != nil {
		span.Finish(tracer.WithError(err))
		return err
	}

	// we insert the tracks
	tracks := getAllTracksForPlaylists(playlists)
	err = mongoclient.InsertTracks(tracks, ctx)

	if err != nil {
		span.Finish(tracer.WithError(err))
		return err
	}

	// we insert the room
	mongoPlaylists := convertPlaylistsToMongoPlaylists(playlists, room)

	// IMPORTANT: we remove the tokens to not introduce them in long term storage once the processing is over
	roomOwner, errOwner := recreateUsersWithoutToken([]*clientcommon.User{room.Owner})
	roomUsers, errUsers := recreateUsersWithoutToken(room.Users)

	if errOwner != nil  {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("An error occurred while copying users to remove token %v %v", errOwner, span)
		return errOwner
	}

	if errUsers != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("An error occurred while copying users to remove token %v %v", errUsers, span)
		return errUsers
	}

	room.Owner = roomOwner[0]
	room.Users = roomUsers

	mongoRoom := MongoRoom{
		room,
		mongoPlaylists,
	}

	insertResult, err := mongoclient.GetDatabase().Collection(roomCollection).InsertOne(ctx, mongoRoom)

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to insert room in mongo %v %v", err, span)
		return err
	}

	logger.Logger.Infof("Room was inserted successfully in mongo %v %v", insertResult.InsertedID, span)

	return nil
}

// we prevent introducing in the database for the processed rooms the user tokens (even if they are encrypted)
func recreateUsersWithoutToken(users []*clientcommon.User) ([]*clientcommon.User, error) {
	newUsers := make([]*clientcommon.User, 0)

	// copy as we don't want ot alter users, which should be immutable
	err := copier.Copy(&newUsers, &users)

	if err != nil {
		return nil, err
	}

	for _, user := range newUsers {
		user.Token = ""
	}

	return newUsers, nil
}

func GetRoom(roomId string, ctx context.Context) (*app.Room, error) {
	var mongoRoom MongoRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := mongoclient.GetDatabase().Collection(roomCollection).FindOne(ctx, filter).Decode(&mongoRoom)

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

func GetRoomsForUser(user *clientcommon.User, ctx context.Context) ([]*app.Room, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "mongo.rooms.get.for.user")
	span.SetTag("user", user.GetUserId())
	defer span.Finish()

	mongoRooms := make([]*MongoRoom, 0)
	rooms := make([]*app.Room, 0)

	filter := bson.D{{
		"users._id",
		user.GetId(),
	}}

	otherSpan, ctx := tracer.StartSpanFromContext(ctx, "mongo.cursor.find")
	cursor, err := mongoclient.GetDatabase().Collection(roomCollection).Find(ctx, filter)

	if err != nil {
		logger.Logger.
			WithError(err).
			Errorf("Failed to find rooms for user in mongo %v", span)
		otherSpan.Finish(tracer.WithError(err))
		return nil, err
	}
	otherSpan.Finish()

	otherSpan, ctx = tracer.StartSpanFromContext(ctx, "mongo.cursor.all")
	err = cursor.All(ctx, &mongoRooms)

	if err != nil {
		logger.Logger.
			WithError(err).
			Errorf("Failed to find rooms for user in mongo %v", span)
		otherSpan.Finish(tracer.WithError(err))
		return nil, err
	}
	otherSpan.Finish()

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

	_, err := mongoclient.GetDatabase().Collection(roomCollection).UpdateOne(context.TODO(), filter, update)

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

	trackPerId, err := mongoclient.GetTracks(allTrackIds)

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
		isrc, _ := clientcommon.GetTrackISRC(track)
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
