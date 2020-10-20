package spotifyclient

import (
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

var maxPage = 50

func (user *User) GetAllSongs() (*[]spotify.FullTrack, error) {
	// Get the liked songs
	savedTracks, err := user.GetSavedSongs()

	if err != nil {
		logger.Logger.Error("Failed to fetch all tracks for user", err)
		return nil, err
	}

	// Get the playlist songs
	playlistTracks, err := user.GetAllPlaylistSongs()

	if err != nil {
		logger.Logger.Error("Failed to fetch all tracks for user", err)
		return nil, err
	}

	// Merge all the songs here
	allTracks := make([]spotify.FullTrack, 0)
	allTracks = append(allTracks, *savedTracks...)
	allTracks = append(allTracks, *playlistTracks...)

	return &allTracks, nil
}

// This method gets all the songs "liked" by a user
func (user *User) GetSavedSongs() (*[]spotify.FullTrack, error) {
	client := user.Client

	allTracks := make([]spotify.FullTrack, 0)
	savedTrackPage, err := client.CurrentUsersTracksOpt(&spotify.Options{Limit: &maxPage})

	if err != nil {
		logger.Logger.Error("Failed to get tracks", err)
		return nil, err
	}

	logger.Logger.Infof("Playlist has %d total tracks", savedTrackPage.Total)

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d tracks", page, len(savedTrackPage.Tracks))

		// Transform all the SavedTrack into FullTrack and add them to the list
		for _, savedTracks := range savedTrackPage.Tracks {
			allTracks = append(allTracks, savedTracks.FullTrack)
		}

		// Go to next page
		err = client.NextPage(savedTrackPage)

		if err == spotify.ErrNoMorePages {
			break
		}

		if err != nil {
			logger.Logger.Error(err)
			return nil, err
		}
	}

	return &allTracks, nil
}

// This method gets all the songs from the playlists of the user
func (user *User) GetAllPlaylistSongs() (*[]spotify.FullTrack, error) {
	client := user.Client

	allTracks := make([]spotify.FullTrack, 0)

	simplePlaylistPage, err := client.CurrentUsersPlaylistsOpt(&spotify.Options{Limit: &maxPage})

	if err != nil {
		logger.Logger.Error("Failed to get playlists", err)
		return nil, err
	}

	logger.Logger.Infof("User has %d total playlists", simplePlaylistPage.Total)

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d playlists", page, len(simplePlaylistPage.Playlists))

		// For each playlist, get the associated tracks
		for _, simplePlaylist := range simplePlaylistPage.Playlists {
			playlistId := simplePlaylist.ID.String()
			tracks, err := user.getSongsForPlaylist(playlistId)

			if err != nil {
				return nil, err
			}

			logger.Logger.Infof("Got %d tracks from playlist %s", len(*tracks), playlistId)

			allTracks = append(allTracks, *tracks...)
		}

		// Go to next page
		err = client.NextPage(simplePlaylistPage)

		if err == spotify.ErrNoMorePages {
			break
		}

		if err != nil {
			logger.Logger.Error(err)
			return nil, err
		}
	}

	return &allTracks, nil
}

func (user *User) getSongsForPlaylist(playlistId string) (*[]spotify.FullTrack, error) {
	client := user.Client

	allTracks := make([]spotify.FullTrack, 0)
	playlistTrackPage, err := client.GetPlaylistTracksOpt(spotify.ID(playlistId), &spotify.Options{Limit: &maxPage}, "")

	if err != nil {
		logger.Logger.Error("Failed to get tracks for playlist %s", playlistId, err)
		return nil, err
	}

	logger.Logger.Infof("Playlist %s has %d total tracks", playlistId, playlistTrackPage.Total)

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d tracks for playlist %s", page, len(playlistTrackPage.Tracks),
			playlistId)

		// Transform all the PlaylistTrack into FullTrack and add them to the list
		for _, playlistTrack := range playlistTrackPage.Tracks {
			allTracks = append(allTracks, playlistTrack.Track)
		}

		// Go to next page
		err = client.NextPage(playlistTrackPage)

		if err == spotify.ErrNoMorePages {
			break
		}

		if err != nil {
			logger.Logger.Error(err)
			return nil, err
		}
	}

	return &allTracks, nil
}