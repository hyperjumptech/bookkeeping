package connector

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	//Anonymous import for mysql initialization
	_ "github.com/go-sql-driver/mysql"
)

var (
	log = logrus.WithField("module", "DBConnector")
)

// AccountRecord an entity representative of Account table
type AccountRecord struct {
	// AccountNumber related to account_number column
	AccountNumber string
	// Name related to name column
	Name string
	// CurrencyCode related to currency_code column
	CurrencyCode string
	// Description related to description column
	Description string
	// Alignment related to alignment column
	Alignment string
	// Balance related to ballance column
	Balance int64
	// Coa related to coa column
	Coa string
	// CreatedAt related to created_at column
	CreatedAt time.Time
	// CreatedBy related to created_by column
	CreatedBy string
	// UpdatedAt related to updated_at column
	UpdatedAt time.Time
	// UpdatedBy related to updated_by column
	UpdatedBy string
}

// JournalRecord an entity representative of Journal table
type JournalRecord struct {
	// JournalID related to journal_id column
	JournalID string
	// JournalingTime related to journaling_time column
	JournalingTime time.Time
	// Description related to description column
	Description string
	// IsReversal related to is_reversal column
	IsReversal bool
	// ReversedJournalID related to reversed_jounal_id column
	ReversedJournalID string
	// TotalAmount related to total_amount column
	TotalAmount int64
	// CreatedAt related to created_at column
	CreatedAt time.Time
	// CreatedBy related to created_by column
	CreatedBy string
}

// TransactionRecord an entity representative of Transaction table
type TransactionRecord struct {
	// TransactionID related to transaction_id column
	TransactionID string
	// TransactionTime relate to transaction_time column. is the time when transaction happen, could differ to created at.
	TransactionTime time.Time
	// AccountNumber related to account_number column
	AccountNumber string
	// JournalID related to journal_id column
	JournalID string
	// Description related to desc column
	Description string
	// Alignment related to alignment column
	Alignment string
	// Amount related to amount column
	Amount int64
	// Balance related to balance column
	Balance int64
	// CreatedAt related to created_at column
	CreatedAt time.Time
	// CreatedBy related to created_by column
	CreatedBy string
}

// CurrenciesRecord an entity representative of Currency table
type CurrenciesRecord struct {
	// Code related to code column
	Code string
	// Name related to name column
	Name string
	// Exchange related to exchange column
	Exchange float64
	// CreatedAt related to created_at column
	CreatedAt time.Time
	// CreatedBy related to created_by column
	CreatedBy string
	// UpdatedAt related to updated_at column
	UpdatedAt time.Time
	// UpdatedBy related to updated_by column
	UpdatedBy string
}

