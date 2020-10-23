package main

import (
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
	r.HandleFunc("/rooms/{roomId:[0-9]+}/musics", app.RoomMusicHandler)

	// Setup cors policies
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8080", "http://localhost:3000"},
		AllowCredentials: true,
	}).Handler(r)

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
