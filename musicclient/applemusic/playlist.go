package applemusic

import (
	"context"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const maxISRCPerApiCall = 15
const maxTrackPerPlaylistAddCall = 100
const maxRetryAddSongs = 3

func CreatePlaylist(user *clientcommon.User, playlistName string, tracks []*spotify.FullTrack, ctx context.Context) (*string, error) {
	rootSpan, rootCtx := tracer.StartSpanFromContext(ctx, "playlist.create.applemusic")
	defer rootSpan.Finish()

	client := user.AppleMusicClient

	// we get the storefront
	storefront, err := GetStorefront(user)

	if err != nil {
		rootSpan.Finish(tracer.WithError(err))
		return nil, err
	}

	// we create the isrc mapping to be able later to select the best songs
	trackToISRC := make(map[string]*spotify.FullTrack)

	for _, track := range tracks {
		t := track
		isrc, ok := clientcommon.GetTrackISRC(t)

		if !ok {
			continue
		}

		trackToISRC[isrc] = t
	}

	// we fetch the songs
	span, ctx := tracer.StartSpanFromContext(rootCtx, "playlist.create.applemusic.convert")
	allSongs := make(map[string]*applemusic.Song, 0)

	for i := 0; i < len(tracks); i += maxISRCPerApiCall {
		upperBound := i + maxISRCPerApiCall

		if upperBound > len(tracks) {
			upperBound = len(tracks)
		}

		trackIsrcs := make([]string, 0)

		for _, track := range tracks {
			isrc, ok := clientcommon.GetTrackISRC(track)

			if !ok {
				continue
			}

			trackIsrcs = append(trackIsrcs, isrc)
		}

		songs, err := GetsongsByIsrc(
			user,
			*storefront,
			trackIsrcs[i:upperBound])

		if err != nil {
			span.Finish(tracer.WithError(err))
			logger.
				WithUser(user.GetUserId()).
				WithError(err).
				Errorf("Failed to get apple songs by id to add to playlist %v", span)
			return nil, err
		}

		for _, s := range songs.Data {
			song := s

			if _, ok := allSongs[song.Attributes.ISRC]; !ok {
				allSongs[song.Attributes.ISRC] = &song

			} else {
				track := trackToISRC[song.Attributes.ISRC]

				// make sure it has the play params
				if song.Attributes.PlayParams != nil {
					allSongs[song.Attributes.ISRC] = &song
				}

				if song.Attributes.PlayParams != nil && song.Attributes.AlbumName == track.Album.Name {
					// if we see a second song that has the same ISRC, we take the one that has the most similar
					// album name
					allSongs[song.Attributes.ISRC] = &song
				}
			}
		}

		logger.Logger.Infof("Fetched %d apple songs successfully to add to playlist %v", upperBound-i, span)
	}
	span.Finish()

	span, ctx = tracer.StartSpanFromContext(rootCtx, "playlist.create.applemusic.empty")

	playlists, _, err := client.Me.CreateLibraryPlaylist(
		ctx,
		applemusic.CreateLibraryPlaylist{
			Attributes: applemusic.CreateLibraryPlaylistAttributes{Name: playlistName, Description: clientcommon.PlaylistDescription}},
		nil,
	)

	clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypePlaylistCreated, true, err)

	if err != nil {
		logger.
			WithUser(user.GetUserId()).
			WithError(err).
			Error("Failed to created apple music playlist %v", span)
		span.Finish(tracer.WithError(err))
		return nil, err
	}

	playlist := playlists.Data[0]

	logger.WithUser(user.GetUserId()).Infof("Playlist '%s' successfully created for user %v", playlistName, span)
	span.Finish()

	// we add the tracks
	span, ctx = tracer.StartSpanFromContext(rootCtx, "playlist.create.applemusic.add.tracks")
	tracksToAdd := make([]applemusic.CreateLibraryPlaylistTrack, 0)

	for _, song := range allSongs {
		tracksToAdd = append(tracksToAdd, applemusic.CreateLibraryPlaylistTrack{Id: song.Id, Type: "music"})
	}

	// Send the track by batch of maxTrackPerPlaylistAddCall, as we are limited on the number of songs we can
	// add at once
	for i := 0; i < len(tracksToAdd); i += maxTrackPerPlaylistAddCall {
		upperBound := i + maxTrackPerPlaylistAddCall

		if upperBound > len(tracksToAdd) {
			upperBound = len(tracksToAdd)
		}

		var err error
		retryCount := 0

		// Add retry as apple endpoint to add songs is not super resilient and often sends back 5xx
		for retryCount <= maxRetryAddSongs {
			_, err = client.Me.AddLibraryTracksToPlaylist(
				ctx,
				playlist.Id,
				applemusic.CreateLibraryPlaylistTrackData{Data: tracksToAdd[i:upperBound]})

			clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypePlaylistSongsAdded, true, err)

			if err == nil {
				break
			}

			retryCount += 1
			logger.
				WithUser(user.GetUserId()).
				WithError(err).
				Warningf("Failed to add songs to playlist %s - retryCount=%d %v", playlistName, retryCount, span)
		}

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
		Infof("Added %d tracks to playlist %s for user %v", len(tracksToAdd), playlistName, span)
	span.Finish()

	// FIXME: we cannot get straight way the public link to the playlist as apple indexes it later
	//   for this reason, we can only redirect the user at best to is apple music library where he will find the playlist
	externalLink := "https://music.apple.com/library"

	return &externalLink, nil
}
