package mongoclient

import (
	"context"
	"errors"
	"github.com/shared-spotify/appmodels"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/zmb3/spotify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const roomCollection = "rooms"
const trackCollection = "tracks"

var NotFound = errors.New("Not found")

type MongoRoom struct {
	*appmodels.Room
	Playlists map[string]*MongoPlaylist
}

type MongoPlaylist struct {
	appmodels.PlaylistMetadata
	TrackIdsPerSharedCount map[int][]string
	UsersPerSharedTracks   map[string][]*spotifyclient.User
}

func InsertRoom(room *appmodels.Room) error {
	playlists := room.GetPlaylists()

	tracks := getAllTracksForPlaylists(playlists)
	err := InsertTracks(tracks)

	if err != nil {
		return err
	}

	mongoPlaylists := convertPlaylistsToMongoPlaylists(playlists)

	mongoRoom := MongoRoom{
		room,
		mongoPlaylists,
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
	var mongoRoom MongoRoom

	filter := bson.D{{
		"_id",
		roomId,
	}}

	err := getDatabase().Collection(roomCollection).FindOne(context.TODO(), filter).Decode(&mongoRoom)

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

	for playlistId, playlist := range playlists {
		trackIdsPerSharedCount := make(map[int][]string)

		for sharedCount, tracks := range playlist.TracksPerSharedCount {
			trackIdsPerSharedCount[sharedCount] = getTrackIds(tracks)
		}

		mongoPlaylist := MongoPlaylist{
			playlist.PlaylistMetadata,
			trackIdsPerSharedCount,
			playlist.UsersPerSharedTracks,
		}

		mongoPlaylists[playlistId] = &mongoPlaylist
	}

	return mongoPlaylists
}

func convertMongoPlaylistsToPlaylists(mongoPlaylists map[string]*MongoPlaylist) (map[string]*appmodels.Playlist, error) {
	playlists := make(map[string]*appmodels.Playlist)
	trackIds := make([]string, 0)

	for _, mongoPlaylist := range mongoPlaylists {
		for _, trackIds := range mongoPlaylist.TrackIdsPerSharedCount {
			trackIds = append(trackIds, trackIds...)
		}
	}

	trackPerId, err := GetTracks(trackIds)

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

		playlists[playlistId] = &appmodels.Playlist{
			PlaylistMetadata:     mongoPlaylist.PlaylistMetadata,
			TracksPerSharedCount: tracksPerSharedCount,
			UsersPerSharedTracks: mongoPlaylist.UsersPerSharedTracks,
		}
	}

	return playlists, nil
}

func getTrackIds(tracks []*spotify.FullTrack) []string {
	trackIds := make([]string, 0)

	for _, track := range tracks {
		isrc, _ := spotifyclient.GetTrackISRC(track)
		trackIds = append(trackIds, isrc)
	}

	return trackIds
}

func getAllTracksForPlaylists(playlists map[string]*appmodels.Playlist) []*spotify.FullTrack {
	allTracks := make([]*spotify.FullTrack, 0)

	for _, playlist := range playlists {
		tracks := playlist.GetAllTracks()
		allTracks = append(allTracks, tracks...)
	}

	return allTracks
}