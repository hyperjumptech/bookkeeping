package accounting



import (
	"context"
	"github.com/IDN-Media/awards/internal/config"
	"github.com/IDN-Media/awards/internal/connector"
	"github.com/hyperjumptech/acccore"
	"math/big"
	"testing"
	"time"
)

func TestAccounting_CreateNewAccount(t *testing.T) {

	ctx := context.WithValue(context.Background(), connector.UserIDContextKey, "TESTING")

	var journalManager acccore.JournalManager
	var accountManager acccore.AccountManager
	var transactionManager acccore.TransactionManager
	var exchangeManager acccore.ExchangeManager
	var uniqueIDGenerator acccore.UniqueIDGenerator

	if testing.Short() {
		accountManager = &acccore.InMemoryAccountManager{}
		transactionManager = &acccore.InMemoryTransactionManager{}
		journalManager= &acccore.InMemoryJournalManager{}
		exchangeManager= &acccore.InMemoryExchangeManager{}
		uniqueIDGenerator =  &acccore.RandomGenUniqueIDGenerator {
			Length:        10,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
		acccore.ClearInMemoryTables()
	} else {
		config.GetInt("")
		config.Set("db.host", "localhost")
		config.Set("db.port", "3306")
		config.Set("db.user", "devuser")
		config.Set("db.password", "devpassword")
		config.Set("db.name", "devdb")

		repo := &connector.MySqlDBRepository{}
		err := repo.Connect(ctx)
		if err != nil {
			t.Errorf("cannot connect to db. got %s", err.Error())
			t.FailNow()
		}

		journalManager = NewMySQLJournalManager(repo)
		accountManager = NewMySQLAccountManager(repo)
		transactionManager = NewMySQLTransactionManager(repo)
		exchangeManager = NewMySQLExchangeManager(repo)
		uniqueIDGenerator =  &acccore.RandomGenUniqueIDGenerator {
			Length:        10,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
	}

	err := exchangeManager.SetExchangeValueOf(ctx, "GLD", big.NewFloat(1.0), ctx.Value(connector.UserIDContextKey).(string))
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	acc := acccore.NewAccounting(accountManager, transactionManager, journalManager,uniqueIDGenerator)

	account, err := acc.CreateNewAccount(ctx, "Test Account", "Gold base test user account", "1.1", "GLD", acccore.CREDIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	exist, err := acc.GetAccountManager().IsAccountIdExist(ctx, account.GetAccountNumber())
	if err != nil {
		t.Error(err.Error())
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
	ctx := context.Background()

	mysqlJournalManager := &MySQLJournalManager{}
	mysqlAccountManager := &MySQLAccountManager{}
	mysqlTransactionManager := &MySQLTransactionManager{}

	acc := acccore.NewAccounting(mysqlAccountManager, mysqlTransactionManager, mysqlJournalManager, &acccore.RandomGenUniqueIDGenerator{
		Length:        10,
		LowerAlpha:    false,
		UpperAlpha:    true,
		Numeric:       true,
		CharSetBuffer: nil,
	})

	goldLoan, err := acc.CreateNewAccount(ctx, "Gold Loan", "Gold base loan reserve", "1.1", "GOLD", acccore.DEBIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	alphaCreditor, err := acc.CreateNewAccount(ctx, "Gold Creditor Alpha", "Gold base debitor alpha", "2.1", "GOLD", acccore.CREDIT, "aCreator")
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	betaDebitor, err := acc.CreateNewAccount(ctx, "Gold Debitor Alpha", "Gold base creditor beta", "3.1", "GOLD", acccore.DEBIT, "aCreator")
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
