package clientcommon

import (
	"github.com/shared-spotify/logger"
	"net/http"
	"net/url"
	"os"
	"time"
)

const LoginTypeCookieName = "login_type" // spotify, apple, deezer...
const TokenCookieName = "token"

// Login types
const SpotifyLoginType = "spotify"
const AppleMusicLoginType = "applemusic"

var TokenEncryptionKey = os.Getenv("TOKEN_ENCRYPTION_KEY")

var BackendUrl = os.Getenv("BACKEND_URL")
var FrontendUrl = os.Getenv("FRONTEND_URL")

func GetLoginTypeCookie(loginType string) (*http.Cookie, error) {
	expiration := time.Now().Add(365 * 24 * time.Hour)

	urlParsed, err := url.Parse(BackendUrl)

	if err != nil {
		logger.Logger.Error("Failed to parse urls ", err)
		return nil, err
	}

	secure := true
	sameSite := http.SameSiteNoneMode

	// for localhost development
	if urlParsed.Scheme == "http" {
		secure = false
		sameSite = http.SameSiteDefaultMode
	}

	return &http.Cookie{
		Name:    LoginTypeCookieName,
		Value:   loginType,
		Expires: expiration,
		// we send the cookie cross domain, so we need all this
		Domain:   urlParsed.Host,
		Path:     "/",
		Secure:   secure,
		SameSite: sameSite,
	}, nil
}
