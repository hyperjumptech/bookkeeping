package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperjumptech/acccore"
	"github.com/hyperjumptech/hyperwallet/internal/accounting"

	"github.com/gorilla/mux"
	"github.com/hyperjumptech/hyperwallet/internal/config"
	"github.com/hyperjumptech/hyperwallet/internal/connector"
	"github.com/hyperjumptech/hyperwallet/internal/health"
	"github.com/hyperjumptech/hyperwallet/internal/logger"
	"github.com/hyperjumptech/hyperwallet/internal/router"
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

	// Address of server
	address string

	// dbRepo database repository
	dbRepo connector.DBRepository
)

// InitializeServer initializes all server connections
func InitializeServer() error {
	logf := srvLog.WithField("fn", "InitializeServer")

	// configure logging
	logger.ConfigureLogging()

	startUpTime = time.Now()
	// load system / env configs
	config.LoadConfig()

	t := time.Duration(config.GetInt("server.context.timeout"))
	ctx, cancel := context.WithTimeout(context.Background(), t*time.Second)
	defer cancel()

	logf.Info("setting up routing...")
	appRouter = router.NewRouter()
	appRouter.Router = mux.NewRouter()

	// setup db connection
	dbRepo = &connector.MySQLDBRepository{}
	err := dbRepo.Connect(ctx)
	if err != nil {
		logf.Fatal("could not connect to db. Error: ", err)
		panic("DB connection failed. please check log.")
	}

	accounting.AccountMgr = accounting.NewMySQLAccountManager(dbRepo)
	accounting.JournalMgr = accounting.NewMySQLJournalManager(dbRepo)
	accounting.TransactionMgr = accounting.NewMySQLTransactionManager(dbRepo)
	accounting.ExchangeMgr = accounting.NewMySQLExchangeManager(dbRepo)
	accounting.UniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
		Length:     16,
		LowerAlpha: false,
		UpperAlpha: true,
		Numeric:    true,
	}

	// setup health monitoring
	err = health.InitializeHealthCheck(ctx, dbRepo.DB())
	if err != nil {
		logf.Warn("health monitor error: ", err)
	}

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
	logf.Info("App version: ", config.Get("app.version"), ", listening at: ", address)
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
