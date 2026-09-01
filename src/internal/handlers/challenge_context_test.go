package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
)

func testChallengeContext() challengeContext {
	return challengeContext{
		ChallengeID: "challenge-1",
		UserID:      "user-1",
		Channel:     "email",
		Destination: "user@example.com",
	}
}

func TestMemoryChallengeContextLifecycle(t *testing.T) {
	store := newMemoryChallengeContextStore()
	ctx := context.Background()
	want := testChallengeContext()

	testza.AssertNoError(t, store.Put(ctx, want, 20*time.Millisecond))
	got, found, err := store.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, found)
	testza.AssertEqual(t, want, got)

	// Failed verification only reads the context, so subsequent attempts retain it.
	_, found, err = store.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, found)

	time.Sleep(30 * time.Millisecond)
	_, found, err = store.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, found)
}

func TestChallengeContextSuccessfulConsumption(t *testing.T) {
	store := newMemoryChallengeContextStore()
	ctx := context.Background()
	want := testChallengeContext()
	testza.AssertNoError(t, store.Put(ctx, want, time.Minute))
	testza.AssertNoError(t, store.Delete(ctx, want.ChallengeID))
	_, found, err := store.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, found)
}

func TestChallengeAuditContextIgnoresLoginDeliveryPreference(t *testing.T) {
	store := newMemoryChallengeContextStore()
	SetChallengeContextStore(store)
	t.Cleanup(func() { SetChallengeContextStore(nil) })
	ctx := context.Background()
	want := testChallengeContext()
	testza.AssertNoError(t, store.Put(ctx, want, time.Minute))

	// The login request may claim a different delivery preference, but audit
	// context is recovered solely by the server-issued challenge ID.
	clientDeliverVia := "sms"
	got, err := loadChallengeContext(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "sms", clientDeliverVia)
	testza.AssertEqual(t, "email", got.Channel)
	testza.AssertEqual(t, "user@example.com", got.Destination)
	testza.AssertEqual(t, "user-1", got.UserID)
	testza.AssertEqual(t, "challenge-1", got.ChallengeID)
}

func TestRedisChallengeContextSharedAcrossInstances(t *testing.T) {
	server := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: server.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = clientA.Close()
		_ = clientB.Close()
	})

	previousPrefix := config.SessionStorageRedisKeyPrefix.Value
	config.SessionStorageRedisKeyPrefix.Value = "stargate:test:"
	t.Cleanup(func() { config.SessionStorageRedisKeyPrefix.Value = previousPrefix })

	firstReplica := NewChallengeContextStore(clientA)
	secondReplica := NewChallengeContextStore(clientB)
	ctx := context.Background()
	want := testChallengeContext()
	testza.AssertNoError(t, firstReplica.Put(ctx, want, time.Minute))

	got, found, err := secondReplica.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, found)
	testza.AssertEqual(t, want, got)

	// A successful verification on another replica consumes the shared record.
	testza.AssertNoError(t, secondReplica.Delete(ctx, want.ChallengeID))
	_, found, err = firstReplica.Get(ctx, want.ChallengeID)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, found)
}
