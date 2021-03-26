package app

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/utils"
	"github.com/zmb3/spotify"
	"sort"
	"strings"
)

const playlistNameShared = "All songs in common"
const playlistNameDance = "Dance songs"
const playlistNamePopular = "Popular songs"
const playlistNameUnpopular = "Uncommon songs"
const playlistNameGenre = "%s songs"
// for music period
const playlistNameRecentRelease = "Recent release" // less than 1 year
const playlistName2010 = "2010s"
const playlistName2000 = "2000s"
const playlistName1990 = "1990s"
const playlistName1980 = "1980s"
const playlistNameOld = "1970s and before"

const playlistTypeShared = "shared"
const playlistTypePopularity = "popularity"
const playlistTypeDance = "dance"
const playlistTypeGenre = "genre"
const playlistTypePeriod = "music period"

const playlistRankShared = 1
const playlistRankPopular = 2
const playlistRankDance = 3
const playlistRankMusicPeriod = 4
const playlistRankGenre = 5

const minNumberOfUserForCommonMusic = 2

const periodRecentTrackCountThreshold = 2
const periodTrackCountThreshold = 5
const genreTrackCountThreshold = 5 // min count to have a playlist to be included
const maxGenrePlaylists = 4

const popularityThreshold = 60 // out of 100
const unpopularThreshold = 20  // out of 100

type CommonPlaylists struct {
	// all playlists in a map with key playlist generated id
	Playlists map[string]*Playlist `json:"-"`

	// These are fields used for computation of the playlists, they are not useful once Playlists is populated
	*CommonPlaylistComputation `bson:"-"`
}

type CommonPlaylistComputation struct {
	// all users in a map with key user id
	Users map[string]*clientcommon.User `json:"-"`
	// all tracks for a user in a map with key track id
	TracksPerUser map[string][]*spotify.FullTrack `json:"-"`
	// all users sharing track in a map with key track id
	SharedTracksRank map[string][]*clientcommon.User `json:"-"`
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
	RankForType      int    `json:"rank_for_type"`
	SharedTrackCount int    `json:"shared_track_count"`
}

