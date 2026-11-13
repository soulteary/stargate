package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

// passwordHeaderCheckStatus sends one /_auth request with the given
// Stargate-Password value through CheckRoute and returns the status.
func passwordHeaderCheckStatus(t *testing.T, handler func(fiber.Ctx) error, password string) int {
	t.Helper()
	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
		"Stargate-Password": password,
		"Accept":            "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, handler(ctx))
	return ctx.Response().StatusCode()
}

func setupPasswordHeaderLimitTest(t *testing.T) (func(fiber.Ctx) error, int) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	require.NoError(t, config.Initialize(testLogger()))
	InitForwardAuthHandler(testLogger())

	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	require.Positive(t, max)
	return CheckRoute(setupTestStore()), max
}

// TestCheckRoute_WrongPasswordHeaderIsRateLimited drives wrong
// Stargate-Password values through the real /_auth handler (CheckRoute)
// without a session, the way a brute-force client would. The session checker
// in forwardauth-kit replaces the password error, so the limiter cannot rely
// on seeing forwardauth.ErrInvalidPassword.
func TestCheckRoute_WrongPasswordHeaderIsRateLimited(t *testing.T) {
	handler, max := setupPasswordHeaderLimitTest(t)

	for attempt := 1; attempt <= max; attempt++ {
		require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"), "attempt %d", attempt)
	}
	for attempt := max + 1; attempt <= max+5; attempt++ {
		require.Equal(t, fiber.StatusTooManyRequests, passwordHeaderCheckStatus(t, handler, "wrong-guess"), "attempt %d", attempt)
	}
}

// Once the quota is used up the password is no longer evaluated, so a correct
// guess is refused like a wrong one and the limit cannot be used as an oracle.
func TestCheckRoute_CorrectPasswordHeaderIsRefusedAfterQuota(t *testing.T) {
	handler, max := setupPasswordHeaderLimitTest(t)

	for attempt := 1; attempt <= max; attempt++ {
		require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"))
	}

	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
		"Stargate-Password": "test123",
		"Accept":            "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, handler(ctx))
	require.Equal(t, fiber.StatusTooManyRequests, ctx.Response().StatusCode())
	require.NotEmpty(t, ctx.GetRespHeader(fiber.HeaderRetryAfter))
	require.Empty(t, ctx.GetRespHeader(xRateLimitLimit))
}

// Only failures are counted: a client that always sends the right password is
// never limited, and its successes do not use up the quota of a later mistake.
func TestCheckRoute_CorrectPasswordHeaderIsNotCounted(t *testing.T) {
	handler, max := setupPasswordHeaderLimitTest(t)

	for attempt := 1; attempt <= max+5; attempt++ {
		require.Equal(t, fiber.StatusOK, passwordHeaderCheckStatus(t, handler, "test123"), "attempt %d", attempt)
	}
	require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"))
	require.Equal(t, fiber.StatusOK, passwordHeaderCheckStatus(t, handler, "test123"))
}

// A wrong password that falls back to an authenticated session is still a
// failed guess, even though the request itself is allowed.
func TestCheckRoute_WrongPasswordHeaderWithSessionIsCounted(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	require.NoError(t, config.Initialize(testLogger()))
	InitForwardAuthHandler(testLogger())
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)

	store := setupTestStore()
	handler := CheckRoute(store)
	request := func() int {
		ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
			"Stargate-Password": "wrong-guess",
			"Accept":            "application/json",
		}, "")
		sess, err := store.Get(ctx)
		require.NoError(t, err)
		require.NoError(t, auth.Authenticate(sess))
		ctx = rebindSessionContext(t, app, ctx, sess)
		defer app.ReleaseCtx(ctx)
		require.NoError(t, handler(ctx))
		return ctx.Response().StatusCode()
	}

	for attempt := 1; attempt <= max; attempt++ {
		require.Equal(t, fiber.StatusOK, request(), "attempt %d", attempt)
	}
	require.Equal(t, fiber.StatusTooManyRequests, request())
}

func TestCheckRoute_PasswordHeaderLimitDisabledByZero(t *testing.T) {
	handler, _ := setupPasswordHeaderLimitTest(t)
	withRateLimitConfig(t, "0", "5", "1m")

	for attempt := 1; attempt <= 25; attempt++ {
		require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"), "attempt %d", attempt)
	}
}

func TestMemoryRateLimitStoreCountDoesNotRecord(t *testing.T) {
	store := newMemoryRateLimitStore()
	ctx := context.Background()

	count, remaining, err := store.Count(ctx, "client")
	require.NoError(t, err)
	require.Zero(t, count)
	require.Zero(t, remaining)

	_, _, err = store.Incr(ctx, "client", time.Minute)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		count, remaining, err = store.Count(ctx, "client")
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Positive(t, remaining)
	}

	_, _, err = store.Incr(ctx, "expired", time.Nanosecond)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	count, _, err = store.Count(ctx, "expired")
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestRedisRateLimitStoreCountIsSharedAndDoesNotRecord(t *testing.T) {
	server := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: server.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = clientA.Close()
		_ = clientB.Close()
	})
	first, second := NewRateLimitStore(clientA), NewRateLimitStore(clientB)
	ctx := context.Background()

	count, remaining, err := second.Count(ctx, "client")
	require.NoError(t, err)
	require.Zero(t, count)
	require.Zero(t, remaining)

	_, _, err = first.Incr(ctx, "client", time.Minute)
	require.NoError(t, err)
	_, _, err = first.Incr(ctx, "client", time.Minute)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		count, remaining, err = second.Count(ctx, "client")
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Positive(t, remaining)
		require.LessOrEqual(t, remaining, time.Minute)
	}

	// A counter without an expiry is still reported; its window is unknown.
	require.NoError(t, server.Set(first.(*redisRateLimitStore).prefix+"persistent", "4"))
	count, remaining, err = second.Count(ctx, "persistent")
	require.NoError(t, err)
	require.Equal(t, 4, count)
	require.Zero(t, remaining)
}

// During a Redis outage Incr counts in the process-local store, so Count has
// to read the same place or the quota would never be reached.
func TestRedisRateLimitStoreCountFallsBackWhenRedisIsUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRateLimitStore(client)
	server.Close()
	ctx := context.Background()

	_, _, err := store.Incr(ctx, "client", time.Minute)
	require.Error(t, err)

	count, remaining, err := store.Count(ctx, "client")
	require.Error(t, err)
	require.Equal(t, 1, count)
	require.Positive(t, remaining)
}

func TestPasswordHeaderQuotaUsesLocalCountWhenRedisIsUnavailable(t *testing.T) {
	handler, _ := setupPasswordHeaderLimitTest(t)
	// Every command against the stopped server waits for the dial to fail, so
	// keep the quota small.
	const max = 2
	withRateLimitConfig(t, "2", "5", "1m")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	SetRateLimitStore(NewRateLimitStore(client))
	server.Close()

	for attempt := 1; attempt <= max; attempt++ {
		require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"), "attempt %d", attempt)
	}
	require.Equal(t, fiber.StatusTooManyRequests, passwordHeaderCheckStatus(t, handler, "test123"))
}
