package accounting

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hyperjumptech/acccore"
	"github.com/hyperjumptech/hyperwallet/internal/connector"
	"github.com/hyperjumptech/hyperwallet/internal/contextkeys"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
)

var (
	dbLog = logrus.WithField("file", "MySQLManagerImpl.go")
)

// JOURNAL MANAGER ------------------------------------------------------------------

// NewMySQLJournalManager returns the sql journal manager
func NewMySQLJournalManager(repo connector.DBRepository) acccore.JournalManager {
	return &MySQLJournalManager{repo: repo}
}

// MySQLJournalManager implementation of JournalManager using Journal table in MySQL
type MySQLJournalManager struct {
	repo connector.DBRepository
}

// NewJournal will create new blank un-persisted journal
func (jm *MySQLJournalManager) NewJournal(ctx context.Context) acccore.Journal {
	return &acccore.BaseJournal{}
}

// PersistJournal will record a journal entry into database.
// It requires list of transactions for which each of the transaction MUST BE :
//    1.NOT BE PERSISTED. (the journal accountNumber is not exist in DB yet)
//    2.Pointing or owned by a PERSISTED Account
//    3.Each of this account must belong to the same Currency
//    4.Balanced. The total sum of DEBIT and total sum of CREDIT is equal.
//    5.No duplicate transaction that belongs to the same Account.
// If your database support 2 phased commit, you can make all balance changes in
// accounts and transactions. If your db do not support this, you can implement your own 2 phase commits mechanism
// on the CommitJournal and CancelJournal
func (jm *MySQLJournalManager) PersistJournal(ctx context.Context, journalToPersist acccore.Journal) error {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "PersistJournal")

	// First we have to make sure that the journalToPersist is not yet in our database.
	// 1. Checking if anything mandatory is not missing
	if journalToPersist == nil {
		return acccore.ErrJournalNil
	}
	if len(journalToPersist.GetJournalID()) == 0 {
		lLog.Errorf("error persisting journal. journal is missing the journalID")
		return acccore.ErrJournalMissingID
	}
	if len(journalToPersist.GetTransactions()) == 0 {
		lLog.Errorf("error persisting journal %s. journal contains no transactions.", journalToPersist.GetJournalID())
		return acccore.ErrJournalNoTransaction
	}
	if len(journalToPersist.GetCreateBy()) == 0 {
		lLog.Errorf("error persisting journal %s. journal author not known.", journalToPersist.GetJournalID())
		return acccore.ErrJournalMissingAuthor
	}

	// 2. Checking if the journal ID must not in the Database (already persisted)
	//    SQL HINT : SELECT COUNT(*) FROM JOURNAL WHERE JOURNAL.ID = {journalToPersist.GetJournalID()}
	//    If COUNT(*) is > 0 return error
	j, err := jm.repo.GetJournal(ctx, journalToPersist.GetJournalID())
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, acccore.ErrJournalIDNotFound) {
			lLog.Errorf("error while fetching journal %s. got %s", journalToPersist.GetJournalID(), err.Error())
			return err
		}
	}
	if j != nil {
		lLog.Errorf("error persisting journal %s. journal already exist.", journalToPersist.GetJournalID())
		return acccore.ErrJournalAlreadyPersisted
	}

	// 3. Make sure all journal transactions are IDed.
	for idx, trx := range journalToPersist.GetTransactions() {
		if len(trx.GetTransactionID()) == 0 {
			lLog.Errorf("error persisting journal %s. transaction %d is missing transactionID.", journalToPersist.GetJournalID(), idx)
			return acccore.ErrJournalTransactionMissingID
		}
	}

	// 4. Make sure all journal transactions are not persisted.
	for idx, trx := range journalToPersist.GetTransactions() {
		t, err := jm.repo.GetTransaction(ctx, trx.GetTransactionID())
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, acccore.ErrJournalIDNotFound) {
				lLog.Errorf("error while fetching transaction %s. got %s", trx.GetTransactionID(), err.Error())
				return err
			}
		}
		if t != nil {
			lLog.Errorf("error persisting journal %s. transaction %d is already exist.", journalToPersist.GetJournalID(), idx)
			return acccore.ErrJournalAlreadyPersisted
		}
	}

	// 5. Make sure transactions are balanced.
	var creditSum, debitSum int64
	for _, trx := range journalToPersist.GetTransactions() {
		if trx.GetAlignment() == acccore.DEBIT {
			debitSum += trx.GetAmount()
		}
		if trx.GetAlignment() == acccore.CREDIT {
			creditSum += trx.GetAmount()
		}
	}
	if creditSum != debitSum {
		lLog.Errorf("error persisting journal %s. debit (%d) != credit (%d). journal not balance", journalToPersist.GetJournalID(), debitSum, creditSum)
		return acccore.ErrJournalNotBalance
	}

	// 6. Make sure transactions account are not appear twice in the journal
	accountDupCheck := make(map[string]bool)
	for _, trx := range journalToPersist.GetTransactions() {
		if _, exist := accountDupCheck[trx.GetAccountNumber()]; exist {
			lLog.Errorf("error persisting journal %s. multiple transaction belong to the same account (%s)", journalToPersist.GetJournalID(), trx.GetAccountNumber())
			return acccore.ErrJournalTransactionAccountDuplicate
		}
		accountDupCheck[trx.GetAccountNumber()] = true
	}

	// 7. Make sure transactions are all belong to existing accounts
	for _, trx := range journalToPersist.GetTransactions() {
		account, err := jm.repo.GetAccount(ctx, trx.GetAccountNumber())
		if err != nil || account == nil {
			lLog.Errorf("error persisting journal %s. theres a transaction belong to non existent account (%s)", journalToPersist.GetJournalID(), trx.GetAccountNumber())
			return acccore.ErrJournalTransactionAccountNotPersist
		}
	}

	// 8. Make sure transactions are all have the same currency
	var currency string
	for idx, trx := range journalToPersist.GetTransactions() {
		account, err := jm.repo.GetAccount(ctx, trx.GetAccountNumber())
		if err != nil || account == nil {
			return acccore.ErrAccountIDNotFound
		}
		cur := account.CurrencyCode
		if idx == 0 {
			currency = cur
		} else {
			if cur != currency {
				lLog.Errorf("error persisting journal %s. transactions here uses account with different currencies", journalToPersist.GetJournalID())
				return acccore.ErrJournalTransactionMixCurrency
			}
		}
	}

	// 9. If this is a reversal journal, make sure the journal being reversed have not been reversed before.
	if journalToPersist.GetReversedJournal() != nil {
		reversed, err := jm.IsJournalIDReversed(ctx, journalToPersist.GetJournalID())
		if err != nil {
			return err
		}
		if reversed {
			lLog.Errorf("error persisting journal %s. this journal try to make reverse transaction on journals thats already reversed %s", journalToPersist.GetJournalID(), journalToPersist.GetJournalID())
			return acccore.ErrJournalCanNotDoubleReverse
		}
	}

	// ALL is OK. So lets start persisting.

	// BEGIN transaction
	tx, err := jm.repo.DB().BeginTxx(ctx, &sql.TxOptions{
		// todo investigate the use of this.
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		lLog.Errorf("error creating transaction. got %s", err.Error())
		return err
	}

	// 1. Save the Journal
	journalToInsert := &connector.JournalRecord{
		JournalID:         journalToPersist.GetJournalID(),
		JournalingTime:    time.Now(),
		Description:       journalToPersist.GetDescription(),
		IsReversal:        false,
		ReversedJournalID: "",
		TotalAmount:       creditSum,
		CreatedAt:         time.Now(),
		CreatedBy:         journalToPersist.GetCreateBy(),
	}

	if journalToPersist.GetReversedJournal() != nil {
		journalToInsert.ReversedJournalID = journalToPersist.GetReversedJournal().GetJournalID()
		journalToInsert.IsReversal = true
	}

	journalID, err := jm.repo.InsertJournal(ctx, journalToInsert)
	if err != nil {
		lLog.Errorf("error inserting new journal %s . got %s. rolling back transaction.", journalToInsert.JournalID, err.Error())
		err = tx.Rollback()
		if err != nil {
			lLog.Errorf("error rolling back transaction. got %s", err.Error())
		}
		return err
	}

	// 2 Save the Transactions
	for _, trx := range journalToPersist.GetTransactions() {
		transactionToInsert := &connector.TransactionRecord{
			TransactionID:   trx.GetTransactionID(),
			TransactionTime: trx.GetTransactionTime(),
			AccountNumber:   trx.GetAccountNumber(),
			JournalID:       journalID,
			Description:     trx.GetDescription(),
			//Alignment:     string(trx.GetTransactionType()),
			Amount:    trx.GetAmount(),
			Balance:   trx.GetAccountBalance(),
			CreatedAt: time.Now(),
			CreatedBy: trx.GetCreateBy(),
		}

		if trx.GetAlignment() == acccore.DEBIT {
			transactionToInsert.Alignment = "DEBIT"
		} else {
			transactionToInsert.Alignment = "CREDIT"
		}

		account, err := jm.repo.GetAccount(ctx, trx.GetAccountNumber())
		if err != nil {
			lLog.Errorf("error retrieving account %s in transaction. got %s. rolling back transaction.", trx.GetAccountNumber(), err.Error())
			err = tx.Rollback()
			if err != nil {
				lLog.Errorf("error rolling back transaction. got %s", err.Error())
			}
			return err
		}
		balance, accountTrxType := account.Balance, account.Alignment

		newBalance := int64(0)
		if transactionToInsert.Alignment == accountTrxType {
			newBalance = balance + transactionToInsert.Amount
		} else {
			newBalance = balance - transactionToInsert.Amount
		}
		transactionToInsert.Balance = newBalance

		_, err = jm.repo.InsertTransaction(ctx, transactionToInsert)
		if err != nil {
			lLog.Errorf("error inserting new transaction %s in transaction. got %s. rolling back transaction.", transactionToInsert.TransactionID, err.Error())
			err = tx.Rollback()
			if err != nil {
				lLog.Errorf("error rolling back transaction. got %s", err.Error())
			}
			return err
		}

		// Update Account Balance.
		// UPDATE ACCOUNT SET BALANCE = {newBalance},  UPDATEDBY = {trx.GetCreateBy()}, UPDATE_TIME = {time.Now()} WHERE ACCOUNT_ID = {trx.GetAccountNumber()}
		account.Balance = newBalance
		account.UpdatedAt = time.Now()
		account.UpdatedBy = trx.GetCreateBy()
		err = jm.repo.UpdateAccount(ctx, account)
		if err != nil {
			lLog.Errorf("error updating account %s in transaction. got %s. rolling back transaction.", account.AccountNumber, err.Error())
			err = tx.Rollback()
			if err != nil {
				lLog.Errorf("error rolling back transaction. got %s", err.Error())
			}
			return err
		}
	}

	// COMMIT transaction
	err = tx.Commit()
	if err != nil {
		lLog.Errorf("error committing transaction. got %s", err.Error())
		return err
	}

	return nil
}

