package spotify

import (
	"context"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const playlistPublic = false
const maxTrackPerPlaylistAddCall = 100
const spotifyExternalLinkName = "spotify"

func CreatePlaylist(user *clientcommon.User, playlistName string, tracks []*spotify.FullTrack, ctx context.Context) (*string, error) {
	rootSpan, rootCtx := tracer.StartSpanFromContext(ctx, "playlist.create.spotify")
	defer rootSpan.Finish()

	// we create the playlist
	span, ctx := tracer.StartSpanFromContext(rootCtx, "playlist.create.spotify.empty")
	fullPlaylist, err := user.SpotifyClient.CreatePlaylistForUser(user.GetId(), playlistName,
		clientcommon.PlaylistDescription, playlistPublic)

	clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypePlaylistCreated, true, err)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to created playlist ", err)
		span.Finish(tracer.WithError(err))
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Playlist '%s' successfully created for user %s", playlistName, user.GetUserId())
	span.Finish()

	// we add the tracks
	trackIds := make([]spotify.ID, 0)

	for _, track := range tracks {
		trackIds = append(trackIds, track.ID)
	}

	span, ctx = tracer.StartSpanFromContext(rootCtx, "playlist.create.spotify.add.tracks")

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
			logger.
				WithUser(user.GetUserId()).
				WithError(err).
				Errorf("Failed to add songs to playlist %s %v", playlistName, span)
			span.Finish(tracer.WithError(err))
			return nil, err
		}

		logger.
			WithUser(user.GetUserId()).
			Debugf("Add %d tracks to Playlist '%s' successfully created for user %v",
				upperBound-i, playlistName, span)
	}

	logger.
		WithUser(user.GetUserId()).
		Infof("Added %d tracks to playlist %s for user %v", len(trackIds), playlistName, span)
	span.Finish()

	// get the spotify link to the playlist so we return it
	externalLink, ok := fullPlaylist.ExternalURLs[spotifyExternalLinkName]

	if !ok {
		logger.
			WithUser(user.GetUserId()).
			Warningf("No spotify external link for playlist '%s' for user %v", playlistName, span)
		return nil, nil
	}

	return &externalLink, nil
}
