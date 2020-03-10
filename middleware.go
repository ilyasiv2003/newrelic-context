package nrcontext

import (
	"fmt"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type TxnNameFunc func(*http.Request) string

type NewRelicMiddleware struct {
	app      *newrelic.Application
	nameFunc TxnNameFunc
}

// Creates new middleware that will report time in NewRelic and set transaction in context
func NewMiddleware(appname string, license string) (*NewRelicMiddleware, error) {
	app, err := newrelic.NewApplication(newrelic.ConfigAppName(appname), newrelic.ConfigLicense(license))

	if err != nil {
		return nil, err
	}
	return &NewRelicMiddleware{app, makeTransactionName}, nil
}

// Same as NewMiddleware but accepts newrelic.Config
func NewMiddlewareWithConfig(opts newrelic.ConfigOption) (*NewRelicMiddleware, error) {
	app, err := newrelic.NewApplication(opts)
	if err != nil {
		return nil, err
	}
	return &NewRelicMiddleware{app, makeTransactionName}, nil
}

// Same as NewMiddleware but accepts newrelic.Application
func NewMiddlewareWithApp(app *newrelic.Application) *NewRelicMiddleware {
	return &NewRelicMiddleware{app, makeTransactionName}
}

// Allows to change transaction name. By default `fmt.Sprintf("%s %s", r.Method, r.URL.Path)`
func (nr *NewRelicMiddleware) SetTxnNameFunc(fn TxnNameFunc) {
	nr.nameFunc = fn
}

// Creates wrapper for your handler
func (nr *NewRelicMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := nr.app.StartTransaction(nr.nameFunc(r))
		defer txn.End()

		w = txn.SetWebResponse(w)
		txn.SetWebRequestHTTP(r)
		r = r.WithContext(ContextWithTxn(r.Context(), txn))
		h.ServeHTTP(w, r)
	})
}

func makeTransactionName(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}
