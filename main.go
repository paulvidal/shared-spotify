package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/shared-spotify/app"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/spotifyclient"
	"net/http"
	"os"
)

var Prod = os.Getenv("ENV") == "PROD"
var Port = os.Getenv("PORT")

func h(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Infof("Got request for %s://%s%s", r.RequestURI, r.Host, r.URL.String())
}

func startServer() {
	r := mux.NewRouter()

	r.HandleFunc("/login", spotifyclient.Authenticate)
	r.HandleFunc("/callback", spotifyclient.CallbackHandler)

	r.HandleFunc("/api/user", spotifyclient.GetUser)

	r.HandleFunc("/api/rooms", app.RoomsHandler)
	r.HandleFunc("/api/rooms/{roomId:[0-9]+}", app.RoomHandler)
	r.HandleFunc("/api/rooms/{roomId:[0-9]+}/users", app.RoomUsersHandler)
	r.HandleFunc("/api/rooms/{roomId:[0-9]+}/playlists", app.RoomPlaylistsHandler)
	r.HandleFunc("/api/rooms/{roomId:[0-9]+}/playlists/add", app.RoomAddPlaylistsHandler)

	r.PathPrefix("/").HandlerFunc(h)

	// Setup cors policies
	options := cors.Options{}
	if !Prod {
		// For dev environment
		options = cors.Options{
			AllowedOrigins:   []string{"http://localhost:8080", "http://localhost:3000"},
			AllowCredentials: true,
		}
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