// DBRepository is the database structure
type DBRepository interface {
	// Connect connect there repository to the database, it uses the configuration internally for connection arguments and parameters.
	Connect(ctx context.Context) error

	// Disconnect the already established connection. Throws error if the underlying database connection yield an error
	Disconnect() error

	// IsConnected check if the connection is already established
	IsConnected() bool

	// DB the database connection object.
	DB() *sqlx.DB

	// Dump database for backup
	DumpDB(ctx context.Context) (string, error)

	// ClearTables clear all table for testing purpose
	ClearTables(ctx context.Context) error

	// InsertAccount insert an entity record of account into database.
	// Throws error if the underlying connection have problem.
	// The rec argument contains the Account information to be written.
	// It returns the account number that written into database.
	// The AccountNumber contained within the rec MUST NOT be perstited before.
	InsertAccount(ctx context.Context, rec *AccountRecord) (string, error)

	// UpdateAccount update an account entity record in the database.
	// Throws error if the underlying database connection has problem.
	// The rec argument contains the Account information to be updated.
	// The AccountNumber contained within the rec MUST be already persisted before.
	UpdateAccount(ctx context.Context, rec *AccountRecord) error

	// DeleteAccount soft/logical delete an account.
	// Throws error if the underlying database connection has problem.
	// If the account number not exist, it will do nothing and return nil.
	DeleteAccount(ctx context.Context, accountNumber string) error

	// GetAccount retrieves an AccountRecord from database where the account number is specified.
	// Throws error if  the underlying database connection has problem. or, if there is no Account with
	// specified accountNumber.
	// It returns an instance of AccountRecord
	GetAccount(ctx context.Context, accountNumber string) (*AccountRecord, error)

	// ListAccount will list account in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of AcccountRecords
	ListAccount(ctx context.Context, sort string, offset, length int) ([]*AccountRecord, error)

	// CountAccounts will return a number of accounts in database.
	// Throws error if the underlying database connection has problem.
	// It will returns total number of accounts in the database.
	CountAccounts(ctx context.Context) (int, error)

	// ListAccountByCoa will list all account that have the specified COA, the list presented in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of AcccountRecords
	ListAccountByCoa(ctx context.Context, coa string, sort string, offset, length int) ([]*AccountRecord, error)

	// CountAccountByCoa will return a number of accounts in database that belong to the specified COA number.
	// Throws error if the underlying database connection has problem.
	// It will returns total number of accounts in the database.
	CountAccountByCoa(ctx context.Context, coa string) (int, error)

	// FindAccountByName will list all account that have the specified name, the list presented in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return AccountRecords sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of AcccountRecords
	FindAccountByName(ctx context.Context, nameLike string, sort string, offset, length int) ([]*AccountRecord, error)

	// CountAccountByName will return a number of accounts in database that have the name like the specified in the argument..
	// Throws error if the underlying database connection has problem.
	// It will returns total number of accounts in the database.
	CountAccountByName(ctx context.Context, nameLike string) (int, error)

	// InsertJournal will insert the data specified in the rec argument into database
	// will return error if the underlying database connection has problem. or if the
	// journalID, or Transaction ID in the journal already in the database.
	// Will return the JournalID saved if successful.
	InsertJournal(ctx context.Context, rec *JournalRecord) (string, error)

	// UpdateJournal update an journal entity record in the database.
	// Throws error if the underlying database connection has problem.
	// The rec argument contains the Journal information to be updated.
	// The JournalID contained within the rec MUST be already persisted before.
	UpdateJournal(ctx context.Context, rec *JournalRecord) error

	// DeleteJournal soft/logical delete an journal.
	// Throws error if the underlying database connection has problem.
	// If the JournalID not exist, it will do nothing and return nil.
	DeleteJournal(ctx context.Context, journalID string) error

	// ListJournal will list journals in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return JournalRecord sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of JournalRecord
	ListJournal(ctx context.Context, sort string, offset, length int) ([]*JournalRecord, error)

	// GetJournal retrieves an JournalRecord from database where the journalID is specified.
	// Throws error if  the underlying database connection has problem. or, if there is no Journal with
	// specified journalID.
	// It returns an instance of JournalRecord
	GetJournal(ctx context.Context, journalID string) (*JournalRecord, error)

	// GetJournalByReversalID retrieves an JournalRecord from database where the reversedJournalID is specified.
	// Throws error if  the underlying database connection has problem. or, if there is no Journal with
	// specified reversedJournalID.
	// It returns an instance of JournalRecord
	GetJournalByReversalID(ctx context.Context, journalID string) (*JournalRecord, error)

	// ListJournalByTimeRange will list journals in paginated fashion where journal is in the specified time range.
	// Throws error if the underlying database connection has problem.
	// It will return JournalRecord sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of JournalRecord
	ListJournalByTimeRange(ctx context.Context, timeFrom, timeTo time.Time, sort string, offset, length int) ([]*JournalRecord, error)

	// CountJournalByTimeRange will return a number of journals in database that been created within the time range.
	// Throws error if the underlying database connection has problem.
	// It will returns total number of journals in the database.
	CountJournalByTimeRange(ctx context.Context, timeFrom, timeTo time.Time) (int, error)

	// InsertTransaction will insert the data specified in the rec argument into database
	// will return error if the underlying database connection has problem. or if the
	// Transaction ID in the journal already in the database.
	// Will return the TransactionID saved if successful.
	InsertTransaction(ctx context.Context, rec *TransactionRecord) (string, error)

	// UpdateTransaction update an transaction entity record in the database.
	// Throws error if the underlying database connection has problem.
	// The rec argument contains the Transaction information to be updated.
	// The TransactionID contained within the rec MUST be already persisted before.
	UpdateTransaction(ctx context.Context, rec *TransactionRecord) error

	// DeleteTransaction soft/logical delete a transaction.
	// Throws error if the underlying database connection has problem.
	// If the TransactionID not exist, it will do nothing and return nil.
	DeleteTransaction(ctx context.Context, transactionID string) error

	// ListTransaction will list journals in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return TransactionRecord sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of TransactionRecord
	ListTransaction(ctx context.Context, sort string, offset, length int) ([]*TransactionRecord, error)

	// GetTransaction retrieves an TransactionRecord from database where the transactionID is specified.
	// Throws error if  the underlying database connection has problem. or, if there is no Transaction with
	// specified transactionID.
	// It returns an instance of TransactionRecord
	GetTransaction(ctx context.Context, transactionID string) (*TransactionRecord, error)

	// ListTransactionByAccountNumber will list transactions in paginated fashion, the transaction must belong to the
	// specified accountNumber arguments and created within the time rage.
	// Throws error if the underlying database connection has problem.
	// It will return TransactionRecord sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of TransactionRecord
	ListTransactionByAccountNumber(ctx context.Context, accountNumber string, timeFrom, timeTo time.Time, offset, length int) ([]*TransactionRecord, error)

	// CountTransactionByAccountNumber will return a number of accounts in database that belong to a speciffic
	// accountNumber andbeen created within the time range.
	// Throws error if the underlying database connection has problem.
	// It will returns total number of transaction in the database as specified in the argument.
	CountTransactionByAccountNumber(ctx context.Context, accountNumber string, timeFrom, timeTo time.Time) (int, error)

	// ListTransactionByJournalID will list transactions , the transaction must belong to the
	// specified journalID arguments.
	// Throws error if the underlying database connection has problem.
	// It will return TransactionRecord sorted.
	// It returns list of TransactionRecord
	ListTransactionByJournalID(ctx context.Context, journalID string) ([]*TransactionRecord, error)

	// InsertCurrency will insert the data specified in the rec argument into database
	// will return error if the underlying database connection has problem. or if the
	// Currency Code already in the database.
	// Will return the Currency Code saved if successful.
	InsertCurrency(ctx context.Context, rec *CurrenciesRecord) (string, error)

	// UpdateCurrency update an currency entity record in the database.
	// Throws error if the underlying database connection has problem.
	// The rec argument contains the Currency information to be updated.
	// The Currency Code contained within the rec MUST be already persisted before.
	UpdateCurrency(ctx context.Context, rec *CurrenciesRecord) error

	// DeleteCurrency soft/logical delete a currency entity.
	// Throws error if the underlying database connection has problem.
	// If the Currency Code not exist, it will do nothing and return nil.
	DeleteCurrency(ctx context.Context, currencyCode string) error

	// ListCurrency will list currencies in paginated fashion.
	// Throws error if the underlying database connection has problem.
	// It will return CurrenciesRecord sorted, starting from the offset with total maximum number or item, specified
	// in the length argument.
	// It returns list of CurrenciesRecord
	ListCurrency(ctx context.Context, sort string, offset, length int) ([]*CurrenciesRecord, error)

	// GetCurrency retrieves an Currency Record from database where the code is specified.
	// Throws error if  the underlying database connection has problem. or, if there is no Currency with
	// specified code.
	// It returns an instance of CurrenciesRecord
	GetCurrency(ctx context.Context, code string) (*CurrenciesRecord, error)
}
