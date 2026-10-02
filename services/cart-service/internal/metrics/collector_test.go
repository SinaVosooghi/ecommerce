package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeKey_IsDeterministic(t *testing.T) {
	labels := map[string]string{"method": "GET", "path": "/v1/cart/{userID}", "status_code": "200", "a": "1", "z": "2"}

	want := "requests:a=1:method=GET:path=/v1/cart/{userID}:status_code=200:z=2"
	for range 50 {
		assert.Equal(t, want, makeKey("requests", labels))
	}
}

func TestInMemoryCollector_CountsAcrossLabelOrder(t *testing.T) {
	c := NewInMemoryCollector()
	c.IncrementCounter("hits", map[string]string{"a": "1", "b": "2"})
	c.IncrementCounter("hits", map[string]string{"b": "2", "a": "1"})

	assert.Equal(t, float64(2), c.GetCounter("hits", map[string]string{"a": "1", "b": "2"}))
}
