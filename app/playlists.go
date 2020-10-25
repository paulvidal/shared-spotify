package app

import (
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/zmb3/spotify"
)

type CommonPlaylists struct {
	TracksPerUser          map[string][]*spotify.FullTrack   `json:"-"`
	TracksInCommon         []*spotify.FullTrack              `json:"tracks_in_common"`
	SharedTracksRank       map[string]*int                   `json:"-"`
	SharedTracks           map[string]*spotify.FullTrack     `json:"-"`
}

func CreateCommonPlaylists() *CommonPlaylists {
	return &CommonPlaylists{
		make(map[string][]*spotify.FullTrack),
		make([]*spotify.FullTrack, 0),
		make(map[string]*int ),
		make(map[string]*spotify.FullTrack),
	}
}

func (playlists *CommonPlaylists) addTracks(user *spotifyclient.User, tracks []*spotify.FullTrack) {
	// Add the track for this user
	playlists.TracksPerUser[user.Infos.Id] = tracks

	// a list of tracks from a user can contain multiple times the same track, so we de-duplicate per user
	trackAlreadyInserted := make(map[string]bool)

	for _, track := range tracks {
		trackId := string(track.URI)

		_, ok := trackAlreadyInserted[trackId]

		if ok {
			// if the track has already been inserted for this user, we skip it to prevent adding duplicate songs
			continue
		}

		var newTrackCount int
		trackCount, ok := playlists.SharedTracksRank[trackId]

		if !ok {
			newTrackCount = 1
		} else {
			newTrackCount = *trackCount + 1
		}

		playlists.SharedTracksRank[trackId] = &newTrackCount
		playlists.SharedTracks[trackId] = track
		trackAlreadyInserted[trackId] = true
	}
}

func (playlists *CommonPlaylists) GenerateCommonPlaylists() {
	totalUsers := len(playlists.TracksPerUser)

	logger.Logger.Infof("Finding most common tracks for %d users across %d different tracks",
		totalUsers, len(playlists.SharedTracksRank))

	inCommon := make([]*spotify.FullTrack, 0)

	for trackId, trackCount := range playlists.SharedTracksRank {
		if *trackCount == totalUsers {
			track := playlists.SharedTracks[trackId]
			inCommon = append(inCommon, track)

			logger.Logger.Infof("Common track found: %s", track.Name)
		}
	}

	logger.Logger.Infof("Found %d tracks in common", len(inCommon))

	playlists.TracksInCommon = inCommon
}