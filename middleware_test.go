package nrcontext

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func TestHandler(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		tnx := GetTnxFromContext(r.Context())
		if tnx == nil {
			t.Fatal("can't get tnx from context")
		}

		w.Write([]byte("Test response"))
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	app, _ := newrelic.NewApplication(
		newrelic.ConfigAppName("test"),
		newrelic.ConfigLicense(strings.Repeat("a", 40)),
	)
	nr := &NewRelicMiddleware{
		app:      app,
		nameFunc: func(r *http.Request) string { return r.URL.Path },
	}

	handler := nr.Handler(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("status code is wrong o_O")
	}
}
