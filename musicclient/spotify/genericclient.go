package spotify

import (
	"encoding/json"
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
	"math/rand"
	"os"
	"time"
)

// General clients are spotify clients that can be used at anytime to access spotify info
// they cannot access user info, as all this is linked with another client id and secret
var SpotifyGenericClients []*spotify.Client

func init()  {
	genericClientsCredentials := os.Getenv("SPOTIFY_GENERIC_CLIENT_CREDENTIALS")
	var spotifyClientCredentials ClientsCredentials
	err := json.Unmarshal([]byte(genericClientsCredentials), &spotifyClientCredentials)

	if err != nil {
		logger.Logger.Fatalf(
			"SPOTIFY_GENERIC_CLIENT_CREDENTIALS env var not well formed, found %s, %+v",
			genericClientsCredentials,
			err)
	}

	for _, credential := range spotifyClientCredentials.Credentials {
		client := GenericClient(credential.ClientId, credential.ClientSecret)
		SpotifyGenericClients = append(SpotifyGenericClients, client)
	}
}

type Credential struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type ClientsCredentials struct {
	 Credentials []Credential `json:"credentials"`
}

// a simple idea to prevent rate limits is to just randomly pick one client every time we ask for one
// over time, this should spread the load on different spotify clients
func GetSpotifyGenericClient() *spotify.Client {
	rand.Seed(time.Now().Unix())
	randomPick := rand.Int() % len(SpotifyGenericClients)

	return SpotifyGenericClients[randomPick]
}
