package main

import (
	"github.com/gorilla/handlers"
	"github.com/rs/cors"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"net/http"
	"os"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
)

var Port = os.Getenv("PORT")

func startServer() {
	logger.Logger.Warning("Starting server")

	r := muxtrace.NewRouter(
		muxtrace.WithServiceName("shared-spotify-backend"),
		muxtrace.WithAnalytics(true),
		muxtrace.WithAnalyticsRate(1.0),
	)

	r.HandleFunc("/login", spotifyclient.Authenticate)
	r.HandleFunc("/callback", spotifyclient.CallbackHandler)

	r.HandleFunc("/user", spotifyclient.GetUser)

	r.HandleFunc("/rooms", app.RoomsHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}", app.RoomHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/users", app.RoomUsersHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists", app.RoomPlaylistsHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists/{playlistId:[a-zA-Z0-9]+}", app.RoomPlaylistHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists/{playlistId:[a-zA-Z0-9]+}/add", app.RoomAddPlaylistHandler)

	// Setup cors policies
	options := cors.Options{
		AllowedOrigins: []string{spotifyclient.FrontendUrl},
		AllowCredentials: true,
	}
	handler := cors.New(options).Handler(r)

	// Setup request logging
	handler = handlers.LoggingHandler(logger.Logger.Out, handler)

	// Setup recovery in case of panic
	handler = handlers.RecoveryHandler()(handler)

	// Launch the server
	err := http.ListenAndServe(":" + Port, handler)
	if err != nil {
		logger.Logger.Fatal("Failed to start server", err)
	}
}

func main() {
	startServer()
}
