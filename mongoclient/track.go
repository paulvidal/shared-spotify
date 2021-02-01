package mongoclient

import (
	"context"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const trackCollection = "tracks"

type MongoTrack struct {
	TrackId            string `bson:"_id"`
	*spotify.FullTrack `bson:"inline"`
}

func InsertTracks(tracks []*spotify.FullTrack) error {
	tracksToInsert := make([]MongoTrack, 0)

	if len(tracks) == 0 {
		return nil
	}

	for _, track := range tracks {
		id, _ := clientcommon.GetTrackISRC(track)
		tracksToInsert = append(tracksToInsert, MongoTrack{id, track})
	}

	// We do a mongo transaction as we want all the documents to be inserted at once
	ctx := context.Background()

	mongoSession, err := MongoClient.StartSession()

	if err != nil {
		logger.Logger.Error("Failed to start mongo session to insert track ", err)
		return err
	}

	err = mongoSession.StartTransaction()

	if err != nil {
		logger.Logger.Error("Failed to start mongo transaction to insert tracks ", err)
		return err
	}

	defer mongoSession.EndSession(ctx)

	err = mongo.WithSession(ctx, mongoSession, func(sessionContext mongo.SessionContext) error {
		ordered := false
		upsert := true

		writes := make([]mongo.WriteModel, 0)
		for _, track := range tracksToInsert {
			writes = append(writes, &mongo.ReplaceOneModel{Upsert: &upsert, Filter: bson.D{{
				"_id",
				track.TrackId,
			}}, Replacement: track})
		}

		_, err = GetDatabase().Collection(trackCollection).BulkWrite(
			ctx, writes, &options.BulkWriteOptions{Ordered: &ordered})

		if err != nil {
			logger.Logger.Error("Failed to insert tracks in mongo ", err)
			return err
		}

		err = mongoSession.CommitTransaction(ctx)

		if err != nil {
			logger.Logger.Error("Failed to commit mongo transaction to insert tracks ", err)
			return err
		}

		return nil
	})

	if err != nil {
		if abortErr := mongoSession.AbortTransaction(ctx); abortErr != nil {
			logger.Logger.Error("Failed to abort mongo transaction to insert tracks ", err)
			return abortErr
		}

		return err
	}

	logger.Logger.Infof("%d tracks were inserted successfully in mongo ", len(tracksToInsert))

	return nil
}

func GetTracks(trackIds []string) (map[string]*spotify.FullTrack, error) {
	mongoTracks := make([]*MongoTrack, 0)
	tracksPerId := make(map[string]*spotify.FullTrack)

	filter := bson.D{{
		"_id",
		bson.D{{
			"$in",
			trackIds,
		}},
	}}

	cursor, err := GetDatabase().Collection(trackCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find tracks in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &mongoTracks)

	if err != nil {
		logger.Logger.Error("Failed to find tracks in mongo ", err)
		return nil, err
	}

	// we convert the tracks back to their original format
	for _, mongoTrack := range mongoTracks {
		isrc, _ := clientcommon.GetTrackISRC(mongoTrack.FullTrack)
		tracksPerId[isrc] = mongoTrack.FullTrack
	}

	return tracksPerId, nil
}
