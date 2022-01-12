package accounting

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hyperjumptech/acccore"
	"github.com/hyperjumptech/bookkeeping/internal/contextkeys"
	"github.com/hyperjumptech/bookkeeping/internal/helpers"
	"github.com/sirupsen/logrus"
)

var (
	// AccountMgr is the account manager instance used in all rest endpoint
	AccountMgr acccore.AccountManager

	// JournalMgr is the journal manager instance used in all rest endpoint
	JournalMgr acccore.JournalManager

	// TransactionMgr is the transaction manager instance used in all rest endpoint
	TransactionMgr acccore.TransactionManager

	// ExchangeMgr is the exchange manager instance used in all rest endpoint
	ExchangeMgr acccore.ExchangeManager

	// UniqueIDGenerator is the UniqueIDGenerator instance used in all rest endpoint
	UniqueIDGenerator acccore.UniqueIDGenerator

	restLog = logrus.WithField("file", "AccountRest.go")

	// ErrRestPathInvalid base error used to indicate if a URL path is not valid
	ErrRestPathInvalid = errors.New("invalid path error")

	// RestTimeFormat data format for all time.Time typed json string.
	RestTimeFormat = "2006-01-02T15:04:05"
)

// NewAccountEntity is the structure of request body for creating new Account
type NewAccountEntity struct {
	AccountNo   string `json:"account_number"`
	Name        string `json:"name"`
	Description string `json:"description"`
	COA         string `json:"coa"`
	Currency    string `json:"currency"`
	Alignment   string `json:"alignment"`
	Creator     string `json:"creator"`
}

// AccountEntity is the structure of response body that contains an account
type AccountEntity struct {
	AccountNo   string `json:"account_number"`
	Name        string `json:"name"`
	Description string `json:"description"`
	COA         string `json:"coa"`
	Currency    string `json:"currency"`
	Alignment   string `json:"alignment"`
	Balance     int64  `json:"balance"`
}

// PaginatedResponse is the structure of stuff that requires pagination
type PaginatedResponse struct {
	Items      interface{}
	Pagination acccore.PageResult
}

// TransactionListResponse is the structure response
type TransactionListResponse struct {
	Transactions []*TransactionListItem `json:"transactions"`
	Pagination   acccore.PageResult     `json:"pagination"`
}

