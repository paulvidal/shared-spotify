package applemusic

import (
	"context"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"net/http"
)

const maxPage = 50
const maxCatalogSongsPerApiCall = 300
const maxPlaylistPerApiCall = 100
const maxRetryGetSongsByIsrc = 10

func GetAllSongs(user *clientcommon.User) ([]*applemusic.Song, error) {
	// Get the library songs
	logger.WithUser(user.GetUserId()).Info("Fetching all apple library songs for user")

	savedSongs, err := GetLibrarySongs(user)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error(
			"Failed to fetch all apple library songs for user ",
			err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Info("Successfully fetched all apple library songs for user")

	// Get the playlist songs
	logger.WithUser(user.GetUserId()).Info("Fetching all apple library songs for library playlists for user")

	playlistSongs, err := GetAllLibraryPlaylistSongs(user)

	logger.WithUser(user.GetUserId()).Info("Successfully fetched all apple library songs for library playlists for user")

	if err != nil {
		logger.WithUser(user.GetUserId()).Error(
			"Failed to fetch all apple library songs for library playlists for user",
			err)
		return nil, err
	}

	// Merge all the songs here
	allSongs := make([]*applemusic.Song, 0)
	allSongs = append(allSongs, savedSongs...)
	allSongs = append(allSongs, playlistSongs...)

	return allSongs, nil
}

// This method gets all the library songs of a user
func GetLibrarySongs(user *clientcommon.User) ([]*applemusic.Song, error) {
	client := user.AppleMusicClient

	// We fetch all the library songs
	allLibrarySongs := make([]*applemusic.LibrarySong, 0)

	next := true
	offset := 0

	for next {
		logger.WithUser(user.GetUserId()).Debugf("Fetching library songs offset %d", offset)

		librarySongs, _, err := client.Me.GetAllLibrarySongs(
			context.Background(),
			&applemusic.PageOptions{Limit: maxPage, Offset: offset})

		clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSavedSongs, true, err)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library songs ", err)
			return nil, err
		}

		logger.WithUser(user.GetUserId()).Debugf("Found %d library songs for offset %d", len(librarySongs.Data), offset)

		// Add all the songs
		for _, s := range librarySongs.Data {
			song := s
			allLibrarySongs = append(allLibrarySongs, &song)
		}

		logger.WithUser(user.GetUserId()).Debugf("Library songs next=%s href=%s", librarySongs.Next, librarySongs.Href)

		if librarySongs.Next == "" {
			next = false
		}

		offset += maxPage
	}

	allTracks, err := getFullSongsForLibrarySongs(user, allLibrarySongs)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to convert apple library songs to catalog songs ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Found %d apple library songs", len(allTracks))

	return allTracks, nil
}

// This method gets all the songs from the playlists of the user
func GetAllLibraryPlaylistSongs(user *clientcommon.User) ([]*applemusic.Song, error) {
	client := user.AppleMusicClient

	// We fetch all the library playlists
	allLibraryPlaylists := make([]*applemusic.LibraryPlaylist, 0)

	next := true
	offset := 0

	for next {
		logger.WithUser(user.GetUserId()).Debugf("Fetching library playlists songs offset %d", offset)

		playlists, _, err := client.Me.GetAllLibraryPlaylists(
			context.Background(),
			&applemusic.PageOptions{Offset: offset, Limit: maxPlaylistPerApiCall})

		clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypePlaylists, true, err)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library playlists ", err)
			return nil, err
		}

		logger.WithUser(user.GetUserId()).Debugf("Found %d library playlists songs for offset %d", len(playlists.Data), offset)

		// Add all the playlists
		for _, p := range playlists.Data {
			playlist := p
			allLibraryPlaylists = append(allLibraryPlaylists, &playlist)
		}

		if playlists.Next == "" {
			next = false
		}

		logger.WithUser(user.GetUserId()).Debugf("Library playlist songs next=%s", playlists.Next)

		offset += maxPlaylistPerApiCall
	}

	logger.WithUser(user.GetUserId()).Infof("User %s has a total of %d apple playlists", user.GetUserId(), len(allLibraryPlaylists))

	// We fetch all the songs for each library playlist

	// These are not real song objects, we need to fetch storefront to have real songs with all the info
	allIncompleteSongs := make([]*applemusic.Song, 0)

	for _, playlist := range allLibraryPlaylists {

		// Do not take into account playlists which the user did not create
		// (we find this by checking edit and delete permissions)
		if !playlist.Attributes.CanEdit {
			logger.WithUser(user.GetUserId()).Warningf(
				"Skipped apple playlist %s as user had not write access edit=%t",
				playlist.Attributes.Name,
				playlist.Attributes.CanEdit)
			continue
		}

		librarySongs, err := client.Me.GetLibraryPlaylistTracks(
			context.Background(),
			playlist.Id,
			nil)

		if err != nil {
			success := true

			if errResponse, ok := err.(*applemusic.ErrorResponse); ok {

				// we need to make sure status 404 does not throw an error
				if errResponse.Response.StatusCode != http.StatusNotFound {
					success = false
				}

			} else {
				success = false
			}

			if !success {
				clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSongs, true, err)
				logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library playlist songs", err)
				return nil, err
			}
		}

		clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSongs, true, nil)
		logger.WithUser(user.GetUserId()).Infof("Found %d apple songs for playlists %s",
			len(librarySongs),
			playlist.Attributes.Name)

		for _, l := range librarySongs {
			librarySong := l
			allIncompleteSongs = append(allIncompleteSongs, &librarySong)
		}
	}

	allTracks, err := getFullSongsForIncompleteSongs(user, allIncompleteSongs)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to convert apple playlist library songs to catalog songs ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Found %d apple playlist library songs", len(allTracks))

	return allTracks, nil
}

