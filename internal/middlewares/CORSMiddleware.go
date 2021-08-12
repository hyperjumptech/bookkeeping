package middlewares

import (
	"github.com/rs/cors"
	"net/http"
)

var (
	theCors *cors.Cors
)

func init() {
	theCors = cors.New(cors.Options{
		AllowedOrigins:         []string{"*"},
		AllowOriginFunc:        nil,
		AllowOriginRequestFunc: nil,
		AllowedMethods:         []string{http.MethodOptions, http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPut, http.MethodHead},
		AllowedHeaders:         []string{"*", "Authorization"},
		ExposedHeaders:         []string{"*", "Authorization"},
		MaxAge:                 60 * 60 * 24 * 365,
		AllowCredentials:       true,
		OptionsPassthrough:     false,
		Debug:                  true,
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return theCors.Handler(next)
}