// DrawAccount draws the account activity
func DrawAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "ListTransactionByAccount")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/accounts/{AccountNumber}/draw", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/accounts/{AccountNumber}/draw. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	var from, until time.Time
	var page, size int

	qfrom := r.URL.Query()["from"]
	if qfrom == nil || len(qfrom[0]) == 0 {
		llog.Errorf("error missing from field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing from", "missing from", 1)
		return
	}
	from, err = time.Parse(RestTimeFormat, qfrom[0])
	if err != nil {
		llog.Errorf("invalid from date format : %s", qfrom[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid from date format", "invalid from date format", 1)
		return
	}
	quntil := r.URL.Query()["until"]
	if quntil == nil || len(quntil[0]) == 0 {
		llog.Errorf("error missing until field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing until", "missing until", 1)
		return
	}
	until, err = time.Parse(RestTimeFormat, quntil[0])
	if err != nil {
		llog.Errorf("invalid until date format : %s", quntil[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid until date format", "invalid from date format", 1)
		return
	}

	qpage := r.URL.Query()["page"]
	if qpage == nil || len(qpage[0]) == 0 {
		llog.Errorf("error missing page field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing from", "missing from", 1)
		return
	}
	page, err = strconv.Atoi(qpage[0])
	if err != nil {
		llog.Errorf("invalid page number format : %s", qpage[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid page number format", "invalid page number format", 1)
		return
	}

	qsize := r.URL.Query()["size"]
	if qsize == nil || len(qsize[0]) == 0 {
		llog.Errorf("error missing size field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing until", "missing until", 1)
		return
	}
	size, err = strconv.Atoi(qsize[0])
	if err != nil {
		llog.Errorf("invalid size number format : %s", qsize[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid size number format", "invalid size number format", 1)
		return
	}

	accountNo := m["AccountNumber"]
	account, err := AccountMgr.GetAccountByID(r.Context(), accountNo)
	if err != nil {
		llog.Errorf("error while calling AccountMgr.GetAccountByID. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "backend error", err.Error(), 2)
		return
	}
	if account == nil {
		llog.Errorf("error account number not found : %s", accountNo)
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "backend error", "account not found", 2)
		return
	}

	str, err := TransactionMgr.RenderTransactionsOnAccount(r.Context(), from, until, account, acccore.PageRequest{
		PageNo:   page,
		ItemSize: size,
	})
	if err != nil {
		llog.Errorf("error while calling TransactionMgr.RenderTransactionsOnAccount. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "backend error", err.Error(), 2)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(str))
}

// GetAccount is the controller to handle retrieval of single account
func GetAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "GetAccount")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/accounts/{AccountNumber}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/accounts/{AccountNumber}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}
	accountNo := m["AccountNumber"]
	account, err := AccountMgr.GetAccountByID(r.Context(), accountNo)
	if err != nil {
		llog.Errorf("error while calling AccountMgr.GetAccountByID. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "backend error", err.Error(), 2)
		return
	}
	if account == nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "account number not found", "account number not found", 3)
		return
	}
	ret := &AccountEntity{
		AccountNo:   account.GetAccountNumber(),
		Name:        account.GetName(),
		Description: account.GetDescription(),
		COA:         account.GetCOA(),
		Currency:    account.GetCurrency(),
		//Alignment:   account.GetBaseTransactionType(),
		Balance: account.GetBalance(),
	}
	if account.GetAlignment() == acccore.DEBIT {
		ret.Alignment = "DEBIT"
	} else {
		ret.Alignment = "CREDIT"
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "account "+account.GetAccountNumber(), ret, 0)
}

// ListTransactionByAccount lists transactions given an account
func ListTransactionByAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "ListTransactionByAccount")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/accounts/{AccountNumber}/transactions", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/accounts/{AccountNumber}/transactions. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	var from, until time.Time
	var page, size int

	qfrom := r.URL.Query()["from"]
	if qfrom == nil || len(qfrom[0]) == 0 {
		llog.Errorf("error missing from field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing from", "missing from", 1)
		return
	}
	from, err = time.Parse(RestTimeFormat, qfrom[0])
	if err != nil {
		llog.Errorf("invalid from date format : %s", qfrom[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid from date format", "invalid from date format", 1)
		return
	}
	quntil := r.URL.Query()["until"]
	if quntil == nil || len(quntil[0]) == 0 {
		llog.Errorf("error missing until field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing until", "missing until", 1)
		return
	}
	until, err = time.Parse(RestTimeFormat, quntil[0])
	if err != nil {
		llog.Errorf("invalid until date format : %s", quntil[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid until date format", "invalid from date format", 1)
		return
	}

	qpage := r.URL.Query()["page"]
	if qpage == nil || len(qpage[0]) == 0 {
		llog.Errorf("error missing page field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing from", "missing from", 1)
		return
	}
	page, err = strconv.Atoi(qpage[0])
	if err != nil {
		llog.Errorf("invalid page number format : %s", qpage[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid page number format", "invalid page number format", 1)
		return
	}

	qsize := r.URL.Query()["size"]
	if qsize == nil || len(qsize[0]) == 0 {
		llog.Errorf("error missing size field")
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "missing until", "missing until", 1)
		return
	}
	size, err = strconv.Atoi(qsize[0])
	if err != nil {
		llog.Errorf("invalid size number format : %s", qsize[0])
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid size number format", "invalid size number format", 1)
		return
	}

	accountNo := m["AccountNumber"]
	account, err := AccountMgr.GetAccountByID(r.Context(), accountNo)
	if err != nil {
		llog.Errorf("error while calling AccountMgr.GetAccountByID. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "backend error", err.Error(), 2)
		return
	}
	if account == nil {
		llog.Errorf("error account number not found : %s", accountNo)
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "backend error", err.Error(), 2)
		return
	}

	pr, transactions, err := TransactionMgr.ListTransactionsOnAccount(r.Context(), from, until, account, acccore.PageRequest{
		PageNo:   page,
		ItemSize: size,
	})
	if err != nil {
		llog.Errorf("error while calling TransactionMgr.ListTransactionsOnAccount. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "backend error", err.Error(), 2)
		return
	}

	retTransac := make([]*TransactionListItem, len(transactions))
	for idx, trx := range transactions {
		align := "DEBIT"
		if trx.GetAlignment() == acccore.CREDIT {
			align = "CREDIT"
		}
		retTransac[idx] = &TransactionListItem{
			TransactionID:   trx.GetTransactionID(),
			TransactionTime: trx.GetTransactionTime().Format(time.RFC3339),
			AccountNumber:   trx.GetAccountNumber(),
			JournalID:       trx.GetJournalID(),
			Description:     trx.GetDescription(),
			TransactionType: align,
			Amount:          trx.GetAmount(),
			AccountBalance:  trx.GetAccountBalance(),
			CreateTime:      trx.GetCreateTime().Format(time.RFC3339),
			CreateBy:        trx.GetCreateBy(),
		}
	}

	resp := &TransactionListResponse{
		Transactions: retTransac,
		Pagination:   pr,
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "transaction list", resp, 2)
}

// JournalDetail is the journal detail struct
type JournalDetail struct {
	JournalID       string `json:"journal_id"`
	JournalingTime  string `json:"journaling_time"`
	Description     string `json:"description"`
	Reversal        bool   `json:"reversal"`
	ReversedJournal string `json:"reversed_journal"`
	Amount          int64  `json:"amount"`
	Transactions    []*TransactionListItem
	CreateTime      string `json:"create_time"`
	CreateBy        string `json:"create_by"`
}

// TransactionListItem is the transaction detail
type TransactionListItem struct {
	TransactionID   string `json:"transaction_id"`
	TransactionTime string `json:"transaction_time"`
	AccountNumber   string `json:"account_number"`
	JournalID       string `json:"journal_id"`
	Description     string `json:"description"`
	TransactionType string `json:"transaction_type"`
	Amount          int64  `json:"amount"`
	AccountBalance  int64  `json:"account_balance"`
	CreateTime      string `json:"create_time"`
	CreateBy        string `json:"create_by"`
}

// AccountResponseBody with the account details
type AccountResponseBody struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	COA           string `json:"coa"`
	Currency      string `json:"currency"`
	Alignment     string `json:"alignment"`
	Creator       string `json:"creator"`
}

// AccountItemsResponseBody response for account items
type AccountItemsResponseBody struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	COA           string `json:"coa"`
	Currency      string `json:"currency"`
	Alignment     string `json:"alignment"`
	Balance       int64  `json:"balance"`
}

// FromAccorePageResult returns the pagination detail
func FromAccorePageResult(pr acccore.PageResult) *PageResultBody {
	return &PageResultBody{
		RequestPage:  pr.Request.PageNo,
		RequestSize:  pr.Request.ItemSize,
		TotalEntries: pr.TotalEntries,
		TotalPages:   pr.TotalPages,
		Page:         pr.Page,
		PageSize:     pr.PageSize,
		NextPage:     pr.NextPage,
		PreviousPage: pr.PreviousPage,
		LastPage:     pr.LastPage,
		FirstPage:    pr.FirstPage,
		IsFirst:      pr.IsFirst,
		IsLast:       pr.IsLast,
		HavePrevious: pr.HavePrev,
		HaveNext:     pr.HaveNext,
		Offset:       pr.Offset,
	}
}

// PageResultBody is the pagination payload
type PageResultBody struct {
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

// FindAccount searches for a given account id
func FindAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "FindAccount")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	name, ok := r.URL.Query()["name"]
	if !ok || len(name[0]) == 0 {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "missing name", 0)
		return
	}
	if len(name[0]) < 3 {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "name query length is too short", 0)
		return
	}
	page, ok := r.URL.Query()["page"]
	if !ok || len(page[0]) == 0 {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "missing page", 0)
		return
	}
	size, ok := r.URL.Query()["size"]
	if !ok || len(size[0]) == 0 {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "missing size", 0)
		return
	}

	npage, err := strconv.Atoi(page[0])
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "page is not a number", 0)
		return
	}
	nsize, err := strconv.Atoi(size[0])
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request", "size is not a number", 0)
		return
	}

	pr, accounts, err := AccountMgr.FindAccounts(r.Context(), fmt.Sprintf("%%%s%%", name[0]), acccore.PageRequest{
		PageNo:   npage,
		ItemSize: nsize,
		Sorts:    nil,
	})
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 0)
		return
	}

	accountSet := make([]*AccountItemsResponseBody, 0)
	for _, acc := range accounts {
		acci := &AccountItemsResponseBody{
			AccountNumber: acc.GetAccountNumber(),
			Name:          acc.GetName(),
			Description:   acc.GetDescription(),
			COA:           acc.GetCOA(),
			Currency:      acc.GetCurrency(),

			//Alignment:     "",
			Balance: acc.GetBalance(),
		}
		if acc.GetAlignment() == acccore.DEBIT {
			acci.Alignment = "DEBIT"
		} else {
			acci.Alignment = "CREDIT"
		}
		accountSet = append(accountSet, acci)
	}

	resp := &struct {
		Accounts   []*AccountItemsResponseBody `json:"accounts"`
		Pagination *PageResultBody             `json:"pagination"`
	}{
		Accounts:   accountSet,
		Pagination: FromAccorePageResult(pr),
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "accounts", resp, 0)
}