type Playlist struct {
	PlaylistMetadata       `bson:"inline"`
	TracksPerSharedCount   map[int][]*spotify.FullTrack  `json:"tracks_per_shared_count"`
	UserIdsPerSharedTracks map[string][]string           `json:"user_ids_per_shared_tracks"`
	Users                  map[string]*clientcommon.User `json:"users"`
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
		make(map[string]*clientcommon.User),
		make(map[string][]*spotify.FullTrack),
		make(map[string][]*clientcommon.User),
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

func (playlists *CommonPlaylists) getAUser() *clientcommon.User {
	for _, user := range playlists.Users {
		return user
	}

	return nil
}

func (playlists *CommonPlaylists) addTracks(user *clientcommon.User, tracks []*spotify.FullTrack) {
	// Remember the user
	playlists.Users[user.GetId()] = user

	// Add the track for this user
	playlists.TracksPerUser[user.GetId()] = tracks

	// a list of tracks from a user can contain multiple times the same track, so we de-duplicate per user
	trackAlreadyInserted := make(map[string]bool)

	for _, track := range tracks {
		trackISCR, ok := clientcommon.GetTrackISRC(track)

		if !ok {
			logger.WithUser(user.GetUserId()).Warning("ISRC does not exist, track=", track)
			continue
		}

		_, ok = trackAlreadyInserted[trackISCR]

		if ok {
			// if the track has already been inserted for this user, we skip it to prevent adding duplicate songs
			continue
		}

		users, ok := playlists.SharedTracksRank[trackISCR]

		if !ok {
			users = make([]*clientcommon.User, 1)
			users[0] = user
		} else {
			users = append(users, user)
			logger.Logger.Debugf("Song %s present multiple times %d, id is %s, user is %s, track=%v",
				track.Name, len(users), track.ID, user.GetUserId(), track)
		}

		playlists.SharedTracksRank[trackISCR] = users
		playlists.SharedTracks[trackISCR] = track
		trackAlreadyInserted[trackISCR] = true
	}

	// insert the ISRC to spotify ID mapping to keep a record and be quicker next time
	isrcMapping := make([]mongoclient.IsrcMapping, 0)
	for _, track := range tracks {
		isrc, _ := clientcommon.GetTrackISRC(track)
		isrcMapping = append(isrcMapping, mongoclient.IsrcMapping{Isrc: isrc, SpotifyId: track.ID.String()})
	}

	err := mongoclient.InsertIsrcMapping(isrcMapping)

	if err != nil {
		logger.Logger.Errorf("Failed to insert %d isrc mapping", len(isrcMapping))
	}
}

func (playlists *CommonPlaylists) GeneratePlaylists() error {
	// Generate the shared track playlist
	sharedTrackPlaylist := playlists.GenerateCommonPlaylistType()

	// get all the shared track so it can be used to get infos on those tracks
	allSharedTracks := sharedTrackPlaylist.GetAllTracks()

	// get audio features among common songs
	audioFeatures, err := musicclient.GetAudioFeatures(allSharedTracks)

	if err != nil {
		return err
	}

	// set the audio features
	playlists.AudioFeaturesPerTrack = audioFeatures

	// get artists among common songs
	artists, err := musicclient.GetArtists(allSharedTracks)

	if err != nil {
		return err
	}

	// set the artists
	playlists.ArtistsPerTrack = artists

	// get the albums among common songs
	albums, err := musicclient.GetAlbums(allSharedTracks)

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

	// Generate the music period playlists
	playlists.GenerateMusicPeriodPlaylistType(sharedTrackPlaylist)

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

	commonPlaylistType := playlists.createPlaylist(playlistNameShared, playlistTypeShared,
		playlistRankShared, 1, tracksInCommon)

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
	playlists.createPlaylist(playlistNamePopular, playlistTypePopularity, playlistRankPopular, 1, popularTracksInCommon)
	playlists.createPlaylist(playlistNameUnpopular, playlistTypePopularity, playlistRankPopular, 2, unpopularTracksInCommon)
}

// This could be refactored but for now, let's say it's ok
func (playlists *CommonPlaylists) GenerateMusicPeriodPlaylistType(sharedTrackPlaylist *Playlist) {
	period1970TracksInCommon := make(map[int][]*spotify.FullTrack)
	period1980TracksInCommon := make(map[int][]*spotify.FullTrack)
	period1990TracksInCommon := make(map[int][]*spotify.FullTrack)
	period2000TracksInCommon := make(map[int][]*spotify.FullTrack)
	period2010TracksInCommon := make(map[int][]*spotify.FullTrack)
	periodRecentTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		period1970TracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)
		period1980TracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)
		period1990TracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)
		period2000TracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)
		period2010TracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)
		periodRecentTracksInCommonForSharedTrack := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			// get the song release data
			year := track.Album.ReleaseDateTime().Year()

			if year <= 1979 {
				logger.Logger.Debugf("Found song 1970s track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				period1970TracksInCommonForSharedTrack = append(period1970TracksInCommonForSharedTrack, track)

			} else if year <= 1989 {
				logger.Logger.Debugf("Found song 1980s track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				period1980TracksInCommonForSharedTrack = append(period1980TracksInCommonForSharedTrack, track)

			} else if year <= 1999 {
				logger.Logger.Debugf("Found song 1990s track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				period1990TracksInCommonForSharedTrack = append(period1990TracksInCommonForSharedTrack, track)

			} else if year <= 2009 {
				logger.Logger.Debugf("Found song 2000s track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				period2000TracksInCommonForSharedTrack = append(period2000TracksInCommonForSharedTrack, track)

			} else if year <= 2019 {
				logger.Logger.Debugf("Found song 2010s track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				period2010TracksInCommonForSharedTrack = append(period2010TracksInCommonForSharedTrack, track)

			} else { // > 2020 is recent
				logger.Logger.Debugf("Found song recent track track for %d person: %s by %v",
					sharedCount, track.Name, track.Artists)
				periodRecentTracksInCommonForSharedTrack = append(periodRecentTracksInCommonForSharedTrack, track)
			}
		}

		period1970TracksInCommon[sharedCount] = period1970TracksInCommonForSharedTrack
		period1980TracksInCommon[sharedCount] = period1980TracksInCommonForSharedTrack
		period1990TracksInCommon[sharedCount] = period1990TracksInCommonForSharedTrack
		period2000TracksInCommon[sharedCount] = period2000TracksInCommonForSharedTrack
		period2010TracksInCommon[sharedCount] = period2010TracksInCommonForSharedTrack
		periodRecentTracksInCommon[sharedCount] = periodRecentTracksInCommonForSharedTrack
	}

	// Generate the playlist per period era
	playlists.createPlaylistForMinCount(playlistNameOld, playlistTypePeriod, playlistRankMusicPeriod, 6,
		period1970TracksInCommon, periodTrackCountThreshold)
	playlists.createPlaylistForMinCount(playlistName1980, playlistTypePeriod, playlistRankMusicPeriod, 5,
		period1980TracksInCommon, periodTrackCountThreshold)
	playlists.createPlaylistForMinCount(playlistName1990, playlistTypePeriod, playlistRankMusicPeriod, 4,
		period1990TracksInCommon, periodTrackCountThreshold)
	playlists.createPlaylistForMinCount(playlistName2000, playlistTypePeriod, playlistRankMusicPeriod, 3,
		period2000TracksInCommon, periodTrackCountThreshold)
	playlists.createPlaylistForMinCount(playlistName2010, playlistTypePeriod, playlistRankMusicPeriod, 2,
		period2010TracksInCommon, periodTrackCountThreshold)
	playlists.createPlaylistForMinCount(playlistNameRecentRelease, playlistTypePeriod, playlistRankMusicPeriod, 1,
		periodRecentTracksInCommon, periodRecentTrackCountThreshold)
}

func (playlists *CommonPlaylists) GenerateDancePlaylist(sharedTrackPlaylist *Playlist) {
	danceTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		danceTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := clientcommon.GetTrackISRC(track)
			audioFeatures := playlists.AudioFeaturesPerTrack[isrc]

			if audioFeatures.Danceability >= 0.7 {
				danceTracksInCommonForSharedCount = append(danceTracksInCommonForSharedCount, track)
			}
		}

		danceTracksInCommon[sharedCount] = danceTracksInCommonForSharedCount
	}

	playlists.createPlaylist(playlistNameDance, playlistTypeDance, playlistRankDance, 1, danceTracksInCommon)
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

	allGenres := make([]string, 0)
	for genre := range genres {
		allGenres = append(allGenres, genre)
	}

	// we sort the allGenres array, placing the ost popular ones in front
	sort.Slice(allGenres, func(i, j int) bool {
		return genres[allGenres[i]] > genres[allGenres[j]]
	})

	logger.Logger.Debug("Genres in order of popularity are: ", allGenres)

	// if there are less genres then we want to select, stop
	genreToSelectCount := maxGenrePlaylists
	if len(allGenres) < maxGenrePlaylists {
		genreToSelectCount = len(allGenres)
	}

	for _, genre := range allGenres[:genreToSelectCount] {
		playlists.GenerateGenrePlaylist(sharedTrackPlaylist, genre)
	}
}

