package router

import (
	"net/http"
	"strings"

	"github.com/hyperjumptech/hyperwallet/internal/accounting"
	"github.com/hyperjumptech/hyperwallet/internal/middlewares"
	"github.com/hyperjumptech/hyperwallet/static"

	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/gorilla/mux"
	"github.com/hyperjumptech/hyperwallet/internal/health"
	log "github.com/sirupsen/logrus"
)

// Router is a wrapper for all the router connections
type Router struct {
	Router *mux.Router // point to mux routers
	// Handlers *handlers.Handlers // point to handlers
}

var rLog = log.WithField("module", "router")

// NewRouter get new Instance
func NewRouter() *Router {
	return &Router{}
}

// InitRoutes creates our routes
func InitRoutes(router *Router) {
	rLog.WithField("fn", "InitRoutes()").Info("Initializing routes...")

	r := router.Router

	// register middlewares
	// r.Use(apmgorilla.Middleware()) // apmgorilla.Instrument(r.MuxRouter) // elastic apm: DISABLED
	r.Use(middlewares.CORSMiddleware, middlewares.SetupContextMiddleware, middlewares.Logger, middlewares.HMACMiddleware) // your faithfull logger

	// health check endpoint. Not in a version path as it will seems to be a permanent endpoint (famous last words)
	r.HandleFunc("/health", healthhttp.HandleHealthJSON(health.H)).Methods("GET", "OPTIONS")
	r.HandleFunc("/devkey", middlewares.DevKey).Methods("PUT", "OPTIONS")

	r.HandleFunc("/api/v1/accounts/{AccountNumber}", accounting.GetAccount).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/accounts/{accountNumber}/draw", accounting.DrawAccount).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/accounts/{AccountNumber}/transactions", accounting.ListTransactionByAccount).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/accounts", accounting.FindAccount).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/accounts", accounting.CreateAccount).Methods("POST", "OPTIONS")

	r.HandleFunc("/api/v1/journals", accounting.CreateJournal).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/journals", accounting.ListJournal).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/journals/reversal", accounting.CreateReversalJournal).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/journals/{JournalID}", accounting.GetJournal).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/journals/{JournalID}/draw", accounting.DrawJournal).Methods("GET", "OPTIONS")

	r.HandleFunc("/api/v1/transactions/{TransactionID}", accounting.GetTransaction).Methods("GET", "OPTIONS")

	r.HandleFunc("/api/v1/exchange/denom", accounting.GetCommonDenominator).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/exchange/denom", accounting.SetCommonDenominator).Methods("PUT", "OPTIONS")

	r.HandleFunc("/api/v1/currencies", accounting.ListCurrencies).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/currencies/{code}", accounting.GetCurrency).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/currencies/{code}", accounting.SetCurrency).Methods("PUT", "OPTIONS")

	r.HandleFunc("/api/v1/exchange/{codefrom}/{codeto}", accounting.CalculateExchangeRate).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/exchange/{codefrom}/{codeto}/{amount}", accounting.CalculateExchange).Methods("GET", "OPTIONS")

	r.HandleFunc("/docs", StaticServer("")).Methods("GET")
	r.HandleFunc("/docs/", StaticServer("")).Methods("GET")

	r.HandleFunc("/dashboard", StaticServer("")).Methods("GET")
	r.HandleFunc("/dashboard/", StaticServer("")).Methods("GET")

	tree := static.GetPathTree("api")
	for _, t := range tree {
		pth := strings.ReplaceAll(t, "api/swagger", "/docs")
		if pth[:5] != "[DIR]" {
			r.HandleFunc(pth, StaticServer(t)).Methods("GET")
		}
	}

	tree = static.GetPathTree("dashboard")
	for _, t := range tree {
		pth := strings.ReplaceAll(t, "dashboard", "/dashboard")
		if pth[:5] != "[DIR]" {
			r.HandleFunc("/"+t, StaticServer(t)).Methods("GET")
		}
	}

	if log.GetLevel() == log.DebugLevel {
		walk(*r)
	}
}

// StaticServer is a http handler used to serve all static endpoints such as docs and dashboard
func StaticServer(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
			w.Header().Set("Location", "/docs/index.html")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		} else if r.URL.Path == "/dashboard" || r.URL.Path == "/dashboard/" {
			w.Header().Set("Location", "/dashboard/index.html")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		} else if strings.HasSuffix(r.URL.Path, "/") {
			w.Header().Set("Location", r.URL.Path+"index.html")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		var filePath string
		if strings.Contains(r.URL.Path, "/docs/") {
			filePath = strings.ReplaceAll(r.URL.Path, "/docs/", "api/swagger/")
		}
		if strings.Contains(r.URL.Path, "/dashboard/") {
			filePath = strings.ReplaceAll(r.URL.Path, "/dashboard/", "dashboard/")
		}
		dirFilePath := "[DIR]" + filePath

		if path == dirFilePath {
			w.Header().Set("Location", r.URL.Path+"/index.html")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		} else if path == filePath {
			fdata, err := static.GetFile(filePath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			w.Header().Set("Content-Type", fdata.ContentType)
			w.WriteHeader(http.StatusOK)
			w.Write(fdata.Bytes)
			return
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// walk runs the mux.Router.Walk method to print all the registered routes
func walk(r mux.Router) {
	log.Debugf("REGISTERED PATHS:")
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		var pathTemplates, methods string
		pathTemplates, err := route.GetPathTemplate()
		if err != nil {
			pathTemplates = "err"
		}
		methodArr, _ := route.GetMethods()
		methods = strings.Join(methodArr, ",")
		log.Debugf("    Path : %s. Methods : %s", pathTemplates, methods)
		return nil
	})
	if err != nil {
		log.Error(err)
	}
	log.Debugf("END REGISTERED PATHS")
}