// CreateAccount creates an account
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "CreateAccount")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	bodyByte, err := ioutil.ReadAll(r.Body)
	if err != nil {
		llog.Errorf("got %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "error reading body", err.Error(), 0)
		return
	}

	newEnt := &NewAccountEntity{}
	err = json.Unmarshal(bodyByte, newEnt)
	if err != nil {
		llog.Errorf("got %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "error reading body", err.Error(), 0)
		return
	}

	nctx := context.WithValue(r.Context(), contextkeys.UserIDContextKey, newEnt.Creator)

	acc := &acccore.BaseAccount{}
	acc.SetAccountNumber(newEnt.AccountNo).SetUpdateTime(time.Now()).SetUpdateBy(newEnt.Creator).
		SetCreateBy(newEnt.Creator).SetCreateTime(time.Now()).SetBalance(0).SetName(newEnt.Name).
		SetCOA(newEnt.COA).SetCurrency(newEnt.Currency).SetDescription(newEnt.Description)

	if strings.ToUpper(newEnt.Alignment) == "DEBIT" {
		acc.SetAlignment(acccore.DEBIT)
	} else {
		acc.SetAlignment(acccore.CREDIT)
	}

	if len(acc.GetAccountNumber()) == 0 {
		acc.SetAccountNumber(UniqueIDGenerator.NewUniqueID())
	}

	err = AccountMgr.PersistAccount(nctx, acc)
	if err != nil {
		llog.Errorf("got %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "error reading body", err.Error(), 0)
		return
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "create account", acc.AccountNumber, 0)
}

// GetJournal fetches a journal from journal ID
func GetJournal(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "GetJournal")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/journals/{JournalID}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/journals/{JournalID}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	j, err := JournalMgr.GetJournalByID(r.Context(), m["JournalID"])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrJournalIDNotFound) {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "journal not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	if j == nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "journal not found", 1)
		return
	}

	reversedJournal := ""
	if j.IsReversal() && j.GetReversedJournal() != nil {
		reversedJournal = j.GetReversedJournal().GetJournalID()
	}

	retJournal := &JournalDetail{
		JournalID:       j.GetJournalID(),
		JournalingTime:  j.GetJournalingTime().Format(time.RFC3339),
		Description:     j.GetDescription(),
		Reversal:        j.IsReversal(),
		ReversedJournal: reversedJournal,
		Amount:          j.GetAmount(),
		Transactions:    nil,
		CreateTime:      j.GetCreateTime().Format(time.RFC3339),
		CreateBy:        j.GetCreateBy(),
	}
	retTrxes := make([]*TransactionListItem, len(j.GetTransactions()))
	for idx, trx := range j.GetTransactions() {
		align := "DEBIT"
		if trx.GetAlignment() == acccore.CREDIT {
			align = "CREDIT"
		}
		retTrxes[idx] = &TransactionListItem{
			TransactionID:   trx.GetTransactionID(),
			TransactionTime: trx.GetTransactionTime().Format(time.RFC3339),
			AccountNumber:   trx.GetAccountNumber(),
			JournalID:       trx.GetJournalID(),
			Description:     trx.GetDescription(),
			TransactionType: align,
			Amount:          trx.GetAmount(),
			AccountBalance:  trx.GetAccountBalance(),
			CreateTime:      trx.GetCreateTime().Format(time.RFC3339),
			CreateBy:        trx.GetCreateBy(),
		}
	}
	retJournal.Transactions = retTrxes

	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", retJournal, 1)
}

