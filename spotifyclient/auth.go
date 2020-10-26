package spotifyclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/shared-spotify/httputils"
	"golang.org/x/oauth2"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

// Cache 100 states max
var states, _ = lru.New(100)

const stateMaxSize = 100000000000
const tokenCookieName = "token"

var BackendUrl = os.Getenv("BACKEND_URL")
var FrontendUrl = os.Getenv("FRONTEND_URL")
var clientId = os.Getenv("CLIENT_ID")
var clientSecret = os.Getenv("CLIENT_SECRET_KEY")

var CallbackUrl = fmt.Sprintf("%s/callback", BackendUrl)

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

type UserInfos struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	ImageUrl string `json:"image"`
}

type User struct {
	Infos  UserInfos        `json:"user_infos"`
	Client *spotify.Client  `json:"-"` // we ignore this field
}

func (user *User) GetUserId() string {
	return user.Infos.Name
}

func CreateUserFromRequest(r *http.Request) (*User, error) {
	tokenCookie, err := r.Cookie(tokenCookieName)

	if err == http.ErrNoCookie {
		errMsg := "failed to create user from request - no token cookie found "
		logger.Logger.Warning(errMsg, err)  // this is normal if user is not logged in, so show it as a warning
		return nil, errors.New(errMsg)
	}

	token, err := decryptToken(tokenCookie)

	if err != nil {
		errMsg := "failed to create user from request - failed to decrypt token "
		logger.Logger.Error(errMsg, err)
		return nil, errors.New(errMsg)
	}

	user, err := createUserFromToken(token)

	if err != nil {
		logger.Logger.Error("failed to create user from request - create user from token failed ", err)
		return nil, err
	}

	return user, nil
}

func createUserFromToken(token *oauth2.Token) (*User, error) {
	client := auth.NewClient(token)
	privateUser, err := client.CurrentUser()

	if err != nil {
		logger.Logger.Error("Failed to create user from token ", err)
		return nil, err
	}

	userInfos := toUserInfos(privateUser)

	return &User{userInfos, &client}, nil
}

func toUserInfos(user *spotify.PrivateUser) UserInfos {
	displayName := user.DisplayName
	var image string

	if user.Images != nil && len(user.Images) > 0 {
		image = user.Images[0].URL
	}

	return UserInfos{user.ID, displayName, image}
}

func (user *User) ToJson() ([]byte, error) {
	jsonUserInfos, err := json.Marshal(user.Infos)

	if err != nil {
		return nil, err
	}

	return jsonUserInfos, nil
}

func (user *User) IsEqual(otherUser *User) bool {
	return otherUser.Infos.Id == user.Infos.Id
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	user, err := CreateUserFromRequest(r)

	if err != nil {
		httputils.AuthenticationError(w, r)
		return
	}

	httputils.SendJson(w, user)
}

func Authenticate(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Headers for request to authenticate are ", r.Header)

	// if you didn't store your ID and secret key in the specified environment variables,
	// you can set them manually here
	auth.SetAuthInfo(clientId, clientSecret)

	// We extract the referer if it exists, to redirect to it once th auth is finished
	var redirectUrl string
	referer:= r.Header.Get("Referer")
	refererParsedUrl, err := url.Parse(referer)

	if err != nil || referer == "" {
		redirectUrl = FrontendUrl

	} else {
		redirectUrl = FrontendUrl + refererParsedUrl.RequestURI()
	}

	logger.Logger.Info("Redirect Url for user after auth will be: ", redirectUrl)

	// we generate a random state and remember the redirect url so we use it once we are redirected
	randomState := randomState()
	states.Add(randomState, redirectUrl)

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	authUrl := auth.AuthURL(randomState)

	logger.Logger.Info("Url to login is: ", authUrl)

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
		logger.Logger.Errorf("Couldn't get token", err)
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}

	// we delete the state entry
	states.Remove(st)

	logger.Logger.Infof("token is: %+v\n", token)

	// Add the token as an encrypted cookie
	cookie, err := encryptToken(token)
	if err != nil {
		logger.Logger.Errorf("Failed to set token", err)
		http.Error(w, "Failed to set token", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, cookie)

	logger.Logger.Info("Redirecting to ", redirectUrl)

	http.Redirect(w, r, redirectUrl.(string), http.StatusFound)
}

func randomState() string {
	// we initialise the random seed
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(stateMaxSize))
}

func decryptToken(tokenCookie *http.Cookie) (*oauth2.Token, error) {
	var token oauth2.Token

	base64JsonToken, err := base64.StdEncoding.DecodeString(tokenCookie.Value)

	if err != nil {
		logger.Logger.Error("Failed to decode base64 token")
		return nil, err
	}

	err = json.Unmarshal(base64JsonToken, &token)

	if err != nil {
		logger.Logger.Error("Failed to deserialise json token")
		return nil, err
	}

	return &token, nil
}

func encryptToken(token *oauth2.Token) (*http.Cookie, error) {
	jsonToken, err := json.Marshal(*token)

	if err != nil {
		logger.Logger.Error("Failed to serialise json token")
		return nil, err
	}

	base64JsonToken := base64.StdEncoding.EncodeToString(jsonToken)
	expiration := time.Now().Add(365 * 24 * time.Hour)

	urlParsed, err := url.Parse(BackendUrl)

	if err != nil {
		logger.Logger.Error("Failed to parse urls")
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
		Name: tokenCookieName,
		Value: base64JsonToken,
		Expires: expiration,
		// we send the cookie cross domain, so we need all this
		Domain: urlParsed.Host,
		Secure: secure,
		SameSite: sameSite,
	}

	return &cookie, nil
}