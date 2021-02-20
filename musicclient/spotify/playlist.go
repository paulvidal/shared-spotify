package spotify

import (
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
)

const playlistPublic = false
const maxTrackPerPlaylistAddCall = 100
const spotifyExternalLinkName = "spotify"

func CreatePlaylist(user *clientcommon.User, playlistName string, tracks []*spotify.FullTrack) (*string, error) {
	// we create the playlist
	fullPlaylist, err := user.SpotifyClient.CreatePlaylistForUser(user.GetId(), playlistName,
		clientcommon.PlaylistDescription, playlistPublic)

	clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypePlaylistCreated, true, err)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to created playlist ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Playlist '%s' successfully created for user %s", playlistName, user.GetUserId())

	// we add the tracks
	trackIds := make([]spotify.ID, 0)

	for _, track := range tracks {
		trackIds = append(trackIds, track.ID)
	}

	// Send the track by batch of maxTrackPerPlaylistAddCall, as we are limited on the number of songs we can
	// add at once
	for i := 0; i < len(trackIds); i += maxTrackPerPlaylistAddCall {
		upperBound := i + maxTrackPerPlaylistAddCall

		if upperBound > len(trackIds) {
			upperBound = len(trackIds)
		}

		_, err := user.SpotifyClient.AddTracksToPlaylist(fullPlaylist.ID, trackIds[i:upperBound]...)

		clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypePlaylistSongsAdded, true, err)

		if err != nil {
			logger.WithUser(user.GetUserId()).Errorf("Failed to add songs to playlist %s - %v", playlistName, err)
			return nil, err
		}

		logger.WithUser(user.GetUserId()).Infof("Add %d tracks to Playlist '%s' successfully created for user %s",
			upperBound-i, playlistName, user.GetUserId())
	}

	// get the spotify link to the playlist so we return it
	externalLink, ok := fullPlaylist.ExternalURLs[spotifyExternalLinkName]

	if !ok {
		logger.WithUser(user.GetUserId()).Warningf("No spotify external link for playlist '%s' for user %s",
			playlistName, user.GetUserId())
		return nil, nil
	}

	return &externalLink, nil
}
