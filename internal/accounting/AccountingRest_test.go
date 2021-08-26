package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/hyperjumptech/acccore"
	"github.com/hyperjumptech/hyperwallet/internal/config"
	"github.com/hyperjumptech/hyperwallet/internal/connector"
	"github.com/hyperjumptech/hyperwallet/internal/contextkeys"
	"github.com/hyperjumptech/hyperwallet/internal/middlewares"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	journalManager     acccore.JournalManager
	accountManager     acccore.AccountManager
	transactionManager acccore.TransactionManager
	exchangeManager    acccore.ExchangeManager
	uniqueIDGenerator  acccore.UniqueIDGenerator
	Router             *mux.Router
)

func TestRestAll(t *testing.T) {

	logrus.SetLevel(logrus.DebugLevel)

	ctx := context.WithValue(context.Background(), contextkeys.XRequestID, "1234567890")
	ctx = context.WithValue(ctx, contextkeys.UserIDContextKey, "TESTING")

	if testing.Short() {
		t.Log("Running test in short mode")
		accountManager = &acccore.InMemoryAccountManager{}
		transactionManager = &acccore.InMemoryTransactionManager{}
		journalManager = &acccore.InMemoryJournalManager{}
		exchangeManager = acccore.NewInMemoryExchangeManager()
		uniqueIDGenerator = &acccore.RandomGenUniqueIDGenerator{
			Length:        16,
			LowerAlpha:    false,
			UpperAlpha:    true,
			Numeric:       true,
			CharSetBuffer: nil,
		}
		acccore.ClearInMemoryTables()
	} else {
		t.Log("Running test in normal mode")
		config.GetInt("")
		config.Set("db.host", "localhost")
		config.Set("db.port", "6603")
		config.Set("db.user", "devuser")
		config.Set("db.password", "devuser")
		config.Set("db.name", "devdb")

		repo := &connector.MySQLDBRepository{}
		assert.NoError(t, repo.Connect(ctx))
		assert.NoError(t, repo.ClearTables(ctx))

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

	AccountMgr = accountManager
	JournalMgr = journalManager
	TransactionMgr = transactionManager
	ExchangeMgr = exchangeManager
	UniqueIDGenerator = uniqueIDGenerator

	Router = mux.NewRouter()

	Router.Use(middlewares.SetupContextMiddleware, middlewares.Logger, middlewares.HMACMiddleware)
	Router.HandleFunc("/api/v1/accounts/{AccountNumber}", GetAccount).Methods("GET")
	Router.HandleFunc("/api/v1/accounts/{AccountNumber}/transactions", ListTransactionByAccount).Methods("GET")
	Router.HandleFunc("/api/v1/accounts", FindAccount).Methods("GET")
	Router.HandleFunc("/api/v1/accounts", CreateAccount).Methods("POST")

	Router.HandleFunc("/api/v1/journals", CreateJournal).Methods("POST")
	Router.HandleFunc("/api/v1/journals", ListJournal).Methods("GET")
	Router.HandleFunc("/api/v1/journals/reversal", CreateReversalJournal).Methods("POST")
	Router.HandleFunc("/api/v1/journals/{JournalID}", GetJournal).Methods("GET")

	Router.HandleFunc("/api/v1/transactions/{TransactionID}", GetTransaction).Methods("GET")

	Router.HandleFunc("/api/v1/exchange/denom", GetCommonDenominator).Methods("GET")
	Router.HandleFunc("/api/v1/exchange/denom", SetCommonDenominator).Methods("PUT")

	Router.HandleFunc("/api/v1/currencies", ListCurrencies).Methods("GET")
	Router.HandleFunc("/api/v1/currencies/{code}", GetCurrency).Methods("GET")
	Router.HandleFunc("/api/v1/currencies/{code}", SetCurrency).Methods("PUT")

	Router.HandleFunc("/api/v1/exchange/{codefrom}/{codeto}", CalculateExchangeRate).Methods("GET")
	Router.HandleFunc("/api/v1/exchange/{codefrom}/{codeto}/{amount}", CalculateExchange).Methods("GET")

	// ---------------- here comes the tests

	t.Run("Test Listing Empty Currencies", RunningTestListCurrenciesEmpty)

	t.Run("Test Create GOLD Currencies", MakeTestCreateCurrency("GOLD", "Gold Currency", 1.0, "max", http.StatusOK, "SUCCESS"))
	t.Run("Test Create POINT Currencies", MakeTestCreateCurrency("POINT", "Point Currency", 10.0, "max", http.StatusOK, "SUCCESS"))

	t.Run("Test Listing Currencies", RunningTestListCurrenciesContainsGoldPoint)
	t.Run("Test Get Individual Currencies", RunningTestFetchIndividualCurrency)
	t.Run("Test Common Denominator", RunningTestCommonDenominator)
	t.Run("Testing Exchange", RunningTestExchange)

	t.Run("Test Listing Empty Accounts", RunningTestListAccountEmpty)

	t.Run("Test Creating GoldReserve Accounts",
		MakeCreateAccountTest("GOLDRESERVE", "Gold Reserve", "Gold Reservation",
			"1.1.1", "GOLD", "DEBIT", "max", http.StatusOK, &GoldReserveAccountNo))
	t.Run("Test Creating Ferdinand Gold Accounts",
		MakeCreateAccountTest("", "Ferdinand Gold", "Ferdinand Gold Account",
			"1.1.2", "GOLD", "DEBIT", "max", http.StatusOK, &FerdinandGoldAccountNo))
	t.Run("Test Creating Budhi Gold  Accounts",
		MakeCreateAccountTest("", "Budhi Gold", "Budhi Gold Account",
			"1.1.2", "GOLD", "DEBIT", "max", http.StatusOK, &BudhiGoldAccountNo))
	t.Run("Test Creating PointReserve Accounts",
		MakeCreateAccountTest("POINTRESERVE", "Point Reserve", "Point Reservation",
			"1.2.1", "POINT", "DEBIT", "max", http.StatusOK, &PointReserveAccountNo))
	t.Run("Test Creating Ferdinand Point Accounts",
		MakeCreateAccountTest("", "Ferdinand Point", "Ferdinand Point Account",
			"1.2.2", "POINT", "DEBIT", "max", http.StatusOK, &FerdinandPointAccountNo))
	t.Run("Test Creating Budhi Point  Accounts",
		MakeCreateAccountTest("", "Budhi Point", "Budhi Point Account",
			"1.2.2", "POINT", "DEBIT", "max", http.StatusOK, &BudhiPointAccountNo))
	t.Run("Test Creating GoldCommit Accounts",
		MakeCreateAccountTest("GOLDCOMMIT", "Gold Committed", "The total commitment of gold",
			"2.1.1", "GOLD", "CREDIT", "max", http.StatusOK, &GoldCommitmentAccountNo))
	t.Run("Test Creating PointCommit Accounts",
		MakeCreateAccountTest("POINTCOMMIT", "Point Committed", "The total commitment of point",
			"2.2.1", "POINT", "CREDIT", "max", http.StatusOK, &PointCommitmentAccountNo))

	t.Run("Test Listing Filled Accounts", RunningTestListAccountFilled)
	t.Run("Test Get GOLDRESERVE Account",
		MakeFetchIndividualAccountTest("GOLDRESERVE", "Gold Reserve", "1.1.1", "GOLD", "DEBIT", 0, http.StatusOK, "SUCCESS"))

	t.Run("Test Journal Commit 2,000,000 Gold",
		MakeJournalTest("Committing Gold Reserve",
			GoldReserveAccountNo, "Reserving Gold",
			GoldCommitmentAccountNo, "Commiting Gold",
			2000000))
	t.Run("Test Journal Commit 2,000,000 Point",
		MakeJournalTest("Committing Point Reserve",
			PointReserveAccountNo, "Reserving Point",
			PointCommitmentAccountNo, "Commiting Point",
			2000000))

	t.Run("Test Ferdinand Topup 500,000 Gold",
		MakeJournalTest("TopUp Gold",
			FerdinandGoldAccountNo, "Topup",
			GoldReserveAccountNo, "Disburse To Ferdinand",
			500000))

	t.Run("Test Budhi Topup 300,000 Gold",
		MakeJournalTest("TopUp Gold",
			BudhiGoldAccountNo, "Topup",
			GoldReserveAccountNo, "Disburse To Budhi",
			300000))

	t.Run("Test Ferdinand TransferTo Budhi 50,000 Gold",
		MakeJournalTest("Transfer Gold",
			BudhiGoldAccountNo, "Receive From Ferdinand",
			FerdinandGoldAccountNo, "Send To Budhi",
			50000))
}

type AccountIndividual struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	COA           string `json:"coa"`
	Currency      string `json:"currency"`
	Alignment     string `json:"alignment"`
	Balance       int64  `json:"balance"`
}
type IndividualAccountResponse struct {
	Message   string             `json:"message"`
	Status    string             `json:"status"`
	Data      *AccountIndividual `json:"data"`
	ErrorCode int                `json:"error_code"`
}
type AccountItem struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	COA           string `json:"coa"`
	Currency      string `json:"currency"`
	Alignment     string `json:"alignment"`
	Creator       string `json:"creator"`
}
type Pagination struct {
	RequestPage  int  `json:"request_page"`
	RequestSize  int  `json:"request_size"`
	TotalEntries int  `json:"total_entries"`
	TotalPages   int  `json:"total_pages"`
	Page         int  `json:"page"`
	PageSize     int  `json:"page_size"`
	NextPage     int  `json:"next_page"`
	PreviousPage int  `json:"previous_page"`
	LastPage     int  `json:"last_page"`
	FirstPage    int  `json:"first_page"`
	IsFirst      bool `json:"is_first"`
	IsLast       bool `json:"is_last"`
	HavePrevious bool `json:"have_previous"`
	HaveNext     bool `json:"have_next"`
	Offset       int  `json:"offset"`
}

