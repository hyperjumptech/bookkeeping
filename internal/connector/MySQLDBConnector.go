package connector

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/IDN-Media/awards/internal/contextkeys"
	"time"

	"github.com/IDN-Media/awards/errors"
	"github.com/IDN-Media/awards/internal/config"
	"github.com/hyperjumptech/acccore"
	"github.com/jmoiron/sqlx"
)

var (
	mysqlLog = log.WithField("file", "MySQLDBConnector.go")
)

// MySqlDBRepository is implementation of DBRepository specified for MySQL database
type MySqlDBRepository struct {
	db        *sqlx.DB
	connected bool
}

// ClearTables clear all table for testing purpose
func (repo *MySqlDBRepository) ClearTables(ctx context.Context) error {
	lLog := mysqlLog.WithField("function", "ClearTables")
	tablesToDrop := []string{"accounts", "currencies", "journals", "transactions"}
	for _, t := range tablesToDrop {
		_, err := repo.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", t))
		if err != nil {
			lLog.Errorf("error dropping table %s. got %s", t, err.Error())
			return err
		}
	}
	return nil
}

// Connect connect the repository to the database, it uses the configuration internally for connection arguments and parameters.
func (repo *MySqlDBRepository) Connect(ctx context.Context) error {
	lLog := mysqlLog.WithField("function", "Connect")

	dbHost := config.Get("db.host")
	dbPort := config.Get("db.port")
	dbUser := config.Get("db.user")
	dbPass := config.Get("db.password")
	dbName := config.Get("db.name")

	sqlConnStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sqlx.ConnectContext(ctx, "mysql", sqlConnStr)
	if err != nil {
		lLog.Errorf("Connection to database error. got %s", err)
		return errors.ErrDBConnectingFailed
	}
	lLog.Info("DB opened and PINGed successfully")

	// Connect and check the server version
	var version string
	err = db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		lLog.Warnf("unable to obtain DB server version")
		version = "UNKNOWN"
	}
	lLog.Info("DB server version:", version)
	repo.db = db
	repo.connected = true
	return nil
}

// Disconnect the already establshed connection. Throws error if the underlying database connection yield an error
func (repo *MySqlDBRepository) Disconnect() error {
	lLog := mysqlLog.WithField("function", "Disconnect")

	defer func() {
		repo.connected = false
		repo.db = nil
	}()
	err := repo.db.Close()
	if err != nil {
		lLog.Errorf("error while disconnecting. Got %s", err.Error())
	}
	return err
}

// IsConnected check if the connection is already established
func (repo *MySqlDBRepository) IsConnected() bool {
	if repo.db == nil || !repo.connected {
		return false
	}
	return true
}

// DB the database connection object.
func (repo *MySqlDBRepository) DB() *sqlx.DB {
	return repo.db
}

// InsertAccount insert an entity record of account into database.
// Throws error if the underlying connection have problem.
// The rec argument contains the Account information to be written.
// It returns the account number that written into database.
// The AccountNumber contained within the rec MUST NOT be persisted before.
func (repo *MySqlDBRepository) InsertAccount(ctx context.Context, rec *AccountRecord) (string, error) {
	lLog := mysqlLog.WithField("function", "InsertAccount")

	if len(rec.CurrencyCode) > 10 {
		lLog.Errorf("Currency code %s is too long. Should not more than 10 digit", rec.CurrencyCode)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.Name) > 128 {
		lLog.Errorf("Account name %s is too long. Should not more than 128 digit", rec.Name)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.AccountNumber) > 20 {
		lLog.Errorf("Account Number %s is too long. Should not more than 20 digit", rec.AccountNumber)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.Coa) > 10 {
		lLog.Errorf("COA %s is too long. Should not more than 10 digit", rec.Coa)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}
	if len(rec.UpdatedBy) > 16 {
		rec.CreatedBy = rec.UpdatedBy[:16]
	}

	theUser, ok := ctx.Value(contextkeys.UserIDContextKey).(string)
	if !ok {
		lLog.Errorf("UserContext Key %s is not in context", contextkeys.UserIDContextKey)
		return "", errors.ErrUserContextKeyMissing
	}
	rec.UpdatedBy = theUser
	rec.UpdatedAt = time.Now()
	rec.CreatedBy = theUser
	rec.CreatedAt = time.Now()
	q := "INSERT INTO accounts(" +
		"account_number, name, currency_code, description, alignment, balance, coa, created_at, created_by, updated_at, updated_by, is_deleted" +
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, false)"
	args := []interface{}{
		rec.AccountNumber, rec.Name, rec.CurrencyCode, rec.Description, rec.Alignment, rec.Balance, rec.Coa, rec.CreatedAt, rec.CreatedBy, rec.UpdatedAt, rec.UpdatedBy,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error when inserting account. got %s", err.Error())
		return "", err
	}
	return rec.AccountNumber, nil
}

