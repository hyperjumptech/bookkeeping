package middlewares

import (
	"context"
	"github.com/IDN-Media/awards/internal/contextkeys"
	"github.com/hyperjumptech/acccore"
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

func SetupContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()
		if rctx == nil {
			rctx = context.Background()
		}
		xRequestID := r.Header.Get("X-Request-ID")
		if len(xRequestID) == 0 {
			xRequestID = reqIDUniqueGen.NewUniqueID()
		}
		keyedContext := context.WithValue(rctx, contextkeys.XRequestID, xRequestID)
		next.ServeHTTP(w, r.WithContext(keyedContext))
	})
}
