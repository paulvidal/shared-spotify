package spotify

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"strings"
	"time"
)

var maxPerPage = 50

const maxWaitBetweenCalls = 100 * time.Millisecond
const maxWaitBetweenSearchCalls = 40 * time.Millisecond

func GetAllSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	// Get the liked songs
	logger.WithUser(user.GetUserId()).Info("Fetching all spotify saved songs for user")

	savedTracks, err := getSavedSongs(user)

	if err != nil {
		logger.WithUser(user.GetUserId()).Errorf("Failed to fetch all spotify tracks for user %s %v", user.GetUserId(), err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Info("Successfully fetched all spotify saved songs for user")

	// Get the playlist songs
	logger.WithUser(user.GetUserId()).Info("Fetching all spotify playlist tracks for user")

	playlistTracks, err := getAllPlaylistSongs(user)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to fetch all spotify playlist tracks for user ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Info("Successfully fetched all spotify playlist tracks for user")

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
	savedTrackPage, err := client.CurrentUsersTracksOpt(&spotify.Options{Limit: &maxPerPage})

	if err != nil {
		logger.WithUser(user.GetUserId()).Errorf("Failed to get tracks for user %v", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Playlist has %d total tracks for user", savedTrackPage.Total)

	for page := 1; ; page++ {
		logger.WithUser(user.GetUserId()).Infof("Page %d has %d tracks for user", page, len(savedTrackPage.Tracks))

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

	logger.WithUser(user.GetUserId()).Infof("Found %d saved tracks for user", len(allTracks))

	return allTracks, nil
}

// This method gets all the songs from the playlists of the user
func getAllPlaylistSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	client := user.SpotifyClient

	allTracks := make([]*spotify.FullTrack, 0)

	simplePlaylistPage, err := client.CurrentUsersPlaylistsOpt(&spotify.Options{Limit: &maxPerPage})

	if err != nil {
		logger.WithUser(user.GetUserId()).Errorf("Failed to get playlists for user %v", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("User has %d total playlists for user", simplePlaylistPage.Total)

	for page := 1; ; page++ {
		logger.WithUser(user.GetUserId()).Infof("Page %d has %d playlists for user", page, len(simplePlaylistPage.Playlists))

		// For each playlist, get the associated tracks
		for _, simplePlaylist := range simplePlaylistPage.Playlists {

			// If the playlist is owned by someone else and was just "liked" by the user, do not include it
			if simplePlaylist.Owner.ID != user.GetId() {
				continue
			}

			// FIXME: be more robust to this later, as in case the playlist is renamed, it will still count
			//    although we could say if the user renamed the playlists he want to keep the songs for it
			// If the playlist has the name credits and was created by us, we don't include it
			if strings.Contains(simplePlaylist.Name, clientcommon.NameCredits) {
				continue
			}

			playlistId := simplePlaylist.ID.String()
			tracks, err := getSongsForPlaylist(user, playlistId)

			if err != nil {
				return nil, err
			}

			logger.WithUser(user.GetUserId()).Infof("Got %d tracks from playlist %s for user", len(tracks), playlistId)

			allTracks = append(allTracks, tracks...)
		}

		time.Sleep(maxWaitBetweenCalls)

		// Go to next page
		err = client.NextPage(simplePlaylistPage)

		if err == spotify.ErrNoMorePages {
			break
		}

		if err != nil {
			logger.WithUser(user.GetUserId()).Error(err)
			return nil, err
		}
	}

	logger.WithUser(user.GetUserId()).Infof("Found %d playlist tracks for user", len(allTracks))

	return allTracks, nil
}

func getSongsForPlaylist(user *clientcommon.User, playlistId string) ([]*spotify.FullTrack, error) {
	client := user.SpotifyClient

	allTracks := make([]*spotify.FullTrack, 0)
	playlistTrackPage, err := client.GetPlaylistTracksOpt(spotify.ID(playlistId), &spotify.Options{Limit: &maxPerPage}, "")

	if err != nil {
		logger.WithUser(user.GetUserId()).Errorf("Failed to get tracks for playlist %s for user %v", playlistId, err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Playlist %s has %d total tracks for user", playlistId, playlistTrackPage.Total)

	for page := 1; ; page++ {
		logger.WithUser(user.GetUserId()).Infof("Page %d has %d tracks for playlist %s for user", page,
			len(playlistTrackPage.Tracks), playlistId)

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

func GetTracks(client *spotify.Client, spotifyIds []spotify.ID) ([]*spotify.FullTrack, error) {
	allTracks := make([]*spotify.FullTrack, 0)

	for i := 0; i < len(spotifyIds); i += maxPerPage {
		upperBound := i + maxPerPage

		if upperBound > len(spotifyIds) {
			upperBound = len(spotifyIds)
		}

		tracks, err := client.GetTracks(spotifyIds[i:upperBound]...)

		if err != nil {
			return nil, err
		}

		allTracks = append(allTracks, tracks...)
	}

	return allTracks, nil
}

func GetTrackForISRCs(user *clientcommon.User, isrcs []string) ([]*spotify.FullTrack, error) {
	tracks := make([]*spotify.FullTrack, 0)

	isrcMapping, err := mongoclient.GetIsrcmappings(isrcs)
	tracksToSearch := make([]spotify.ID, 0)

	if err != nil {
		// if we have a mongo error, we continue normally
		isrcMapping = make(map[string]string)
		logger.WithUser(user.GetUserId()).Warning("Failed to get isrc mappings ", err)
	}

	for _, isrc := range isrcs {
		// if we already had the spotify id, we don't make the search call
		if spotifyId, ok := isrcMapping[isrc]; ok {
			tracksToSearch = append(tracksToSearch, spotify.ID(spotifyId))
			continue
		}

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

	// search all the tracks for which we already had the spotify id
	client := GetSpotifyGenericClient()
	foundTracks, err := GetTracks(client, tracksToSearch)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to fetch spotify songs by Id ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Used ISRC cache for %d songs for a total of %d songs",
		len(foundTracks), len(isrcs))
	tracks = append(tracks, foundTracks...)

	logger.WithUser(user.GetUserId()).Infof(
		"Converted %d apple music songs to %d spotify tracks",
		len(isrcs),
		len(tracks))

	return tracks, nil
}