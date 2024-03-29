package spotify

import (
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
)

const maxArtistsPerApiCall = 50

func GetArtists(tracks []*spotify.FullTrack) (map[string][]*spotify.FullArtist, error) {
	logger.Logger.Infof("Fetching artists for %d tracks", len(tracks))

	artistsPerTrack := make(map[string][]*spotify.FullArtist)
	artists := make([]*spotify.FullArtist, 0)

	artistIds := make([]spotify.ID, 0)
	artistSeen := make(map[spotify.ID]bool) // this acts like a set to know if we added an artist or not
	TrackISCRsPerArtistId := make(map[spotify.ID][]string)

	for _, track := range tracks {
		for _, artist := range track.Artists {
			trackISCR, _ := clientcommon.GetTrackISRC(track)
			artistId := artist.ID

			// we only add the artistId if we have not seen it already
			if _, seen := artistSeen[artistId]; !seen {
				artistIds = append(artistIds, artistId)
				artistSeen[artistId] = true
			}

			// artist can have multiple tracks
			trackISCRs := TrackISCRsPerArtistId[artistId]
			if trackISCRs == nil {
				trackISCRs = make([]string, 1)
				trackISCRs[0] = trackISCR
			} else {
				trackISCRs = append(trackISCRs, trackISCR)
			}

			TrackISCRsPerArtistId[artistId] = trackISCRs
		}
	}

	// Send the artists query by batch of maxArtistsPerApiCall, as we are limited on the number of artists
	// we can query at once
	for i := 0; i < len(artistIds); i += maxArtistsPerApiCall {
		upperBound := i + maxArtistsPerApiCall

		if upperBound > len(artistIds) {
			upperBound = len(artistIds)
		}

		// we change client often to spread the load
		client, err := GetSpotifyGenericClient()

		if err != nil {
			return nil, err
		}

		artistsPart, err := client.GetArtists(artistIds[i:upperBound]...)

		clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypeArtists, false, err)

		if err != nil {
			logger.Logger.Errorf("Failed to get artists - %v", err)
			return nil, err
		}

		artists = append(artists, artistsPart...)
		logger.Logger.Debugf("Fetched %d artists successfully", upperBound-i)
	}

	for _, artist := range artists {
		trackISCRs := TrackISCRsPerArtistId[artist.ID]

		for _, trackISCR := range trackISCRs {
			artistsForTrack := artistsPerTrack[trackISCR]

			if artistsForTrack == nil {
				artistsForTrack = make([]*spotify.FullArtist, 1)
				artistsForTrack[0] = artist
			} else {
				artistsForTrack = append(artistsForTrack, artist)
			}

			artistsPerTrack[trackISCR] = artistsForTrack
		}
	}

	return artistsPerTrack, nil
}
