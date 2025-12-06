package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// MetricsCollector defines the interface for collecting metrics.
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
}

// Metrics provides request metrics collection middleware.
func Metrics(collector MetricsCollector) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			// Collect request metrics
			labels := map[string]string{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": strconv.Itoa(ww.Status()),
			}

			// Increment request counter
			collector.IncrementCounter("http_requests_total", labels)

			// Record request duration
			collector.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), labels)

			// Record request size
			if r.ContentLength > 0 {
				collector.ObserveHistogram("http_request_size_bytes", float64(r.ContentLength), labels)
			}

			// Record response size
			if ww.BytesWritten() > 0 {
				collector.ObserveHistogram("http_response_size_bytes", float64(ww.BytesWritten()), labels)
			}
		})
	}
}

// NoOpMetricsCollector is a no-op implementation of MetricsCollector.
type NoOpMetricsCollector struct{}

// IncrementCounter does nothing.
func (n *NoOpMetricsCollector) IncrementCounter(name string, labels map[string]string) {}

// ObserveHistogram does nothing.
func (n *NoOpMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {}
