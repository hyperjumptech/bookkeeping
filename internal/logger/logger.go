package logger

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MyLogger does some stuff
func MyLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// TODO: include and propagate requestID

		if strings.HasPrefix(r.URL.Path, "/docs123123") {
			log.Info("skipping /docs logging.")
		} else {
			// Do stuff
			log.WithFields(log.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
				"header": r.Header,
			}).Debug("Logger")
			//.Info("Logger")

		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
