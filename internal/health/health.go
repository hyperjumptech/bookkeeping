package health

import (
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/IDN-Media/awards/internal/config"
	log "github.com/sirupsen/logrus"
)

var (
	healthLog = log.WithField("module", "health")

	// H health instance
	H gosundheit.Health
)

// InitializeHealthCheck initializes health monitors
func InitializeHealthCheck() error {
	logf := healthLog.WithField("fn", "InitializeHealthCheck")

	// create a new health instance
	H = gosundheit.New()

	// dependency check to localhost:8080
	url := config.Get("health.local")
	httpCheckConf := checks.HTTPCheckConfig{
		CheckName: "DNS.url.check",
		Timeout:   1 * time.Second,
		URL:       url,
	}

	// For checking database connections
	// db, err := sql.Open(...)
	// dbCheck, err := checks.NewPingCheck("db.check", db)
	// _ = h.RegisterCheck(&gosundheit.Config{
	// 	Check: dbCheck,
	// 	// ...
	// })

	httpCheck, err := checks.NewHTTPCheck(httpCheckConf)
	if err != nil {
		logf.Error("could not setup httpCheck")
	}

	err = H.RegisterCheck(
		httpCheck,
		gosundheit.InitialDelay(time.Second),       // the check will run once after 1 sec
		gosundheit.ExecutionPeriod(60*time.Second), // the check will be executed every 60 sec
	)
	if err != nil {
		logf.Error("Failed to register check(s): ", err)
	}

	return nil

}
