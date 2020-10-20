package logger

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger = logrus.StandardLogger()

func init() {
	Logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetReportCaller(false)
}