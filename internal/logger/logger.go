package logger

import (
	"context"
	"net/http"
	"strings"

	"github.com/IDN-Media/awards/internal/config"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// ContextKey is used for context.Context value. The value requires a key that is not primitive type.
type ContextKey string

const (
	// ContextKeyRequestID is the contextKey key name (string)
	ContextKeyRequestID ContextKey = "requestID"
)

// ConfigureLogging set logging lever from config
func ConfigureLogging() {
	lLevel := config.Get("server.log.level")
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Setting log level to: ", lLevel)
	switch strings.ToUpper(lLevel) {
	default:
		log.Info("Unknown level [", lLevel, "]. Log level set to ERROR")
		log.SetLevel(log.ErrorLevel)
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	}
}

// Logger middleware handles some logging with logrus
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		requestHdr := r.Header.Get("x-request-id") // propagate request-id if exist to all context
		if requestHdr == "" {
			requestHdr = uuid.New().String()
		}
		r = r.WithContext(context.WithValue(ctx, ContextKeyRequestID, requestHdr))

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

// GetRequestID will get reqID from a http request and return it as a string
func GetRequestID(ctx context.Context) string {
	reqID := ctx.Value(ContextKeyRequestID)
	if ret, ok := reqID.(string); ok {
		return ret
	}
	return "-"
}
