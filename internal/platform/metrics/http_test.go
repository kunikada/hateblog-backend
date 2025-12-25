package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPMetrics_MiddlewareAndHandler(t *testing.T) {
	m := NewHTTPMetrics()

	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	srv := m.Middleware(base)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	metricsRec := httptest.NewRecorder()
	m.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "http://example.com/observability/metrics", nil))
	require.Equal(t, http.StatusOK, metricsRec.Code)

	body, err := io.ReadAll(metricsRec.Body)
	require.NoError(t, err)
	text := string(body)
	require.Contains(t, text, "http_requests_total")
	require.Contains(t, text, `method="GET",path="/test",status="201"`)
}
