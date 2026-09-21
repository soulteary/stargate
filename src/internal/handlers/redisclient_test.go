package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisClientMissing covers the distinction the plain nil check could not
// make: an interface holding a nil pointer is not a usable client.
func TestRedisClientMissing(t *testing.T) {
	var typedNil *redis.Client

	server := miniredis.RunT(t)
	live := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = live.Close() })

	testza.AssertTrue(t, redisClientMissing(nil),
		"an untyped nil is no client")
	testza.AssertTrue(t, redisClientMissing(typedNil),
		"a nil *redis.Client carries no client, whatever the interface header says")
	testza.AssertFalse(t, redisClientMissing(live),
		"a live client must not be mistaken for a missing one")
}

// TestRedisBackedStoresTreatTypedNilClientAsAbsent pins the nil handling of
// every store that takes an optional Redis client.
//
// setupSessionStore declares its client as *redis.Client and leaves it nil when
// session storage is disabled. Passing that to a redis.Cmdable parameter yields
// a non-nil interface holding a nil pointer, so a plain `client == nil` guard
// does not fire and the first command panics. The stores must fall back to
// their in-memory implementations instead.
func TestRedisBackedStoresTreatTypedNilClientAsAbsent(t *testing.T) {
	// Exactly what setupSessionStore returns when SESSION_STORAGE_ENABLED=false.
	var client *redis.Client

	t.Run("rate limit", func(t *testing.T) {
		store := NewRateLimitStore(client)
		count, retryAfter, err := store.Incr(context.Background(), "typed-nil", time.Minute)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, 1, count)
		testza.AssertTrue(t, retryAfter > 0)
	})

	t.Run("challenge context", func(t *testing.T) {
		store := NewChallengeContextStore(client)
		ctx := context.Background()
		stored, err := store.PutIfAbsent(ctx, challengeContext{
			ChallengeID: "typed-nil",
			UserID:      "u-1",
		}, time.Minute)
		testza.AssertNoError(t, err)
		testza.AssertTrue(t, stored)

		got, found, err := store.Get(ctx, "typed-nil")
		testza.AssertNoError(t, err)
		testza.AssertTrue(t, found)
		testza.AssertEqual(t, "u-1", got.UserID)
	})

	t.Run("session exchange replay", func(t *testing.T) {
		store := NewSessionExchangeReplayStore(client)
		first, err := store.Consume(context.Background(), "typed-nil", time.Minute)
		testza.AssertNoError(t, err)
		testza.AssertTrue(t, first, "a ticket should be consumable once")
	})
}
