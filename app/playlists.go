package app

import (
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
)

const minNumberOfUserForCommonMusic = 2

const playlistTypeShared = "Common songs"

type CommonPlaylists struct {
	TracksPerUser    map[string][]*spotify.FullTrack `json:"-"`
	SharedTracksRank map[string]*int                 `json:"-"`
	SharedTracks     map[string]*spotify.FullTrack   `json:"-"`
	PlaylistTypes    map[string]*PlaylistType        `json:"playlist_types"`
}

type PlaylistType struct {
	Id                   string                       `json:"id"`
	Type                 string                       `json:"type"`
	TracksPerSharedCount map[int][]*spotify.FullTrack `json:"tracks_per_shared_count"`
}

func CreateCommonPlaylists() *CommonPlaylists {
	return &CommonPlaylists{
		make(map[string][]*spotify.FullTrack),
		make(map[string]*int),
		make(map[string]*spotify.FullTrack),
		make(map[string]*PlaylistType, 0),
	}
}

func (playlists *CommonPlaylists) addTracks(user *spotifyclient.User, tracks []*spotify.FullTrack) {
	// Add the track for this user
	playlists.TracksPerUser[user.Infos.Id] = tracks

	// a list of tracks from a user can contain multiple times the same track, so we de-duplicate per user
	trackAlreadyInserted := make(map[string]bool)

	for _, track := range tracks {
		// Unique id representing a track
		// https://en.wikipedia.org/wiki/International_Standard_Recording_Code
		trackId, ok := track.ExternalIDs["isrc"]

		if !ok {
			logger.WithUser(user.GetUserId()).Error("ISRC does not exist, found=", track.ExternalIDs)
			continue
		}

		_, ok = trackAlreadyInserted[trackId]

		if ok {
			// if the track has already been inserted for this user, we skip it to prevent adding duplicate songs
			continue
		}

		var newTrackCount int
		trackCount, ok := playlists.SharedTracksRank[trackId]

		if !ok {
			logger.Logger.Infof("New song %s, id is %s, user is %s, track=%+v",
				track.Name, track.ID, user.GetUserId(), track)
			newTrackCount = 1
		} else {
			newTrackCount = *trackCount + 1
			logger.Logger.Infof("Song %s present multiple times %d, id is %s, user is %s, track=%+v",
				track.Name, newTrackCount, track.ID, user.GetUserId(), track)
		}

		playlists.SharedTracksRank[trackId] = &newTrackCount
		playlists.SharedTracks[trackId] = track
		trackAlreadyInserted[trackId] = true
	}
}

func (playlists *CommonPlaylists) GenerateCommonPlaylistType() {
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
	playlists.PlaylistTypes[id] = &PlaylistType{id, playlistTypeShared, tracksInCommon}
}
