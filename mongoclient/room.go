package mongoclient

import (
	"context"
	"errors"
	"github.com/shared-spotify/appmodels"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const roomCollection = "rooms"
const trackCollection = "tracks"

var NotFound = errors.New("Not found")

type MongoRoom struct {
	*appmodels.Room
	Playlists map[string]*MongoPlaylist
}

type MongoPlaylist struct {
	*appmodels.Playlist
	TrackIdsPerSharedCount map[int][]string
}

/**
  TODO: We need to make sure when we insert a room, and when we deserialise it, we end up with the room and all its infos
    we should only touch this class and adapt objects for now
    => goal is to have rooms and tracks in separate collections otherwise size is too big
 */

func InsertRoom(room *appmodels.Room) error {
	mongoRoom := MongoRoom{
		room,
		convertPlaylistsToMongoPlaylists(room.MusicLibrary.CommonPlaylists.Playlists),
	}

	insertResult, err := getDatabase().Collection(roomCollection).InsertOne(context.TODO(), mongoRoom)

	if err != nil {
		logger.Logger.Error("Failed to insert room in mongo ", err)
		return err
	}

	logger.Logger.Info("Room was inserted successfully in mongo", insertResult.InsertedID)

	return nil
}

func GetRoom(roomId string) (*appmodels.Room, error) {
	var result MongoRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := getDatabase().Collection(roomCollection).FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, NotFound
		}

		logger.Logger.Error("Failed to find room in mongo ", err)
	}

	return &result, err
}

func GetRoomsForUser(user *spotifyclient.User) ([]*appmodels.Room, error) {
	results := make([]*appmodels.Room, 0)

	filter := bson.D{{
		"users.userinfos.id",
		user.GetId(),
	}}

	cursor, err := getDatabase().Collection(roomCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find rooms for user in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &results)

	if err != nil {
		logger.Logger.Error("Failed to find rooms for user in mongo ", err)
		return nil, err
	}

	return results, nil
}

func convertPlaylistsToMongoPlaylists(playlists map[string]*appmodels.Playlist) map[string]*MongoPlaylist {
	mongoPlaylists := make(map[string]*MongoPlaylist)

	// TODO
}

func convertMongoPlaylistToPlaylist()  {

}

/*
  Tracks
 */

type MongoTrack struct {
	TrackId string `bson:"_id"`
	*spotify.FullTrack `bson:"track"`
}

func InsertTracks(tracks []*spotify.FullTrack) error {
	tracksToInsert := make([]interface{}, 0)

	for _, track := range tracks {
		id, _ := spotifyclient.GetTrackISRC(track)
		tracksToInsert = append(tracksToInsert, MongoTrack{id, track})
	}

	ordered := false // to prevent duplicates from making the whole operation fail, we will just ignore them
	_, err := getDatabase().Collection(trackCollection).InsertMany(
		context.TODO(),
		tracksToInsert,
		&options.InsertManyOptions{Ordered: &ordered})

	if err != nil {
		logger.Logger.Error("Failed to insert tracks in mongo ", err)
		return err
	}

	return nil
}

func GetTracks(trackIds []string) ([]*spotify.FullTrack, error) {
	results := make([]*MongoTrack, 0)
	alltracks := make([]*spotify.FullTrack, 0)

	filter := bson.D{{
		"_id",
		bson.D{{
			"$in",
			bson.A{
				trackIds,
			},
		}},
	}}

	cursor, err := getDatabase().Collection(trackCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find tracks in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &results)

	if err != nil {
		logger.Logger.Error("Failed to find tracks in mongo ", err)
		return nil, err
	}

	// we convert the tracks back to their original format
	for _, r := range results {
		alltracks = append(alltracks, r.FullTrack)
	}

	return alltracks, nil
}