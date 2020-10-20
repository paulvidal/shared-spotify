package spotifyclient

import (
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

var maxPage = 50

func (user *User) GetSongs() (*[]spotify.SavedTrack, error) {
	client := user.Client

	allTracks := make([]spotify.SavedTrack, 0)
	savedTrackPage, err := client.CurrentUsersTracksOpt(&spotify.Options{Limit: &maxPage})

	if err != nil {
		logger.Logger.Error("Failed to get tracks", err)
		return nil, err
	}

	logger.Logger.Infof("Playlist has %d total tracks", savedTrackPage.Total)

	for page := 1; ; page++ {
		logger.Logger.Infof(" Page %d has %d tracks", page, len(savedTrackPage.Tracks))
		allTracks = append(allTracks, savedTrackPage.Tracks...)

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
