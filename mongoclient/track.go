package mongoclient

import (
	"context"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoTrack struct {
	TrackId            string `bson:"_id"`
	*spotify.FullTrack `bson:"track"`
}

func InsertTracks(tracks []*spotify.FullTrack) error {
	tracksToInsert := make([]interface{}, 0)

	for _, track := range tracks {
		id, _ := spotifyclient.GetTrackISRC(track)
		tracksToInsert = append(tracksToInsert, MongoTrack{id, track})
	}

	// We do a mongo transaction as we want all the documents to be inserted at once
	ctx := context.Background()

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Important: You must pass sessCtx as the Context parameter to the operations for them to be executed in the
		// transaction.

		ordered := false // to prevent duplicates from making the whole operation fail, we will just ignore them
		result, err := getDatabase().Collection(trackCollection).InsertMany(
			sessCtx,
			tracksToInsert,
			&options.InsertManyOptions{Ordered: &ordered})

		return result, err
	}

	mongoSession, err := MongoClient.StartSession()

	if err != nil {
		logger.Logger.Error("Failed to start mongo session ", err)
	}

	defer mongoSession.EndSession(ctx)

	_, err = mongoSession.WithTransaction(ctx, callback)

	if err != nil {
		logger.Logger.Error("Failed to insert tracks in mongo ", err)
		return err
	}

	return nil
}

func GetTracks(trackIds []string) (map[string]*spotify.FullTrack, error) {
	mongotracks := make([]*MongoTrack, 0)
	tracksPerId := make(map[string]*spotify.FullTrack)

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

	err = cursor.All(context.TODO(), &mongotracks)

	if err != nil {
		logger.Logger.Error("Failed to find tracks in mongo ", err)
		return nil, err
	}

	// we convert the tracks back to their original format
	for _, mongoTrack := range mongotracks {
		isrc, _ := spotifyclient.GetTrackISRC(mongoTrack.FullTrack)
		tracksPerId[isrc] = mongoTrack.FullTrack
	}

	return tracksPerId, nil
}