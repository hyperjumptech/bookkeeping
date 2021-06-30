package router

import (
	"fmt"
	"strings"

	"github.com/IDN-Media/awards/internal/config"
	"github.com/IDN-Media/awards/internal/health"
	"github.com/IDN-Media/awards/internal/logger"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Router is a wrapper for all the router connections
type Router struct {
	Router *mux.Router // point to mux routers
	// Handlers *handlers.Handlers // point to handlers
}

var rLog = log.WithField("module", "router")

// NewRouter get new Instance
func NewRouter() *Router {
	return &Router{}
}

// InitRoutes creates our routes
func InitRoutes(router *Router) {
	rLog.WithField("fn", "InitRoutes()").Info("Initializing routes...")

	r := router.Router

	// register middlewares
	// r.Use(apmgorilla.Middleware()) // apmgorilla.Instrument(r.MuxRouter) // elastic apm
	r.Use(logger.Logger) // your faithfull logger

	// health check endpoint. Not in a version path as it will seems to be a permanent endpoint (famous last words)
	r.HandleFunc("/health", health.Health).Methods("GET")

	// display routes under development
	if config.Get("app.env") == "development" {
		walk(*r)
	}
}

// walk runs the mux.Router.Walk method to print all the registered routes
func walk(r mux.Router) {
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})

	if err != nil {
		log.Error(err)
	}
}
