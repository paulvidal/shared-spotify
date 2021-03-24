package musicclient

import (
	"context"
	"errors"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/applemusic"
	"github.com/shared-spotify/musicclient/clientcommon"
	spotifyclient "github.com/shared-spotify/musicclient/spotify"
	"github.com/zmb3/spotify"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
)

// The goal of this client is to provide a general abstraction regardless of the underlying music service
// used by the user
//
// We use the spotify objects for now as the reference objects - e.g. FullTrack object
// Eventually, we should create out own data model so we can get rid of spotify and have real abstraction

const retryFailCreateUserFromRequestSpotify = 5

func Logout(w http.ResponseWriter, r *http.Request)  {
	// delete the cookies
	tokenDeleteCookie, errToken := clientcommon.GetDeletedCookie(clientcommon.TokenCookieName)
	loginTypeDeleteCookie, errLoginType := clientcommon.GetDeletedCookie(clientcommon.LoginTypeCookieName)

	if errToken != nil || errLoginType != nil {
		logger.Logger.Error("Got an error while creating deletion cookies")
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, tokenDeleteCookie)
	http.SetCookie(w, loginTypeDeleteCookie)

	tag := "unknown"
	tokenCookie, err := r.Cookie(clientcommon.LoginTypeCookieName)

	if err != nil {
		// nothing, do not crash

	} else if tokenCookie.Value == clientcommon.SpotifyLoginType {
		tag = datadog.SpotifyProvider

	} else if tokenCookie.Value == clientcommon.AppleMusicLoginType {
		tag = datadog.AppleMusicProvider
	}

	datadog.Increment(1, datadog.UserLogout, datadog.Provider.Tag(tag))

	// we redirect to home page
	http.Redirect(w, r, clientcommon.FrontendUrl, http.StatusFound)
}

/**
  Create user abstraction
*/

func GetUser(w http.ResponseWriter, r *http.Request) {
	user, err := CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w, r)
		return
	}

	httputils.SendJson(w, user)
}

func CreateUserFromRequest(r *http.Request) (*clientcommon.User, error) {
	span, ctx := tracer.StartSpanFromContext(r.Context(), "create.user.from.request")
	defer span.Finish()

	loginTypeCookie, err := r.Cookie(clientcommon.LoginTypeCookieName)

	if err != nil {
		errMsg := "failed to create user from request - no login cookie found "
		logger.Logger.Warning(errMsg, err)
		return nil, errors.New(errMsg)
	}

	tokenCookie, err := r.Cookie(clientcommon.TokenCookieName)

	if err != nil {
		errMsg := "failed to create user from request - no token cookie found "
		logger.Logger.Warning(errMsg, err) // this is normal if user is not logged in, so show it as a warning
		return nil, errors.New(errMsg)
	}

	return CreateUserFromToken(tokenCookie.Value, loginTypeCookie.Value, ctx)
}

func CreateUserFromToken(token string, loginType string, ctx context.Context) (*clientcommon.User, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "create.user.from.cache")
	defer span.Finish()

	if token == "" {
		err := errors.New("no token provided for user")
		span.Finish(tracer.WithError(err))
		return nil, err
	}

	// try to use our local user cache
	if user, ok := clientcommon.GetUserFromCache(token); ok {
		return user, nil
	}

	if loginType == clientcommon.SpotifyLoginType {
		span.SetOperationName("create.user.from.spotify_token")
		user, err := createUserFromTokenSpotify(token)

		if user != nil {
			clientcommon.AddUserToCache(token, user)
		}

		span.Finish(tracer.WithError(err))
		return user, err

	} else if loginType == clientcommon.AppleMusicLoginType {
		span.SetOperationName("create.user.from.apple_token")
		user, err := createUserFromTokenAppleMusic(token)

		if user != nil {
			clientcommon.AddUserToCache(token, user)
		}

		span.Finish(tracer.WithError(err))
		return user, err

	} else {
		msg := "Unknown token login type, found " + loginType
		err := errors.New(msg)
		logger.Logger.Warning(msg)
		span.Finish(tracer.WithError(err))
		return nil, err
	}
}

func createUserFromTokenSpotify(tokenStr string) (*clientcommon.User, error) {
	token, err := spotifyclient.DecryptToken(tokenStr)

	if err != nil {
		errMsg := "failed to create user from request - failed to decrypt token "
		logger.Logger.Error(errMsg, err)
		return nil, errors.New(errMsg)
	}

	var user *clientcommon.User
	retry := 0

	// We retry for spotify because the api throws randomly 503 sometimes
	for retry < retryFailCreateUserFromRequestSpotify {
		user, err = spotifyclient.CreateUserFromToken(token, tokenStr)

		if user != nil {
			break
		}

		retry += 1
		logger.Logger.Warningf("Failed to create user from request, retrying with retry count=%d, %+v", retry, err)
	}

	if err != nil {
		logger.Logger.Error("failed to create user from request - create user from token failed ", err)
		return nil, err
	}

	return user, nil
}

func createUserFromTokenAppleMusic(tokenStr string) (*clientcommon.User, error) {
	appleLogin, err := applemusic.DecryptToken(tokenStr)

	if err != nil {
		errMsg := "failed to create user from request - failed to decrypt token "
		logger.Logger.Error(errMsg, err)
		return nil, errors.New(errMsg)
	}

	user, err := applemusic.CreateUserFromToken(appleLogin, tokenStr)

	if err != nil {
		logger.Logger.Error("failed to create user from request - create user from token failed ", err)
		return nil, err
	}

	return user, nil
}

/**
  Get all songs abstraction
*/

func GetAllSongs(user *clientcommon.User) ([]*spotify.FullTrack, error) {
	var allSongs []*spotify.FullTrack

	if user.IsSpotify() {
		songs, err := spotifyclient.GetAllSongs(user)

		if err != nil {
			return nil, err
		}

		allSongs = songs

	} else if user.IsAppleMusic() {
		// we first query the songs
		appleMusicSongs, err := applemusic.GetAllSongs(user)

		if err != nil {
			return nil, err
		}

		isrcs := make([]string, 0)

		for _, song := range appleMusicSongs {
			isrcs = append(isrcs, song.Attributes.ISRC)
		}

		// we then convert the data to spotify tracks
		songs, err := spotifyclient.GetTrackForISRCs(user, isrcs)

		if err != nil {
			return nil, err
		}

		allSongs = songs
	}

	return allSongs, nil
}

/**
  Get additional information abstractions
*/

func GetAlbums(tracks []*spotify.FullTrack) (map[string]*spotify.FullAlbum, error) {
	return spotifyclient.GetAlbums(tracks)
}

func GetArtists(tracks []*spotify.FullTrack) (map[string][]*spotify.FullArtist, error) {
	return spotifyclient.GetArtists(tracks)
}

func GetAudioFeatures(tracks []*spotify.FullTrack) (map[string]*spotify.AudioFeatures, error) {
	return spotifyclient.GetAudioFeatures(tracks)
}

/**
  Create playlists
*/

func CreatePlaylist(user *clientcommon.User, playlistName string, tracks []*spotify.FullTrack) (*string, error) {
	var link *string

	if user.IsSpotify() {
		externalLink, err := spotifyclient.CreatePlaylist(user, playlistName, tracks)

		if err != nil {
			return nil, err
		}

		link = externalLink

	} else if user.IsAppleMusic() {
		externalLink, err := applemusic.CreatePlaylist(user, playlistName, tracks)

		if err != nil {
			return nil, err
		}

		link = externalLink
	}

	return link, nil
}
