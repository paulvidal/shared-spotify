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
)

const unprocessedRoomCollection = "unprocessed_rooms"

type MongoUnprocessedRoom struct {
	*app.Room `bson:"inline"`
}

func UpdateUnprocessedRoom(room *app.Room) error {
	mongoRoom := MongoUnprocessedRoom{Room: room}

	upsert := true

	insertResult, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).ReplaceOne(
		context.TODO(),
		bson.D{{
			"_id",
			mongoRoom.Room.Id,
		}},
		mongoRoom,
		&options.ReplaceOptions{Upsert: &upsert})

	if err != nil {
		logger.Logger.Error("Failed to update unprocessed room in mongo ", err)
		return err
	}

	logger.Logger.Info("Unprocessed room was updated successfully in mongo ",
		insertResult.UpsertedCount + insertResult.ModifiedCount)

	return nil
}

func GetUnprocessedRoom(roomId string) (*app.Room, error) {
	var mongoRoom MongoUnprocessedRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).FindOne(context.TODO(), filter).Decode(&mongoRoom)

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

func DeleteUnprocessedRoom(roomId string) error {
	deleteResult, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).DeleteOne(
		context.TODO(),
		bson.D{{
			"_id",
			roomId,
		}})

	if err != nil {
		logger.Logger.Errorf("Failed to delete unprocessed ropm with id %s %+v", roomId, err)
		return err
	}

	logger.Logger.Infof("Successfully deleted %d unprocessed room %s", deleteResult.DeletedCount, roomId)

	return nil
}

func GetUnprocessedRoomsForUser(user *clientcommon.User) ([]*app.Room, error) {
	mongoRooms := make([]*MongoUnprocessedRoom, 0)
	rooms := make([]*app.Room, 0)

	filter := bson.D{{
		"users._id",
		user.GetId(),
	}}

	cursor, err := mongoclient.GetDatabase().Collection(unprocessedRoomCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find unprocessed rooms for user in mongo ", err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &mongoRooms)

	if err != nil {
		logger.Logger.Error("Failed to find unprocessed rooms for user in mongo ", err)
		return nil, err
	}

	for _, mongoRoom := range mongoRooms {
		rooms = append(rooms, mongoRoom.Room)
	}

	return rooms, nil
}