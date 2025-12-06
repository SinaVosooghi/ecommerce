// Package features provides feature flag functionality for the cart service.
package features

import (
	"context"
	"sync"
)

// Flags defines the interface for feature flag evaluation.
type Flags interface {
	IsEnabled(ctx context.Context, flag string, userID string) bool
	GetVariant(ctx context.Context, flag string, userID string) string
	Close() error
}

// Known feature flags
const (
	FlagNewPricingEngine      = "cart.new_pricing_engine"
	FlagExpressCheckout       = "cart.express_checkout"
	FlagRecommendationWidget  = "cart.recommendation_widget"
	FlagOptimisticLocking     = "cart.optimistic_locking"
	FlagEventPublishing       = "cart.event_publishing"
)

// InMemoryFlags is an in-memory implementation for testing.
type InMemoryFlags struct {
	flags    map[string]bool
	variants map[string]string
	mu       sync.RWMutex
}

// NewInMemoryFlags creates a new in-memory feature flags instance.
func NewInMemoryFlags() *InMemoryFlags {
	return &InMemoryFlags{
		flags:    make(map[string]bool),
		variants: make(map[string]string),
	}
}

// IsEnabled checks if a feature flag is enabled.
func (f *InMemoryFlags) IsEnabled(ctx context.Context, flag string, userID string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.flags[flag]
}

// GetVariant returns the variant for a feature flag.
func (f *InMemoryFlags) GetVariant(ctx context.Context, flag string, userID string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.variants[flag]
}

// SetFlag sets a feature flag value (for testing).
func (f *InMemoryFlags) SetFlag(flag string, enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flags[flag] = enabled
}

// SetVariant sets a variant value (for testing).
func (f *InMemoryFlags) SetVariant(flag, variant string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variants[flag] = variant
}

// Close closes the feature flags instance.
func (f *InMemoryFlags) Close() error {
	return nil
}

// StaticFlags provides static feature flags from configuration.
type StaticFlags struct {
	flags    map[string]bool
	variants map[string]string
}

// NewStaticFlags creates a new static feature flags instance.
func NewStaticFlags(flags map[string]bool, variants map[string]string) *StaticFlags {
	if flags == nil {
		flags = make(map[string]bool)
	}
	if variants == nil {
		variants = make(map[string]string)
	}
	return &StaticFlags{
		flags:    flags,
		variants: variants,
	}
}

// IsEnabled checks if a feature flag is enabled.
func (f *StaticFlags) IsEnabled(ctx context.Context, flag string, userID string) bool {
	return f.flags[flag]
}

// GetVariant returns the variant for a feature flag.
func (f *StaticFlags) GetVariant(ctx context.Context, flag string, userID string) string {
	return f.variants[flag]
}

// Close closes the feature flags instance.
func (f *StaticFlags) Close() error {
	return nil
}

// PercentageFlags provides percentage-based rollout.
type PercentageFlags struct {
	percentages map[string]int // 0-100
	mu          sync.RWMutex
}

// NewPercentageFlags creates a new percentage-based feature flags instance.
func NewPercentageFlags(percentages map[string]int) *PercentageFlags {
	if percentages == nil {
		percentages = make(map[string]int)
	}
	return &PercentageFlags{
		percentages: percentages,
	}
}

// IsEnabled checks if a feature flag is enabled for a user.
func (f *PercentageFlags) IsEnabled(ctx context.Context, flag string, userID string) bool {
	f.mu.RLock()
	percentage, ok := f.percentages[flag]
	f.mu.RUnlock()

	if !ok {
		return false
	}

	// Use hash of userID for consistent bucketing
	hash := hashString(userID + flag)
	bucket := int(hash % 100)
	return bucket < percentage
}

// GetVariant returns empty string (percentage flags don't support variants).
func (f *PercentageFlags) GetVariant(ctx context.Context, flag string, userID string) string {
	return ""
}

// SetPercentage sets the rollout percentage for a flag.
func (f *PercentageFlags) SetPercentage(flag string, percentage int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	f.percentages[flag] = percentage
}

// Close closes the feature flags instance.
func (f *PercentageFlags) Close() error {
	return nil
}

// hashString returns a simple hash of a string.
func hashString(s string) uint32 {
	var hash uint32 = 5381
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}
