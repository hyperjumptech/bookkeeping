package middlewares

import (
	"net/http"

	"github.com/hyperjumptech/bookkeeping/internal/contextkeys"
	log "github.com/sirupsen/logrus"
)

// Logger middleware handles some logging with logrus
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/devkey" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		requestHdr := ctx.Value(contextkeys.XRequestID).(string)
		// Log this request
		log.WithFields(log.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"header":     r.Header,
			"request-id": requestHdr,
		}).Debug("Logger")
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
		log.Debug("Done with http, request-id: ", requestHdr)
	})
}
