package health

import (
	"context"
	"errors"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/hyperjumptech/bookkeeping/internal/config"
	"github.com/hyperjumptech/bookkeeping/internal/connector"
	log "github.com/sirupsen/logrus"
)

var (
	healthLog = log.WithField("module", "health")

	// H health instance
	H gosundheit.Health
)

// InitializeHealthCheck initializes health monitors
func InitializeHealthCheck(ctx context.Context, repo *connector.MySQLDBRepository) error {
	logf := healthLog.WithField("fn", "InitializeHealthCheck")

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !repo.IsConnected() {
		return errors.New("Db not connected, healthcheck disabled")
	}
	// create a new health instance
	H = gosundheit.New()

	// dependency check to localhost:8080
	url := config.Get("health.local")
	httpCheckConf := checks.HTTPCheckConfig{
		CheckName: "DNS.url.check",
		Timeout:   1 * time.Second,
		URL:       url,
	}

	d := time.Duration(config.GetInt("health.delay"))
	i := time.Duration(config.GetInt("health.interval"))

	httpCheck, err := checks.NewHTTPCheck(httpCheckConf)
	if err != nil {
		logf.Error("could not setup httpCheck")
	}
	err = H.RegisterCheck(
		httpCheck,
		gosundheit.InitialDelay(d*time.Second),    // the check will run once after 1 sec
		gosundheit.ExecutionPeriod(i*time.Second), // the check will be executed every 60 sec
	)
	if err != nil {
		logf.Error("Failed to register check(s): ", err)
	}

	// For checking database connections
	dbCheck, err := checks.NewPingCheck("db.check", repo.DB().DB)
	if err != nil {
		logf.Error("could not setup dbCheck")
	}
	err = H.RegisterCheck(
		dbCheck,
		gosundheit.InitialDelay(d*time.Second), // the check will run once after 1 sec
		gosundheit.ExecutionPeriod(i*time.Second), // the check will be executed every 60 sec
	)
	if err != nil {
		logf.Error("Failed to register check(s): ", err)
	}

	return nil
}
