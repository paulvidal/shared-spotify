package clientcommon

import "github.com/zmb3/spotify"

func GetTrackISRC(track *spotify.FullTrack) (string, bool) {
	// Unique id representing a track
	// https://en.wikipedia.org/wiki/International_Standard_Recording_Code
	trackId, ok := track.ExternalIDs["isrc"]

	return trackId, ok
}