// DrawJournal draws the journal activity for easier debugging
func DrawJournal(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "DrawJournal")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/journals/{JournalID}/draw", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/journals/{JournalID}/draw. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	j, err := JournalMgr.GetJournalByID(r.Context(), m["JournalID"])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrJournalIDNotFound) {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "journal not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	if j == nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "journal not found", 1)
		return
	}

	drawing := JournalMgr.RenderJournal(r.Context(), j)

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(drawing))
}

// PaginatedJournalsResponse is the journal response paginated
type PaginatedJournalsResponse struct {
	Journals   []acccore.Journal  `json:"journals"`
	Pagination acccore.PageResult `json:"pagination"`
}

// ListJournal lists the journal given
func ListJournal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := ctx.Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "ListJournal")
	if ctx.Err() != nil {
		llog.Errorf("context is canceled : %s", ctx.Err().Error())
		helpers.HTTPResponseBuilder(ctx, w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	fromA, fOk := r.URL.Query()["from"]
	untilA, uOk := r.URL.Query()["until"]
	pageA, pOk := r.URL.Query()["page"]
	sizeA, sOk := r.URL.Query()["size"]

	if !fOk || !uOk || !pOk || !sOk {
		helpers.HTTPResponseBuilder(ctx, w, r, 400, "invalid request", "either from, until, page or size is missing", 0)
		return
	}
	fTime, ferr := time.Parse(RestTimeFormat, fromA[0])
	uTime, uerr := time.Parse(RestTimeFormat, untilA[0])
	if ferr != nil || uerr != nil {
		helpers.HTTPResponseBuilder(ctx, w, r, 400, "invalid request", "either from, until time format not correct", 0)
		return
	}
	page, perr := strconv.Atoi(pageA[0])
	size, serr := strconv.Atoi(sizeA[0])
	if perr != nil || serr != nil {
		helpers.HTTPResponseBuilder(ctx, w, r, 400, "invalid request", "either page, size is not number", 0)
		return
	}

	pr, journals, err := JournalMgr.ListJournals(r.Context(), fTime, uTime, acccore.PageRequest{
		PageNo:   page,
		ItemSize: size,
		Sorts:    nil,
	})
	if err != nil {
		helpers.HTTPResponseBuilder(ctx, w, r, 500, "internal server error", err.Error(), 0)
		return
	}
	helpers.HTTPResponseBuilder(ctx, w, r, 200, "OK", &PaginatedJournalsResponse{
		Journals:   journals,
		Pagination: pr,
	}, 0)
}

// CreateReversalRequest is the create reversal request payload
type CreateReversalRequest struct {
	Description string `json:"description"`
	JournalID   string `json:"journal_id"`
	Creator     string `json:"creator"`
}

// CreateJournalRequest is the create journal request paylaod
type CreateJournalRequest struct {
	Description  string                `json:"description"`
	Creator      string                `json:"creator"`
	Transactions []*TransactionRequest `json:"transactions"`
}

// TransactionRequest is the create transaction request payload
type TransactionRequest struct {
	AccountNumber string `json:"account_number"`
	Description   string `json:"description"`
	Alignment     string `json:"alignment"`
	Amount        int64  `json:"amount"`
}

// CreateJournal creates a journal
func CreateJournal(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "CreateJournal")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	reqBod := &CreateJournalRequest{}
	bodBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error when reading body", err.Error(), 0)
		return
	}
	err = json.Unmarshal(bodBytes, &reqBod)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "malformed json body", err.Error(), 0)
		return
	}

	journal := &acccore.BaseJournal{
		JournalID:       UniqueIDGenerator.NewUniqueID(),
		JournalingTime:  time.Now(),
		Description:     reqBod.Description,
		Reversal:        false,
		ReversedJournal: nil,
		Amount:          0,
		Transactions:    make([]acccore.Transaction, 0),
		CreateTime:      time.Now(),
		CreatedBy:       reqBod.Creator,
	}

	for _, tx := range reqBod.Transactions {
		ntx := &acccore.BaseTransaction{
			TransactionID:   UniqueIDGenerator.NewUniqueID(),
			TransactionTime: time.Now(),
			AccountNumber:   tx.AccountNumber,
			JournalID:       journal.JournalID,
			Description:     tx.Description,
			Amount:          tx.Amount,
			AccountBalance:  0,
			CreateTime:      time.Now(),
			CreateBy:        reqBod.Creator,
		}
		if strings.ToUpper(tx.Alignment) == "DEBIT" {
			ntx.TransactionType = acccore.DEBIT
		} else {
			ntx.TransactionType = acccore.CREDIT
		}
		journal.Transactions = append(journal.Transactions, ntx)
	}

	journalContext := context.WithValue(r.Context(), contextkeys.UserIDContextKey, reqBod.Creator)

	err = JournalMgr.PersistJournal(journalContext, journal)
	if err != nil {
		helpers.HTTPResponseBuilder(journalContext, w, r, 400, "malformed json body", err.Error(), 0)
		return
	}
	helpers.HTTPResponseBuilder(journalContext, w, r, 200, "OK", journal.JournalID, 0)

}

