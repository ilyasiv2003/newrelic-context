package nrcontext

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Gets NewRelic transaction from context and wraps client transport with newrelic RoundTripper
func WrapHTTPClient(ctx context.Context, c *http.Client, rq func() (*http.Request, error)) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}
	c.Transport = newrelic.NewRoundTripper(c.Transport)
	req, _ := rq()
	request := newrelic.RequestWithTransactionContext(req, txn)

	c.Do(request)
}
