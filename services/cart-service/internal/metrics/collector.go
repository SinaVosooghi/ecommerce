// Package metrics provides observability instrumentation for the cart service.
package metrics

import (
	"sync"
)

// Collector defines the interface for metrics collection.
type Collector interface {
	IncrementCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// Metric types
const (
	// Request metrics
	MetricHTTPRequestsTotal          = "http_requests_total"
	MetricHTTPRequestDuration        = "http_request_duration_seconds"
	MetricHTTPRequestSize            = "http_request_size_bytes"
	MetricHTTPResponseSize           = "http_response_size_bytes"

	// Business metrics
	MetricCartOperationsTotal        = "cart_operations_total"
	MetricCartItemsTotal             = "cart_items_total"
	MetricCartValueDollars           = "cart_value_dollars"

	// Infrastructure metrics
	MetricPersistenceOperationsTotal = "persistence_operations_total"
	MetricPersistenceDuration        = "persistence_operation_duration_seconds"
	MetricEventPublishTotal          = "event_publish_total"
	MetricCircuitBreakerState        = "circuit_breaker_state"
)

// InMemoryCollector is an in-memory implementation of Collector for testing.
type InMemoryCollector struct {
	counters   map[string]float64
	histograms map[string][]float64
	gauges     map[string]float64
	mu         sync.RWMutex
}

// NewInMemoryCollector creates a new in-memory collector.
func NewInMemoryCollector() *InMemoryCollector {
	return &InMemoryCollector{
		counters:   make(map[string]float64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

// IncrementCounter increments a counter metric.
func (c *InMemoryCollector) IncrementCounter(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := makeKey(name, labels)
	c.counters[key]++
}

// ObserveHistogram records a histogram observation.
func (c *InMemoryCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := makeKey(name, labels)
	c.histograms[key] = append(c.histograms[key], value)
}

// SetGauge sets a gauge metric.
func (c *InMemoryCollector) SetGauge(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := makeKey(name, labels)
	c.gauges[key] = value
}

// GetCounter returns a counter value for testing.
func (c *InMemoryCollector) GetCounter(name string, labels map[string]string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := makeKey(name, labels)
	return c.counters[key]
}

// GetHistogram returns histogram values for testing.
func (c *InMemoryCollector) GetHistogram(name string, labels map[string]string) []float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := makeKey(name, labels)
	result := make([]float64, len(c.histograms[key]))
	copy(result, c.histograms[key])
	return result
}

// GetGauge returns a gauge value for testing.
func (c *InMemoryCollector) GetGauge(name string, labels map[string]string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := makeKey(name, labels)
	return c.gauges[key]
}

// Reset clears all metrics (for testing).
func (c *InMemoryCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters = make(map[string]float64)
	c.histograms = make(map[string][]float64)
	c.gauges = make(map[string]float64)
}

func makeKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ":" + k + "=" + v
	}
	return key
}

// NoOpCollector is a no-op implementation of Collector.
type NoOpCollector struct{}

// IncrementCounter does nothing.
func (n *NoOpCollector) IncrementCounter(name string, labels map[string]string) {}

// ObserveHistogram does nothing.
func (n *NoOpCollector) ObserveHistogram(name string, value float64, labels map[string]string) {}

// SetGauge does nothing.
func (n *NoOpCollector) SetGauge(name string, value float64, labels map[string]string) {}