// Allow us to transform incomplete songs into catalog songs where we can get all info related to a song such as ISRC
func getFullSongsForIncompleteSongs(user *clientcommon.User, librarySongs []*applemusic.Song) ([]*applemusic.Song, error) {
	songByIds := make([]string, 0)

	for _, librarySong := range librarySongs {
		// playParams can be null
		if librarySong.Attributes.PlayParams != nil {
			songByIds = append(songByIds, librarySong.Attributes.PlayParams.CatalogId)
		}
	}

	return getFullSongs(user, songByIds)
}

// Allow us to transform library songs into catalog songs where we can get all info related to a song such as ISRC
func getFullSongsForLibrarySongs(user *clientcommon.User, librarySongs []*applemusic.LibrarySong) ([]*applemusic.Song, error) {
	songByIds := make([]string, 0)

	for _, librarySong := range librarySongs {
		songByIds = append(songByIds, librarySong.Attributes.PlayParams.CatalogId)
	}

	return getFullSongs(user, songByIds)
}

func getFullSongs(user *clientcommon.User, songIds []string) ([]*applemusic.Song, error) {
	client := user.AppleMusicClient

	storefront, err := GetStorefront(user)

	if err != nil {
		return nil, err
	}

	allSongs := make([]*applemusic.Song, 0)

	for i := 0; i < len(songIds); i += maxCatalogSongsPerApiCall {
		upperBound := i + maxCatalogSongsPerApiCall

		if upperBound > len(songIds) {
			upperBound = len(songIds)
		}

		songs, _, err := client.Catalog.GetSongsByIds(
			context.Background(),
			*storefront,
			songIds[i:upperBound],
			nil)

		clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSongs, true, err)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to get apple songs by id ", err)
			return nil, err
		}

		for _, s := range songs.Data {
			song := s
			allSongs = append(allSongs, &song)
		}

		logger.Logger.Infof("Fetched %d apple songs successfully", upperBound-i)
	}

	return allSongs, nil
}

func GetsongsByIsrc(user *clientcommon.User, storefront string, isrcs []string) (*applemusic.Songs, error) {
	var songs *applemusic.Songs
	var resp *applemusic.Response
	var err error

	for retry := 1; retry <= maxRetryGetSongsByIsrc; retry++ {
		songs, resp, err = user.AppleMusicClient.Catalog.GetSongsByIsrcs(
			context.Background(),
			storefront,
			isrcs,
			nil)

		// Apple randomly return 504 sometimes, so we need to retry
		if resp != nil && resp.StatusCode != http.StatusGatewayTimeout {
			clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSongs, true, nil)
			return songs, err
		}

		clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeSongs, true, err)
		logger.WithUser(user.GetUserId()).Errorf("Failed to get songs by ISRC - attempt count=%d - %v ", retry, err)
	}

	return songs, err
}
