package accounting

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/hyperjumptech/acccore"
	"github.com/hyperjumptech/hyperwallet/internal/config"
	"github.com/hyperjumptech/hyperwallet/internal/connector"
	"github.com/hyperjumptech/hyperwallet/internal/contextkeys"
)

func TestAccounting_CreateNewAccount(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkeys.XRequestID, "1234567890")
	ctx = context.WithValue(ctx, contextkeys.UserIDContextKey, "TESTING")

	var journalManager acccore.JournalManager
	var accountManager acccore.AccountManager
	var transactionManager acccore.TransactionManager
	var exchangeManager acccore.ExchangeManager
	var uniqueIDGenerator acccore.UniqueIDGenerator

	if testing.Short() {
		accountManager = &acccore.InMemoryAccountManager{}
		transactionManager = &acccore.InMemoryTransactionManager{}
		journalManager = &acccore.InMemoryJournalManager{}
		exchangeManager = &acccore.InMemoryExchangeManager{}
		uniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
			Length:        16,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
		acccore.ClearInMemoryTables()
	} else {
		config.GetInt("")
		config.Set("db.host", "localhost")
		config.Set("db.port", "6603")
		config.Set("db.user", "devuser")
		config.Set("db.password", "devuser")
		config.Set("db.name", "devdb")

		repo := &connector.MySQLDBRepository{}
		err := repo.Connect(ctx)
		if err != nil {
			t.Errorf("cannot connect to db. got %s", err.Error())
			t.FailNow()
		}

		err = repo.ClearTables(ctx)
		if err != nil {
			t.Errorf("cannot clear tables. got %s", err.Error())
			t.FailNow()
		}

		journalManager = NewMySQLJournalManager(repo)
		accountManager = NewMySQLAccountManager(repo)
		transactionManager = NewMySQLTransactionManager(repo)
		exchangeManager = NewMySQLExchangeManager(repo)
		uniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
			Length:        16,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
	}

	_, err := exchangeManager.CreateCurrency(ctx, "GLD", "Gold Bullion", big.NewFloat(1.0), ctx.Value(contextkeys.UserIDContextKey).(string))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	acc := acccore.NewAccounting(accountManager, transactionManager, journalManager, uniqueIDGenerator)

	account, err := acc.CreateNewAccount(ctx, "", "Test Account", "Gold base test user account", "1.1", "GLD", acccore.CREDIT, "aCreator")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	exist, err := acc.GetAccountManager().IsAccountIdExist(ctx, account.GetAccountNumber())
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !exist {
		t.Error("account should exist after creation")
		t.FailNow()
	}
	render, err := acc.GetTransactionManager().RenderTransactionsOnAccount(ctx, time.Now().Add(-2*time.Hour), time.Now().Add(2*time.Hour), account, acccore.PageRequest{
		PageNo:   1,
		ItemSize: 10,
		Sorts:    nil,
	})
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	} else {
		t.Log(render)
	}
}

func TestAccounting_CreateNewJournal(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkeys.XRequestID, "1234567890")
	ctx = context.WithValue(ctx, contextkeys.UserIDContextKey, "TESTING")

	var journalManager acccore.JournalManager
	var accountManager acccore.AccountManager
	var transactionManager acccore.TransactionManager
	var exchangeManager acccore.ExchangeManager
	var uniqueIDGenerator acccore.UniqueIDGenerator

	if testing.Short() {
		accountManager = &acccore.InMemoryAccountManager{}
		transactionManager = &acccore.InMemoryTransactionManager{}
		journalManager = &acccore.InMemoryJournalManager{}
		exchangeManager = &acccore.InMemoryExchangeManager{}
		uniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
			Length:        16,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
		acccore.ClearInMemoryTables()
	} else {
		config.GetInt("")
		config.Set("db.host", "localhost")
		config.Set("db.port", "6603")
		config.Set("db.user", "devuser")
		config.Set("db.password", "devuser")
		config.Set("db.name", "devdb")

		repo := &connector.MySQLDBRepository{}
		err := repo.Connect(ctx)
		if err != nil {
			t.Errorf("cannot connect to db. got %s", err.Error())
			t.FailNow()
		}

		err = repo.ClearTables(ctx)
		if err != nil {
			t.Errorf("cannot clear tables. got %s", err.Error())
			t.FailNow()
		}

		journalManager = NewMySQLJournalManager(repo)
		accountManager = NewMySQLAccountManager(repo)
		transactionManager = NewMySQLTransactionManager(repo)
		exchangeManager = NewMySQLExchangeManager(repo)
		uniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
			Length:        16,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
	}

	acc := acccore.NewAccounting(accountManager, transactionManager, journalManager, uniqueIDGenerator)

	_, err := exchangeManager.CreateCurrency(ctx, "GOLD", "Gold Bullion", big.NewFloat(1.0), ctx.Value(contextkeys.UserIDContextKey).(string))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	goldLoan, err := acc.CreateNewAccount(ctx, "", "Gold Loan", "Gold base loan reserve", "1.1", "GOLD", acccore.DEBIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	alphaCreditor, err := acc.CreateNewAccount(ctx, "", "Gold Creditor Alpha", "Gold base debitor alpha", "2.1", "GOLD", acccore.CREDIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	betaDebitor, err := acc.CreateNewAccount(ctx, "", "Gold Debitor Alpha", "Gold base creditor beta", "3.1", "GOLD", acccore.DEBIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	topupTransactions := []acccore.TransactionInfo{
		{
			AccountNumber: goldLoan.GetAccountNumber(),
			Description:   "Added Gold Reserve",
			TxType:        acccore.DEBIT,
			Amount:        1000000,
		},
		{
			AccountNumber: alphaCreditor.GetAccountNumber(),
			Description:   "Added Gold Equity",
			TxType:        acccore.CREDIT,
			Amount:        1000000,
		},
	}
	journal, err := acc.CreateNewJournal(ctx, "Creditor Topup Gold", topupTransactions, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	if journal == nil {
		t.Error("Journal is nil")
		t.FailNow()
	}
	t.Log(acc.GetJournalManager().RenderJournal(ctx, journal))

	goldPurchaseTransaction := []acccore.TransactionInfo{
		{
			AccountNumber: betaDebitor.GetAccountNumber(),
			Description:   "Add debitor AR",
			TxType:        acccore.DEBIT,
			Amount:        200000,
		},
		{
			AccountNumber: goldLoan.GetAccountNumber(),
			Description:   "Gold Disbursement",
			TxType:        acccore.CREDIT,
			Amount:        200000,
		},
	}
	journal, err = acc.CreateNewJournal(ctx, "GOLD purchase transaction", goldPurchaseTransaction, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	t.Log(acc.GetJournalManager().RenderJournal(ctx, journal))

	pr, trxs, _ := acc.GetTransactionManager().ListTransactionsOnAccount(ctx, time.Now().Add(-2*time.Hour), time.Now().Add(2*time.Hour), goldLoan, acccore.PageRequest{
		PageNo:   1,
		ItemSize: 10,
		Sorts:    nil,
	})
	if len(trxs) == 0 {
		t.Error("Empty transaction")
		t.Fail()
	}
	if pr.TotalEntries == 0 {
		t.Error("Empty TotalEntries")
		t.Fail()
	}

	render, err := acc.GetTransactionManager().RenderTransactionsOnAccount(ctx, time.Now().Add(-2*time.Hour), time.Now().Add(2*time.Hour), goldLoan, acccore.PageRequest{
		PageNo:   1,
		ItemSize: 10,
		Sorts:    nil,
	})
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	} else {
		t.Log(render)
	}
}