// CommitJournal will commit the journal into the system
// Only non committed journal can be committed.
// use this if the implementation database do not support 2 phased commit.
// if your database support 2 phased commit, you should do all commit in the PersistJournal function
// and this function should simply return nil.
func (jm *MySQLJournalManager) CommitJournal(ctx context.Context, journalToCommit acccore.Journal) error {
	return nil
}

// CancelJournal Cancel a journal
// Only non committed journal can be committed.
// use this if the implementation database do not support 2 phased commit.
// if your database do not support 2 phased commit, you should do all roll back in the PersistJournal function
// and this function should simply return nil.
func (jm *MySQLJournalManager) CancelJournal(ctx context.Context, journalToCancel acccore.Journal) error {
	return nil
}

// IsJournalIDReversed check if the journal with specified ID has been reversed
func (jm *MySQLJournalManager) IsJournalIDReversed(ctx context.Context, journalID string) (bool, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "IsJournalIdReversed")
	// SELECT COUNT(*) FROM JOURNAL WHERE REVERSED_JOURNAL_ID = {journalID}
	// return false if COUNT = 0
	// return true if COUNT > 0
	journal, err := jm.repo.GetJournalByReversalID(ctx, journalID)
	if err != nil {
		lLog.Errorf("error while calling GetJournalByReversalID. got %s", err.Error())
		return false, nil
	}
	if journal == nil { // journal == nil AND err == nil, then journal is not reversed
		return false, nil
	}
	return true, nil
}

