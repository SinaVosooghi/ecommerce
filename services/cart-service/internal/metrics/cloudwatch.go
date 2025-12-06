package metrics

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// CloudWatchConfig holds CloudWatch EMF configuration.
type CloudWatchConfig struct {
	Namespace   string
	ServiceName string
	Environment string
}

// CloudWatchCollector implements CloudWatch Embedded Metric Format (EMF).
type CloudWatchCollector struct {
	namespace   string
	dimensions  map[string]string
	mu          sync.Mutex
}

// NewCloudWatchCollector creates a new CloudWatch EMF collector.
func NewCloudWatchCollector(cfg CloudWatchConfig) *CloudWatchCollector {
	return &CloudWatchCollector{
		namespace: cfg.Namespace,
		dimensions: map[string]string{
			"ServiceName": cfg.ServiceName,
			"Environment": cfg.Environment,
		},
	}
}

// EMFMetric represents a CloudWatch EMF metric.
type EMFMetric struct {
	AWS       EMFAWSBlock            `json:"_aws"`
	Metrics   map[string]interface{} `json:"-"`
	Timestamp int64                  `json:"Timestamp"`
}

// EMFAWSBlock represents the _aws block in EMF format.
type EMFAWSBlock struct {
	Timestamp         int64              `json:"Timestamp"`
	CloudWatchMetrics []CloudWatchMetric `json:"CloudWatchMetrics"`
}

// CloudWatchMetric represents a metric definition in EMF.
type CloudWatchMetric struct {
	Namespace  string           `json:"Namespace"`
	Dimensions [][]string       `json:"Dimensions"`
	Metrics    []MetricDefinition `json:"Metrics"`
}

// MetricDefinition defines a single metric.
type MetricDefinition struct {
	Name string `json:"Name"`
	Unit string `json:"Unit"`
}

// IncrementCounter increments a counter and outputs EMF.
func (c *CloudWatchCollector) IncrementCounter(name string, labels map[string]string) {
	c.emitMetric(name, 1, "Count", labels)
}

// ObserveHistogram records a histogram observation and outputs EMF.
func (c *CloudWatchCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	unit := "Seconds"
	if contains(name, "bytes") {
		unit = "Bytes"
	}
	c.emitMetric(name, value, unit, labels)
}

// SetGauge sets a gauge and outputs EMF.
func (c *CloudWatchCollector) SetGauge(name string, value float64, labels map[string]string) {
	c.emitMetric(name, value, "None", labels)
}

func (c *CloudWatchCollector) emitMetric(name string, value float64, unit string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixMilli()

	// Merge dimensions
	dimensions := make(map[string]string)
	for k, v := range c.dimensions {
		dimensions[k] = v
	}
	for k, v := range labels {
		dimensions[k] = v
	}

	// Build dimension keys
	dimensionKeys := make([]string, 0, len(dimensions))
	for k := range dimensions {
		dimensionKeys = append(dimensionKeys, k)
	}

	// Build EMF output
	emf := map[string]interface{}{
		"_aws": EMFAWSBlock{
			Timestamp: now,
			CloudWatchMetrics: []CloudWatchMetric{
				{
					Namespace:  c.namespace,
					Dimensions: [][]string{dimensionKeys},
					Metrics: []MetricDefinition{
						{Name: name, Unit: unit},
					},
				},
			},
		},
		name: value,
	}

	// Add dimension values
	for k, v := range dimensions {
		emf[k] = v
	}

	// Output as JSON to stdout (CloudWatch agent picks this up)
	output, _ := json.Marshal(emf)
	os.Stdout.Write(output)
	os.Stdout.Write([]byte("\n"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
