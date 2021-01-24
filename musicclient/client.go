package musicclient

import (
	"errors"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/applemusic"
	"github.com/shared-spotify/musicclient/clientcommon"
	spotifyclient "github.com/shared-spotify/musicclient/spotify"
	"net/http"
)

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
	tokenCookie, err := r.Cookie(clientcommon.LoginTypeCookieName)

	if err != nil {
		errMsg := "failed to create user from request - no login cookie found "
		logger.Logger.Warning(errMsg, err)
		return nil, errors.New(errMsg)
	}

	if tokenCookie.Value == clientcommon.SpotifyLoginType {
		return createUserFromRequestSpotify(r)

	} else if tokenCookie.Value == clientcommon.AppleMusicLoginType {
		return createUserFromRequestAppleMusic(r)
	}

	msg := "Unknown token login type, found " + tokenCookie.Value
	logger.Logger.Warning(msg)
	return nil, errors.New(msg)
}

func createUserFromRequestSpotify(r *http.Request) (*clientcommon.User, error) {
	tokenCookie, err := r.Cookie(clientcommon.TokenCookieName)

	if err == http.ErrNoCookie {
		errMsg := "failed to create user from request - no token cookie found "
		logger.Logger.Warning(errMsg, err) // this is normal if user is not logged in, so show it as a warning
		return nil, errors.New(errMsg)
	}

	token, err := spotifyclient.DecryptToken(tokenCookie)

	if err != nil {
		errMsg := "failed to create user from request - failed to decrypt token "
		logger.Logger.Error(errMsg, err)
		return nil, errors.New(errMsg)
	}

	user, err := spotifyclient.CreateUserFromToken(token)

	if err != nil {
		logger.Logger.Error("failed to create user from request - create user from token failed ", err)
		return nil, err
	}

	return user, nil
}

func createUserFromRequestAppleMusic(r *http.Request) (*clientcommon.User, error) {
	tokenCookie, err := r.Cookie(clientcommon.TokenCookieName)

	if err == http.ErrNoCookie {
		errMsg := "failed to create user from request - no token cookie found "
		logger.Logger.Warning(errMsg, err) // this is normal if user is not logged in, so show it as a warning
		return nil, errors.New(errMsg)
	}

	appleLogin, err := applemusic.DecryptToken(tokenCookie)

	if err != nil {
		errMsg := "failed to create user from request - failed to decrypt token "
		logger.Logger.Error(errMsg, err)
		return nil, errors.New(errMsg)
	}

	user, err := applemusic.CreateUserFromToken(appleLogin)

	if err != nil {
		logger.Logger.Error("failed to create user from request - create user from token failed ", err)
		return nil, err
	}

	return user, nil
}
