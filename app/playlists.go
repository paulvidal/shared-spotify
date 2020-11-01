package app

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
)

const playlistNameShared = "Shared songs"
const playlistNameDance = "Dance shared songs"
const playlistNamePopular = "Most popular shared songs"
const playlistNameGenre = "Genre [%s] shared songs"

const playlistTypeShared = "shared"
const playlistTypePopular = "popular"
const playlistTypeDance = "dance"
const playlistTypeGenre = "genre"

const playlistRankShared = 1
const playlistRankPopular = 2
const playlistRankDance = 3
const playlistRankGenre = 4

const minNumberOfUserForCommonMusic = 2

const genreTrackCountThreshold = 5 // min count to have a playlist to be included

const popularityThreshold = 60 // out of 100

type CommonPlaylists struct {
	// all users in a map with key user id
	Users map[string]*spotifyclient.User `json:"users"`
	// all tracks for a user in a map with key track id
	TracksPerUser map[string][]*spotify.FullTrack `json:"-"`
	// all users sharing track in a map with key track id
	SharedTracksRank map[string][]*spotifyclient.User `json:"-"`
	// all users sharing track above the min threshold minNumberOfUserForCommonMusic in a map with key track id
	SharedTracksRankAboveMinThreshold map[string][]*spotifyclient.User `json:"shared_tracks_rank"`
	// all tracks of all users in a map with key track id
	SharedTracks map[string]*spotify.FullTrack `json:"-"`
	// all playlists in a map with key playlist generated id
	PlaylistTypes map[string]*PlaylistType `json:"playlist_types"`
	// audio records in a map with key track id
	AudioFeaturesPerTrack map[string]*spotify.AudioFeatures `json:"-"`
	// artist list in a map with key track id
	ArtistsPerTrack map[string][]*spotify.FullArtist `json:"-"`
	// album in a map with key track id
	AlbumPerTrack map[string]*spotify.FullAlbum `json:"-"`
}

type PlaylistType struct {
	Id                   string                       `json:"id"`
	Name                 string                       `json:"name"`
	Type                 string                       `json:"type"`
	Rank                 int                          `json:"rank"`
	TracksPerSharedCount map[int][]*spotify.FullTrack `json:"tracks_per_shared_count"`
}

func (playlistType *PlaylistType) getAllTracks() []*spotify.FullTrack {
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
		make(map[string][]*spotifyclient.User),
		make(map[string][]*spotifyclient.User),
		make(map[string]*spotify.FullTrack),
		make(map[string]*PlaylistType, 0),
		nil,
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

		users, ok := playlists.SharedTracksRank[trackISCR]

		if !ok {
			users = make([]*spotifyclient.User, 1)
			users[0] = user
			logger.Logger.Infof("New song %s, id is %s, user is %s, track=%+v",
				track.Name, track.ID, user.GetUserId(), track)

		} else {
			users = append(users, user)
			logger.Logger.Infof("Song %s present multiple times %d, id is %s, user is %s, track=%+v",
				track.Name, len(users), track.ID, user.GetUserId(), track)
		}

		playlists.SharedTracksRank[trackISCR] = users
		playlists.SharedTracks[trackISCR] = track
		trackAlreadyInserted[trackISCR] = true
	}
}

func (playlists *CommonPlaylists) GeneratePlaylists() error {
	// Generate the shared track playlist
	sharedTrackPlaylist := playlists.GenerateCommonPlaylistType()

	// get all the shared track so it can be used to get infos on those tracks
	allSharedTracks := sharedTrackPlaylist.getAllTracks()

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

	// get the albums among common songs
	albums, err := user.GetAlbums(allSharedTracks)

	if err != nil {
		return err
	}

	// set the albums
	playlists.AlbumPerTrack = albums

	/*
	  We generate new playlists here
	*/

	// Generate the popular songs playlist
	playlists.GeneratePopularPlaylistType(sharedTrackPlaylist)

	//TODO: activate back dance playlists
	// Generate the dance track playlist
	//playlists.GenerateDancePlaylist(sharedTrackPlaylist)

	// Generate the genre playlists
	playlists.GenerateGenrePlaylists(sharedTrackPlaylist)

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

	for trackId, users := range playlists.SharedTracksRank {
		userCount := len(users)

		if userCount >= minNumberOfUserForCommonMusic {
			// playlist containing as key the number of user that share this music, and in value the number of tracks
			trackListForUserCount := tracksInCommon[userCount]

			track := playlists.SharedTracks[trackId]
			tracksInCommon[userCount] = append(trackListForUserCount, track)

			// Add shared track rank above min threshold, so we can in the frontend keep record of who liked the song
			playlists.SharedTracksRankAboveMinThreshold[trackId] = users

			logger.Logger.Infof("Common track found for %d person: %s by %v", userCount, track.Name, track.Artists)
		}
	}

	for commonUserCount, tracks := range tracksInCommon {
		logger.Logger.Infof("Found %d tracks shared between %d users", len(tracks), commonUserCount)
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &PlaylistType{
		id,
		playlistNameShared,
		playlistTypeShared,
		playlistRankShared,
		tracksInCommon,
	}
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
	commonPlaylistType := &PlaylistType{
		id,
		playlistNamePopular,
		playlistTypePopular,
		playlistRankPopular,
		popularTracksInCommon,
	}
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
	commonPlaylistType := &PlaylistType{
		id,
		playlistNameDance,
		playlistTypeDance,
		playlistRankDance,
		danceTracksInCommon,
	}
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
		playlists.GenerateGenrePlaylist(sharedTrackPlaylist, genre)
	}
}

func (playlists *CommonPlaylists) GenerateGenrePlaylist(sharedTrackPlaylist *PlaylistType, playlistGenre string) {
	genreTrackCount := 0
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
				genreTrackCount += 1
			}
		}

		genreTracksInCommon[sharedCount] = genreTracksInCommonForSharedCount
	}

	if genreTrackCount >= genreTrackCountThreshold {
		id := utils.GenerateStrongHash()
		palylistType := fmt.Sprintf(playlistNameGenre, playlistGenre)

		commonPlaylistType := &PlaylistType{
			id,
			palylistType,
			playlistTypeGenre,
			playlistRankGenre,
			genreTracksInCommon,
		}
		playlists.PlaylistTypes[id] = commonPlaylistType
	}
}