// IsJournalIDExist will check if a journal ID/number exist in the database.
func (jm *MySQLJournalManager) IsJournalIDExist(ctx context.Context, journalID string) (bool, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "IsJournalIdExist")

	journal, err := jm.repo.GetJournal(ctx, journalID)
	if err != nil || journal == nil {
		lLog.Errorf("error while calling GetJournal. got %s", err.Error())
		return false, nil
	}
	return true, nil
}

// GetJournalByID retrieved a Journal information identified by its ID.
// the provided ID must be exactly the same, not uses the LIKE select expression.
func (jm *MySQLJournalManager) GetJournalByID(ctx context.Context, journalID string) (acccore.Journal, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "GetJournalById")

	journal, err := jm.repo.GetJournal(ctx, journalID)
	if err != nil {
		lLog.Errorf("error while calling GetJournal. got %s", err.Error())
		return nil, err
	}
	if journal == nil {
		lLog.Errorf("error while calling GetJournal, journal is NIL but not throwing any error.")
		return nil, fmt.Errorf("error while calling GetJournal, journal is NIL but not throwing any error")
	}
	ret := &acccore.BaseJournal{}
	ret.SetAmount(journal.TotalAmount).SetDescription(journal.Description).SetReversal(journal.IsReversal).
		SetJournalingTime(journal.JournalingTime).SetCreateBy(journal.CreatedBy).SetCreateTime(journal.CreatedAt).
		SetJournalID(journal.JournalID)

	if journal.IsReversal {
		reversed, err := jm.GetJournalByID(ctx, journal.ReversedJournalID)
		if err != nil {
			lLog.Errorf("error while calling GetJournal. got %s", err.Error())
			return nil, acccore.ErrJournalLoadReversalInconsistent
		}
		ret.SetReversedJournal(reversed)
	}

	// Populate all transactions from DB.
	transactions := make([]acccore.Transaction, 0)
	trxs, err := jm.repo.ListTransactionByJournalID(ctx, journalID)
	if err != nil {
		lLog.Errorf("error while calling jm.repo.ListTransactionByJournalID. got %s", err.Error())
		return nil, err
	}
	for _, trx := range trxs {
		transaction := &acccore.BaseTransaction{}
		transaction.SetJournalID(trx.JournalID).SetTransactionTime(trx.TransactionTime).
			SetAccountNumber(trx.AccountNumber).SetTransactionID(trx.TransactionID).SetDescription(trx.Description).
			SetCreateTime(trx.CreatedAt).SetCreateBy(trx.CreatedBy).SetAccountBalance(trx.Balance).SetAmount(trx.Amount)
		if strings.ToUpper(trx.Alignment) == "DEBIT" {
			transaction.SetAlignment(acccore.DEBIT)
		} else {
			transaction.SetAlignment(acccore.CREDIT)
		}
		transactions = append(transactions, transaction)
	}
	ret.SetTransactions(transactions)

	return ret, nil
}

