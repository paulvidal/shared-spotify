package mongoclient

import (
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
)

func IsOnlyDuplicateError(err error) bool {
	bulkWriteErrors, ok := err.(mongo.BulkWriteException)

	if !ok {
		return false
	}

	for _, bulkWriteErr := range bulkWriteErrors.WriteErrors {
		if !isDup(bulkWriteErr.WriteError) {
			return false
		}
	}

	return true
}

func isDup(err mongo.WriteError) bool {
	return err.Code == 11000 ||
		err.Code == 11001 ||
		err.Code == 12582 ||
		err.Code == 16460 &&
		strings.Contains(err.Message, " E11000 ")
}