// CreateReversalJournal creates a reversal journal response
func CreateReversalJournal(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "CreateReversalJournal")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	byteBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error when reading body", err.Error(), 0)
		return
	}

	rBody := &CreateReversalRequest{}
	err = json.Unmarshal(byteBody, &rBody)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "invalid request body", err.Error(), 0)
		return
	}

	rJournal, err := JournalMgr.GetJournalByID(r.Context(), rBody.JournalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrJournalIDNotFound) {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "journal not found", "journal to reverse not found", 0)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error when fetching journal", err.Error(), 0)
		return
	}
	if rJournal == nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "journal not found", "journal to reverse not found", 0)
		return
	}

	journal := &acccore.BaseJournal{
		JournalID:       UniqueIDGenerator.NewUniqueID(),
		JournalingTime:  time.Now(),
		Reversal:        true,
		ReversedJournal: rJournal,
		Description:     rBody.Description,
		CreatedBy:       rBody.Creator,
		CreateTime:      time.Now(),
	}

	transacs := make([]acccore.Transaction, 0)

	// make sure all Transactions have accounts of the same Currency
	for _, txinfo := range rJournal.GetTransactions() {
		tx := acccore.DEBIT
		if txinfo.GetAlignment() == acccore.DEBIT {
			tx = acccore.CREDIT
		}

		newTransaction := &acccore.BaseTransaction{
			TransactionID:   UniqueIDGenerator.NewUniqueID(),
			TransactionTime: time.Time{},
			AccountNumber:   txinfo.GetAccountNumber(),
			JournalID:       journal.JournalID,
			Description:     fmt.Sprintf("%s - reversed", txinfo.GetDescription()),
			TransactionType: tx,
			Amount:          txinfo.GetAmount(),
			AccountBalance:  txinfo.GetAccountBalance(),
			CreateTime:      time.Now(),
			CreateBy:        rBody.Creator,
		}
		transacs = append(transacs, newTransaction)
	}

	journal.SetTransactions(transacs)
	journalContext := context.WithValue(r.Context(), contextkeys.UserIDContextKey, rBody.Creator)

	err = JournalMgr.PersistJournal(journalContext, journal)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error when reversing journal", err.Error(), 0)
		return
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", journal.JournalID, 0)
}

