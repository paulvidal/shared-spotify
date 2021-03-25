package main

import (
	"context"
	"github.com/gorilla/handlers"
	"github.com/rs/cors"
	"github.com/shared-spotify/api"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/env"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient"
	"github.com/shared-spotify/musicclient/applemusic"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/musicclient/spotify"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Port = os.Getenv("PORT")
var ReleaseVersion = os.Getenv("HEROKU_RELEASE_VERSION")

const Service = "shared-spotify-backend"

var srv *http.Server

// Allows us to wait for all connection to be closed
var idleConnsClosed = make(chan struct{})

func startServer() {
	logger.Logger.Warning("Starting server")

	// Create the router
	r := muxtrace.NewRouter()

	r.HandleFunc("/health", api.Health)

	r.HandleFunc("/login", spotify.Authenticate)
	r.HandleFunc("/logout", musicclient.Logout)

	r.HandleFunc("/callback", spotify.CallbackHandler)
	r.HandleFunc("/callback/apple", applemusic.CallbackHandler)
	r.HandleFunc("/callback/apple/user", applemusic.UserHandler)

	r.HandleFunc("/user", musicclient.GetUser)

	r.HandleFunc("/rooms", api.RoomsHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}", api.RoomHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/users", api.RoomUsersHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists", api.RoomPlaylistsHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists/{playlistId:[a-zA-Z0-9]+}", api.RoomPlaylistHandler)
	r.HandleFunc("/rooms/{roomId:[a-zA-Z0-9]+}/playlists/{playlistId:[a-zA-Z0-9]+}/add", api.RoomAddPlaylistHandler)

	// Setup cors policies
	options := cors.Options{
		AllowedOrigins:   []string{clientcommon.FrontendUrl},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
	}
	handler := cors.New(options).Handler(r)

	// Setup request logging
	handler = handlers.LoggingHandler(logger.Logger.Out, handler)

	// Setup recovery in case of panic
	handler = handlers.RecoveryHandler(
		handlers.RecoveryLogger(logger.Logger),
		handlers.PrintRecoveryStack(true),
	)(handler)

	// Launch the server
	srv = &http.Server{
		Addr: ":" + Port,
		Handler: handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  1200 * time.Second,
	}
	err := srv.ListenAndServe()

	if err != http.ErrServerClosed {
		logger.Logger.Fatal("Failed to start server ", err)
	}

	<-idleConnsClosed
}

func RegisterGracefulShutdown() {
	go func() {
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)
		<-sigterm

		logger.Logger.Warningf("Shutting down gracefully...")

		// close all api resources
		logger.Logger.Warningf("Shutting down api...")
		api.Shutdown()

		// Close tracer and profiler
		logger.Logger.Warningf("Shutting down tracer and profiler...")
		tracer.Stop()
		profiler.Stop()

		// Shutdown the server
		logger.Logger.Warningf("Shutting down server...")
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Logger.Errorf("Got an error when shutting down server: %v", err)
		}
		close(idleConnsClosed)

		logger.Logger.Warningf("Server has gracefully shutdown")
	}()
}

func connectToMongo() {
	mongoclient.Initialise()
}

func startTracing() {
	// Activate datadog tracer
	rules := []tracer.SamplingRule{tracer.RateRule(1)}
	tracer.Start(
		tracer.WithSamplingRules(rules),
		tracer.WithAnalytics(true),
		tracer.WithService(Service),
		tracer.WithEnv(env.GetEnv()),
		tracer.WithServiceVersion(ReleaseVersion),
		tracer.WithRuntimeMetrics(),
	)

	logger.Logger.Warning("Datadog tracer started")

	// Activate datadog profiler
	err := profiler.Start(
		profiler.WithService(Service),
		profiler.WithEnv(env.GetEnv()),
		profiler.WithVersion(ReleaseVersion),
		profiler.WithProfileTypes(
			profiler.HeapProfile,
			profiler.CPUProfile,
			profiler.BlockProfile,
			profiler.GoroutineProfile,
			profiler.MutexProfile,
		),
	)

	if err != nil {
		logger.Logger.Fatal("Failed to start profiler ", err)
	}

	logger.Logger.Warning("Datadog profiler started")
}

func startMetricClient() {
	datadog.Initialise()
}

func main() {
	if env.IsProd() {
		startTracing()
		startMetricClient()
	}
	connectToMongo()

	RegisterGracefulShutdown()
	startServer()
}
