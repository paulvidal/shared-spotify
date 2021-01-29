package applemusic

import (
	"context"
	"fmt"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
)

const maxISRCPerApiCall = 25
const maxTrackPerPlaylistAddCall = 100

func CreatePlaylist(user *clientcommon.User, playlistName string, tracks []*spotify.FullTrack) (*string, error) {
	client := user.AppleMusicClient

	// we get the storefront
	storefront, err := GetStorefront(user)

	if err != nil {
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

		songs, _, err := client.Catalog.GetSongsByIsrcs(
			context.Background(),
			*storefront,
			trackIsrcs[i:upperBound],
			nil)

		if err != nil {
			logger.WithUser(user.GetUserId()).Error("Failed to get apple songs by id to add to playlist", err)
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

		logger.Logger.Infof("Fetched %d apple songs successfully to add to playlist", upperBound-i)
	}

	playlists, _, err := client.Me.CreateLibraryPlaylist(
		context.Background(),
		applemusic.CreateLibraryPlaylist{
			applemusic.CreateLibraryPlaylistAttributes{playlistName, clientcommon.PlaylistDescription},
			nil},
			nil,
		)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to created apple music playlist ", err)
		return nil, err
	}

	playlist := playlists.Data[0]

	logger.WithUser(user.GetUserId()).Infof("Playlist '%s' successfully created for user %s", playlistName, user.GetUserId())

	// we add the tracks
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

		_, err := client.Me.AddLibraryTracksToPlaylist(
			context.Background(),
			playlist.Id,
			applemusic.CreateLibraryPlaylistTrackData{Data: tracksToAdd[i:upperBound]})

		if err != nil {
			logger.WithUser(user.GetUserId()).Errorf("Failed to add songs to playlist %s - %v", playlistName, err)
			return nil, err
		}

		logger.WithUser(user.GetUserId()).Infof("Add %d tracks to Playlist '%s' successfully created for user %s",
			upperBound-i, playlistName, user.GetUserId())
	}

	// get the spotify link to the playlist so we return it
	externalLink := fmt.Sprintf("https://music.apple.com/library/playlist/%s", playlist.Id)

	return &externalLink, nil
}