// UpdateAccount update an account entity record in the database.
// Throws error if the underlying database connection has problem.
// The rec argument contains the Account information to be updated.
// The AccountNumber contained within the rec MUST be already persisted before.
func (repo *MySqlDBRepository) UpdateAccount(ctx context.Context, rec *AccountRecord) error {
	lLog := mysqlLog.WithField("function", "UpdateAccount")

	if len(rec.CurrencyCode) > 10 {
		lLog.Errorf("Currency code %s is too long. Should not more than 10 digit", rec.CurrencyCode)
		return errors.ErrStringDataTooLong
	}
	if len(rec.Name) > 128 {
		lLog.Errorf("Account name %s is too long. Should not more than 128 digit", rec.Name)
		return errors.ErrStringDataTooLong
	}
	if len(rec.AccountNumber) > 20 {
		lLog.Errorf("Account Number %s is too long. Should not more than 20 digit", rec.AccountNumber)
		return errors.ErrStringDataTooLong
	}
	if len(rec.Coa) > 10 {
		lLog.Errorf("COA %s is too long. Should not more than 10 digit", rec.Coa)
		return errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}
	if len(rec.UpdatedBy) > 16 {
		rec.CreatedBy = rec.UpdatedBy[:16]
	}

	theUser, ok := ctx.Value(contextkeys.UserIDContextKey).(string)
	if !ok {
		lLog.Errorf("UserContext Key %s is not in context", contextkeys.UserIDContextKey)
		return errors.ErrUserContextKeyMissing
	}
	rec.UpdatedBy = theUser
	rec.UpdatedAt = time.Now()
	q := "UPDATE accounts set" +
		" name=?, currency_code=?, description=?, alignment=?, balance=?, coa=?, created_at=?, created_by=?, updated_at=?, updated_by=?" +
		" WHERE account_number=? AND is_deleted=false"
	args := []interface{}{
		rec.Name, rec.CurrencyCode, rec.Description, rec.Alignment, rec.Balance, rec.Coa, rec.CreatedAt, rec.CreatedBy, rec.UpdatedAt, rec.UpdatedBy, rec.AccountNumber,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while updating account. got %s", err.Error())
		return err
	}
	return nil
}

// DeleteAccount soft/logical delete an account.
// Throws error if the underlying database connection has problem.
// If the account number not exist, it will do nothing and return nil.
func (repo *MySqlDBRepository) DeleteAccount(ctx context.Context, accountNumber string) error {
	lLog := mysqlLog.WithField("function", "DeleteAccount")
	q := "UPDATE accounts " +
		"set is_deleted=true" +
		" WHERE account_number=? && is_deleted=true"
	args := []interface{}{
		accountNumber,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while deleting account. got %s", err.Error())
		return err
	}
	return nil
}

