package mongoclient

import (
	"context"
	"github.com/shared-spotify/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const isrcCollection = "isrc"

type IsrcMapping struct {
	Isrc      string `bson:"_id"`
	SpotifyId string `bson:"spotify_id"`
}

func InsertIsrcMapping(isrcMappings []IsrcMapping) error {
	// We do a mongo transaction as we want all the documents to be inserted at once
	ctx := context.Background()

	mongoSession, err := MongoClient.StartSession()

	if err != nil {
		logger.Logger.Error("Failed to start mongo session to insert isrcMappings ", err)
		return err
	}

	err = mongoSession.StartTransaction()

	if err != nil {
		logger.Logger.Error("Failed to start mongo transaction to insert isrcMappings ", err)
		return err
	}

	defer mongoSession.EndSession(ctx)

	err = mongo.WithSession(ctx, mongoSession, func(sessionContext mongo.SessionContext) error {

		ordered := false
		upsert := true

		writes := make([]mongo.WriteModel, 0)
		for _, isrcMapping := range isrcMappings {
			writes = append(writes, &mongo.ReplaceOneModel{Upsert: &upsert, Filter: bson.D{{
				"_id",
				isrcMapping.Isrc,
			}}, Replacement: isrcMapping})
		}

		_, err = GetDatabase().Collection(isrcCollection).BulkWrite(
			ctx, writes, &options.BulkWriteOptions{Ordered: &ordered})

		if err != nil {
			logger.Logger.Error("Failed to insert isrcMappings in mongo ", err)
			return err
		}

		err = mongoSession.CommitTransaction(ctx)

		if err != nil {
			logger.Logger.Error("Failed to commit mongo transaction to insert isrcMappings ", err)
			return err
		}

		return nil
	})

	if err != nil {
		if abortErr := mongoSession.AbortTransaction(ctx); abortErr != nil {
			logger.Logger.Error("Failed to abort mongo transaction to insert isrcMappings ", err)
			return abortErr
		}

		return err
	}

	logger.Logger.Infof("%d IsrcMappings were inserted successfully in mongo ", len(isrcMappings))

	return nil
}

func GetIsrcmappings(isrcs []string) (map[string]string, error) {
	isrcMappings := make([]IsrcMapping, 0)

	filter := bson.D{{
		"_id",
		bson.D{{
			"$in",
			isrcs,
		}},
	}}

	cursor, err := GetDatabase().Collection(isrcCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find isrcs in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &isrcMappings)

	if err != nil {
		logger.Logger.Error("Failed to find isrcs in mongo ", err)
		return nil, err
	}

	mapping := make(map[string]string)
	for _, isrc := range isrcMappings {
		mapping[isrc.Isrc] = isrc.SpotifyId
	}

	return mapping, nil
}
