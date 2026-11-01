package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestChallengeContextRejectsCorruptOrUnavailableRedis(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	store := NewChallengeContextStore(client).(*redisChallengeContextStore)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, store.prefix+"corrupt", "not JSON", time.Minute).Err())
	_, found, err := store.Get(ctx, "corrupt")
	require.Error(t, err)
	require.False(t, found)
	server.Close()
	_, found, err = store.Get(ctx, "corrupt")
	require.Error(t, err)
	require.False(t, found)
	SetChallengeContextStore(nil)
	t.Cleanup(func() { SetChallengeContextStore(nil) })
	value, err := loadChallengeContext(ctx, "")
	require.NoError(t, err)
	require.Empty(t, value.ChallengeID)
	_, err = loadChallengeContext(ctx, "missing")
	require.NoError(t, err)
}

func TestChallengeContextExpiresBeforeCleanupCallback(t *testing.T) {
	store := newMemoryChallengeContextStore()
	ctx := context.Background()
	want := testChallengeContext()
	_, err := store.PutIfAbsent(ctx, want, time.Hour)
	require.NoError(t, err)
	// Model a delayed cleanup callback: expiry must still be enforced on read.
	store.mu.Lock()
	entry := store.entries[want.ChallengeID]
	entry.expiresAt = time.Now().Add(-time.Second)
	store.entries[want.ChallengeID] = entry
	store.mu.Unlock()
	_, found, err := store.Get(ctx, want.ChallengeID)
	require.NoError(t, err)
	require.False(t, found)
	_, err = store.PutIfAbsent(ctx, want, time.Hour)
	require.NoError(t, err)
	store.mu.Lock()
	entry = store.entries[want.ChallengeID]
	entry.expiresAt = time.Now().Add(-time.Second)
	store.entries[want.ChallengeID] = entry
	store.mu.Unlock()
	stored, err := store.PutIfAbsent(ctx, want, time.Hour)
	require.NoError(t, err)
	require.True(t, stored)
	require.NoError(t, store.Delete(ctx, want.ChallengeID))
}
