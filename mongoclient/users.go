package mongoclient

import (
	"context"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const userCollection = "users"

type MongoUser struct {
	*clientcommon.User `bson:"inline"`
}

func InsertUsers(users []*clientcommon.User) error {
	usersToInsert := make([]interface{}, 0)

	for _, user := range users {
		usersToInsert = append(usersToInsert, MongoUser{user})
	}

	// We do a mongo transaction as we want all the documents to be inserted at once
	ctx := context.Background()

	mongoSession, err := MongoClient.StartSession()

	if err != nil {
		logger.Logger.Error("Failed to start mongo session to insert users ", err)
		return err
	}

	err = mongoSession.StartTransaction()

	if err != nil {
		logger.Logger.Error("Failed to start mongo transaction to insert users ", err)
		return err
	}

	ordered := false // to prevent duplicates from making the whole operation fail, we will just ignore them
	result, err := getDatabase().Collection(userCollection).InsertMany(
		ctx,
		usersToInsert,
		&options.InsertManyOptions{Ordered: &ordered})

	if err != nil && !IsOnlyDuplicateError(err) {
		logger.Logger.Error("Failed to insert users in mongo ", err)
		abortErr := mongoSession.AbortTransaction(ctx)

		if abortErr != nil {
			logger.Logger.Error("Failed to abort mongo transaction to insert users ", err)
			return abortErr
		}

		return err
	}

	err = mongoSession.CommitTransaction(ctx)

	if err != nil {
		logger.Logger.Error("Failed to commit mongo transaction to insert users ", err)
		return err
	}

	mongoSession.EndSession(ctx)

	newUsersCount := len(result.InsertedIDs)
	datadog.Increment(newUsersCount, datadog.UsersNewCount)

	logger.Logger.Info("Users were inserted successfully in mongo ", result.InsertedIDs)

	return nil
}

func GetUsers(userIds []string) (map[string]*clientcommon.User, error) {
	mongoUsers := make([]*MongoUser, 0)
	usersPerId := make(map[string]*clientcommon.User)

	filter := bson.D{{
		"_id",
		bson.D{{
			"$in",
			userIds,
		}},
	}}

	cursor, err := getDatabase().Collection(userCollection).Find(context.TODO(), filter)

	if err != nil {
		logger.Logger.Error("Failed to find users in mongo ", err)
		return nil, err
	}

	logger.Logger.Info("here16")

	err = cursor.All(context.TODO(), &mongoUsers)

	logger.Logger.Info("here13")

	if err != nil {
		logger.Logger.Error("Failed to find users in mongo ", err)
		return nil, err
	}

	// we convert the users back to their original format
	for _, mongoUser := range mongoUsers {
		usersPerId[mongoUser.Id] = mongoUser.User
	}

	return usersPerId, nil
}
