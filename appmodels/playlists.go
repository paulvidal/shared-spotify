package appmodels

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
)

const playlistNameShared = "Songs in common"
const playlistNameDance = "Dance songs in common"
const playlistNamePopular = "Most popular songs in common"
const playlistNameUnpopular = "Unpopular songs in common"
const playlistNameGenre = "Genre [%s] songs in common"

const playlistTypeShared = "shared"
const playlistTypePopular = "popular"
const playlistTypeUnpopular = "unpopular"
const playlistTypeDance = "dance"
const playlistTypeGenre = "genre"

const playlistRankShared = 1
const playlistRankPopular = 2
const playlistRankUnpopular = 3
const playlistRankDance = 4
const playlistRankGenre = 5

const minNumberOfUserForCommonMusic = 2

const genreTrackCountThreshold = 5 // min count to have a playlist to be included

const popularityThreshold = 60 // out of 100
const unpopularThreshold = 25 // out of 100

type CommonPlaylists struct {
	// all playlists in a map with key playlist generated id
	Playlists map[string]*Playlist `json:"-"`

	// These are fields used for computation of the playlists, they are not useful once Playlists is populated
	*CommonPlaylistComputation `bson:"-"`
}

type CommonPlaylistComputation struct {
	// all users in a map with key user id
	Users map[string]*spotifyclient.User `json:"-"`
	// all tracks for a user in a map with key track id
	TracksPerUser map[string][]*spotify.FullTrack `json:"-"`
	// all users sharing track in a map with key track id
	SharedTracksRank map[string][]*spotifyclient.User `json:"-"`
	// all user ids sharing track above the min threshold minNumberOfUserForCommonMusic in a map with key track id
	SharedTracksRankAboveMinThreshold map[string][]string `json:"-"`
	// all tracks of all users in a map with key track id
	SharedTracks map[string]*spotify.FullTrack `json:"-"`
	// audio records in a map with key track id
	AudioFeaturesPerTrack map[string]*spotify.AudioFeatures `json:"-"`
	// artist list in a map with key track id
	ArtistsPerTrack map[string][]*spotify.FullArtist `json:"-"`
	// album in a map with key track id
	AlbumPerTrack map[string]*spotify.FullAlbum `json:"-"`
}

type PlaylistsMetadata map[string]*PlaylistMetadata

type PlaylistMetadata struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Rank             int    `json:"rank"`
	SharedTrackCount int    `json:"shared_track_count"`
}

type Playlist struct {
	PlaylistMetadata                                        `bson:"inline"`
	TracksPerSharedCount   map[int][]*spotify.FullTrack     `json:"tracks_per_shared_count"`
	UserIdsPerSharedTracks map[string][]string              `json:"user_ids_per_shared_tracks"`
	Users                  map[string]*spotifyclient.User   `json:"users"`
}

func (playlist *Playlist) GetAllTracks() []*spotify.FullTrack {
	tracks := make([]*spotify.FullTrack, 0)

	for _, tracksPart := range playlist.TracksPerSharedCount {
		tracks = append(tracks, tracksPart...)
	}

	return tracks
}

func (playlists *CommonPlaylists) GetPlaylistsMetadata() PlaylistsMetadata {
	playlistsMetadata := make(PlaylistsMetadata)

	for playlistId, playlist := range playlists.Playlists {
		playlistsMetadata[playlistId] = &playlist.PlaylistMetadata
	}

	return playlistsMetadata
}

