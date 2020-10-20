package main

import (
	"fmt"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"net/http"
	"os"
)

func startServer() {
	http.HandleFunc("/callback", spotifyclient.CallbackHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infof("Got request for:", r.URL.String())
	})
	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		logger.Logger.Fatal("Failed to start server", err)
	}
}

func main() {
	// Launch the server
	go startServer()

	// Initiate an auth flow
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET_KEY")
	user := spotifyclient.Authenticate(clientId, clientSecret)

	logger.Logger.Infof("user is: %+v\n", user.Infos.DisplayName)

	songs, err := user.GetSongs()

	if err != nil {
		logger.Logger.Fatal("Could not retrieve songs for user", err)
	}

	fmt.Printf("Found %d songs for user %s\n", len(*songs), user.Infos.DisplayName)

	for _, song := range *songs {
		fmt.Println(song.Name, " | ", song.Artists[0].Name)
	}

}
