package applemusic

import (
	"context"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
)

const maxPage = 50
const maxCatalogSongsPerApiCall = 300
const maxPlaylistPerApiCall = 100


func GetAllSongs(user *clientcommon.User) ([]*applemusic.Song, error) {
	// Get the library songs
	savedSongs, err := GetLibrarySongs(user)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error(
			"Failed to fetch all apple library songs for user ",
			err)
		return nil, err
	}

	// Get the playlist songs
	playlistSongs, err := GetAllLibraryPlaylistSongs(user)

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
		librarySongs, _, err := client.Me.GetAllLibrarySongs(
			context.Background(),
			&applemusic.PageOptions{Limit: maxPage, Offset: offset})

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library songs ", err)
			return nil, err
		}

		// Add all the songs
		for _, song := range librarySongs.Data {
			allLibrarySongs = append(allLibrarySongs, &song)
		}

		if librarySongs.Next == "" {
			next = false
		}

		offset += 1
	}

	logger.WithUser(user.GetUserId()).Infof("Found %d apple library songs", len(allLibrarySongs))

	allTracks, err := getFullSongsForLibrarySongs(user, allLibrarySongs)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to convert apple library songs to catalog songs ", err)
		return nil, err
	}

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
		playlists, _, err := client.Me.GetAllLibraryPlaylists(
			context.Background(),
			&applemusic.PageOptions{Offset: offset, Limit: maxPlaylistPerApiCall})

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library playlists ", err)
			return nil, err
		}

		// Add all the playlists
		for _, playlist := range playlists.Data {
			allLibraryPlaylists = append(allLibraryPlaylists, &playlist)
		}

		if playlists.Next == "" {
			next = false
		}

		offset += 1
	}

	logger.WithUser(user.GetUserId()).Infof("User %d has a total of %s apple playlists", user.GetUserId(), len(allLibraryPlaylists))

	// We fetch all the songs for each library playlist

	// These are not real song objects, we need to fetch storefront to have real songs with all the info
	allIncompleteSongs := make([]*applemusic.Song, 0)

	for _, playlist := range allLibraryPlaylists {

		// Do not take into account playlists which the user did not create
		// (we find this by checking edit and delete permissions)
		if !playlist.Attributes.CanEdit || !playlist.Attributes.CanDelete {
			logger.WithUser(user.GetUserId()).Warningf(
				"Skipped apple playlist %s as user had not write access edit=%s delete=%s",
				playlist.Attributes.Name,
				playlist.Attributes.CanEdit,
				playlist.Attributes.CanDelete)
			continue
		}

		librarySongs, err := client.Me.GetLibraryPlaylistTracks(
			context.Background(),
			playlist.Id,
			nil)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to fetch apple library playlist songs", err)
			return nil, err
		}

		logger.WithUser(user.GetUserId()).Infof("Found %s apple songs for playlists %s",
			len(librarySongs),
			playlist.Attributes.Name)

		for _, librarySong := range librarySongs {
			allIncompleteSongs = append(allIncompleteSongs, &librarySong)
		}
	}

	return getFullSongsForIncompleteSongs(user, allIncompleteSongs)
}

// Allow us to transform incomplete songs into catalog songs where we can get all info related to a song such as ISRC
func getFullSongsForIncompleteSongs(user *clientcommon.User, librarySongs []*applemusic.Song) ([]*applemusic.Song, error) {
	songByIds := make([]string, 0)

	for _, librarySong := range librarySongs {
		songByIds = append(songByIds, librarySong.Attributes.PlayParams.CatalogId)
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

	storefronts, _, err := client.Me.GetStorefront(
		context.Background(),
		&applemusic.PageOptions{Offset: 0, Limit: maxPage})

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to fetch storefront for apple user ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Found %s storefronts for apple user", len(storefronts.Data))

	// We always take the first one, users should generally only have 1
	storefront := storefronts.Data[0].Id

	allSongs := make([]*applemusic.Song, 0)

	for i := 0; i < len(songIds); i += maxCatalogSongsPerApiCall {
		upperBound := i + maxCatalogSongsPerApiCall

		if upperBound > len(songIds) {
			upperBound = len(songIds)
		}

		songs, _, err := client.Catalog.GetSongsByIds(
			context.Background(),
			storefront,
			songIds,
			nil)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to get apple songs by id", err)
			return nil, err
		}

		for _, song := range songs.Data {
			allSongs = append(allSongs, &song)
		}

		logger.Logger.Infof("Fetched %d apple songs successfully", upperBound-i)
	}

	return allSongs, nil
}
