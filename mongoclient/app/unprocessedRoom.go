package app

import (
	"context"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const unprocessedRoomCollection = "unprocessed_rooms"

type MongoUnprocessedRoom struct {
	*app.Room `bson:"inline"`
}

func UpdateUnprocessedRoom(room *app.Room, ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "mongo.room.unprocessed.update")
	defer span.Finish()

	mongoRoom := MongoUnprocessedRoom{Room: room}

	upsert := true

	insertResult, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).ReplaceOne(
		ctx,
		bson.D{{
			"_id",
			mongoRoom.Room.Id,
		}},
		mongoRoom,
		&options.ReplaceOptions{Upsert: &upsert})

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to update unprocessed room in mongo %v %v", err, span)
		return err
	}

	logger.Logger.Infof("Unprocessed room was updated successfully in mongo %d %v",
		insertResult.UpsertedCount + insertResult.ModifiedCount, span)

	return nil
}

func GetUnprocessedRoom(roomId string, ctx context.Context) (*app.Room, error) {
	var mongoRoom MongoUnprocessedRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).FindOne(ctx, filter).Decode(&mongoRoom)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, NotFound
		}

		logger.Logger.Error("Failed to find unprocessed room in mongo ", err)
	}

	room := mongoRoom.Room

	logger.Logger.Infof("Fetched unprocessed room %s successfully", roomId)

	return room, err
}

func DeleteUnprocessedRoom(roomId string, ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "mongo.room.unprocessed.delete")
	defer span.Finish()

	deleteResult, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).DeleteOne(
		ctx,
		bson.D{{
			"_id",
			roomId,
		}})

	if err != nil {
		span.Finish(tracer.WithError(err))
		logger.Logger.Errorf("Failed to delete unprocessed ropm with id %s %v %v", roomId, err, span)
		return err
	}

	logger.Logger.Infof("Successfully deleted %d unprocessed room %s %v", deleteResult.DeletedCount, roomId, span)

	return nil
}

func GetUnprocessedRoomsForUser(user *clientcommon.User, ctx context.Context) ([]*app.Room, error) {
	mongoRooms := make([]*MongoUnprocessedRoom, 0)
	rooms := make([]*app.Room, 0)

	filter := bson.D{{
		"users._id",
		user.GetId(),
	}}

	cursor, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).Find(ctx, filter)

	if err != nil {
		logger.Logger.Error("Failed to find unprocessed rooms for user in mongo ", err)
		return nil, err
	}

	err = cursor.All(ctx, &mongoRooms)

	if err != nil {
		logger.Logger.Error("Failed to find unprocessed rooms for user in mongo ", err)
		return nil, err
	}

	for _, mongoRoom := range mongoRooms {
		rooms = append(rooms, mongoRoom.Room)
	}

	return rooms, nil
}