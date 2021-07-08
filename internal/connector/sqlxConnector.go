package connector

import (
	"context"
	"fmt"

	"github.com/IDN-Media/awards/internal/config"
	"github.com/jmoiron/sqlx"

	//Anonymous import for mysql initialization
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// Repository is the database structure
type Repository struct {
	DB *sqlx.DB
}

// DbConnector is a low level interface for sqlx.DB, declared as interface here so we can mock later
type DbConnector interface {
	Connect(string, string) (*sqlx.DB, error)
}

var (
	// Repo is the Repository instance
	Repo    *Repository // TODO: should this be not global?
	sqlxLog = log.WithField("module", "sqlx")
)

// Connect implementation for the connect interface
func (r *Repository) Connect(ctx context.Context, driverName string, dataSourceName string) (*sqlx.DB, error) {
	return sqlx.ConnectContext(ctx, driverName, dataSourceName)
}

// InitDBInstance creates a repository instance, will read from config and returns *Repository, error if any
func (r *Repository) InitDBInstance(ctx context.Context) (*Repository, error) {
	logf := sqlxLog.WithField("fn", "InitDBInstance")

	dbHost := config.Get("db.host")
	dbPort := config.Get("db.port")
	dbUser := config.Get("db.user")
	dbPass := config.Get("db.password")
	dbName := config.Get("db.name")

	sqlConnStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	// sqlx.Connect("dbType", "dbUser:dbPassword@(dbURL:PORT)/dbName")
	db, err := r.Connect(ctx, "mysql", sqlConnStr)
	if err != nil {
		logf.Error("Connection to database error: ", err)
		return nil, err
	}
	logf.Info("db opened and PINGed successfully")

	// Connect and check the server version
	var version string
	db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	logf.Info("DB server version:", version)
	r.DB = db
	Repo = r // assign db instances back to package variable
	return Repo, nil
}

// CloseDB closes the db connection
func (r *Repository) CloseDB() error {
	return r.DB.Close()
}

// GetRepo returns the instace Repo
func GetRepo() *Repository {
	return Repo
}