type ListAccountItem struct {
	Accounts   []*AccountItem `json:"accounts"`
	Pagination *Pagination    `json:"pagination"`
}
type ListAccountResponse struct {
	Message   string           `json:"message"`
	Status    string           `json:"status"`
	Data      *ListAccountItem `json:"data"`
	ErrorCode int              `json:"error_code"`
}
type CreateAccountResponse struct {
	Message   string `json:"message"`
	Status    string `json:"status"`
	Data      string `json:"data"`
	ErrorCode int    `json:"error_code"`
}

func MakeJournalTest(desc, accDebit, descDebit, accCredit, descCredit string, amount int64) func(t *testing.T) {
	return func(t *testing.T) {
		time.Sleep(1500 * time.Millisecond)
		body := fmt.Sprintf(`
{
  "description": "%s",
  "transactions": [
    {
      "account_number": "%s",
      "description": "%s",
      "alignment": "DEBIT",
      "amount": %d
    },
	{
      "account_number": "%s",
      "description": "%s",
      "alignment": "CREDIT",
      "amount": %d
    }
  ],
  "creator": "max"
}
`, desc, accDebit, descDebit, amount, accCredit, descCredit, amount)
		hmac := middlewares.GenHMAC()
		req, err := http.NewRequest(http.MethodPost, "http://localhost/api/v1/journals", bytes.NewBuffer([]byte(body)))
		assert.NoError(t, err)
		req.Header.Add("Authorization", hmac)
		recorder := httptest.NewRecorder()
		Router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		bodyObj := &CreateAccountResponse{}
		err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
		assert.True(t, len(bodyObj.Data) > 0)
	}
}

