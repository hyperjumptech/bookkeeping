package health

import (
	"net/http"
	"time"

	"github.com/IDN-Media/awards/internal/config"
	"github.com/IDN-Media/awards/internal/helpers"
)

// Health returns a simple health status
func Health(w http.ResponseWriter, r *http.Request) {

	data := map[string]string{
		"version": config.Get("app.version"),
		"app.id":  config.Get("app.id"),
		"status":  "OK",
		"time":    time.Now().Format(time.RFC3339),
	}

	helpers.HTTPResponseBuilder(r.Context(), w, r, http.StatusOK, "Health status", data, 0)
}
