package app

import (
	"context"
	"errors"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/zmb3/spotify"
	"runtime/debug"
	"time"
)

const TimeoutRoomProcessing = 18 * time.Minute      // wait 18min before we kill all processing linked to room
const TimeoutRoomForReProcessing = 20 * time.Minute // 20min before we can re trigger a processing

var ErrorPlaylistTypeNotFound = errors.New("playlist type id not found")

type SharedMusicLibrary struct {
	TotalUsers           int                      `json:"total_users"`
	ProcessingStatus     *ProcessingStatus        `json:"processing_status"`
	MusicFetchingChannel chan MusicFetchingResult `json:"-"`
	MusicProcessingChannel chan MusicProcessingResult `json:"-"`
	CommonPlaylists      *CommonPlaylists         `json:"-"`
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

func (musicLibrary *SharedMusicLibrary) HasProcessingSucceeded() bool {
	return musicLibrary.ProcessingStatus.Success != nil && *musicLibrary.ProcessingStatus.Success
}

func (musicLibrary *SharedMusicLibrary) HasProcessingFinished() bool {
	return musicLibrary.ProcessingStatus.Success != nil
}

func (musicLibrary *SharedMusicLibrary) HasTimedOut() bool {
	return !musicLibrary.HasProcessingFinished() &&
		time.Now().Sub(musicLibrary.ProcessingStatus.CheckpointTime) > TimeoutRoomForReProcessing
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

type MusicFetchingResult struct {
	User   *clientcommon.User
	Tracks []*spotify.FullTrack
	Error  error
}

type MusicProcessingResult struct {
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
		make(chan MusicFetchingResult, totalUsers), // Channel needs to be only as big as the number of users
		make(chan MusicProcessingResult, 1), // only 1 message in this channel
		nil,
	}
}

/*
  These are the Go routine functions to process the shared music library
*/

// Will process the common library and find all the common songs
func (musicLibrary *SharedMusicLibrary) Process(room *Room, notifyProcessingOver func(success bool),
	saveMusicLibrary func() error, ctx context.Context) error {
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

	for _, user := range room.Users {
		// launch one routine per user to fetch all the songs
		logger.WithUser(user.GetUserId()).Infof("Launching processing for user %s", user.GetUserId())
		go musicLibrary.fetchSongsForUser(room, user, ctx)
	}

	// launch a single routine to wait for the songs from users, add them to the library and the fidn the most commons
	logger.Logger.Infof("Launching processing gatherer of information")
	go musicLibrary.addSongsToLibraryAndFindMostCommonSongs(room, notifyProcessingOver, saveMusicLibrary, ctx)

	return nil
}

func (musicLibrary *SharedMusicLibrary) fetchSongsForUser(room *Room, user *clientcommon.User, ctx context.Context) {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.WithUser(user.GetUserId()).Errorf("An unknown error happened while fetching song for "+
				"user %s - error: %s \n%s", user.GetUserId(), err, string(debug.Stack()))
			musicLibrary.MusicFetchingChannel <- MusicFetchingResult{user, nil, errors.New("Unknown error")}
		}
	}()

	logger.WithUser(user.GetUserId()).Infof("Fetching songs for user %s", user.GetUserId())

	tracks, err := musicclient.GetAllSongs(user)

	if err != nil {
		logger.WithUserAndRoom(user.GetUserId(), room.Id).
			WithError(err).
			Error("Failed to fetch all songs for user")
	} else {
		logger.WithUserAndRoom(user.GetUserId(), room.Id).
			Infof("Fetching songs for user finished successfully with %d tracks found", len(tracks))
	}

	// We send in the channel the result after processing the music for this user
	musicLibrary.MusicFetchingChannel <- MusicFetchingResult{user, tracks, err}
}

func (musicLibrary *SharedMusicLibrary) addSongsToLibraryAndFindMostCommonSongs(room *Room,
	notifyProcessingOver func(success bool), saveMusicLibrary func() error, ctx context.Context) {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.WithRoom(room.Id).Errorf(
				"An unknown error happened while adding song and finding common songs - error %s \n%s",
				err, string(debug.Stack()))

			// we notify that the processing is over
			notifyProcessingOver(false)
		}
	}()

	logger.Logger.Infof("Starting to wait for music processing results")

	success := true

	// Fetch all the music for each user, setting the success result on success/failure
	musicLibrary.getUserMusic(room, &success, saveMusicLibrary, ctx)

	logger.Logger.Infof("All music fetching results received - success=%t", success)

	if success {
		musicLibrary.processUserMusic(room, &success, ctx)
	}

	// we add the last step done once all the processing is good
	musicLibrary.ProcessingStatus.AlreadyProcessed += 1
	_ = saveMusicLibrary()

	// we notify that the processing is over
	notifyProcessingOver(success)
}

func (musicLibrary *SharedMusicLibrary) getUserMusic(room *Room, success *bool, saveMusicLibrary func() error,
	ctx context.Context) {

	for {
		if musicLibrary.ProcessingStatus.AlreadyProcessed == musicLibrary.ProcessingStatus.TotalToProcess - 1 {
			// we break once we have received a message for every user
			return
		}

		select {

		// We receive from the channel a messages for each user
		case musicProcessingResult := <-musicLibrary.MusicFetchingChannel:
			user := musicProcessingResult.User
			userId := user.GetUserId()

			logger.WithUser(user.GetUserId()).Info("Received music fetching result for user")

			if musicProcessingResult.Error != nil {
				logger.WithUserAndRoom(user.GetUserId(), room.Id).
					WithError(musicProcessingResult.Error).
					Error("Music fetching failed for user")
				*success = false
				return

			} else {
				logger.WithUser(user.GetUserId()).Infof("Music fetching succeeded for user %s, " +
					"finding %d tracks",
					userId, len(musicProcessingResult.Tracks))

				// we add the tracks as all went fine
				musicLibrary.CommonPlaylists.addTracks(musicProcessingResult.User, musicProcessingResult.Tracks)
			}

			// Mark one user's music as processed
			musicLibrary.ProcessingStatus.AlreadyProcessed += 1

			// we mark the change in mongo
			_ = saveMusicLibrary()

		// this happens if processing takes too much time
		case <-ctx.Done():
			logger.WithRoom(room.Id).Error("Music fetching timeout")
			*success = false
			return
		}
	}
}

func (musicLibrary *SharedMusicLibrary) processUserMusic(room *Room, success *bool, ctx context.Context) {
	go musicLibrary.generatePlaylists(room)

	select {

	case musicProcessingResult := <-musicLibrary.MusicProcessingChannel:
		if musicProcessingResult.Error != nil {
			logger.Logger.Error("An error when generating playlists occurred ", musicProcessingResult.Error)
			*success = false
		}

	case <-ctx.Done():
		logger.WithRoom(room.Id).Error("Music processing timeout")
		*success = false
	}
}

func (musicLibrary *SharedMusicLibrary) generatePlaylists(room *Room) {
	// Recovery for the goroutine
	defer func() {
		if err := recover(); err != nil {
			logger.
				WithRoom(room.Id).
				Errorf("An unknown error happened while generating playlists - error %s \n%s",
				err, string(debug.Stack()))
			musicLibrary.MusicProcessingChannel <- MusicProcessingResult{errors.New("Unknown error")}
		}
	}()

	// if everything went well, we now generate the playlists for the users in the room
	err := musicLibrary.CommonPlaylists.GeneratePlaylists()

	if err != nil {
		logger.
			WithRoom(room.Id).
			WithError(err).
			Error("An error when generating playlists occurred")
	}

	musicLibrary.MusicProcessingChannel <- MusicProcessingResult{err}
}