// ListJournals retrieve list of journals with transaction date between the `from` and `until` time range inclusive.
// This function uses pagination.
func (jm *MySQLJournalManager) ListJournals(ctx context.Context, from time.Time, until time.Time, request acccore.PageRequest) (acccore.PageResult, []acccore.Journal, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "ListJournals")

	count, err := jm.repo.CountJournalByTimeRange(ctx, from, until)
	if err != nil {
		lLog.Errorf("error while calling jm.repo.CountJournalByTimeRange. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	pResult := acccore.PageResultFor(request, count)
	jRecords, err := jm.repo.ListJournal(ctx, "journaling_time", pResult.Offset, pResult.PageSize)
	if err != nil {
		lLog.Errorf("error while calling jm.repo.ListJournal. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	ret := make([]acccore.Journal, 0)
	for _, jrnl := range jRecords {
		journal, err := jm.GetJournalByID(ctx, jrnl.JournalID)
		if err != nil {
			lLog.Errorf("Error while retrieving journal %s. got %s. skipping", jrnl.JournalID, err.Error())
		} else {
			ret = append(ret, journal)
		}
	}
	return pResult, ret, nil
}

// RenderJournal Render this journal into string for easy inspection
func (jm *MySQLJournalManager) RenderJournal(ctx context.Context, journal acccore.Journal) string {

	var buff bytes.Buffer
	table := tablewriter.NewWriter(&buff)
	table.SetHeader([]string{"TRX ID", "Account", "Description", "DEBIT", "CREDIT"})
	table.SetFooter([]string{"", "", "", fmt.Sprintf("%d", acccore.GetTotalDebit(journal)), fmt.Sprintf("%d", acccore.GetTotalCredit(journal))})

	for _, t := range journal.GetTransactions() {
		if t.GetAlignment() == acccore.DEBIT {
			table.Append([]string{t.GetTransactionID(), t.GetAccountNumber(), t.GetDescription(), fmt.Sprintf("%d", t.GetAmount()), ""})
		}
	}
	for _, t := range journal.GetTransactions() {
		if t.GetAlignment() == acccore.CREDIT {
			table.Append([]string{t.GetTransactionID(), t.GetAccountNumber(), t.GetDescription(), "", fmt.Sprintf("%d", t.GetAmount())})
		}
	}
	buff.WriteString(fmt.Sprintf("Journal Entry : %s\n", journal.GetJournalID()))
	buff.WriteString(fmt.Sprintf("Journal Date  : %s\n", journal.GetJournalingTime().String()))
	buff.WriteString(fmt.Sprintf("Description   : %s\n", journal.GetDescription()))
	table.Render()
	return buff.String()
}

// TRANSACTION MANAGER ------------------------------------------------------------------

// NewMySQLTransactionManager returns new SQL Transaction Manager
func NewMySQLTransactionManager(repo connector.DBRepository) acccore.TransactionManager {
	return &MySQLTransactionManager{repo: repo}
}

// MySQLTransactionManager implementation of TransactionManager using Transaction table in MySQL
type MySQLTransactionManager struct {
	repo connector.DBRepository
}

// NewTransaction will create new blank un-persisted Transaction
func (am *MySQLTransactionManager) NewTransaction(ctx context.Context) acccore.Transaction {
	return &acccore.BaseTransaction{}
}

// IsTransactionIDExist will check if an Transaction ID/number is exist in the database.
func (am *MySQLTransactionManager) IsTransactionIDExist(ctx context.Context, id string) (bool, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "IsTransactionIdExist")

	tx, err := am.repo.GetTransaction(ctx, id)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetTransaction. got %s", err.Error())
		return false, err
	}
	if tx == nil {
		return false, nil
	}
	return true, nil
}

// GetTransactionByID will retrieve one single transaction that identified by some ID
func (am *MySQLTransactionManager) GetTransactionByID(ctx context.Context, id string) (acccore.Transaction, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "GetTransactionById")

	tx, err := am.repo.GetTransaction(ctx, id)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetTransaction. got %s", err.Error())
		return nil, err
	}
	if tx == nil {
		lLog.Errorf("error transaction not found")
		return nil, acccore.ErrTransactionNotFound
	}
	trx := &acccore.BaseTransaction{}
	trx.SetAmount(tx.Amount).SetAccountBalance(tx.Balance).SetCreateBy(tx.CreatedBy).SetCreateTime(tx.CreatedAt).
		SetDescription(tx.Description).SetTransactionID(tx.TransactionID).SetAccountNumber(tx.AccountNumber).
		SetTransactionTime(tx.TransactionTime).SetJournalID(tx.JournalID)

	if strings.ToUpper(tx.Alignment) == "DEBIT" {
		trx.SetAlignment(acccore.DEBIT)
	} else {
		trx.SetAlignment(acccore.CREDIT)
	}
	return trx, nil
}

