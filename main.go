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
		logger.Logger.Infof("Got request for %s", r.URL.String())
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

	logger.Logger.Infof("user is: %s\n", user.Infos.DisplayName)

	songs, err := user.GetAllSongs()

	if err != nil {
		logger.Logger.Fatal("Could not retrieve songs for user", err)
	}

	fmt.Printf("Found %d songs for user %s\n", len(*songs), user.Infos.DisplayName)

	for i, song := range *songs {
		artists := song.Artists
		fmt.Printf("%d - %s | %s\n", i, song.Name, artists[0].Name)
	}
}