// GetTransaction retrieves a transaction from its ID
func GetTransaction(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "GetTransaction")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/transactions/{TransactionID}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/transactions/{TransactionID}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	tx, err := TransactionMgr.GetTransactionByID(r.Context(), m["TransactionID"])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrTransactionNotFound) {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "transaction id not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 0)
		return
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", tx, 0)
}

// SetCommonDenominator sets the common denominator
func SetCommonDenominator(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "SetCommonDenominator")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	denomArr, ok := r.URL.Query()["denom"]
	if !ok {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "Malformed request", "missing denom", 0)
		return
	}

	f, err := strconv.ParseFloat(denomArr[0], 64)
	if err != nil {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "Malformed request", "denom must be a number (could be float)", 0)
		return
	}
	ExchangeMgr.SetDenom(r.Context(), big.NewFloat(f))
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", f, 0)
}

// GetCommonDenominator returns the current common denominator
func GetCommonDenominator(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "GetCommonDenominator")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	bf := ExchangeMgr.GetDenom(r.Context())
	f, _ := bf.Float64()
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", f, 0)
}

// SetCurrency sets the currency details
func SetCurrency(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "SetCurrency")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/currencies/{code}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/currencies/{code}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	bodyByte, err := ioutil.ReadAll(r.Body)
	if err != nil {
		llog.Errorf("error while reading body. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	setBody := &SetCurrencyBody{}
	err = json.Unmarshal(bodyByte, &setBody)
	if err != nil {
		llog.Errorf("error while parsing json body. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "malformed json", err.Error(), 1)
		return
	}

	createNew := false

	cur, err := ExchangeMgr.GetCurrency(r.Context(), m["code"])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrCurrencyNotFound) {
			createNew = true
		} else {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
			return
		}
	}
	if cur == nil {
		createNew = true
	}

	if createNew {
		nCur, err := ExchangeMgr.CreateCurrency(r.Context(), m["code"], setBody.Name, big.NewFloat(setBody.Exchange), setBody.Author)
		if err != nil {
			if err == acccore.ErrCurrencyAlreadyPersisted {
				helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "malformed request", "currnecy already persist", 1)
				return
			}
			helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", &CurrencyRet{
			Code:     nCur.GetCode(),
			Name:     nCur.GetName(),
			Exchange: nCur.GetExchange(),
		}, 1)
		return
	}
	cur.SetExchange(setBody.Exchange).SetName(setBody.Name)
	err = ExchangeMgr.UpdateCurrency(r.Context(), m["code"], cur, setBody.Author)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, acccore.ErrCurrencyNotFound) || err.Error() == "currency not found" {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "currnecy not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", &CurrencyRet{
		Code:     cur.GetCode(),
		Name:     cur.GetName(),
		Exchange: cur.GetExchange(),
	}, 1)

}

