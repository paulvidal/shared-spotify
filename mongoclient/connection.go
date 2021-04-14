package mongoclient

import (
	"context"
	"github.com/shared-spotify/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongotrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go.mongodb.org/mongo-driver/mongo"
	"os"
	"reflect"
	"time"
)

var MongoClient *mongo.Client
var MongoUrl = os.Getenv("MONGO_URL")

const database = "spotify"

func Initialise() {
	// Create the struct codec decoder
	structcodec, err := bsoncodec.NewStructCodec(JSONFallbackStructTagParser, bsonoptions.StructCodec().
		SetDecodeZeroStruct(true).
		SetEncodeOmitDefaultStruct(true).
		SetAllowUnexportedFields(true))

	if err != nil {
		logger.Logger.Fatal("Failed to load struct codec ", err)
	}

	// Set client options
	clientOptions := options.
		Client().
		ApplyURI(MongoUrl).
		SetRegistry(
			bson.NewRegistryBuilder(). // Add the new struct codec
							RegisterDefaultDecoder(reflect.Struct, structcodec).
							RegisterDefaultEncoder(reflect.Struct, structcodec).
							Build(),
		).
		// Add timeouts
		SetServerSelectionTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetSocketTimeout(60 * time.Second).
		SetMonitor(mongotrace.NewMonitor(mongotrace.WithAnalytics(true))) // enable tracing of mongo calls

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)

	// setup the client so we don't lose reference
	MongoClient = client

	if err != nil {
		logger.Logger.Fatalf("Failed to connect to mongo with url %s %s", MongoUrl, err)
	}

	// Check the connection
	err = MongoClient.Ping(context.TODO(), nil)

	if err != nil {
		logger.Logger.Fatalf("Failed to ping mongo with url %s %s", MongoUrl, err)
	}

	logger.Logger.Warningf("Connection to mongo successful, with url %s", MongoUrl)
}

func GetDatabase() *mongo.Database {
	return MongoClient.Database(database)
}
