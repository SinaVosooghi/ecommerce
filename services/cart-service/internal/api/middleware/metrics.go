package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// MetricsCollector defines the interface for collecting metrics.
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
}

// Metrics provides request metrics collection middleware. It must be mounted on a chi
// router so the matched route pattern (not the raw URL) can be used as the path label;
// raw paths contain user and item IDs and would create unbounded label cardinality.
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
				"path":        routePattern(r),
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

// routePattern returns the matched chi route pattern, e.g. "/v1/cart/{userID}/items".
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