// ListTransactionsOnAccount retrieves list of transactions that belongs to this account
// that transaction happens between the `from` and `until` time range.
// This function uses pagination
func (am *MySQLTransactionManager) ListTransactionsOnAccount(ctx context.Context, from time.Time, until time.Time, account acccore.Account, request acccore.PageRequest) (acccore.PageResult, []acccore.Transaction, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "ListTransactionsOnAccount")

	count, err := am.repo.CountTransactionByAccountNumber(ctx, account.GetAccountNumber(), from, until)
	if err != nil {
		lLog.Errorf("error while calling am.repo.CountTransactionByAccountNumber. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	pageResult := acccore.PageResultFor(request, count)
	records, err := am.repo.ListTransactionByAccountNumber(ctx, account.GetAccountNumber(), from, until, pageResult.Offset, pageResult.PageSize)
	if err != nil {
		lLog.Errorf("error while calling am.repo.ListTransactionByAccountNumber. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	ret := make([]acccore.Transaction, 0)
	for _, tx := range records {
		trx := &acccore.BaseTransaction{}
		trx.SetAmount(tx.Amount).SetAccountBalance(tx.Balance).SetCreateBy(tx.CreatedBy).SetCreateTime(tx.CreatedAt).
			SetDescription(tx.Description).SetTransactionID(tx.TransactionID).SetAccountNumber(tx.AccountNumber).
			SetTransactionTime(tx.TransactionTime).SetJournalID(tx.JournalID)

		if strings.ToUpper(tx.Alignment) == "DEBIT" {
			trx.SetAlignment(acccore.DEBIT)
		} else {
			trx.SetAlignment(acccore.CREDIT)
		}
		ret = append(ret, trx)
	}
	return pageResult, ret, nil
}

// RenderTransactionsOnAccount Render list of transaction been down on an account in a time span
func (am *MySQLTransactionManager) RenderTransactionsOnAccount(ctx context.Context, from time.Time, until time.Time, account acccore.Account, request acccore.PageRequest) (string, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "RenderTransactionsOnAccount")

	result, transactions, err := am.ListTransactionsOnAccount(ctx, from, until, account, request)
	if err != nil {
		lLog.Errorf("error while calling am.repo.ListTransactionsOnAccount. got %s", err.Error())
		return "Error rendering", err
	}

	var buff bytes.Buffer
	table := tablewriter.NewWriter(&buff)
	table.SetHeader([]string{"TRX ID", "TIME", "JOURNAL ID", "Description", "DEBIT", "CREDIT", "BALANCE"})

	for _, t := range transactions {
		if t.GetAlignment() == acccore.DEBIT {
			table.Append([]string{t.GetTransactionID(), t.GetTransactionTime().String(), t.GetJournalID(), t.GetDescription(), fmt.Sprintf("%d", t.GetAmount()), "", fmt.Sprintf("%d", t.GetAccountBalance())})
		}
		if t.GetAlignment() == acccore.CREDIT {
			table.Append([]string{t.GetTransactionID(), t.GetTransactionTime().String(), t.GetJournalID(), t.GetDescription(), "", fmt.Sprintf("%d", t.GetAmount()), fmt.Sprintf("%d", t.GetAccountBalance())})
		}
	}

	buff.WriteString(fmt.Sprintf("Account Number    : %s\n", account.GetAccountNumber()))
	buff.WriteString(fmt.Sprintf("Account Name      : %s\n", account.GetName()))
	buff.WriteString(fmt.Sprintf("Description       : %s\n", account.GetDescription()))
	buff.WriteString(fmt.Sprintf("Currency          : %s\n", account.GetCurrency()))
	buff.WriteString(fmt.Sprintf("COA               : %s\n", account.GetCOA()))
	buff.WriteString(fmt.Sprintf("Current Balance   : %d\n", account.GetBalance()))
	buff.WriteString(fmt.Sprintf("Transactions From : %s\n", from.String()))
	buff.WriteString(fmt.Sprintf("             To   : %s\n", until.String()))
	buff.WriteString(fmt.Sprintf("#Transactions     : %d\n", result.TotalEntries))
	buff.WriteString(fmt.Sprintf("Showing page      : %d/%d\n", result.Page, result.TotalPages))
	table.Render()
	return buff.String(), err
}

// ACCOUNT MANAGER ------------------------------------------------------------------

// NewMySQLAccountManager returns new sql account manager
func NewMySQLAccountManager(repo connector.DBRepository) acccore.AccountManager {
	return &MySQLAccountManager{repo: repo}
}

// MySQLAccountManager implementation of AccountManager using Account table in MySQL
type MySQLAccountManager struct {
	repo connector.DBRepository
}

// NewAccount will create a new blank un-persisted account.
func (am *MySQLAccountManager) NewAccount(ctx context.Context) acccore.Account {
	return &acccore.BaseAccount{}
}

// PersistAccount will save the account into database.
// will throw error if the account already persisted
func (am *MySQLAccountManager) PersistAccount(ctx context.Context, AccountToPersist acccore.Account) error {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "PersistAccount")

	if len(AccountToPersist.GetAccountNumber()) == 0 {
		return acccore.ErrAccountMissingID
	}
	if len(AccountToPersist.GetName()) == 0 {
		return acccore.ErrAccountMissingName
	}
	if len(AccountToPersist.GetDescription()) == 0 {
		return acccore.ErrAccountMissingDescription
	}
	if len(AccountToPersist.GetCreateBy()) == 0 {
		return acccore.ErrAccountMissingCreator
	}

	curRec, err := am.repo.GetCurrency(ctx, AccountToPersist.GetCurrency())
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetCurrency. got %s", err.Error())
		return err
	}
	if curRec == nil {
		logrus.Errorf("can not persist. currency do not exist %s", AccountToPersist.GetCurrency())
		return acccore.ErrCurrencyNotFound
	}

	ar := &connector.AccountRecord{
		AccountNumber: AccountToPersist.GetAccountNumber(),
		Name:          AccountToPersist.GetName(),
		CurrencyCode:  AccountToPersist.GetCurrency(),
		Description:   AccountToPersist.GetDescription(),
		// Alignment:     AccountToPersist.GetBaseTransactionType(),
		Balance:   AccountToPersist.GetBalance(),
		Coa:       AccountToPersist.GetCOA(),
		CreatedAt: time.Now(),
		CreatedBy: AccountToPersist.GetCreateBy(),
		UpdatedAt: time.Now(),
		UpdatedBy: AccountToPersist.GetUpdateBy(),
	}
	if AccountToPersist.GetAlignment() == acccore.DEBIT {
		ar.Alignment = "DEBIT"
	} else {
		ar.Alignment = "CREDIT"
	}

	_, err = am.repo.InsertAccount(ctx, ar)
	return err
}

