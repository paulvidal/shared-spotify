package app

import (
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
)

const minNumberOfUserForCommonMusic = 2

const playlistTypeShared = "Shared songs"
const playlistTypeDance = "Dance shared songs"

type CommonPlaylists struct {
	Users                 map[string]*spotifyclient.User    `json:"users"`
	TracksPerUser         map[string][]*spotify.FullTrack   `json:"-"`
	SharedTracksRank      map[string]*int                   `json:"-"`
	SharedTracks          map[string]*spotify.FullTrack     `json:"-"`
	PlaylistTypes         map[string]*PlaylistType          `json:"playlist_types"`
	AudioFeaturesPerTrack map[string]*spotify.AudioFeatures `json:"-"`
}

type PlaylistType struct {
	Id                   string                       `json:"id"`
	Type                 string                       `json:"type"`
	TracksPerSharedCount map[int][]*spotify.FullTrack `json:"tracks_per_shared_count"`
}

func CreateCommonPlaylists() *CommonPlaylists {
	return &CommonPlaylists{
		make(map[string]*spotifyclient.User),
		make(map[string][]*spotify.FullTrack),
		make(map[string]*int),
		make(map[string]*spotify.FullTrack),
		make(map[string]*PlaylistType, 0),
		nil,
	}
}

func (playlists *CommonPlaylists) getAUser() *spotifyclient.User {
	for _, user := range playlists.Users {
		return user
	}

	return nil
}

func (playlists *CommonPlaylists) addTracks(user *spotifyclient.User, tracks []*spotify.FullTrack) {
	// Remember the user
	playlists.Users[user.GetId()] = user

	// Add the track for this user
	playlists.TracksPerUser[user.GetId()] = tracks

	// a list of tracks from a user can contain multiple times the same track, so we de-duplicate per user
	trackAlreadyInserted := make(map[string]bool)

	for _, track := range tracks {
		trackISCR, ok := spotifyclient.GetTrackISRC(track)

		if !ok {
			logger.WithUser(user.GetUserId()).Error("ISRC does not exist, track=", track)
			continue
		}

		_, ok = trackAlreadyInserted[trackISCR]

		if ok {
			// if the track has already been inserted for this user, we skip it to prevent adding duplicate songs
			continue
		}

		var newTrackCount int
		trackCount, ok := playlists.SharedTracksRank[trackISCR]

		if !ok {
			logger.Logger.Infof("New song %s, id is %s, user is %s, track=%+v",
				track.Name, track.ID, user.GetUserId(), track)
			newTrackCount = 1
		} else {
			newTrackCount = *trackCount + 1
			logger.Logger.Infof("Song %s present multiple times %d, id is %s, user is %s, track=%+v",
				track.Name, newTrackCount, track.ID, user.GetUserId(), track)
		}

		playlists.SharedTracksRank[trackISCR] = &newTrackCount
		playlists.SharedTracks[trackISCR] = track
		trackAlreadyInserted[trackISCR] = true
	}
}

func (playlists *CommonPlaylists) GeneratePlaylists() error {
	// Generate the shared track playlist
	playlists.GenerateCommonPlaylistType()

	// TODO: activate back dance playlists
	//// get audio features among common songs
	//user := playlists.getAUser()
	//audioFeatures, err := user.GetAudioFeatures(playlists.SharedTracks)
	//
	//if err != nil {
	//	return err
	//}
	//
	//// set the audio features
	//playlists.AudioFeaturesPerTrack = audioFeatures
	//
	//// Generate the dance track playlist
	//playlists.GenerateDancePlaylist(sharedTrackPlaylist)

	return nil
}

func (playlists *CommonPlaylists) GenerateCommonPlaylistType() *PlaylistType {
	totalUsers := len(playlists.TracksPerUser)

	logger.Logger.Infof("Finding most common tracks for %d users across %d different tracks",
		totalUsers, len(playlists.SharedTracksRank))

	tracksInCommon := make(map[int][]*spotify.FullTrack)

	// Create the track list for each user count possibility
	for i := minNumberOfUserForCommonMusic; i <= totalUsers; i++ {
		tracksInCommon[i] = make([]*spotify.FullTrack, 0)
	}

	for trackId, trackCount := range playlists.SharedTracksRank {
		if *trackCount >= minNumberOfUserForCommonMusic {
			// playlist containing as key the number of user that share this music, and in value the number of tracks
			trackListForUserCount := tracksInCommon[*trackCount]

			track := playlists.SharedTracks[trackId]
			tracksInCommon[*trackCount] = append(trackListForUserCount, track)

			logger.Logger.Infof("Common track found for %d person: %s by %v", *trackCount, track.Name, track.Artists)
		}
	}

	for commonUserCount, tracks := range tracksInCommon {
		logger.Logger.Infof("Found %d tracks shared between %d users", len(tracks), commonUserCount)
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &PlaylistType{id, playlistTypeShared, tracksInCommon}
	playlists.PlaylistTypes[id] = commonPlaylistType

	return commonPlaylistType
}

func (playlists *CommonPlaylists) GenerateDancePlaylist(sharedTrackPlaylist *PlaylistType) {
	danceTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		danceTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := spotifyclient.GetTrackISRC(track)
			audioFeatures := playlists.AudioFeaturesPerTrack[isrc]

			logger.Logger.Infof("Track %s has audio features %+v", track.Name, audioFeatures)

			if audioFeatures.Danceability >= 0.7 {
				danceTracksInCommonForSharedCount = append(danceTracksInCommonForSharedCount, track)
			}
		}

		danceTracksInCommon[sharedCount] = danceTracksInCommonForSharedCount
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &PlaylistType{id, playlistTypeDance, danceTracksInCommon}
	playlists.PlaylistTypes[id] = commonPlaylistType
}