func MakeCreateAccountTest(accountNo, name, description, coa, currency, alignment, creator string, expectCode int, targetVar *string) func(t *testing.T) {
	return func(t *testing.T) {
		hmac := middlewares.GenHMAC()

		body := fmt.Sprintf(`
{
  "account_number": "%s",
  "name": "%s",
  "description": "%s",
  "coa": "%s",
  "currency": "%s",
  "alignment": "%s",
  "creator": "%s"
}
`, accountNo, name, description, coa, currency, alignment, creator)
		req, err := http.NewRequest(http.MethodPost, "http://localhost/api/v1/accounts", bytes.NewBuffer([]byte(body)))
		assert.NoError(t, err)
		req.Header.Add("Authorization", hmac)
		recorder := httptest.NewRecorder()
		Router.ServeHTTP(recorder, req)
		assert.Equal(t, expectCode, recorder.Code)
		bodyObj := &CreateAccountResponse{}
		err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
		assert.NoError(t, err)
		if len(accountNo) > 0 {
			assert.Equal(t, accountNo, bodyObj.Data)
		} else {
			assert.True(t, len(bodyObj.Data) > 0)
		}

		rstr := reflect.ValueOf(targetVar).Elem()
		rdata := reflect.ValueOf(bodyObj).Elem()
		rf := rdata.FieldByName("Data")
		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
		rstr.Set(rf)
		rf.Set(rstr)
	}
}

var (
	FerdinandGoldAccountNo   string
	BudhiGoldAccountNo       string
	FerdinandPointAccountNo  string
	BudhiPointAccountNo      string
	GoldReserveAccountNo     = "GOLDRESERVE"
	GoldCommitmentAccountNo  = "GOLDCOMMIT"
	PointReserveAccountNo    = "POINTRESERVE"
	PointCommitmentAccountNo = "POINTCOMMIT"
)

func MakeFetchIndividualAccountTest(accountNo, name, coa, currency, alignment string, balance int, expectCode int, expectStatus string) func(t *testing.T) {
	return func(t *testing.T) {
		hmac := middlewares.GenHMAC()
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost/api/v1/accounts/%s", accountNo), nil)
		assert.NoError(t, err)
		req.Header.Add("Authorization", hmac)
		recorder := httptest.NewRecorder()
		Router.ServeHTTP(recorder, req)
		assert.Equal(t, expectCode, recorder.Code)

		bodyObj := &IndividualAccountResponse{}
		err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
		assert.NoError(t, err)
		assert.Equal(t, expectStatus, bodyObj.Status)
		assert.Equal(t, accountNo, bodyObj.Data.AccountNumber)
		assert.Equal(t, name, bodyObj.Data.Name)
		assert.Equal(t, coa, bodyObj.Data.COA)
		assert.Equal(t, currency, bodyObj.Data.Currency)
		assert.Equal(t, alignment, bodyObj.Data.Alignment)
		assert.Equal(t, int64(balance), bodyObj.Data.Balance)
	}
}

