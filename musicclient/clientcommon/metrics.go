package clientcommon

import "github.com/shared-spotify/datadog"

func SendRequestMetric(provider string, requestType string, authenticated bool, err error) {
	success := err == nil

	datadog.Increment(1, datadog.ApiRequests,
		datadog.Provider.Tag(provider),
		datadog.RequestType.Tag(requestType),
		datadog.Authenticated.TagBool(authenticated),
		datadog.Success.TagBool(success),
	)
}
