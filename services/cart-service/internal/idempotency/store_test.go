package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewMemoryStore(ctx)

	existing, err := s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, existing, "first reservation owns the key")

	existing, err = s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.False(t, existing.Completed, "second caller sees the in-flight reservation")

	require.NoError(t, s.Complete(ctx, "k", &Record{Fingerprint: "fp", StatusCode: 201, Body: []byte("{}")}, time.Hour))
	existing, err = s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.True(t, existing.Completed)
	assert.Equal(t, 201, existing.StatusCode)

	require.NoError(t, s.Release(ctx, "k"))
	existing, err = s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, existing, "released keys can be reserved again")
}

func TestMemoryStore_ExpiredReservationCanBeReclaimed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewMemoryStore(ctx)
	now := time.Now()
	s.now = func() time.Time { return now }

	_, err := s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)

	now = now.Add(2 * time.Minute)
	existing, err := s.Reserve(ctx, "k", "fp", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, existing)
}
