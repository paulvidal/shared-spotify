package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

const redirectParam = "redirect_uri"
var loginUrl = clientcommon.FrontendUrl + "/login"

// Cache 1000 states max
var states, _ = lru.New(1000)

var ClientId = os.Getenv("CLIENT_ID")
var ClientSecret = os.Getenv("CLIENT_SECRET_KEY")

var CallbackUrl = fmt.Sprintf("%s/callback", clientcommon.BackendUrl)

// the redirect URL must be an exact match of a URL you've registered for your application
// scopes determine which permissions the user is prompted to authorize
var auth = spotify.NewAuthenticator(
	CallbackUrl,
	spotify.ScopeUserReadPrivate,
	spotify.ScopeUserReadEmail,
	spotify.ScopePlaylistReadPrivate,
	spotify.ScopePlaylistReadCollaborative,
	spotify.ScopePlaylistModifyPrivate,
	spotify.ScopePlaylistModifyPublic,
	spotify.ScopeUserLibraryRead)

func init() {
	// set client id and secret here for spotify
	auth.SetAuthInfo(ClientId, ClientSecret)
}

func CreateUserFromToken(token *oauth2.Token, tokenStr string) (*clientcommon.User, error) {
	client := auth.NewClient(token)
	client.AutoRetry = true // enable auto retries when rate limited

	privateUser, err := client.CurrentUser()

	clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypeUserInfo, true, nil)

	if err != nil {
		logger.Logger.Warning("Failed to create user from token ", err)
		return nil, err
	}

	userInfos := toUserInfos(privateUser)

	return &clientcommon.User{
		UserInfos:     &userInfos,
		SpotifyClient: &client,
		LoginType:     clientcommon.SpotifyLoginType,
		Token:         tokenStr,
	}, nil
}

func toUserInfos(user *spotify.PrivateUser) clientcommon.UserInfos {
	displayName := user.DisplayName
	email := user.Email
	var image string

	if user.Images != nil && len(user.Images) > 0 {
		image = user.Images[0].URL
	}

	return clientcommon.UserInfos{Id: user.ID, Name: displayName, ImageUrl: image, Email: email, JoinDate: time.Now()}
}

func Authenticate(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Headers for request to authenticate are ", r.Header)
	datadog.Increment(1, datadog.UserLoginStarted, datadog.Provider.Tag(datadog.SpotifyProvider))

	// We extract the redirect_uri if it exists, to redirect to it once the auth is finished
	redirectUri := r.URL.Query().Get(redirectParam)
	redirect := clientcommon.FrontendUrl

	if redirect != "" {
		redirect = clientcommon.FrontendUrl + redirectUri
	}

	logger.Logger.Info("Redirect Url for user after auth will be: ", redirect)

	// we generate a random state and remember the redirect url so we use it once we are redirected
	randomState := utils.GenerateStrongHash()
	states.Add(randomState, redirect)

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	authUrl := auth.AuthURLWithDialog(randomState)

	logger.Logger.Info("Url to login is: ", authUrl)
	clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypeAuth, false, nil)

	// We redirect to the correct url
	http.Redirect(w, r, authUrl, http.StatusFound)
}