// ListAccount will list account in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of AcccountRecords
func (repo *MySqlDBRepository) ListAccount(ctx context.Context, sort string, offset, length int) ([]*AccountRecord, error) {
	lLog := mysqlLog.WithField("function", "ListAccount")
	q := "SELECT account_number, name, currency_code, description, alignment, balance, coa, created_at, created_by, updated_at, updated_by" +
		" FROM accounts WHERE is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	lLog.Infof("Q = %s", q)
	rows, err := repo.db.QueryxContext(ctx, q, offset, length)
	if err != nil {
		lLog.Errorf("error while listing account. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*AccountRecord, 0)
	for rows.Next() {
		ar := &AccountRecord{}
		err := rows.Scan(&ar.AccountNumber, &ar.Name, &ar.CurrencyCode, &ar.Description, &ar.Alignment, &ar.Balance, &ar.Coa, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// CountAccounts will return a number of accounts in database.
// Throws error if the underlying database connection has problem.
// It will returns total number of accounts in the database.
func (repo *MySqlDBRepository) CountAccounts(ctx context.Context) (int, error) {
	lLog := mysqlLog.WithField("function", "CountAccounts")
	q := "SELECT COUNT(*) as accountCounts" +
		" FROM accounts WHERE is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q)
	if row.Err() != nil {
		lLog.Errorf("error while counting account. got %s", row.Err().Error())
		return 0, row.Err()
	}
	count := 0
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListAccountByCoa will list all account that have the specified COA, the list presented in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of AcccountRecords
func (repo *MySqlDBRepository) ListAccountByCoa(ctx context.Context, coa string, sort string, offset, length int) ([]*AccountRecord, error) {
	lLog := mysqlLog.WithField("function", "ListAccountByCoa")
	q := "SELECT account_number, name, currency_code, description, alignment, balance, coa, created_at, created_by, updated_at, updated_by" +
		" FROM accounts WHERE coa LIKE ? AND is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, coa, offset, length)
	if err != nil {
		lLog.Errorf("error while listing account by coa. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*AccountRecord, 0)
	for rows.Next() {
		ar := &AccountRecord{}
		err := rows.Scan(&ar.AccountNumber, &ar.Name, &ar.CurrencyCode, &ar.Description, &ar.Alignment, &ar.Balance, &ar.Coa, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// CountAccountByCoa will return a number of accounts in database that belong to the specified COA number.
// Throws error if the underlying database connection has problem.
// It will returns total number of accounts in the database.
func (repo *MySqlDBRepository) CountAccountByCoa(ctx context.Context, coa string) (int, error) {
	lLog := mysqlLog.WithField("function", "CountAccountByCoa")
	q := "SELECT COUNT(*) as accountCounts" +
		" FROM accounts WHERE coa LIKE ? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, coa)
	if row.Err() != nil {
		lLog.Errorf("error while counting account by coa. got %s", row.Err().Error())
		return 0, row.Err()
	}
	count := 0
	err := row.Scan(&count)
	if err != nil {
		lLog.Errorf("error while scanning count of account by coa. got %s", err.Error())
		return 0, err
	}
	return count, nil
}

// FindAccountByName will list all account that have the specified name, the list presented in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of AcccountRecords
func (repo *MySqlDBRepository) FindAccountByName(ctx context.Context, nameLike string, sort string, offset, length int) ([]*AccountRecord, error) {
	lLog := mysqlLog.WithField("function", "FindAccountByName")
	q := "SELECT account_number, name, currency_code, description, alignment, balance, coa, created_at, created_by, updated_at, updated_by" +
		" FROM accounts WHERE name LIKE ? AND is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, nameLike, offset, length)
	if err != nil {
		lLog.Errorf("error while finding accounts by name. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*AccountRecord, 0)
	for rows.Next() {
		ar := &AccountRecord{}
		err := rows.Scan(&ar.AccountNumber, &ar.Name, &ar.CurrencyCode, &ar.Description, &ar.Alignment, &ar.Balance, &ar.Coa, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// CountAccountByName will return a number of accounts in database that have the name like the specified in the argument..
// Throws error if the underlying database connection has problem.
// It will returns total number of accounts in the database.
func (repo *MySqlDBRepository) CountAccountByName(ctx context.Context, nameLike string) (int, error) {
	lLog := mysqlLog.WithField("function", "CountAccountByName")
	q := "SELECT COUNT(*) as accountCounts" +
		" FROM accounts WHERE name LIKE ? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, nameLike)
	if row.Err() != nil {
		lLog.Errorf("error while counting account by name. got %s", row.Err().Error())
		return 0, row.Err()
	}
	count := 0
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAccount retrieves an AccountRecord from database where the account number is specified.
// Throws error if  the underlying database connection has problem.
// It returns an instance of AccountRecord or nil if there is no Account with
// specified accountNumber.
func (repo *MySqlDBRepository) GetAccount(ctx context.Context, accountNumber string) (*AccountRecord, error) {
	lLog := mysqlLog.WithField("function", "GetAccount")
	q := "SELECT account_number, name, currency_code, description, alignment, balance, coa, created_at, created_by, updated_at, updated_by" +
		" FROM accounts WHERE account_number=? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, accountNumber)
	if row.Err() != nil {
		lLog.Errorf("error while retrieving account by account number. got %s", row.Err().Error())
		return nil, row.Err()
	}
	ar := &AccountRecord{}
	err := row.Scan(&ar.AccountNumber, &ar.Name, &ar.CurrencyCode, &ar.Description, &ar.Alignment, &ar.Balance, &ar.Coa, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		lLog.Errorf("error while scanning count of account by account number. got %s", err.Error())
		return nil, err
	}
	return ar, nil
}

// InsertJournal will insert the data specified in the rec argument into database
// will return error if the underlying database connection has problem. or if the
// journalID, or Transaction ID in the journal already in the database.
// Will return the JournalID saved if successful.
func (repo *MySqlDBRepository) InsertJournal(ctx context.Context, rec *JournalRecord) (string, error) {
	lLog := mysqlLog.WithField("function", "InsertJournal")

	theUser, ok := ctx.Value(contextkeys.UserIDContextKey).(string)
	if !ok {
		lLog.Errorf("UserContext Key %s is not in context", contextkeys.UserIDContextKey)
		return "", errors.ErrUserContextKeyMissing
	}

	if len(rec.JournalID) > 20 {
		lLog.Errorf("JournalID %s is too long. Should not more than 20 digit", rec.JournalID)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.ReversedJournalId) > 128 {
		lLog.Errorf("Reversed journal id %s is too long. Should not more than 20 digit", rec.ReversedJournalId)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}

	rec.CreatedBy = theUser
	rec.CreatedAt = time.Now()
	q := "INSERT INTO journals(" +
		"journal_id, journaling_time, description, is_reversal, reversed_journal_id, total_amount, created_at, created_by, updated_at, updated_by, is_deleted" +
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	args := []interface{}{
		rec.JournalID, rec.JournalingTime, rec.Description, rec.IsReversal, rec.ReversedJournalId, rec.TotalAmount, rec.CreatedAt, rec.CreatedBy, rec.CreatedAt, rec.CreatedBy, false,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while inserting journal. got %s", err.Error())
		return "", err
	}
	return rec.JournalID, nil
}

// UpdateJournal update an journal entity record in the database.
// Throws error if the underlying database connection has problem.
// The rec argument contains the Journal information to be updated.
// The JournalID contained within the rec MUST be already persisted before.
func (repo *MySqlDBRepository) UpdateJournal(ctx context.Context, rec *JournalRecord) error {
	lLog := mysqlLog.WithField("function", "UpdateJournal")
	theUser, ok := ctx.Value(contextkeys.UserIDContextKey).(string)
	if !ok {
		lLog.Errorf("UserContext Key %s is not in context", contextkeys.UserIDContextKey)
		return errors.ErrUserContextKeyMissing
	}

	if len(rec.JournalID) > 20 {
		lLog.Errorf("JournalID %s is too long. Should not more than 20 digit", rec.JournalID)
		return errors.ErrStringDataTooLong
	}
	if len(rec.ReversedJournalId) > 128 {
		lLog.Errorf("Reversed journal id %s is too long. Should not more than 20 digit", rec.ReversedJournalId)
		return errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}

	rec.CreatedBy = theUser
	rec.CreatedAt = time.Now()
	q := "UPDATE journals " +
		"set journaling_time=?, description=?, is_reversal=?, reversed_journal_id=?, total_amount=?, updated_at=?, updated_by=?" +
		" WHERE journal_id=? AND is_deleted=false"
	args := []interface{}{
		rec.JournalingTime, rec.Description, rec.IsReversal, rec.ReversedJournalId, rec.TotalAmount, time.Now(), theUser, rec.JournalID,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while updating journal. got %s", err.Error())
		return err
	}
	return nil
}

// DeleteJournal soft/logical delete an journal.
// Throws error if the underlying database connection has problem.
// If the JournalID not exist, it will do nothing and return nil.
func (repo *MySqlDBRepository) DeleteJournal(ctx context.Context, journalID string) error {
	lLog := mysqlLog.WithField("function", "DeleteJournal")
	q := "UPDATE journals " +
		"set is_deleted=true" +
		" WHERE journal_id=? && is_deleted=true"
	args := []interface{}{
		journalID,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while deleting journal. got %s", err.Error())
		return err
	}
	return nil
}

// ListJournal will list journals in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return JournalRecord sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of JournalRecord
func (repo *MySqlDBRepository) ListJournal(ctx context.Context, sort string, offset, length int) ([]*JournalRecord, error) {
	lLog := mysqlLog.WithField("function", "ListJournal")
	q := "SELECT journal_id, journaling_time, description, is_reversal, reversed_journal_id, total_amount, created_at, created_by" +
		" FROM journals WHERE is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, offset, length)
	if err != nil {
		lLog.Errorf("error while listing journals. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*JournalRecord, 0)
	for rows.Next() {
		ar := &JournalRecord{}
		err := rows.Scan(&ar.JournalID, &ar.JournalingTime, &ar.Description, &ar.Description, &ar.IsReversal, &ar.ReversedJournalId, &ar.TotalAmount, &ar.CreatedAt, &ar.CreatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// GetJournal retrieves an JournalRecord from database where the journalID is specified.
// Throws error if  the underlying database connection has problem.
// It returns an instance of JournalRecord or nil if there is no Journal with
// specified journalID.
func (repo *MySqlDBRepository) GetJournal(ctx context.Context, journalID string) (*JournalRecord, error) {
	lLog := mysqlLog.WithField("function", "GetJournal")
	q := "SELECT  journal_id, journaling_time, description, is_reversal, reversed_journal_id, total_amount, created_at, created_by" +
		" FROM journals WHERE journal_id=? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, journalID)
	if row.Err() != nil {
		lLog.Errorf("error while retrieving journal by journalID. got %s", row.Err().Error())
		return nil, row.Err()
	}
	ar := &JournalRecord{}
	err := row.Scan(&ar.JournalID, &ar.JournalingTime, &ar.Description, &ar.IsReversal, &ar.ReversedJournalId, &ar.TotalAmount, &ar.CreatedAt, &ar.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		lLog.Errorf("error while scanning listing row. got %s", err.Error())
		return nil, err
	}
	return ar, nil
}

// GetJournalByReversalID retrieves an JournalRecord from database where the reversedJournalID is specified.
// Throws error if  the underlying database connection has problem.
// It returns an instance of JournalRecord or nil if there is no Journal with
// specified reversedJournalID.
func (repo *MySqlDBRepository) GetJournalByReversalID(ctx context.Context, journalID string) (*JournalRecord, error) {
	lLog := mysqlLog.WithField("function", "GetJournalByReversalID")
	q := "SELECT  journal_id, journaling_time, description, is_reversal, reversed_journal_id, total_amount, created_at, created_by" +
		" FROM journals WHERE reversed_journal_id=? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, journalID)
	if row.Err() != nil {
		lLog.Errorf("error while retriving journals by reversal id. got %s", row.Err().Error())
		return nil, row.Err()
	}
	ar := &JournalRecord{}
	err := row.Scan(&ar.JournalID, &ar.JournalingTime, &ar.Description, &ar.Description, &ar.IsReversal, &ar.ReversedJournalId, &ar.TotalAmount, &ar.CreatedAt, &ar.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		lLog.Errorf("error while scanning record when retrieving journal. got %s", err.Error())
		return nil, err
	}
	return ar, nil
}

// ListJournalByTimeRange will list journals in paginated fashion where journal is in the specified time range.
// Throws error if the underlying database connection has problem.
// It will return JournalRecord sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of JournalRecord
func (repo *MySqlDBRepository) ListJournalByTimeRange(ctx context.Context, timeFrom, timeTo time.Time, sort string, offset, length int) ([]*JournalRecord, error) {
	lLog := mysqlLog.WithField("function", "ListJournalByTimeRange")
	q := "SELECT journal_id, journaling_time, description, is_reversal, reversed_journal_id, total_amount, created_at, created_by" +
		" FROM journals WHERE journaling_time > ? AND journaling_time < ? AND is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, timeFrom, timeTo, offset, length)
	if err != nil {
		lLog.Errorf("error while listing journals by time range. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*JournalRecord, 0)
	for rows.Next() {
		ar := &JournalRecord{}
		err := rows.Scan(&ar.JournalID, &ar.JournalingTime, &ar.Description, &ar.Description, &ar.IsReversal, &ar.ReversedJournalId, &ar.TotalAmount, &ar.CreatedAt, &ar.CreatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// CountJournalByTimeRange will return a number of journals in database that been created within the time range.
// Throws error if the underlying database connection has problem.
// It will returns total number of journals in the database.
func (repo *MySqlDBRepository) CountJournalByTimeRange(ctx context.Context, timeFrom, timeTo time.Time) (int, error) {
	lLog := mysqlLog.WithField("function", "CountJournalByTimeRange")
	q := "SELECT COUNT(*) as journalCount" +
		" FROM journals WHERE journaling_time > ? AND journaling_time < ? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, timeFrom, timeTo)
	if row.Err() != nil {
		lLog.Errorf("error while counting journals by time range. got %s", row.Err().Error())
		return 0, row.Err()
	}
	count := 0
	err := row.Scan(&count)
	if err != nil {
		lLog.Errorf("error while scanning journals count when finding journal by time range. got %s", err.Error())
		return 0, err
	}
	return count, nil
}

// InsertTransaction will insert the data specified in the rec argument into database
// will return error if the underlying database connection has problem. or if the
// Transaction ID in the journal already in the database.
// Will return the TransactionID saved if successful.
func (repo *MySqlDBRepository) InsertTransaction(ctx context.Context, rec *TransactionRecord) (string, error) {
	lLog := mysqlLog.WithField("function", "InsertTransaction")

	if len(rec.TransactionID) > 20 {
		lLog.Errorf("TransactionID %s is too long. Should not more than 20 digit", rec.TransactionID)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.JournalID) > 20 {
		lLog.Errorf("JournalID %s is too long. Should not more than 20 digit", rec.JournalID)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.AccountNumber) > 20 {
		lLog.Errorf("AccountNumber %s is too long. Should not more than 20 digit", rec.AccountNumber)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}

	q := "INSERT INTO transactions(" +
		"transaction_id, transaction_time, account_number, journal_id, description, alignment, amount, balance, created_at, created_by, is_deleted" +
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, false)"
	args := []interface{}{
		rec.TransactionID, rec.TransactionTime, rec.AccountNumber, rec.JournalID, rec.Description, rec.Alignment, rec.Amount, rec.Balance, rec.CreatedAt, rec.CreatedBy,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while inserting transaction. got %s", err.Error())
		return "", err
	}
	return rec.JournalID, nil
}

// UpdateTransaction update an transaction entity record in the database.
// Throws error if the underlying database connection has problem.
// The rec argument contains the Transaction information to be updated.
// The TransactionID contained within the rec MUST be already persisted before.
func (repo *MySqlDBRepository) UpdateTransaction(ctx context.Context, rec *TransactionRecord) error {
	lLog := mysqlLog.WithField("function", "UpdateTransaction")

	if len(rec.TransactionID) > 20 {
		lLog.Errorf("TransactionID %s is too long. Should not more than 20 digit", rec.TransactionID)
		return errors.ErrStringDataTooLong
	}
	if len(rec.JournalID) > 20 {
		lLog.Errorf("JournalID %s is too long. Should not more than 20 digit", rec.JournalID)
		return errors.ErrStringDataTooLong
	}
	if len(rec.AccountNumber) > 20 {
		lLog.Errorf("AccountNumber %s is too long. Should not more than 20 digit", rec.AccountNumber)
		return errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}

	q := "UPDATE transactions " +
		"set transaction_time=?, account_number=?, journal_id=?, description=?, alignment=?, amount=?, balance=?, created_at=?, created_by=?" +
		" WHERE journal_id=? and is_deleted=false"
	args := []interface{}{
		rec.TransactionTime, rec.AccountNumber, rec.JournalID, rec.Description, rec.Alignment, rec.Amount, rec.Balance, rec.CreatedAt, rec.CreatedBy, rec.JournalID,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while updating transaction. got %s", err.Error())
		return err
	}
	return nil
}

// DeleteTransaction soft/logical delete a transaction.
// Throws error if the underlying database connection has problem.
// If the TransactionID not exist, it will do nothing and return nil.
func (repo *MySqlDBRepository) DeleteTransaction(ctx context.Context, transactionID string) error {
	lLog := mysqlLog.WithField("function", "DeleteTransaction")
	q := "UPDATE transactions " +
		"set is_deleted=true" +
		" WHERE transaction_id=? && is_deleted=true"
	args := []interface{}{
		transactionID,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while deleting transaction. got %s", err.Error())
		return err
	}
	return nil
}

// ListTransaction will list journals in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return TransactionRecord sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of TransactionRecord
func (repo *MySqlDBRepository) ListTransaction(ctx context.Context, sort string, offset, length int) ([]*TransactionRecord, error) {
	lLog := mysqlLog.WithField("function", "ListTransaction")
	q := "SELECT transaction_id, transaction_time, account_number, journal_id, description, alignment, amount, balance, created_at, created_by" +
		" FROM transactions WHERE is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, offset, length)
	if err != nil {
		lLog.Errorf("error while listing transaction in time-range. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*TransactionRecord, 0)
	for rows.Next() {
		ar := &TransactionRecord{}
		err := rows.Scan(&ar.TransactionID, &ar.TransactionTime, &ar.AccountNumber, &ar.JournalID, &ar.Description, &ar.Alignment, &ar.Amount, &ar.Balance, &ar.CreatedAt, &ar.CreatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// GetTransaction retrieves an TransactionRecord from database where the transactionID is specified.
// Throws error if  the underlying database connection has problem.
// It returns an instance of TransactionRecord  or nil if there is no Transaction with
// specified transactionID.
func (repo *MySqlDBRepository) GetTransaction(ctx context.Context, transactionID string) (*TransactionRecord, error) {
	lLog := mysqlLog.WithField("function", "GetTransaction")
	q := "SELECT  transaction_id, transaction_time, account_number, journal_id, description, alignment, amount, balance, created_at, created_by" +
		" FROM transactions WHERE transaction_id=? and is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, transactionID)
	if row.Err() != nil {
		lLog.Errorf("error while retrieving transaction. got %s", row.Err().Error())
		return nil, row.Err()
	}
	ar := &TransactionRecord{}
	err := row.Scan(&ar.TransactionID, &ar.TransactionTime, &ar.AccountNumber, &ar.JournalID, &ar.Description, &ar.Alignment, &ar.Amount, &ar.Balance, &ar.CreatedAt, &ar.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		lLog.Errorf("error while scanning transaction record. got %s", err.Error())
		return nil, err
	}
	return ar, nil
}

// ListTransactionByAccountNumber will list transactions in paginated fashion, the transaction must belong to the
// specified accountNumber arguments and created within the time rage.
// Throws error if the underlying database connection has problem.
// It will return TransactionRecord sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of TransactionRecord
func (repo *MySqlDBRepository) ListTransactionByAccountNumber(ctx context.Context, accountNumber string, timeFrom, timeTo time.Time, offset, length int) ([]*TransactionRecord, error) {
	lLog := mysqlLog.WithField("function", "ListTransactionByAccountNumber")
	q := "SELECT transaction_id, transaction_time, account_number, journal_id, description, alignment, amount, balance, created_at, created_by" +
		" FROM transactions WHERE account_number=? AND transaction_time > ? AND transaction_time < ? AND is_deleted=false ORDER BY transaction_time ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, accountNumber, timeFrom, timeTo, offset, length)
	if err != nil {
		lLog.Errorf("error while listing transaction by account number. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*TransactionRecord, 0)
	for rows.Next() {
		ar := &TransactionRecord{}
		err := rows.Scan(&ar.TransactionID, &ar.TransactionTime, &ar.AccountNumber, &ar.JournalID, &ar.Description, &ar.Alignment, &ar.Amount, &ar.Balance, &ar.CreatedAt, &ar.CreatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListAccount function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// CountTransactionByAccountNumber will return a number of accounts in database that belong to a specific
// accountNumber andbeen created within the time range.
// Throws error if the underlying database connection has problem.
// It will returns total number of transaction in the database as specified in the argument.
func (repo *MySqlDBRepository) CountTransactionByAccountNumber(ctx context.Context, accountNumber string, timeFrom, timeTo time.Time) (int, error) {
	lLog := mysqlLog.WithField("function", "CountTransactionByAccountNumber")
	q := "SELECT COUNT(*) as trxCount" +
		" FROM transactions WHERE account_number = ? AND transaction_time > ? AND transaction_time < ? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, accountNumber, timeFrom, timeTo)
	if row.Err() != nil {
		lLog.Errorf("error while counting transaction by account number. got %s", row.Err().Error())
		return 0, row.Err()
	}
	count := 0
	err := row.Scan(&count)
	if err != nil {
		lLog.Errorf("error while counting transactions by account number. got %s", err.Error())
		return 0, err
	}
	return count, nil
}

// ListTransactionByJournalID will list transactions , the transaction must belong to the
// specified journalID arguments.
// Throws error if the underlying database connection has problem.
// It will return TransactionRecord sorted.
// It returns list of TransactionRecord
func (repo *MySqlDBRepository) ListTransactionByJournalID(ctx context.Context, journalID string) ([]*TransactionRecord, error) {
	lLog := mysqlLog.WithField("function", "ListTransactionByJournalID")
	q := "SELECT transaction_id, transaction_time, account_number, journal_id, description, alignment, amount, balance, created_at, created_by" +
		" FROM transactions WHERE journal_id=? AND is_deleted=false"
	rows, err := repo.db.QueryxContext(ctx, q, journalID)
	if err != nil {
		lLog.Errorf("error while listing transaction by journalID. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*TransactionRecord, 0)
	for rows.Next() {
		ar := &TransactionRecord{}
		err := rows.Scan(&ar.TransactionID, &ar.TransactionTime, &ar.AccountNumber, &ar.JournalID, &ar.Description, &ar.Alignment, &ar.Amount, &ar.Balance, &ar.CreatedAt, &ar.CreatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListTransactionByJournalID function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// InsertCurrency will insert the data specified in the rec argument into database
// will return error if the underlying database connection has problem. or if the
// Currency Code already in the database.
// Will return the Currency Code saved if successful.
func (repo *MySqlDBRepository) InsertCurrency(ctx context.Context, rec *CurrenciesRecord) (string, error) {
	lLog := mysqlLog.WithField("function", "InsertCurrency")
	if len(rec.Code) > 10 {
		lLog.Errorf("Currency code %s is too long. Should not more than 10 digit", rec.Code)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.Name) > 30 {
		lLog.Errorf("Currency name %s is too long. Should not more than 30 digit", rec.Name)
		return "", errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}
	if len(rec.UpdatedBy) > 16 {
		rec.CreatedBy = rec.UpdatedBy[:16]
	}
	q := "INSERT INTO currencies(" +
		"code, name, exchange, created_at, created_by, updated_at, updated_by, is_deleted" +
		") VALUES(?, ?, ?, ?, ?, ?, ?, false)"
	args := []interface{}{
		rec.Code, rec.Name, rec.Exchange, rec.CreatedAt, rec.CreatedBy, rec.UpdatedAt, rec.UpdatedBy,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while listing transaction by journalID. got %s", err.Error())
		return "", err
	}
	return rec.Code, nil
}

// UpdateCurrency update an currency entity record in the database.
// Throws error if the underlying database connection has problem.
// The rec argument contains the Currency information to be updated.
// The Currency Code contained within the rec MUST be already persisted before.
func (repo *MySqlDBRepository) UpdateCurrency(ctx context.Context, rec *CurrenciesRecord) error {
	lLog := mysqlLog.WithField("function", "UpdateCurrency")
	if len(rec.Code) > 10 {
		lLog.Errorf("Currency code %s is too long. Should not more than 10 digit", rec.Code)
		return errors.ErrStringDataTooLong
	}
	if len(rec.Name) > 30 {
		lLog.Errorf("Currency name %s is too long. Should not more than 30 digit", rec.Code)
		return errors.ErrStringDataTooLong
	}
	if len(rec.CreatedBy) > 16 {
		rec.CreatedBy = rec.CreatedBy[:16]
	}
	if len(rec.UpdatedBy) > 16 {
		rec.CreatedBy = rec.UpdatedBy[:16]
	}
	q := "UPDATE currencies " +
		"set name=?, exchange=?, created_at=?, created_by=?, updated_at=?, updated_by=?" +
		" WHERE code=? AND is_deleted=false"
	args := []interface{}{
		rec.Name, rec.Exchange, rec.CreatedAt, rec.CreatedBy, rec.UpdatedAt, rec.UpdatedBy, rec.Code,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while listing transaction by journalID. got %s", err.Error())
		return err
	}
	return nil
}

// DeleteCurrency soft/logical delete a currency entity.
// Throws error if the underlying database connection has problem.
// If the Currency Code not exist, it will do nothing and return nil.
func (repo *MySqlDBRepository) DeleteCurrency(ctx context.Context, currencyCode string) error {
	lLog := mysqlLog.WithField("function", "DeleteCurrency")
	q := "UPDATE currencies " +
		"set is_deleted=true" +
		" WHERE code=? && is_deleted=true"
	args := []interface{}{
		currencyCode,
	}
	_, err := repo.db.ExecContext(ctx, q, args...)
	if err != nil {
		lLog.Errorf("error while deleting currency. got %s", err.Error())
		return err
	}
	return nil
}

// ListCurrency will list currencies in paginated fashion.
// Throws error if the underlying database connection has problem.
// It will return CurrenciesRecord sorted, starting from the offset with total maximum number or item, specified
// in the length argument.
// It returns list of CurrenciesRecord
func (repo *MySqlDBRepository) ListCurrency(ctx context.Context, sort string, offset, length int) ([]*CurrenciesRecord, error) {
	lLog := mysqlLog.WithField("function", "ListCurrency")
	q := "SELECT code, name, exchange, created_at, created_by, updated_at, updated_by" +
		" FROM currencies WHERE is_deleted=false ORDER BY " + sort + " ASC LIMIT ?,?"
	rows, err := repo.db.QueryxContext(ctx, q, offset, length)
	if err != nil {
		lLog.Errorf("error while listing currencies. got %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	ret := make([]*CurrenciesRecord, 0)
	for rows.Next() {
		ar := &CurrenciesRecord{}
		err := rows.Scan(&ar.Code, &ar.Name, &ar.Exchange, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
		if err != nil {
			lLog.Errorf("error while scanning rows in ListCurrency function. got %s", err.Error())
		} else {
			ret = append(ret, ar)
		}
	}
	return ret, nil
}

// GetCurrency retrieves an Currency Record from database where the code is specified.
// Throws error if  the underlying database connection has problem.
// It returns an instance of CurrenciesRecord or nil if record not found
func (repo *MySqlDBRepository) GetCurrency(ctx context.Context, code string) (*CurrenciesRecord, error) {
	lLog := mysqlLog.WithField("function", "GetCurrency")
	q := "SELECT  code, name, exchange, created_at, created_by, updated_at, updated_by" +
		" FROM currencies WHERE code=? AND is_deleted=false"
	row := repo.db.QueryRowxContext(ctx, q, code)
	if row.Err() != nil {
		if row.Err() == sql.ErrNoRows {
			return nil, acccore.ErrCurrencyNotFound
		}
		lLog.Errorf("error while retrieving currencies. got %s", row.Err().Error())
		return nil, row.Err()
	}
	ar := &CurrenciesRecord{}
	err := row.Scan(&ar.Code, &ar.Name, &ar.Exchange, &ar.CreatedAt, &ar.CreatedBy, &ar.UpdatedAt, &ar.UpdatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		lLog.Errorf("error while scanning currency record. got %s", err.Error())
		return nil, err
	}
	return ar, nil
}
