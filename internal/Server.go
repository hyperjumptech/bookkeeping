package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IDN-Media/awards/internal/config"
	"github.com/IDN-Media/awards/internal/logger"
	"github.com/IDN-Media/awards/internal/router"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var (
	srvLog = log.WithField("module", "server")

	// StartUpTime records first ime up
	startUpTime time.Time
	// ServerVersion is a semver versioning
	serverVersion string

	// HTTPServer object
	HTTPServer *http.Server

	// AppRouter object
	appRouter *router.Router
	// AppHandlers is all the handlers
	// appHandlers *handlers.Handlers
	// Address of server
	address string
	// Repo is the database object
	// sqxRepo *connector.DbPool

)

// InitializeServer initializes all server connections
func InitializeServer() error {
	logf := srvLog.WithField("fn", "InitializeServer")

	startUpTime = time.Now()

	config.LoadConfig()
	logger.ConfigureLogging() // configure logging

	logf.Info("setting up routing...")
	appRouter = router.NewRouter()
	appRouter.Router = mux.NewRouter()

	// appHandlers = handlers.NewHandlers(repo)
	// appRouter.Handlers = appHandlers

	logf.Info("initializing routes...")
	router.InitRoutes(appRouter)

	address = fmt.Sprintf("%s:%s", config.Get("server.host"), config.Get("server.port"))
	HTTPServer = &http.Server{
		Addr:         address,
		WriteTimeout: time.Second * 15, // Good practice to set timeouts to avoid Slowloris attacks.
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      appRouter.Router, // Pass our instance of gorilla/mux in.
	}

	return nil
}

// shutdownServer handles shutdown gracefully, clossing connections, flushing caches etc.
func shutdownServer() error {
	logf := srvLog.WithField("fn", "shutdownServer")

	// ctx := context.Background()
	// sqxRepo.CloseConnection()
	logf.Info("done: db closed")

	return nil
}

// StartServer starts listening at given port
func StartServer() {

	var wait time.Duration
	logf := srvLog.WithField("fn", "StartServer")

	logf.Info("initializing server...")
	err := InitializeServer()
	if err != nil {
		logf.Error(err)
	}
	defer shutdownServer()

	logf.Info("starting server...")
	logf.Info("App version: ", serverVersion, ", listening at: ", address)
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := HTTPServer.ListenAndServe(); err != nil {
			logf.Error(err)
		}
	}()

	gracefulStop := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(gracefulStop, os.Interrupt)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	// Block until we receive our signal.
	<-gracefulStop

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	HTTPServer.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	logf.Info("shutting down........ bye")

	t := time.Now()
	upTime := t.Sub(startUpTime)
	fmt.Println("server was up for : ", upTime.String(), " *******")
	os.Exit(0)
}
