package main

import (
	"github.com/rs/cors"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"net/http"
)

func startServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/user", spotifyclient.GetUser)
	mux.HandleFunc("/login", spotifyclient.Authenticate)
	mux.HandleFunc("/callback", spotifyclient.CallbackHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infof("Got request for %s", r.URL.String())
	})

	// Setup cors policies
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8080", "http://localhost:3000"},
		AllowCredentials: true,
	}).Handler(mux)

	// Launch the server
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		logger.Logger.Fatal("Failed to start server", err)
	}
}

func main() {
	// Launch the server
	startServer()

	// Initiate an auth flow

	//user := spotifyclient.Authenticate(clientId, clientSecret)
	//
	//logger.Logger.Infof("user is: %s\n", user.Infos.DisplayName)

	//songs, err := user.GetAllSongs()
	//
	//if err != nil {
	//	logger.Logger.Fatal("Could not retrieve songs for user", err)
	//}
	//
	//fmt.Printf("Found %d songs for user %s\n", len(*songs), user.Infos.DisplayName)
	//
	//for i, song := range *songs {
	//	artists := song.Artists
	//	fmt.Printf("%d - %s | %s\n", i, song.Name, artists[0].Name)
	//}
}
