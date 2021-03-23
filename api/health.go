package api

import (
	"github.com/shared-spotify/httputils"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	// Add healthchecks here
	httputils.SendOk(w)
}
