package connector

import (
	"context"
	"fmt"

	"github.com/JamesStewy/go-mysqldump"

	//Anonymous import for mysql initialization
	_ "github.com/go-sql-driver/mysql"
)

// DumpDB dumps the repository into a file
func (r *MySQLDBRepository) DumpDB(ctx context.Context) (string, error) {
	logf := mysqlLog.WithField("fn", "dumpDB")

	if !r.IsConnected() {
		logf.Error("database is not connected, exiting dumpDB")
		return "", fmt.Errorf("database is not connected, exiting dumpDB")
	}

	dumper, err := mysqldump.Register(r.db.DB, "./", "bookeepingBackup-20060102T1504")
	if err != nil {
		logf.Error("error registering db, got: ", err)
		return "", fmt.Errorf("error registering db, got: %v", err)
	}

	// Dump database to file
	resultFilename, err := dumper.Dump()
	if err != nil {
		logf.Error("error dumping db, got: ", err)
		return "", fmt.Errorf("error dumping db, got: %v", err)
	}
	logf.Info("Dump file saved to ", resultFilename)

	return resultFilename, nil
}
