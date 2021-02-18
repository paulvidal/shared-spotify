package spotify

import (
	"encoding/json"
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
	"math/rand"
	"os"
	"time"
)

const expirationTime = 30 * time.Minute  // change the client every 30 mins

// General clients are spotify clients that can be used at anytime to access spotify info
// they cannot access user info, as all this is linked with another client id and secret
var SpotifyGenericClients []*GenericClient

type GenericClient struct {
	ClientId      string
	ClientSecret  string
	TimeCreatedAt *time.Time
	Client        *spotify.Client
}

func (c *GenericClient) GetClient() *spotify.Client {
	timeNow := time.Now()

	// if the the client is older than expiration time or does not exist, change it
	if c.Client == nil {
		c.TimeCreatedAt = &timeNow
		c.Client = CreateGenericClient(c.ClientId, c.ClientSecret)

		logger.Logger.Warningf("Creating spotify generic client with client id %s", c.ClientId)

	} else if timeNow.After(c.TimeCreatedAt.Add(expirationTime)) {
		c.TimeCreatedAt = &timeNow
		c.Client = CreateGenericClient(c.ClientId, c.ClientSecret)

		logger.Logger.Warningf("Refreshing expired spotify generic client with client id %s", c.ClientId)
	}

	return c.Client
}

func init() {
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
		client := &GenericClient{ClientId: credential.ClientId, ClientSecret: credential.ClientSecret}
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

	return SpotifyGenericClients[randomPick].GetClient()
}
