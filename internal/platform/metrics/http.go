package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics collects basic HTTP request metrics.
type HTTPMetrics struct {
	registry *prometheus.Registry

	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
}

// NewHTTPMetrics creates a new HTTPMetrics with its own registry.
func NewHTTPMetrics() *HTTPMetrics {
	reg := prometheus.NewRegistry()

	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	latency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	reg.MustRegister(requests, latency)

	return &HTTPMetrics{
		registry: reg,
		requests: requests,
		latency:  latency,
	}
}

// Middleware records request count and duration.
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	if m == nil || m.requests == nil || m.latency == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		status := strconv.Itoa(ww.status)
		path := r.URL.Path
		method := r.Method
		m.requests.WithLabelValues(method, path, status).Inc()
		m.latency.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	})
}

// Handler returns a Prometheus handler that serves metrics.
func (m *HTTPMetrics) Handler() http.Handler {
	if m == nil || m.registry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