// UpdateAccount will update the account database to reflect to the provided account information.
// This update account function will fail if the account ID/number is not existing in the database.
func (am *MySQLAccountManager) UpdateAccount(ctx context.Context, AccountToUpdate acccore.Account) error {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "UpdateAccount")

	if len(AccountToUpdate.GetAccountNumber()) == 0 {
		return acccore.ErrAccountMissingID
	}
	if len(AccountToUpdate.GetName()) == 0 {
		return acccore.ErrAccountMissingName
	}
	if len(AccountToUpdate.GetDescription()) == 0 {
		return acccore.ErrAccountMissingDescription
	}
	if len(AccountToUpdate.GetCreateBy()) == 0 {
		return acccore.ErrAccountMissingCreator
	}

	// First make sure that The account have never been created in DB.
	exist, err := am.IsAccountIDExist(ctx, AccountToUpdate.GetAccountNumber())
	if err != nil {
		lLog.Errorf("error while calling am.IsAccountIdExist. got %s", err.Error())
		return err
	}
	if !exist {
		lLog.Errorf("error account is not persisted")
		return acccore.ErrAccountIsNotPersisted
	}

	ar := &connector.AccountRecord{
		AccountNumber: AccountToUpdate.GetAccountNumber(),
		Name:          AccountToUpdate.GetName(),
		CurrencyCode:  AccountToUpdate.GetCurrency(),
		Description:   AccountToUpdate.GetDescription(),
		// Alignment:     AccountToPersist.GetBaseTransactionType(),
		Balance:   AccountToUpdate.GetBalance(),
		Coa:       AccountToUpdate.GetCOA(),
		CreatedAt: time.Now(),
		CreatedBy: AccountToUpdate.GetCreateBy(),
		UpdatedAt: time.Now(),
		UpdatedBy: AccountToUpdate.GetUpdateBy(),
	}
	if AccountToUpdate.GetAlignment() == acccore.DEBIT {
		ar.Alignment = "DEBIT"
	} else {
		ar.Alignment = "CREDIT"
	}

	return am.repo.UpdateAccount(ctx, ar)

}

// IsAccountIDExist will check if an account ID/number is exist in the database.
func (am *MySQLAccountManager) IsAccountIDExist(ctx context.Context, id string) (bool, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "IsAccountIdExist")

	ar, err := am.repo.GetAccount(ctx, id)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetAccount. got %s", err.Error())
		return false, err
	}
	if ar == nil {
		//lLog.Errorf("error account not found")
		return false, nil
	}
	return true, nil
}

// GetAccountByID retrieve an account information by specifying the ID/number
func (am *MySQLAccountManager) GetAccountByID(ctx context.Context, id string) (acccore.Account, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "GetAccountById")

	rec, err := am.repo.GetAccount(ctx, id)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetAccount. got %s", err.Error())
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	ret := &acccore.BaseAccount{}
	ret.SetAccountNumber(rec.AccountNumber).SetDescription(rec.Description).SetCreateTime(rec.CreatedAt).
		SetCreateBy(rec.CreatedBy).SetCurrency(rec.CurrencyCode).SetCOA(rec.Coa).SetName(rec.Name).
		SetBalance(rec.Balance).SetUpdateBy(rec.UpdatedBy).SetUpdateTime(rec.UpdatedAt)

	if strings.ToUpper(rec.Alignment) == "DEBIT" {
		ret.SetAlignment(acccore.DEBIT)
	} else {
		ret.SetAlignment(acccore.CREDIT)
	}

	return ret, nil
}