// the user will eventually be redirected back to your redirect URL
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Headers for request to callback are ", r.Header)

	st := r.FormValue("state")
	var redirectUrl, ok = states.Get(st)
	logger.Logger.Infof("State is state=%s and states are states=%v", st, states.Keys())

	// check state exists to prevent csrf attacks
	if !ok {
		logger.Logger.Errorf("State not found found=%s actual=%v - redirecting user back to login page",
			st, states)
		http.NotFound(w, r)
		return
	}

	// form the url in case we fail to auth and need to redirect the user again to the login page
	loginRedirectUrl := FormRedirectLoginUrl(redirectUrl.(string))

	// use the same state string here that you used to generate the URL
	token, err := auth.Token(st, r)

	if err != nil {
		logger.Logger.
			WithError(err).
			Error("Couldn't get token - redirecting user back to login page")
		http.Redirect(w, r, loginRedirectUrl, http.StatusFound)
		return
	}

	// we delete the state entry
	states.Remove(st)

	// Add the token as an encrypted cookie
	cookie, err := EncryptToken(token)
	if err != nil {
		logger.Logger.
			WithError(err).
			Errorf("Failed to set token - redirecting user back to login page")
		http.Redirect(w, r, loginRedirectUrl, http.StatusFound)
		return
	}
	http.SetCookie(w, cookie)

	// Add the login type cookie name
	loginTypeCookie, err := clientcommon.GetLoginTypeCookie(clientcommon.SpotifyLoginType)
	if err != nil {
		logger.Logger.
			WithError(err).
			Errorf("Failed to set loginType - redirecting user back to login page")
		http.Redirect(w, r, loginRedirectUrl, http.StatusFound)
		return
	}
	http.SetCookie(w, loginTypeCookie)

	// Add the users to the database if we can, but don't fail as we will add him otherwise at another time
	// for example when the room is processed
	user, err := CreateUserFromToken(token, cookie.Value)
	if err == nil {
		_ = mongoclient.InsertUsers([]*clientcommon.User{user})
	}

	logger.Logger.Info("Redirecting to ", redirectUrl)
	datadog.Increment(1, datadog.UserLoginSuccess, datadog.Provider.Tag(datadog.SpotifyProvider))

	http.Redirect(w, r, redirectUrl.(string), http.StatusFound)
}

func DecryptToken(tokenStr string) (*oauth2.Token, error) {
	var token oauth2.Token

	base64JsonToken, err := base64.StdEncoding.DecodeString(tokenStr)

	if err != nil {
		logger.Logger.Error("Failed to decode base64 token ", err)
		return nil, err
	}

	decryptedToken, err := utils.Decrypt(base64JsonToken, clientcommon.TokenEncryptionKey)

	if err != nil {
		logger.Logger.Error("Failed to decrypt token ", err)
		return nil, err
	}

	err = json.Unmarshal(decryptedToken, &token)

	if err != nil {
		logger.Logger.Error("Failed to deserialise json token ", err)
		return nil, err
	}

	return &token, nil
}

func EncryptToken(token *oauth2.Token) (*http.Cookie, error) {
	jsonToken, err := json.Marshal(*token)

	if err != nil {
		logger.Logger.Error("Failed to serialise json token")
		return nil, err
	}

	encryptedToken, err := utils.Encrypt(jsonToken, clientcommon.TokenEncryptionKey)

	if err != nil {
		logger.Logger.Error("Failed to encrypt token ", err)
		return nil, err
	}

	base64EncryptedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	expiration := time.Now().Add(365 * 24 * time.Hour)

	urlParsed, err := url.Parse(clientcommon.BackendUrl)

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

	cookie := http.Cookie{
		Name:    clientcommon.TokenCookieName,
		Value:   base64EncryptedToken,
		Expires: expiration,
		// we send the cookie cross domain, so we need all this
		Domain:   urlParsed.Host,
		Path:     "/",
		Secure:   secure,
		SameSite: sameSite,
	}

	return &cookie, nil
}

func CreateGenericClient(clientId string, clientSecret string) (*spotify.Client, error) {
	config := &clientcredentials.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())

	if err != nil {
		logger.Logger.Warning("Couldn't create oauth token for generic client: ", err)
		return nil, err
	}

	client := spotify.Authenticator{}.NewClient(token)
	client.AutoRetry = true // enable auto retries when rate limited

	return &client, nil
}

// we form here the login url with the redirect uri
func FormRedirectLoginUrl(redirectUrl string) string {
	return fmt.Sprintf(
		"%s?%s=%s",
		loginUrl,
		redirectParam,
		// create the uri by removing the frontend url
		strings.ReplaceAll(redirectUrl, clientcommon.FrontendUrl, ""),
	)
}
