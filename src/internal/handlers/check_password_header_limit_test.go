package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/secure-kit/v2/passwd"
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
	isolateGlobalState(t)
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
	isolateGlobalState(t)
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

// concurrentPasswordHeaderStatuses sends the given Stargate-Password values
// through CheckRoute at the same time and returns how often each status was
// answered.
func concurrentPasswordHeaderStatuses(handler func(fiber.Ctx) error, passwords []string) map[int]int {
	statuses := make(chan int, len(passwords))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, password := range passwords {
		wg.Add(1)
		go func(password string) {
			defer wg.Done()
			ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
				"Stargate-Password": password,
				"Accept":            "application/json",
			}, "")
			defer app.ReleaseCtx(ctx)
			<-start
			if err := handler(ctx); err != nil {
				statuses <- -1
				return
			}
			statuses <- ctx.Response().StatusCode()
		}(password)
	}
	close(start)
	wg.Wait()
	close(statuses)

	counts := make(map[int]int)
	for status := range statuses {
		counts[status]++
	}
	return counts
}

func setupBcryptPasswordHeaderLimitTest(t *testing.T) (func(fiber.Ctx) error, int) {
	t.Helper()
	isolateGlobalState(t)
	hash, err := passwd.NewBcryptHasher().Hash("test123")
	require.NoError(t, err)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "bcrypt:"+hash)
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	require.NoError(t, config.Initialize(testLogger()))
	InitForwardAuthHandler(testLogger())

	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	require.Positive(t, max)
	return CheckRoute(setupTestStore()), max
}

// Wrong values that arrive together must not all be compared. A bcrypt
// comparison takes long enough that, without counting the attempt first, every
// request in the burst has passed the quota check before the first failure is
// recorded.
func TestCheckRoute_ConcurrentWrongPasswordHeadersStayWithinQuota(t *testing.T) {
	handler, max := setupBcryptPasswordHeaderLimitTest(t)

	const burst = 60
	guesses := make([]string, burst)
	for i := range guesses {
		guesses[i] = "wrong-guess"
	}
	counts := concurrentPasswordHeaderStatuses(handler, guesses)

	require.Equal(t, max, counts[fiber.StatusUnauthorized], "compared attempts: %v", counts)
	require.Equal(t, burst-max, counts[fiber.StatusTooManyRequests], "refused attempts: %v", counts)

	// The compared attempts were failures and stay counted.
	require.Equal(t, fiber.StatusTooManyRequests, passwordHeaderCheckStatus(t, handler, "test123"))
}

// Correct values are counted only while they are compared. Up to the quota of
// them may run at the same time, and none of them uses up quota afterwards.
func TestCheckRoute_ConcurrentCorrectPasswordHeadersAreReleased(t *testing.T) {
	handler, max := setupBcryptPasswordHeaderLimitTest(t)

	correct := make([]string, max)
	for i := range correct {
		correct[i] = "test123"
	}
	for round := 1; round <= 3; round++ {
		counts := concurrentPasswordHeaderStatuses(handler, correct)
		require.Equal(t, max, counts[fiber.StatusOK], "round %d: %v", round, counts)
	}

	for attempt := 1; attempt <= max; attempt++ {
		require.Equal(t, fiber.StatusUnauthorized, passwordHeaderCheckStatus(t, handler, "wrong-guess"), "attempt %d", attempt)
	}
	require.Equal(t, fiber.StatusTooManyRequests, passwordHeaderCheckStatus(t, handler, "wrong-guess"))
}

// A refused attempt is taken back, so requests sent while the quota is used up
// do not push the counter further.
func TestCheckRoute_RefusedPasswordHeaderAttemptIsNotKept(t *testing.T) {
	handler, max := setupPasswordHeaderLimitTest(t)
	store := newMemoryRateLimitStore()
	SetRateLimitStore(store)

	for attempt := 1; attempt <= max+5; attempt++ {
		passwordHeaderCheckStatus(t, handler, "wrong-guess")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Len(t, store.buckets, 1)
	for _, bucket := range store.buckets {
		require.Equal(t, max, bucket.count)
	}
}

func TestMemoryRateLimitStoreReserveAndRelease(t *testing.T) {
	store := newMemoryRateLimitStore()
	ctx := context.Background()

	count, remaining, releaseFirst, err := store.Reserve(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Positive(t, remaining)

	count, _, _, err = store.Reserve(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	releaseFirst(ctx)
	count, _, err = store.Incr(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// A release that arrives after its window ended must not take a request
	// out of the next window.
	_, _, releaseExpired, err := store.Reserve(ctx, "expired", time.Nanosecond)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	count, _, err = store.Incr(ctx, "expired", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	releaseExpired(ctx)
	count, _, err = store.Incr(ctx, "expired", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestRedisRateLimitStoreReserveIsSharedAndReleased(t *testing.T) {
	server := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: server.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = clientA.Close()
		_ = clientB.Close()
	})
	first, second := NewRateLimitStore(clientA), NewRateLimitStore(clientB)
	ctx := context.Background()

	count, remaining, releaseFirst, err := first.Reserve(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Positive(t, remaining)
	require.LessOrEqual(t, remaining, time.Minute)

	count, _, _, err = second.Reserve(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	releaseFirst(ctx)
	count, _, err = second.Incr(ctx, "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// The release keeps the expiry of the window.
	key := first.(*redisRateLimitStore).prefix + "client"
	require.Positive(t, server.TTL(key))

	// A release after the window ended neither goes below zero nor leaves a
	// counter without an expiry behind.
	_, _, releaseExpired, err := first.Reserve(ctx, "expired", time.Minute)
	require.NoError(t, err)
	expiredKey := first.(*redisRateLimitStore).prefix + "expired"
	server.Del(expiredKey)
	releaseExpired(ctx)
	require.False(t, server.Exists(expiredKey))
}

// During a Redis outage the attempt is counted in the process-local store, so
// the release has to go to the same place.
func TestRedisRateLimitStoreReserveFallsBackWhenRedisIsUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRateLimitStore(client)
	server.Close()
	ctx := context.Background()

	count, remaining, release, err := store.Reserve(ctx, "client", time.Minute)
	require.Error(t, err)
	require.Equal(t, 1, count)
	require.Positive(t, remaining)

	count, _, _, err = store.Reserve(ctx, "client", time.Minute)
	require.Error(t, err)
	require.Equal(t, 2, count)

	release(ctx)
	count, _, err = store.Incr(ctx, "client", time.Minute)
	require.Error(t, err)
	require.Equal(t, 2, count)
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
