package app

import (
	"errors"
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"github.com/zmb3/spotify"
	"runtime/debug"
)

var errorPlaylistNotFound = errors.New("playlist id not found")

type SharedMusicLibrary struct {
	TotalUsers             int                               `json:"total_users"`
	ProcessingStatus       *ProcessingStatus                 `json:"processing_status"`
	MusicProcessingChannel chan MusicProcessingResult        `json:"-"`
	CommonPlaylists         *CommonPlaylists                 `json:"common_playlists"`
}

type ProcessingStatus struct {
	TotalToProcess        int    `json:"total_to_process"`
	AlreadyProcessed      int   `json:"already_processed"`
	Started               bool  `json:"started"`
	Success               *bool  `json:"success"`
}

func (musicLibrary *SharedMusicLibrary) hasProcessingFailed() bool {
	return musicLibrary.ProcessingStatus.Success != nil && !(*musicLibrary.ProcessingStatus.Success)
}

func (musicLibrary *SharedMusicLibrary) hasProcessingFinished() bool {
	return musicLibrary.ProcessingStatus.Success != nil
}

func (musicLibrary *SharedMusicLibrary) GetPlaylist(id string) (*Playlist, error) {
	playlist, ok := musicLibrary.CommonPlaylists.PlaylistsFound[id]

	if !ok {
		return nil, errorPlaylistNotFound
	}

	return playlist, nil
}

type MusicProcessingResult struct {
	User   *spotifyclient.User
	Tracks []*spotify.FullTrack
	Error  error
}

func CreateSharedMusicLibrary(totalUsers int) *SharedMusicLibrary {
	return &SharedMusicLibrary{
		totalUsers,
		&ProcessingStatus{totalUsers, 0, false, nil},
		make(chan MusicProcessingResult, totalUsers), // Channel needs to be only as big as the number of users
		nil,
	}
}

/*
  These are the Go routine functions to process the shared music library
 */

// Will process the common library and find all the common songs
func (musicLibrary *SharedMusicLibrary) Process(users []*spotifyclient.User) {
	logger.Logger.Infof("Starting processing of room for all users")
	
	// We mark the processing status as started
	musicLibrary.ProcessingStatus.Started = true

	// We create the common playlists
	musicLibrary.CommonPlaylists = CreateCommonPlaylists()

	for _, user := range users {
		// launch one routine per user to fetch all the songs
		logger.Logger.Infof("Launching processing for users %s", user.GetUserId())
		go musicLibrary.fetchSongsForUser(user)
	}

	// launch a single routine to wait for the songs from users, add them to the library and the fidn the most commons
	logger.Logger.Infof("Launching processing gatherer of information")
	go musicLibrary.addSongsToLibraryAndFindMostCommonSongs()
}

func (musicLibrary *SharedMusicLibrary) fetchSongsForUser(user *spotifyclient.User)  {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.Logger.Errorf("An unknown error happened while fetching song for user %s - error: %s",
				user.GetUserId(), err, string(debug.Stack()))
			fmt.Println(string(debug.Stack()))

			musicLibrary.MusicProcessingChannel <- MusicProcessingResult{user, nil, errors.New("")}
		}
	}()

	logger.Logger.Infof("Fetching songs for user %s",user.GetUserId())

	tracks, err := user.GetAllSongs()

	if err != nil {
		logger.Logger.Errorf("Failed to fetch all songs for user %s %v", user.GetUserId(), err)
	} else  {
		logger.Logger.Infof("Fetching songs for user %s finished successfully with %d tracks found",
			user.GetUserId(), len(tracks))
	}

	// We send in the channel the result after processing the music for this user
	musicLibrary.MusicProcessingChannel <- MusicProcessingResult{user, tracks, err}
}

func (musicLibrary *SharedMusicLibrary) addSongsToLibraryAndFindMostCommonSongs() {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.Logger.Errorf(
				"An unknown error happened while adding song and finding common songs - error %s \n%s",
				err, string(debug.Stack()))
			fmt.Println(string(debug.Stack()))

			success := false
			musicLibrary.ProcessingStatus.Success = &success
		}
	}()

	logger.Logger.Infof("Starting to wait for music processing results")

	success := true

	for {
		if musicLibrary.ProcessingStatus.AlreadyProcessed == musicLibrary.ProcessingStatus.TotalToProcess {
			// we break once we have received a message for every user
			break
		}

		// We receive from the channel a messages for each user
		musicProcessingResult := <- musicLibrary.MusicProcessingChannel
		userId := musicProcessingResult.User.GetUserId()

		logger.Logger.Infof("Received music processing result for user %s", userId)

		if musicProcessingResult.Error != nil {
			logger.Logger.Infof("Music processing failed for user %s %v", userId, musicProcessingResult.Error)

			// We mark the processing result as failed
			success = false

		} else {
			logger.Logger.Infof("Music processing succeeded for user %s, finding %d tracks", userId,
				len(musicProcessingResult.Tracks))

			// we add the tracks as all went fine
			musicLibrary.CommonPlaylists.addTracks(musicProcessingResult.User, musicProcessingResult.Tracks)
		}

		// Mark one user's music as processed
		musicLibrary.ProcessingStatus.AlreadyProcessed += 1
	}

	logger.Logger.Infof("All music processing results received - success=%t", success)

	if success {
		// if everything went well, we now generate the common playlist for the users in the room
		musicLibrary.CommonPlaylists.GenerateCommonPlaylists()
	}

	musicLibrary.ProcessingStatus.Success = &success
}