// SetCurrencyBody is the set currency request payload
type SetCurrencyBody struct {
	Name     string  `json:"name"`
	Exchange float64 `json:"exchange"`
	Author   string  `json:"author"`
}

// CurrencyRet is the currency respose
type CurrencyRet struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Exchange float64 `json:"exchange"`
}

// ListCurrencies lists all the currency
func ListCurrencies(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "ListCurrencies")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}
	curs, err := ExchangeMgr.ListCurrencies(r.Context())
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", make([]string, 0), 0)
			return
		}
	}
	arr := make([]*CurrencyRet, 0)
	for _, c := range curs {
		arr = append(arr, &CurrencyRet{
			Code:     c.GetCode(),
			Name:     c.GetName(),
			Exchange: c.GetExchange(),
		})
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", arr, 0)
}

// GetCurrency gets the currency details
func GetCurrency(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "GetCurrency")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/currencies/{code}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/currencies/{code}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	curCode, err := ExchangeMgr.GetCurrency(r.Context(), m["code"])
	if err != nil {
		if err == sql.ErrNoRows || err == acccore.ErrCurrencyNotFound {
			llog.Errorf("error while processing path template /api/v1/currencies/{code}. got : %s", err.Error())
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "currency not found", 1)
			return
		}
	}

	cret := &CurrencyRet{
		Code:     curCode.GetCode(),
		Name:     curCode.GetName(),
		Exchange: curCode.GetExchange(),
	}

	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", cret, 0)
}

