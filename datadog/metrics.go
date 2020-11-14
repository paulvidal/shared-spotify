package datadog

import (
	"github.com/DataDog/datadog-go/statsd"
	"github.com/shared-spotify/logger"
)

var StatsdClient *statsd.Client

func Initialise()  {
	statsdClient, err := statsd.New("127.0.0.1:8125")

	if err != nil {
		logger.Logger.Fatalf("Failed to start statsd client ", err)
	}

	StatsdClient = statsdClient
}

func Increment(count int, metric string, tags ...string) {
	if StatsdClient == nil {
		return
	}

	err := StatsdClient.Count(metric, int64(count), tags, 1)

	if err != nil {
		logger.Logger.Errorf("Failed to increment metric %s with tags %s by %f ", metric, tags, count, err)
	}
}

func Gauge(value int, metric string, tags ...string)  {
	if StatsdClient == nil {
		return
	}

	err := StatsdClient.Gauge(metric, float64(value), tags, 1)

	if err != nil {
		logger.Logger.Errorf("Failed to send gauge value %f to %s with tags %s ", value, metric, tags, err)
	}
}