func CreateCommonPlaylists() *CommonPlaylists {
	computation := CommonPlaylistComputation{
		make(map[string]*spotifyclient.User),
		make(map[string][]*spotify.FullTrack),
		make(map[string][]*spotifyclient.User),
		make(map[string][]string),
		make(map[string]*spotify.FullTrack),
		nil,
		nil,
		nil,
	}

	return &CommonPlaylists{
		make(map[string]*Playlist, 0),
		&computation,
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
		} else {
			users = append(users, user)
			logger.Logger.Debugf("Song %s present multiple times %d, id is %s, user is %s, track=%+v",
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
	allSharedTracks := sharedTrackPlaylist.GetAllTracks()

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
	  We generate new playlists here, with the additional infos gathered
	*/

	// Generate the popular songs playlist
	playlists.GeneratePopularityPlaylistType(sharedTrackPlaylist)

	// TODO: activate back dance playlists once it work
	//playlists.GenerateDancePlaylist(sharedTrackPlaylist)

	// Generate the genre playlists
	playlists.GenerateGenrePlaylists(sharedTrackPlaylist)

	/*
	  We release the memory used for the computation as it won't be used anymore
	*/
	playlists.CommonPlaylistComputation = nil

	return nil
}

func (playlists *CommonPlaylists) GenerateCommonPlaylistType() *Playlist {
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
			userIds := make([]string, 0)
			for _, user := range users {
				userIds = append(userIds, user.GetId())
			}
			playlists.SharedTracksRankAboveMinThreshold[trackId] = userIds 

			logger.Logger.Debugf("Common track found for %d person: %s by %v", userCount, track.Name, track.Artists)
		}
	}

	for commonUserCount, tracks := range tracksInCommon {
		logger.Logger.Infof("Found %d tracks shared between %d users", len(tracks), commonUserCount)
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &Playlist{
		PlaylistMetadata{
			id,
			playlistNameShared,
			playlistTypeShared,
			playlistRankShared,
			getTracksInCommonCount(tracksInCommon),
		},
		tracksInCommon,
		playlists.SharedTracksRankAboveMinThreshold,
		playlists.Users,
	}
	playlists.Playlists[id] = commonPlaylistType

	return commonPlaylistType
}

func (playlists *CommonPlaylists) GeneratePopularityPlaylistType(sharedTrackPlaylist *Playlist) {
	popularTracksInCommon := make(map[int][]*spotify.FullTrack)
	unpopularTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		popularTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)
		unpopularTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			if track.Popularity >= popularityThreshold {
				logger.Logger.Debugf("Found popular track for %d person: %s by %v", sharedCount, track.Name, track.Artists)
				popularTracksInCommonForSharedCount = append(popularTracksInCommonForSharedCount, track)

			} else if track.Popularity != 0 && track.Popularity <= unpopularThreshold {
				logger.Logger.Debugf("Found unpopular track for %d person: %s by %v", sharedCount, track.Name, track.Artists)
				unpopularTracksInCommonForSharedCount = append(unpopularTracksInCommonForSharedCount, track)
			}
		}

		popularTracksInCommon[sharedCount] = popularTracksInCommonForSharedCount
		unpopularTracksInCommon[sharedCount] = unpopularTracksInCommonForSharedCount
	}

	// Popular playlist
	popularId := utils.GenerateStrongHash()
	popularCommonPlaylistType := &Playlist{
		PlaylistMetadata{
			popularId,
			playlistNamePopular,
			playlistTypePopular,
			playlistRankPopular,
			getTracksInCommonCount(popularTracksInCommon),
		},
		popularTracksInCommon,
		playlists.SharedTracksRankAboveMinThreshold,
		playlists.Users,
	}
	playlists.Playlists[popularId] = popularCommonPlaylistType

	// Unpopular playlist
	unpopularId := utils.GenerateStrongHash()
	unpopularCommonPlaylistType := &Playlist{
		PlaylistMetadata{
			unpopularId,
			playlistNameUnpopular,
			playlistTypeUnpopular,
			playlistRankUnpopular,
			getTracksInCommonCount(unpopularTracksInCommon),
		},
		unpopularTracksInCommon,
		playlists.SharedTracksRankAboveMinThreshold,
		playlists.Users,
	}
	playlists.Playlists[unpopularId] = unpopularCommonPlaylistType
}

func (playlists *CommonPlaylists) GenerateDancePlaylist(sharedTrackPlaylist *Playlist) {
	danceTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		danceTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := spotifyclient.GetTrackISRC(track)
			audioFeatures := playlists.AudioFeaturesPerTrack[isrc]

			if audioFeatures.Danceability >= 0.7 {
				danceTracksInCommonForSharedCount = append(danceTracksInCommonForSharedCount, track)
			}
		}

		danceTracksInCommon[sharedCount] = danceTracksInCommonForSharedCount
	}

	id := utils.GenerateStrongHash()
	commonPlaylistType := &Playlist{
		PlaylistMetadata{
			id,
			playlistNameDance,
			playlistTypeDance,
			playlistRankDance,
			getTracksInCommonCount(danceTracksInCommon),
		},
		danceTracksInCommon,
		playlists.SharedTracksRankAboveMinThreshold,
		playlists.Users,
	}
	playlists.Playlists[id] = commonPlaylistType
}

func (playlists *CommonPlaylists) GenerateGenrePlaylists(sharedTrackPlaylist *Playlist) {
	genres := make(map[string]int)

	for _, artists := range playlists.ArtistsPerTrack {
		trackGenres := make(map[string]bool)

		for _, artist := range artists {
			for _, genre := range artist.Genres {
				trackGenres[genre] = true
			}
		}

		for genre := range trackGenres {
			count := genres[genre]
			genres[genre] = count + 1
		}
	}

	logger.Logger.Debug("Genres are: ", genres)

	for genre := range genres {
		playlists.GenerateGenrePlaylist(sharedTrackPlaylist, genre)
	}
}

func (playlists *CommonPlaylists) GenerateGenrePlaylist(sharedTrackPlaylist *Playlist, playlistGenre string) {
	genreTrackCount := 0
	genreTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		genreTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := spotifyclient.GetTrackISRC(track)
			artists := playlists.ArtistsPerTrack[isrc]

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
				logger.Logger.Debugf("Track for genre %s found: %s", playlistGenre, track.Name)
				genreTracksInCommonForSharedCount = append(genreTracksInCommonForSharedCount, track)
				genreTrackCount += 1
			}
		}

		genreTracksInCommon[sharedCount] = genreTracksInCommonForSharedCount
	}

	if genreTrackCount >= genreTrackCountThreshold {
		id := utils.GenerateStrongHash()
		playlistType := fmt.Sprintf(playlistNameGenre, playlistGenre)

		commonPlaylistType := &Playlist{
			PlaylistMetadata{
				id,
				playlistType,
				playlistTypeGenre,
				playlistRankGenre,
				getTracksInCommonCount(genreTracksInCommon),
			},
			genreTracksInCommon,
			playlists.SharedTracksRankAboveMinThreshold,
			playlists.Users,
		}
		playlists.Playlists[id] = commonPlaylistType
	}
}

// Helper to get hte max number of tracks in common
func getTracksInCommonCount(trackList map[int][]*spotify.FullTrack) int {
	tracksInCommonCount := 0

	for _, tracks := range trackList {
		tracksInCommonCount += len(tracks)
	}

	return tracksInCommonCount
}