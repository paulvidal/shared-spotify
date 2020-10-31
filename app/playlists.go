package app

import (
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
)

const playlistTypeShared = "Shared songs"
const playlistTypeDance = "Dance shared songs"
const playlistTypePopular = "Popular shared songs"

const minNumberOfUserForCommonMusic = 2

const popularityThreshold = 60  // out of 100

type CommonPlaylists struct {
	// all users in a map with key user id
	Users                 map[string]*spotifyclient.User    `json:"users"`
	// all tracks for a user in a map with key track id
	TracksPerUser         map[string][]*spotify.FullTrack   `json:"-"`
	// all track shared count of all users in a map with key track id
	SharedTracksRank      map[string]*int                   `json:"-"`
	// all tracks of all users in a map with key track id
	SharedTracks          map[string]*spotify.FullTrack     `json:"-"`
	// all playlists in a map with key playlist generated id
	PlaylistTypes         map[string]*PlaylistType          `json:"playlist_types"`
	// audio records in a map with key track id
	AudioFeaturesPerTrack map[string]*spotify.AudioFeatures `json:"-"`
	// artist list in a map with key track id
	ArtistsPerTrack       map[string][]*spotify.FullArtist  `json:"-"`
}

type PlaylistType struct {
	Id                   string                       `json:"id"`
	Type                 string                       `json:"type"`
	TracksPerSharedCount map[int][]*spotify.FullTrack `json:"tracks_per_shared_count"`
}

func (playlistType *PlaylistType) getAllTracks() []*spotify.FullTrack{
	tracks := make([]*spotify.FullTrack, 0)

	for _, tracksPart := range playlistType.TracksPerSharedCount {
		tracks = append(tracks, tracksPart...)
	}

	return tracks
}

func CreateCommonPlaylists() *CommonPlaylists {
	return &CommonPlaylists{
		make(map[string]*spotifyclient.User),
		make(map[string][]*spotify.FullTrack),
		make(map[string]*int),
		make(map[string]*spotify.FullTrack),
		make(map[string]*PlaylistType, 0),
		nil,
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
	sharedTrackPlaylist := playlists.GenerateCommonPlaylistType()

	// get all the shared track so it can be used to get infos on those tracks
	allSharedTracks := sharedTrackPlaylist.getAllTracks()

	// Generate the popular songs playlist
	//playlists.GeneratePopularPlaylistType(sharedTrackPlaylist)

	// get audio features among common songs
	user := playlists.getAUser()
	audioFeatures, err := user.GetAudioFeatures(allSharedTracks)

	if err != nil {
		return err
	}

	// set the audio features
	playlists.AudioFeaturesPerTrack = audioFeatures

	// get artists among common songs
	artists, err := user.GetArtists(allSharedTracks)

	if err != nil {
		return err
	}

	// set the artists
	playlists.ArtistsPerTrack = artists

	//TODO: activate back dance playlists
	// Generate the dance track playlist
	//playlists.GenerateDancePlaylist(sharedTrackPlaylist)

	// Generate the genre playlists
	//playlists.GenerateGenrePlaylists(sharedTrackPlaylist)

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

func (playlists *CommonPlaylists) GeneratePopularPlaylistType(sharedTrackPlaylist *PlaylistType) {
	popularTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		popularTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			if track.Popularity >= popularityThreshold {
				logger.Logger.Infof("Found popular track for %d person: %s by %v", sharedCount, track.Name, track.Artists)
				popularTracksInCommonForSharedCount = append(popularTracksInCommonForSharedCount, track)
			}
		}

		popularTracksInCommon[sharedCount] = popularTracksInCommonForSharedCount
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &PlaylistType{id, playlistTypePopular, popularTracksInCommon}
	playlists.PlaylistTypes[id] = commonPlaylistType
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

func (playlists *CommonPlaylists) GenerateGenrePlaylists(sharedTrackPlaylist *PlaylistType) {
	genres := make(map[string]int)

	for _, artists := range playlists.ArtistsPerTrack {
		trackGenres := make(map[string]bool)

		for _, artist := range artists {
			for _, genre := range artist.Genres {
				trackGenres[genre] = true
			}
		}

		for genre, _ := range trackGenres {
			count := genres[genre]
			genres[genre] = count + 1
		}
	}

	logger.Logger.Info("Genres are: ", genres)

	for genre, _ := range genres {
		playlists.GenerateGenrePlaylist(sharedTrackPlaylist, genre, genre + " shared songs")
	}
}

func (playlists *CommonPlaylists) GenerateGenrePlaylist(sharedTrackPlaylist *PlaylistType, playlistGenre string, playlistName string) {
	genreTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		genreTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := spotifyclient.GetTrackISRC(track)
			artists := playlists.ArtistsPerTrack[isrc]

			logger.Logger.Infof("Track %s has artists %+v", track.Name, artists)

			genreFound := false

			// We include the song if one artist is of this genre
			for _, artist := range artists {
				for _, genre := range artist.Genres {
					if genre == playlistGenre {
						genreFound = true
						break
					}
				}
			}

			if genreFound {
				logger.Logger.Infof("Track for genre %s found: %s", playlistGenre, track.Name)
				genreTracksInCommonForSharedCount = append(genreTracksInCommonForSharedCount, track)
			}
		}

		genreTracksInCommon[sharedCount] = genreTracksInCommonForSharedCount
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &PlaylistType{id, playlistName, genreTracksInCommon}
	playlists.PlaylistTypes[id] = commonPlaylistType
}