package logger

import (
	"github.com/shared-spotify/env"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
)

const logFilename = "app.log"

var level = os.Getenv("LOG_LEVEL")

var Logger = logrus.StandardLogger()

func init() {
	// Write to stdout and log file
	logFile, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	logrus.SetOutput(mw)

	logLevel, err := logrus.ParseLevel(level)

	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}

	Logger.SetLevel(logLevel)

	if env.IsProd() {
		Logger.SetFormatter(&logrus.JSONFormatter{})
		Logger.SetReportCaller(true)

	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		})
		Logger.SetReportCaller(false)
	}
}

func WithUser(userId string) *logrus.Entry {
	return Logger.WithFields(logrus.Fields{
		"user": userId,
	})
}
