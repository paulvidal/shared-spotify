package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"net/http"
)

func startServer() {
	r := mux.NewRouter()

	r.HandleFunc("/user", spotifyclient.GetUser)
	r.HandleFunc("/login", spotifyclient.Authenticate)
	r.HandleFunc("/callback", spotifyclient.CallbackHandler)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infof("Got request for %s", r.URL.String())
	})
	r.HandleFunc("/rooms", app.RoomsHandler)
	r.HandleFunc("/rooms/{roomId:[0-9]+}", app.RoomHandler)
	r.HandleFunc("/rooms/{roomId:[0-9]+}/users", app.RoomUsersHandler)
	r.HandleFunc("/rooms/{roomId:[0-9]+}/playlists", app.RoomPlaylistsHandler)
	r.HandleFunc("/rooms/{roomId:[0-9]+}/playlists/add", app.RoomAddPlaylistsHandler)

	// Setup cors policies
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8080", "http://localhost:3000"},
		AllowCredentials: true,
	}).Handler(r)

	// Setup request logging
	handler = handlers.LoggingHandler(logger.Logger.Out, handler)

	// Setup recovery in case of panic
	handler = handlers.RecoveryHandler()(handler)

	// Launch the server
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		logger.Logger.Fatal("Failed to start server", err)
	}
}

func main() {
	startServer()
}
