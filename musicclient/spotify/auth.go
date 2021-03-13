package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

// Cache 100 states max
var states, _ = lru.New(100)

var ClientId = os.Getenv("CLIENT_ID")
var ClientSecret = os.Getenv("CLIENT_SECRET_KEY")

var CallbackUrl = fmt.Sprintf("%s/callback", clientcommon.BackendUrl)

// the redirect URL must be an exact match of a URL you've registered for your application
// scopes determine which permissions the user is prompted to authorize
var auth = spotify.NewAuthenticator(
	CallbackUrl,
	spotify.ScopeUserReadPrivate,
	spotify.ScopePlaylistReadPrivate,
	spotify.ScopePlaylistReadCollaborative,
	spotify.ScopePlaylistModifyPrivate,
	spotify.ScopePlaylistModifyPublic,
	spotify.ScopeUserLibraryRead)

func init() {
	// set client id and secret here for spotify
	auth.SetAuthInfo(ClientId, ClientSecret)
}

func CreateUserFromToken(token *oauth2.Token) (*clientcommon.User, error) {
	client := auth.NewClient(token)
	client.AutoRetry = true // enable auto retries when rate limited

	privateUser, err := client.CurrentUser()

	clientcommon.SendRequestMetric(datadog.SpotifyProvider, datadog.RequestTypeUserInfo, true, nil)

	if err != nil {
		logger.Logger.Warning("Failed to create user from token ", err)
		return nil, err
	}

	userInfos := toUserInfos(privateUser)

	return &clientcommon.User{UserInfos: &userInfos, SpotifyClient: &client}, nil
}

func toUserInfos(user *spotify.PrivateUser) clientcommon.UserInfos {
	displayName := user.DisplayName
	var image string

	if user.Images != nil && len(user.Images) > 0 {
		image = user.Images[0].URL
	}

	return clientcommon.UserInfos{Id: user.ID, Name: displayName, ImageUrl: image}
}

func Authenticate(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Headers for request to authenticate are ", r.Header)
	datadog.Increment(1, datadog.UserLoginStarted, datadog.Provider.Tag(datadog.SpotifyProvider))

	// We extract the redirect_uri if it exists, to redirect to it once th auth is finished
	redirectUri := r.URL.Query().Get("redirect_uri")
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

	logger.Logger.Infof("State is state=%s and states are states=%+v", st, states.Keys())

	// check state exists to prevent csrf attacks
	if !ok {
		logger.Logger.Errorf("State not found found=%s actual=%v", st, states)
		http.NotFound(w, r)
		return
	}

	// use the same state string here that you used to generate the URL
	token, err := auth.Token(st, r)
	if err != nil {
		logger.Logger.Error("Couldn't get token ", err)
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}

	// we delete the state entry
	states.Remove(st)

	logger.Logger.Infof("token is: %+v\n", token)

	// Add the token as an encrypted cookie
	cookie, err := EncryptToken(token)
	if err != nil {
		logger.Logger.Errorf("Failed to set token", err)
		http.Error(w, "Failed to set token", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, cookie)

	// Add the login type cookie name
	loginTypeCookie, err := clientcommon.GetLoginTypeCookie(clientcommon.SpotifyLoginType)
	if err != nil {
		logger.Logger.Errorf("Failed to set loginType", err)
		http.Error(w, "Failed to set loginType", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, loginTypeCookie)

	logger.Logger.Info("Redirecting to ", redirectUrl)
	datadog.Increment(1, datadog.UserLoginSuccess, datadog.Provider.Tag(datadog.SpotifyProvider))

	http.Redirect(w, r, redirectUrl.(string), http.StatusFound)
}

func DecryptToken(tokenCookie *http.Cookie) (*oauth2.Token, error) {
	var token oauth2.Token

	base64JsonToken, err := base64.StdEncoding.DecodeString(tokenCookie.Value)

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

// we return the same client as the one used by our app for authentication, not the same as a generic client
func AuthenticatedGenericClient() (*spotify.Client, error) {
	return GetSpotifyGenericClient()
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
