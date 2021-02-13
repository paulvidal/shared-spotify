package spotify

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"time"
)

var maxPage = 50

const maxWaitBetweenCalls = 100 * time.Millisecond
const maxWaitBetweenSearchCalls = 40 * time.Millisecond

func GetAllSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	// Get the liked songs
	savedTracks, err := getSavedSongs(user)

	if err != nil {
		logger.Logger.Errorf("Failed to fetch all tracks for user %s %v", user.GetUserId(), err)
		return nil, err
	}

	// Get the playlist songs
	playlistTracks, err := getAllPlaylistSongs(user)

	if err != nil {
		logger.Logger.Errorf("Failed to fetch all tracks for user %s %v", user.GetUserId(), err)
		return nil, err
	}

	// Merge all the songs here
	allTracks := make([]*spotify.FullTrack, 0)
	allTracks = append(allTracks, savedTracks...)
	allTracks = append(allTracks, playlistTracks...)

	return allTracks, nil
}

// This method gets all the songs "liked" by a user
func getSavedSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	client := user.SpotifyClient

	allTracks := make([]*spotify.FullTrack, 0)
	savedTrackPage, err := client.CurrentUsersTracksOpt(&spotify.Options{Limit: &maxPage})

	if err != nil {
		logger.Logger.Errorf("Failed to get tracks for user %s %v", user.GetUserId(), err)
		return nil, err
	}

	logger.Logger.Infof("Playlist has %d total tracks for user %s", savedTrackPage.Total, user.GetUserId())

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d tracks for user %s", page, len(savedTrackPage.Tracks),
			user.GetUserId())

		// Transform all the SavedTrack into FullTrack and add them to the list
		for _, savedTrack := range savedTrackPage.Tracks {
			fullTrack := savedTrack.FullTrack
			allTracks = append(allTracks, &fullTrack)
		}

		time.Sleep(maxWaitBetweenCalls)

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

	logger.Logger.Infof("Found %d saved tracks for user %s", len(allTracks), user.GetUserId())

	return allTracks, nil
}

// This method gets all the songs from the playlists of the user
func getAllPlaylistSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	client := user.SpotifyClient

	allTracks := make([]*spotify.FullTrack, 0)

	simplePlaylistPage, err := client.CurrentUsersPlaylistsOpt(&spotify.Options{Limit: &maxPage})

	if err != nil {
		logger.Logger.Errorf("Failed to get playlists for user %s %v", user.GetUserId(), err)
		return nil, err
	}

	logger.Logger.Infof("User has %d total playlists for user %s", simplePlaylistPage.Total, user.GetUserId())

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d playlists for user %s", page, len(simplePlaylistPage.Playlists),
			user.GetUserId())

		// For each playlist, get the associated tracks
		for _, simplePlaylist := range simplePlaylistPage.Playlists {

			// If the playlist is owned by someone else and was just "liked" by the user, do not include it
			if simplePlaylist.Owner.ID != user.GetId() {
				continue
			}

			playlistId := simplePlaylist.ID.String()
			tracks, err := getSongsForPlaylist(user, playlistId)

			if err != nil {
				return nil, err
			}

			logger.Logger.Infof("Got %d tracks from playlist %s for user %s", len(tracks), playlistId,
				user.GetUserId())

			allTracks = append(allTracks, tracks...)
		}

		time.Sleep(maxWaitBetweenCalls)

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

	logger.Logger.Infof("Found %d playlist tracks for user %s", len(allTracks), user.GetUserId())

	return allTracks, nil
}

func getSongsForPlaylist(user *clientcommon.User, playlistId string) ([]*spotify.FullTrack, error) {
	client := user.SpotifyClient

	allTracks := make([]*spotify.FullTrack, 0)
	playlistTrackPage, err := client.GetPlaylistTracksOpt(spotify.ID(playlistId), &spotify.Options{Limit: &maxPage}, "")

	if err != nil {
		logger.Logger.Errorf("Failed to get tracks for playlist %s for user %s %v", playlistId,
			user.GetUserId(), err)
		return nil, err
	}

	logger.Logger.Infof("Playlist %s has %d total tracks for user %s", playlistId, playlistTrackPage.Total,
		user.GetUserId())

	for page := 1; ; page++ {
		logger.Logger.Infof("Page %d has %d tracks for playlist %s for user %s", page,
			len(playlistTrackPage.Tracks), playlistId, user.GetUserId())

		// Transform all the PlaylistTrack into FullTrack and add them to the list
		for _, playlistTrack := range playlistTrackPage.Tracks {
			fullTrack := playlistTrack.Track
			allTracks = append(allTracks, &fullTrack)
		}

		// TODO: remove this, we need rate limit in another way
		time.Sleep(maxWaitBetweenCalls)

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

	return allTracks, nil
}

func GetTrackForISRCs(user *clientcommon.User, isrcs []string) ([]*spotify.FullTrack, error) {
	tracks := make([]*spotify.FullTrack, 0)

	for _, isrc := range isrcs {
		// we change client often to spread the load and not be rate limited
		client := GetSpotifyGenericClient()

		isrcQuery := fmt.Sprintf("isrc:%s", isrc)
		results, err := client.Search(isrcQuery, spotify.SearchTypeTrack)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to query track by isrc on spotify", err)
			continue
		}

		trackResults := results.Tracks.Tracks

		if len(trackResults) == 0 {
			logger.WithUser(user.GetUserId()).Warningf("No track found on spotify for isrc: %s", isrc)
			continue
		}

		// Always take the first one, we don't care as we compare later via ISRC
		track := trackResults[0]

		tracks = append(tracks, &track)

		// TODO: remove this, we need rate limit in another way
		time.Sleep(maxWaitBetweenSearchCalls)
	}

	logger.WithUser(user.GetUserId()).Infof(
		"Converted %d apple music songs to %d spotify tracks",
		len(isrcs),
		len(tracks))

	return tracks, nil
}