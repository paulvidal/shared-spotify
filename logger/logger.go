package logger

import (
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
)

const logFilename = "app.log"

var Logger = logrus.StandardLogger()

func init() {
	// Write to stdout and log file
	logFile, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	logrus.SetOutput(mw)

	Logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	Logger.SetLevel(logrus.DebugLevel)
	Logger.SetReportCaller(true)
}

func WithUser(userId string) *logrus.Entry {
	return Logger.WithFields(logrus.Fields{
		"user": userId,
	})
}