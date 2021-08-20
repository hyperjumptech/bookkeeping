package middlewares

import (
	"context"
	"github.com/IDN-Media/awards/internal/contextkeys"
	"github.com/hyperjumptech/acccore"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	reqIDUniqueGen = &acccore.RandomGenUniqueIDGenerator{
		Length:     10,
		LowerAlpha: true,
		UpperAlpha: true,
		Numeric:    true,
	}
)

// SetupContextMiddleware will check if the current request already contains a context or not
// If it do not contain a context, a new context from background will be used and inserted into the request.
// The context is then injected with XRequestID key taken from the request header (or a new request id if
// theres no such header). This will be useful to chain the logs based on the request.
func SetupContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()
		if rctx == nil {
			log.Debug("Creating new context")
			rctx = context.Background()
		} else {
			log.Debug("Using existing context")
		}
		xRequestID := r.Header.Get("X-Request-ID")
		if len(xRequestID) == 0 {
			xRequestID = reqIDUniqueGen.NewUniqueID()
		}
		keyedContext := context.WithValue(rctx, contextkeys.XRequestID, xRequestID)
		next.ServeHTTP(w, r.WithContext(keyedContext))
	})
}