func (playlists *CommonPlaylists) GenerateGenrePlaylist(sharedTrackPlaylist *Playlist, playlistGenre string) {
	genreTrackCount := 0
	genreTracksInCommon := make(map[int][]*spotify.FullTrack)

	for sharedCount, tracks := range sharedTrackPlaylist.TracksPerSharedCount {
		genreTracksInCommonForSharedCount := make([]*spotify.FullTrack, 0)

		for _, track := range tracks {
			isrc, _ := clientcommon.GetTrackISRC(track)
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

	playlistType := fmt.Sprintf(playlistNameGenre, strings.Title(strings.ToLower(playlistGenre)))
	playlists.createPlaylistForMinCount(playlistType, playlistTypeGenre, playlistRankGenre, 1,
		genreTracksInCommon, genreTrackCountThreshold)
}

// Helper to get the max number of tracks in common
func getTracksInCommonCount(trackList map[int][]*spotify.FullTrack) int {
	tracksInCommonCount := 0

	for _, tracks := range trackList {
		tracksInCommonCount += len(tracks)
	}

	return tracksInCommonCount
}

func (playlists *CommonPlaylists) createPlaylistForMinCount(name string, type_ string, rank int, rankForType int,
	tracksPerSharedCount map[int][]*spotify.FullTrack, minCount int) {

	count := 0

	for _, tracks := range tracksPerSharedCount {
		count += len(tracks)
	}

	if count >= minCount {
		playlists.createPlaylist(name, type_, rank, rankForType, tracksPerSharedCount)
	}
}

func (playlists *CommonPlaylists) createPlaylist(
	name string, type_ string, rank int, rankForType int, tracksPerSharedCount map[int][]*spotify.FullTrack) *Playlist {

	playlistId := utils.GenerateStrongHash()
	playlist := &Playlist{
		PlaylistMetadata{
			playlistId,
			name,
			type_,
			rank,
			rankForType,
			getTracksInCommonCount(tracksPerSharedCount),
		},
		tracksPerSharedCount,
		playlists.SharedTracksRankAboveMinThreshold,
		playlists.Users,
	}
	playlists.Playlists[playlistId] = playlist

	return playlist
}
