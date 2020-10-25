package spotifyclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/shared-spotify/httputils"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"time"

	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
)

const redirectURL = "http://localhost:8080/callback"

// TODO: generate different one per user
const state = "state"

const tokenCookieName = "token"

var clientId = os.Getenv("CLIENT_ID")
var clientSecret = os.Getenv("CLIENT_SECRET_KEY")

// the redirect URL must be an exact match of a URL you've registered for your application
// scopes determine which permissions the user is prompted to authorize
var auth = spotify.NewAuthenticator(
	redirectURL,
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
		return nil, errors.New("no token cookie found")
	}

	token, err := decryptToken(tokenCookie)

	if err != nil {
		return nil, errors.New("failed to decrypt token")
	}

	user, err := createUserFromToken(token)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func createUserFromToken(token *oauth2.Token) (*User, error) {
	client := auth.NewClient(token)
	privateUser, err := client.CurrentUser()

	if err != nil {
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
		http.Error(w, "Failed to get current user: " + err.Error(), http.StatusInternalServerError)
		return
	}

	httputils.SendJson(w, user)
}

func Authenticate(w http.ResponseWriter, r *http.Request) {
	// if you didn't store your ID and secret key in the specified environment variables,
	// you can set them manually here
	auth.SetAuthInfo(clientId, clientSecret)

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	url := auth.AuthURL(state)

	logger.Logger.Info("Url to login is: ", url)

	// We redirect to the correct url
	http.Redirect(w, r, url, http.StatusFound)
}

// the user will eventually be redirected back to your redirect URL
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// use the same state string here that you used to generate the URL
	token, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}

	// check state is the same to prevent csrf attacks
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		logger.Logger.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	logger.Logger.Infof("token is: %+v\n", token)

	// Add the token as an encrypted cookie
	cookie, err := encryptToken(token)
	if err != nil {
		http.Error(w, "Failed to set token", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, cookie)

	// Send the client to the channel
	http.Redirect(w, r, "http://localhost:3000", http.StatusFound)
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

	cookie := http.Cookie{
		Name: tokenCookieName,
		Value: base64JsonToken,
		Expires: expiration,
	}

	return &cookie, nil
}