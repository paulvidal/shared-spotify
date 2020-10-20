package spotifyclient

import (
	"github.com/shared-spotify/logger"
	"net/http"

	"github.com/zmb3/spotify"
)

const redirectURL = "http://localhost:8080/callback"

// TODO: generate different one per user
const state = "state"

var channel = make(chan *spotify.Client)

// the redirect URL must be an exact match of a URL you've registered for your application
// scopes determine which permissions the user is prompted to authorize
var auth = spotify.NewAuthenticator(
	redirectURL,
	spotify.ScopeUserReadPrivate,
	spotify.ScopePlaylistReadPrivate,
	spotify.ScopePlaylistReadCollaborative,
	spotify.ScopeUserLibraryRead)

type User struct {
	Infos  spotify.PrivateUser
	Client *spotify.Client
}

func Authenticate(clientId string, clientSecret string) User {
	// if you didn't store your ID and secret key in the specified environment variables,
	// you can set them manually here
	auth.SetAuthInfo(clientId, clientSecret)

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	url := auth.AuthURL(state)
	logger.Logger.Info("Please login to Spotify by visiting the following page in your browser: ", url)

	// wait for auth to complete
	client := <-channel

	// use the client to Makefile calls that require authorization
	user, err := client.CurrentUser()
	if err != nil {
		logger.Logger.Fatal(err)
	}

	logger.Logger.Infof("You are logged in as %s", user.ID)

	return User{
		Infos:   *user,
		Client: client,
	}
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

	// use the token to get an authenticated client
	client := auth.NewClient(token)
	logger.Logger.Info("Login Completed!")

	// Send the client to the channel
	channel <- &client
}
