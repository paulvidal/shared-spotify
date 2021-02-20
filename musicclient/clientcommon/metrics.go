package clientcommon

import "github.com/shared-spotify/datadog"

func SendRequestMetric(metricName string, requestType string, authenticated bool, err error)  {
	success := err == nil

	datadog.Increment(1, metricName,
		datadog.RequestType.Tag(requestType),
		datadog.Authenticated.TagBool(authenticated),
		datadog.Success.TagBool(success),
	)
}