// ListAccounts list all account in the database.
// This function uses pagination
func (am *MySQLAccountManager) ListAccounts(ctx context.Context, request acccore.PageRequest) (acccore.PageResult, []acccore.Account, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "ListAccounts")

	count, err := am.repo.CountAccounts(ctx)
	if err != nil {
		lLog.Errorf("error while calling am.repo.CountAccounts. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	pResult := acccore.PageResultFor(request, count)
	records, err := am.repo.ListAccount(ctx, "name", pResult.Offset, pResult.PageSize)
	if err != nil {
		lLog.Errorf("error while calling am.repo.ListAccount. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}

	ret := make([]acccore.Account, 0)
	for _, rec := range records {
		bacc := &acccore.BaseAccount{}
		bacc.SetAccountNumber(rec.AccountNumber).SetDescription(rec.Description).SetCreateTime(rec.CreatedAt).
			SetCreateBy(rec.CreatedBy).SetCurrency(rec.CurrencyCode).SetCOA(rec.Coa).SetName(rec.Name).
			SetBalance(rec.Balance).SetUpdateBy(rec.UpdatedBy).SetUpdateTime(rec.UpdatedAt)

		if strings.ToUpper(rec.Alignment) == "DEBIT" {
			bacc.SetAlignment(acccore.DEBIT)
		} else {
			bacc.SetAlignment(acccore.CREDIT)
		}

		ret = append(ret, bacc)
	}

	return pResult, ret, nil
}

// ListAccountByCOA returns list of accounts that have the same COA number.
// This function uses pagination
func (am *MySQLAccountManager) ListAccountByCOA(ctx context.Context, coa string, request acccore.PageRequest) (acccore.PageResult, []acccore.Account, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "ListAccountByCOA")

	count, err := am.repo.CountAccountByCoa(ctx, coa)
	if err != nil {
		lLog.Errorf("error while calling am.repo.CountAccountByCoa. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	pResult := acccore.PageResultFor(request, count)
	records, err := am.repo.ListAccountByCoa(ctx, fmt.Sprintf("%s%%", coa), "name", pResult.Offset, pResult.PageSize)
	if err != nil {
		lLog.Errorf("error while calling am.repo.ListAccountByCoa. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}

	ret := make([]acccore.Account, 0)
	for _, rec := range records {
		bacc := &acccore.BaseAccount{}
		bacc.SetAccountNumber(rec.AccountNumber).SetDescription(rec.Description).SetCreateTime(rec.CreatedAt).
			SetCreateBy(rec.CreatedBy).SetCurrency(rec.CurrencyCode).SetCOA(rec.Coa).SetName(rec.Name).
			SetBalance(rec.Balance).SetUpdateBy(rec.UpdatedBy).SetUpdateTime(rec.UpdatedAt)

		if strings.ToUpper(rec.Alignment) == "DEBIT" {
			bacc.SetAlignment(acccore.DEBIT)
		} else {
			bacc.SetAlignment(acccore.CREDIT)
		}

		ret = append(ret, bacc)
	}
	return pResult, ret, nil
}

// FindAccounts returns list of accounts that have their name contains a substring of specified parameter.
// this search should  be case insensitive.
func (am *MySQLAccountManager) FindAccounts(ctx context.Context, nameLike string, request acccore.PageRequest) (acccore.PageResult, []acccore.Account, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "FindAccounts")

	count, err := am.repo.CountAccountByName(ctx, nameLike)
	if err != nil {
		lLog.Errorf("error while calling am.repo.CountAccountByName. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}
	pResult := acccore.PageResultFor(request, count)
	records, err := am.repo.FindAccountByName(ctx, nameLike, "name", pResult.Offset, pResult.PageSize)
	if err != nil {
		lLog.Errorf("error while calling am.repo.FindAccountByName. got %s", err.Error())
		return acccore.PageResult{}, nil, err
	}

	ret := make([]acccore.Account, 0)
	for _, rec := range records {
		bacc := &acccore.BaseAccount{}
		bacc.SetAccountNumber(rec.AccountNumber).SetDescription(rec.Description).SetCreateTime(rec.CreatedAt).
			SetCreateBy(rec.CreatedBy).SetCurrency(rec.CurrencyCode).SetCOA(rec.Coa).SetName(rec.Name).
			SetBalance(rec.Balance).SetUpdateBy(rec.UpdatedBy).SetUpdateTime(rec.UpdatedAt)

		if strings.ToUpper(rec.Alignment) == "DEBIT" {
			bacc.SetAlignment(acccore.DEBIT)
		} else {
			bacc.SetAlignment(acccore.CREDIT)
		}

		ret = append(ret, bacc)
	}
	return pResult, ret, nil
}

// NewMySQLExchangeManager new sqlexcnage amanager
func NewMySQLExchangeManager(repo connector.DBRepository) acccore.ExchangeManager {
	return &MySQLExchangeManager{repo: repo, commonDenominator: 1.0}
}

// MySQLExchangeManager is the manager struct
type MySQLExchangeManager struct {
	repo              connector.DBRepository
	commonDenominator float64
}

// IsCurrencyExist will check in the exchange system for a currency existence
// non-existent currency means that the currency is not supported.
// error should be thrown if only there's an underlying error such as db error.
func (am *MySQLExchangeManager) IsCurrencyExist(ctx context.Context, currency string) (bool, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "IsCurrencyExist")

	cr, err := am.repo.GetCurrency(ctx, currency)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetCurrency. got %s", err.Error())
		return false, err
	}
	if cr == nil {
		return false, nil
	}
	return true, nil
}

// GetDenom get the current common denominator used in the exchange
func (am *MySQLExchangeManager) GetDenom(ctx context.Context) *big.Float {
	//requestID := ctx.Value(contextkeys.XRequestID).(string)
	//lLog := dbLog.WithField("RequestID", requestID).WithField("function", "GetDenom")

	return big.NewFloat(am.commonDenominator)
}

// SetDenom set the current common denominator value into the specified value
func (am *MySQLExchangeManager) SetDenom(ctx context.Context, denom *big.Float) {
	//requestID := ctx.Value(contextkeys.XRequestID).(string)
	//lLog := dbLog.WithField("RequestID", requestID).WithField("function", "SetDenom")

	f, _ := denom.Float64()
	am.commonDenominator = f
}

// GetCurrency retrieve currency data indicated by the code argument
func (am *MySQLExchangeManager) GetCurrency(ctx context.Context, code string) (acccore.Currency, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	llog := dbLog.WithField("RequestID", requestID).WithField("function", "GetCurrency")

	rec, err := am.repo.GetCurrency(ctx, code)
	if err != nil {
		llog.Errorf("error while calling am.repo.GetCurrency. got %s", err.Error())
		return nil, err
	}

	if rec == nil {
		return nil, acccore.ErrCurrencyNotFound
	}

	ret := &acccore.BaseCurrency{
		Code:       rec.Code,
		Name:       rec.Name,
		Exchange:   rec.Exchange,
		CreateTime: rec.CreatedAt,
		CreateBy:   rec.CreatedBy,
		UpdateTime: rec.UpdatedAt,
		UpdateBy:   rec.UpdatedBy,
	}

	return ret, nil
}

// CreateCurrency set the specified value as denominator value for that speciffic Currency.
// This function should return error if the Currency specified is not exist.
func (am *MySQLExchangeManager) CreateCurrency(ctx context.Context, code, name string, exchange *big.Float, author string) (acccore.Currency, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	llog := dbLog.WithField("RequestID", requestID).WithField("function", "CreateCurrency")
	ex, _ := exchange.Float64()
	rec := &connector.CurrenciesRecord{
		Code:      code,
		Name:      name,
		Exchange:  ex,
		CreatedAt: time.Now(),
		CreatedBy: author,
		UpdatedAt: time.Now(),
		UpdatedBy: author,
	}
	key, err := am.repo.InsertCurrency(ctx, rec)
	if err != nil {
		llog.Errorf("error while calling am.repo.InsertCurrency. got %s", err.Error())
		return nil, err
	}
	return &acccore.BaseCurrency{
		Code:       key,
		Name:       name,
		Exchange:   ex,
		CreateTime: rec.CreatedAt,
		CreateBy:   rec.CreatedBy,
		UpdateTime: rec.UpdatedAt,
		UpdateBy:   rec.UpdatedBy,
	}, nil
}

// UpdateCurrency updates the currency data
// Error should be returned if the specified Currency is not exist.
func (am *MySQLExchangeManager) UpdateCurrency(ctx context.Context, code string, currency acccore.Currency, author string) error {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	llog := dbLog.WithField("RequestID", requestID).WithField("function", "UpdateCurrency")

	rec := &connector.CurrenciesRecord{
		Code:      code,
		Name:      currency.GetName(),
		Exchange:  currency.GetExchange(),
		CreatedAt: currency.GetCreateTime(),
		CreatedBy: currency.GetCreateBy(),
		UpdatedAt: currency.GetUpdateTime(),
		UpdatedBy: currency.GetUpdateBy(),
	}

	err := am.repo.UpdateCurrency(ctx, rec)
	if err != nil {
		llog.Errorf("error while calling am.repo.UpdateCurrency. got %s", err.Error())
		return err
	}
	return nil
}

// CalculateExchangeRate Get the currency exchange rate for exchanging between the two currency.
// if any of the currency is not exist, an error should be returned.
// if from and to currency is equal, this must return 1.0
func (am *MySQLExchangeManager) CalculateExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*big.Float, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "CalculateExchangeRate")

	from, err := am.GetCurrency(ctx, fromCurrency)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetCurrency. got %s", err.Error())
		return nil, err
	}
	if from == nil {
		lLog.Errorf("error no currency with code %s", fromCurrency)
		return nil, acccore.ErrCurrencyNotFound
	}

	to, err := am.GetCurrency(ctx, toCurrency)
	if err != nil {
		lLog.Errorf("error while calling am.repo.GetCurrency. got %s", err.Error())
		return nil, err
	}
	if to == nil {
		lLog.Errorf("error no currency with code %s", toCurrency)
		return nil, acccore.ErrCurrencyNotFound
	}

	m1 := new(big.Float).Quo(am.GetDenom(ctx), big.NewFloat(from.GetExchange()))
	m2 := new(big.Float).Mul(m1, big.NewFloat(to.GetExchange()))
	m3 := new(big.Float).Quo(m2, am.GetDenom(ctx))
	return m3, nil
}

// CalculateExchange gets the currency exchange value for the amount of fromCurrency into toCurrency.
// If any of the currency is not exist, an error should be returned.
// if from and to currency is equal, the returned amount must be equal to the amount in the argument.
func (am *MySQLExchangeManager) CalculateExchange(ctx context.Context, fromCurrency, toCurrency string, amount int64) (int64, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	lLog := dbLog.WithField("RequestID", requestID).WithField("function", "CalculateExchange")

	exchange, err := am.CalculateExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		lLog.Errorf("error while calling am.CalculateExchangeRate. got %s", err.Error())
		return 0, err
	}
	m1 := new(big.Float).Mul(exchange, big.NewFloat(float64(amount)))
	f, _ := m1.Float64()
	return int64(f), nil
}

// ListCurrencies will list all currencies.
func (am *MySQLExchangeManager) ListCurrencies(ctx context.Context) ([]acccore.Currency, error) {
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	llog := dbLog.WithField("RequestID", requestID).WithField("function", "ListCurrencies")

	records, err := am.repo.ListCurrency(ctx, "code", 0, 1000)
	if err != nil {
		llog.Errorf("error while calling am.repo.ListCurrency. got %s", err.Error())
		return nil, err
	}

	rets := make([]acccore.Currency, 0)
	for _, rec := range records {
		cur := &acccore.BaseCurrency{
			Code:       rec.Code,
			Name:       rec.Name,
			Exchange:   rec.Exchange,
			CreateTime: rec.CreatedAt,
			CreateBy:   rec.CreatedBy,
			UpdateTime: rec.UpdatedAt,
			UpdateBy:   rec.UpdatedBy,
		}
		rets = append(rets, cur)
	}
	return rets, nil
}