func RunningTestListAccountEmpty(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/accounts?name=ferd&page=1&size=10", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ListAccountResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(bodyObj.Data.Accounts))
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 0, bodyObj.Data.Pagination.TotalEntries)
}

func RunningTestListAccountFilled(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/accounts?name=ferd&page=1&size=10", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ListAccountResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(bodyObj.Data.Accounts))
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 2, bodyObj.Data.Pagination.TotalEntries)
}

type ListCurrencyItem struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Exchange float64 `json:"exchange"`
}

type ListCurrencyResponse struct {
	Message   string              `json:"message"`
	Status    string              `json:"status"`
	Data      []*ListCurrencyItem `json:"data"`
	ErrorCode int                 `json:"error_code"`
}

func RunningTestListCurrenciesEmpty(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/currencies", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ListCurrencyResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(bodyObj.Data))
	assert.Equal(t, "SUCCESS", bodyObj.Status)
}

func RunningTestListCurrenciesContainsGoldPoint(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/currencies", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ListCurrencyResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(bodyObj.Data))
	assert.Equal(t, "GOLD", bodyObj.Data[0].Code)
	assert.Equal(t, "POINT", bodyObj.Data[1].Code)
	assert.Equal(t, "Gold Currency", bodyObj.Data[0].Name)
	assert.Equal(t, "Point Currency", bodyObj.Data[1].Name)
	assert.Equal(t, 1.0, bodyObj.Data[0].Exchange)
	assert.Equal(t, 10.0, bodyObj.Data[1].Exchange)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
}

type CreateCurrencyResponse struct {
	Message   string           `json:"message"`
	Status    string           `json:"status"`
	Data      ListCurrencyItem `json:"data"`
	ErrorCode int              `json:"error_code"`
}

func MakeTestCreateCurrency(code, name string, exchange float64, author string, expectCode int, expectStatus string) func(t *testing.T) {
	return func(t *testing.T) {
		hmac := middlewares.GenHMAC()
		body1 := fmt.Sprintf(`{"name":"%s", "exchange":%f, "author":"%s"}`, name, exchange, author)
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost/api/v1/currencies/%s", code), bytes.NewBuffer([]byte(body1)))
		assert.NoError(t, err)
		req.Header.Add("Authorization", hmac)

		recorder := httptest.NewRecorder()
		Router.ServeHTTP(recorder, req)
		assert.Equal(t, expectCode, recorder.Code)
		bodyObj := &CreateCurrencyResponse{}
		err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
		assert.NoError(t, err)
		assert.Equal(t, expectStatus, bodyObj.Status)
		assert.Equal(t, code, bodyObj.Data.Code)
		assert.Equal(t, name, bodyObj.Data.Name)
		assert.Equal(t, exchange, bodyObj.Data.Exchange)
	}
}

func RunningTestFetchIndividualCurrency(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/currencies/GOLD", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &CreateCurrencyResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "GOLD", bodyObj.Data.Code)
	assert.Equal(t, "Gold Currency", bodyObj.Data.Name)
	assert.Equal(t, 1.0, bodyObj.Data.Exchange)

	req, err = http.NewRequest(http.MethodGet, "http://localhost/api/v1/currencies/EMERALD", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

type ExchangeDenominatorResponse struct {
	Message   string  `json:"message"`
	Status    string  `json:"status"`
	Data      float64 `json:"data"`
	ErrorCode int     `json:"error_code"`
}

func RunningTestCommonDenominator(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/denom", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 1.0, bodyObj.Data)

	req, err = http.NewRequest(http.MethodPut, "http://localhost/api/v1/exchange/denom?denom=0.123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj = &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 0.123, bodyObj.Data)

	req, err = http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/denom", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj = &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 0.123, bodyObj.Data)
}

func RunningTestExchange(t *testing.T) {
	hmac := middlewares.GenHMAC()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/GOLD/POINT", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder := httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj := &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 10.0, bodyObj.Data)

	req, err = http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/POINT/GOLD", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj = &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 0.1, bodyObj.Data)

	req, err = http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/GOLD/POINT/100", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj = &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 1000.0, bodyObj.Data)

	req, err = http.NewRequest(http.MethodGet, "http://localhost/api/v1/exchange/POINT/GOLD/100", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", hmac)
	recorder = httptest.NewRecorder()
	Router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	bodyObj = &ExchangeDenominatorResponse{}
	err = json.Unmarshal(recorder.Body.Bytes(), &bodyObj)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", bodyObj.Status)
	assert.Equal(t, 10.0, bodyObj.Data)
}
