package spotify

import (
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
)

const maxAudioFeaturePerApiCall = 100

func GetAudioFeatures(tracks []*spotify.FullTrack) (map[string]*spotify.AudioFeatures, error) {
	logger.Logger.Infof("Fetching audio features for %d tracks", len(tracks))

	audioFeaturesPerTrack := make(map[string]*spotify.AudioFeatures)
	audioFeatures := make([]*spotify.AudioFeatures, 0)

	trackIds := make([]spotify.ID, 0)
	TrackISCRPerTrackIds := make(map[spotify.ID]string)

	for _, track := range tracks {
		trackISCR, _ := clientcommon.GetTrackISRC(track)
		trackId := track.ID

		trackIds = append(trackIds, track.ID)
		TrackISCRPerTrackIds[trackId] = trackISCR
	}

	// Send the audio features by batch of maxAudioFeaturePerApiCall, as we are limited on the number of audio features
	// we can query at once
	for i := 0; i < len(trackIds); i += maxAudioFeaturePerApiCall {
		upperBound := i + maxAudioFeaturePerApiCall

		if upperBound > len(trackIds) {
			upperBound = len(trackIds)
		}

		// we change client often to spread the load
		client, err := GetSpotifyGenericClient()

		if err != nil {
			return nil, err
		}

		audioFeaturesPart, err := client.GetAudioFeatures(trackIds[i:upperBound]...)

		clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypeAudioFeatures, false, err)

		if err != nil {
			logger.Logger.Errorf("Failed to get audio features for tracks - %v", err)
			return nil, err
		}

		audioFeatures = append(audioFeatures, audioFeaturesPart...)
		logger.Logger.Debugf("Fetched %d track audio features successfully", upperBound-i)
	}

	for _, audioFeature := range audioFeatures {
		trackISCR := TrackISCRPerTrackIds[audioFeature.ID]
		audioFeaturesPerTrack[trackISCR] = audioFeature
	}

	return audioFeaturesPerTrack, nil
}