// CalculateExchangeRate calculates the exchange rate
func CalculateExchangeRate(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "CalculateExchangeRate")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/exchange/{codefrom}/{codeto}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/exchange/{codefrom}/{codeto}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}

	cFrom, fok := m["codefrom"]
	cTo, tok := m["codeto"]
	if !fok || !tok {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "path not valid", "path not valid", 1)
		return
	}

	exc, err := ExchangeMgr.CalculateExchangeRate(r.Context(), cFrom, cTo)
	if err != nil {
		if err == acccore.ErrCurrencyNotFound {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "currency not found", "currency not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	f, _ := exc.Float64()
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", f, 1)
}

// CalculateExchange calculates the exchange betwee two currencies
func CalculateExchange(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(contextkeys.XRequestID).(string)
	llog := restLog.WithField("RequestID", requestID).WithField("function", "CalculateExchange")
	if r.Context().Err() != nil {
		llog.Errorf("context is canceled : %s", r.Context().Err().Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "request is canceled", "request is canceled", 0)
		return
	}

	m, err := helpers.ParsePathParams("/api/v1/exchange/{codefrom}/{codeto}/{amount}", r.URL.Path)
	if err != nil {
		llog.Errorf("error while processing path template /api/v1/exchange/{codefrom}/{codeto}/{amount}. got : %s", err.Error())
		helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "path not found", 1)
		return
	}
	cFrom, fok := m["codefrom"]
	cTo, tok := m["codeto"]
	cAmt, aok := m["amount"]
	if !fok || !tok || !aok {
		helpers.HTTPResponseBuilder(r.Context(), w, r, 400, "path not valid", "path not valid", 1)
		return
	}

	amnt, err := strconv.Atoi(cAmt)
	if err != nil {
		llog.Error("error, couldn't convert the amount: ", cAmt)
		helpers.HTTPResponseBuilder(r.Context(), w, r, http.StatusBadRequest, "path not valid", "path not valid", 400)
		return
	}

	res, err := ExchangeMgr.CalculateExchange(r.Context(), cFrom, cTo, int64(amnt))
	if err != nil {
		if err == sql.ErrNoRows || err == acccore.ErrCurrencyNotFound {
			helpers.HTTPResponseBuilder(r.Context(), w, r, 404, "path not found", "currency not found", 1)
			return
		}
		helpers.HTTPResponseBuilder(r.Context(), w, r, 500, "internal server error", err.Error(), 1)
		return
	}
	helpers.HTTPResponseBuilder(r.Context(), w, r, 200, "OK", res, 0)
}
