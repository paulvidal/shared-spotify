package spotify

import (
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
)

const maxAlbumsPerApiCall = 20

func GetAlbums(tracks []*spotify.FullTrack) (map[string]*spotify.FullAlbum, error) {
	logger.Logger.Infof("Fetching albums for %d tracks", len(tracks))

	albumsPerTrack := make(map[string]*spotify.FullAlbum)
	albums := make([]*spotify.FullAlbum, 0)

	albumIds := make([]spotify.ID, 0)
	albumsSeen := make(map[spotify.ID]bool) // this acts like a set to know if we added an album or not
	TrackISCRsPerAlbumId := make(map[spotify.ID][]string)

	for _, track := range tracks {
		trackISCR, _ := clientcommon.GetTrackISRC(track)
		albumId := track.Album.ID

		// we only add the albumId if we have not seen it already
		if _, seen := albumsSeen[albumId]; !seen {
			albumIds = append(albumIds, albumId)
			albumsSeen[albumId] = true
		}

		// albums can have multiple tracks
		trackISCRs := TrackISCRsPerAlbumId[albumId]
		if trackISCRs == nil {
			trackISCRs = make([]string, 1)
			trackISCRs[0] = trackISCR
		} else {
			trackISCRs = append(trackISCRs, trackISCR)
		}

		TrackISCRsPerAlbumId[albumId] = trackISCRs
	}

	// Send the album query by batch of maxAlbumsPerApiCall, as we are limited on the number of albums
	// we can query at once
	for i := 0; i < len(albumIds); i += maxAlbumsPerApiCall {
		upperBound := i + maxAlbumsPerApiCall

		if upperBound > len(albumIds) {
			upperBound = len(albumIds)
		}

		// we change client often to spread the load
		client := GetSpotifyGenericClient()

		albumsPart, err := client.GetAlbums(albumIds[i:upperBound]...)

		clientcommon.SendRequestMetric(datadog.SpotifyRequest, datadog.RequestTypeAlbums, false, err)

		if err != nil {
			logger.Logger.Errorf("Failed to get albums - %v", err)
			return nil, err
		}

		albums = append(albums, albumsPart...)
		logger.Logger.Infof("Fetched %d albums successfully", upperBound-i)
	}

	for _, album := range albums {
		trackISCRs := TrackISCRsPerAlbumId[album.ID]

		for _, trackISCR := range trackISCRs {
			albumsPerTrack[trackISCR] = album
		}
	}

	return albumsPerTrack, nil
}
