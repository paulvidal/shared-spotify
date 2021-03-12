package app

import (
	"errors"
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"runtime/debug"
	"time"
)

const TimeoutTimeMinRoomProcessing = 20  // 20min before we can re trigger a processing

var ErrorPlaylistTypeNotFound = errors.New("playlist type id not found")

type SharedMusicLibrary struct {
	TotalUsers             int                        `json:"total_users"`
	ProcessingStatus       *ProcessingStatus          `json:"processing_status"`
	MusicProcessingChannel chan MusicProcessingResult `json:"-"`
	CommonPlaylists        *CommonPlaylists           `json:"-"`
}

type ProcessingStatus struct {
	TotalToProcess   int       `json:"total_to_process"`
	AlreadyProcessed int       `json:"already_processed"`
	Started          bool      `json:"started"`
	StartedAt        time.Time `json:"started_at"`
	CheckpointTime   time.Time `json:"checkpoint_time"`  // time for the last time we got an update
	Success          *bool     `json:"success"`
}

func (musicLibrary *SharedMusicLibrary) SetProcessingSuccess(success *bool) {
	musicLibrary.ProcessingStatus.Success = success
}

func (musicLibrary *SharedMusicLibrary) HasProcessingFailed() bool {
	return musicLibrary.ProcessingStatus.Success != nil && !(*musicLibrary.ProcessingStatus.Success)
}

func (musicLibrary *SharedMusicLibrary) HasProcessingFinished() bool {
	return musicLibrary.ProcessingStatus.Success != nil
}

func (musicLibrary *SharedMusicLibrary) HasTimedOut() bool {
	return !musicLibrary.HasProcessingFinished() &&
		time.Now().Sub(musicLibrary.ProcessingStatus.CheckpointTime).Minutes() > TimeoutTimeMinRoomProcessing
}

func (musicLibrary *SharedMusicLibrary) GetProcessingTime() float64 {
	return musicLibrary.ProcessingStatus.CheckpointTime.Sub(musicLibrary.ProcessingStatus.StartedAt).Seconds()
}

func (musicLibrary *SharedMusicLibrary) GetPlaylist(id string) (*Playlist, error) {
	playlist, ok := musicLibrary.CommonPlaylists.Playlists[id]

	if !ok {
		return nil, ErrorPlaylistTypeNotFound
	}

	return playlist, nil
}

type MusicProcessingResult struct {
	User   *clientcommon.User
	Tracks []*spotify.FullTrack
	Error  error
}

func CreateSharedMusicLibrary(totalUsers int) *SharedMusicLibrary {
	return &SharedMusicLibrary{
		totalUsers,
		// we add 1 for total to process so we never reach 100% once we fetched all songs
		// and are in the processing phase
		&ProcessingStatus{
			totalUsers + 1,
			0,
			false,
			time.Now(),
			time.Now(),
			nil},
		make(chan MusicProcessingResult, totalUsers), // Channel needs to be only as big as the number of users
		nil,
	}
}

/*
  These are the Go routine functions to process the shared music library
*/

// Will process the common library and find all the common songs
func (musicLibrary *SharedMusicLibrary) Process(users []*clientcommon.User, notifyProcessingOver func(success bool),
	saveMusicLibrary func() error) error {
	logger.Logger.Infof("Starting processing of room for all users")

	// We mark the processing status as started
	musicLibrary.ProcessingStatus.Started = true
	err := saveMusicLibrary()

	if err != nil {
		logger.Logger.Error("Failed to save processing started ", err)
		return err
	}

	// We create the common playlists
	musicLibrary.CommonPlaylists = CreateCommonPlaylists()

	for _, user := range users {
		// launch one routine per user to fetch all the songs
		logger.WithUser(user.GetUserId()).Infof("Launching processing for user %s", user.GetUserId())
		go musicLibrary.fetchSongsForUser(user)
	}

	// launch a single routine to wait for the songs from users, add them to the library and the fidn the most commons
	logger.Logger.Infof("Launching processing gatherer of information")
	go musicLibrary.addSongsToLibraryAndFindMostCommonSongs(notifyProcessingOver, saveMusicLibrary)

	return nil
}

func (musicLibrary *SharedMusicLibrary) fetchSongsForUser(user *clientcommon.User) {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.WithUser(user.GetUserId()).Errorf("An unknown error happened while fetching song for "+
				"user %s - error: %s", user.GetUserId(), err, string(debug.Stack()))
			fmt.Println(string(debug.Stack()))

			musicLibrary.MusicProcessingChannel <- MusicProcessingResult{user, nil, errors.New("")}
		}
	}()

	logger.WithUser(user.GetUserId()).Infof("Fetching songs for user %s", user.GetUserId())

	tracks, err := musicclient.GetAllSongs(user)

	if err != nil {
		logger.WithUser(user.GetUserId()).Errorf("Failed to fetch all songs for user %s %v",
			user.GetUserId(), err)
	} else {
		logger.WithUser(user.GetUserId()).Infof("Fetching songs for user %s finished successfully with %d"+
			" tracks found", user.GetUserId(), len(tracks))
	}

	// We send in the channel the result after processing the music for this user
	musicLibrary.MusicProcessingChannel <- MusicProcessingResult{user, tracks, err}
}

func (musicLibrary *SharedMusicLibrary) addSongsToLibraryAndFindMostCommonSongs(notifyProcessingOver func(success bool),
	saveMusicLibrary func() error) {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.Logger.Errorf(
				"An unknown error happened while adding song and finding common songs - error %s \n%s",
				err, string(debug.Stack()))
			fmt.Println(string(debug.Stack()))

			// we notify that the processing is over
			notifyProcessingOver(false)
		}
	}()

	logger.Logger.Infof("Starting to wait for music processing results")

	success := true

	for {
		if musicLibrary.ProcessingStatus.AlreadyProcessed == musicLibrary.ProcessingStatus.TotalToProcess - 1 {
			// we break once we have received a message for every user
			break
		}

		// We receive from the channel a messages for each user
		musicProcessingResult := <-musicLibrary.MusicProcessingChannel
		user := musicProcessingResult.User
		userId := user.GetUserId()

		logger.WithUser(user.GetUserId()).Infof("Received music processing result for user %s", userId)

		if musicProcessingResult.Error != nil {
			logger.WithUser(user.GetUserId()).Infof("Music processing failed for user %s %v",
				userId, musicProcessingResult.Error)

			// We mark the processing result as failed
			success = false

		} else {
			logger.WithUser(user.GetUserId()).Infof("Music processing succeeded for user %s, finding %d tracks",
				userId, len(musicProcessingResult.Tracks))

			// we add the tracks as all went fine
			musicLibrary.CommonPlaylists.addTracks(musicProcessingResult.User, musicProcessingResult.Tracks)
		}

		// Mark one user's music as processed
		musicLibrary.ProcessingStatus.AlreadyProcessed += 1

		// we mark the change in mongo
		_ = saveMusicLibrary()
	}

	logger.Logger.Infof("All music processing results received - success=%t", success)

	if success {
		// if everything went well, we now generate the playlists for the users in the room
		err := musicLibrary.CommonPlaylists.GeneratePlaylists()

		if err != nil {
			logger.Logger.Error("An error when generating playlists occurred ", err)
			success = false
		}
	}

	// we add the last step done once all the processing is good
	musicLibrary.ProcessingStatus.AlreadyProcessed += 1
	_ = saveMusicLibrary()

	// we notify that the processing is over
	notifyProcessingOver(success)
}
