package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
)

func withRateLimitConfig(t *testing.T, loginMax, verificationMax, window string) {
	t.Helper()
	previousLogin := config.RateLimitLoginMax.Value
	previousVerification := config.RateLimitVerificationMax.Value
	previousWindow := config.RateLimitWindow.Value
	config.RateLimitLoginMax.Value = loginMax
	config.RateLimitVerificationMax.Value = verificationMax
	config.RateLimitWindow.Value = window
	t.Cleanup(func() {
		config.RateLimitLoginMax.Value = previousLogin
		config.RateLimitVerificationMax.Value = previousVerification
		config.RateLimitWindow.Value = previousWindow
	})
}

func TestMemoryRateLimitStoreCountsWithinWindow(t *testing.T) {
	store := newMemoryRateLimitStore()
	ctx := context.Background()

	for expected := 1; expected <= 3; expected++ {
		count, retryAfter, err := store.Incr(ctx, "client", time.Minute)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, expected, count)
		testza.AssertTrue(t, retryAfter > 0)
	}

	// A separate key keeps its own quota.
	count, _, err := store.Incr(ctx, "other-client", time.Minute)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)
}

func TestMemoryRateLimitStoreResetsAfterWindow(t *testing.T) {
	store := newMemoryRateLimitStore()
	ctx := context.Background()

	count, _, err := store.Incr(ctx, "client", time.Millisecond)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)

	time.Sleep(5 * time.Millisecond)

	count, _, err = store.Incr(ctx, "client", time.Millisecond)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)
}

// A process-local counter gives every replica its own quota, which is exactly
// the dilution the shared store exists to prevent.
func TestRedisRateLimitStoreSharedAcrossReplicas(t *testing.T) {
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

	firstReplica := NewRateLimitStore(clientA)
	secondReplica := NewRateLimitStore(clientB)
	ctx := context.Background()

	count, _, err := firstReplica.Incr(ctx, "203.0.113.7:/_login", time.Minute)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)

	count, retryAfter, err := secondReplica.Incr(ctx, "203.0.113.7:/_login", time.Minute)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, count)
	testza.AssertTrue(t, retryAfter > 0)
	testza.AssertTrue(t, retryAfter <= time.Minute)
}

func TestRedisRateLimitStoreArmsWindowExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRateLimitStore(client)
	ctx := context.Background()

	count, _, err := store.Incr(ctx, "client", time.Minute)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)

	// The window must expire on its own, otherwise the counter would never
	// reset and the client would be locked out permanently.
	server.FastForward(61 * time.Second)

	count, _, err = store.Incr(ctx, "client", time.Minute)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, count)
}

// A Redis outage must neither remove the limit nor deny every request: the
// store degrades to per-replica counting and reports the error.
func TestRedisRateLimitStoreFallsBackWhenRedisIsUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRateLimitStore(client)
	server.Close()
	ctx := context.Background()

	count, retryAfter, err := store.Incr(ctx, "client", time.Minute)
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, 1, count)
	testza.AssertTrue(t, retryAfter > 0)

	count, _, err = store.Incr(ctx, "client", time.Minute)
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, 2, count)
}

func TestNewRateLimitStoreWithoutRedisStaysLocal(t *testing.T) {
	store := NewRateLimitStore(nil)
	_, isMemory := store.(*memoryRateLimitStore)
	testza.AssertTrue(t, isMemory)
}

func TestSetRateLimitStoreRejectsNil(t *testing.T) {
	t.Cleanup(resetRateLimitStateForTesting)

	SetRateLimitStore(nil)
	_, isMemory := getRateLimitStore().(*memoryRateLimitStore)
	testza.AssertTrue(t, isMemory)
}

func TestEndpointRateLimitHonoursConfiguredMaximum(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "2", "5", "1m")

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	}

	resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
	testza.AssertNotEqual(t, "", resp.Header.Get(fiber.HeaderRetryAfter))
}

func TestEndpointRateLimitDisabledByZero(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "0", "0", "1m")

	app := fiber.New()
	app.Post("/_send_verify_code", VerificationRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	for i := 0; i < 20; i++ {
		resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_send_verify_code"))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	}
}

// Two endpoints must not drain each other's quota.
func TestEndpointRateLimitIsScopedPerPath(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "1", "1", "1m")

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Post("/_send_verify_code", VerificationRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)

	resp, err = app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_send_verify_code"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
}

// The shared store is what makes a second replica reject a burst that the
// first replica already counted.
func TestLoginRateLimitSharedAcrossReplicas(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "2", "5", "1m")

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	SetRateLimitStore(NewRateLimitStore(client))

	newReplica := func() *fiber.App {
		app := fiber.New()
		app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusNoContent)
		})
		return app
	}

	firstReplica := newReplica()
	secondReplica := newReplica()

	for _, replica := range []*fiber.App{firstReplica, secondReplica} {
		resp, err := replica.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	}

	// The configured quota is two requests in total, not two per replica.
	resp, err := secondReplica.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

func TestWarnUntrustedForwardedHeadersDoesNotPanicWithoutLogger(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	previousLog := log
	log = nil
	t.Cleanup(func() { log = previousLog })

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := newRequestProtectionTestRequest(fiber.MethodPost, "/_login")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestRateLimitWindowFallsBackToDefault(t *testing.T) {
	withRateLimitConfig(t, "10", "5", "")
	testza.AssertEqual(t, config.DefaultRateLimitWindow, rateLimitWindow())

	withRateLimitConfig(t, "10", "5", "30s")
	testza.AssertEqual(t, 30*time.Second, rateLimitWindow())
}

func TestRetryAfterSecondsIsAtLeastOne(t *testing.T) {
	testza.AssertEqual(t, "1", retryAfterSeconds(0))
	testza.AssertEqual(t, "1", retryAfterSeconds(-time.Second))
	testza.AssertEqual(t, "30", retryAfterSeconds(30*time.Second))
}

func newForwardedRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, path, nil)
	req.Host = "auth.example.com"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	return req
}

// Without a trusted proxy, forwarded headers are ignored for attribution, so
// distinct clients behind one proxy still share a bucket. The warning is the
// only signal an operator gets, so the limit must still apply rather than
// silently letting the burst through.
func TestUntrustedForwardedClientsShareOneQuota(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "1", "1", "1m")

	previousProxies := config.TrustedProxies.Value
	config.TrustedProxies.Value = ""
	t.Cleanup(func() { config.TrustedProxies.Value = previousProxies })

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(newForwardedRequest(t, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)

	resp, err = app.Test(newForwardedRequest(t, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

// The Fiber limiter this package replaced advertised quota state on every
// response; clients that read those headers must keep working.
func TestEndpointRateLimitKeepsRateLimitHeaders(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "2", "5", "1m")

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "2", resp.Header.Get(xRateLimitLimit))
	testza.AssertEqual(t, "1", resp.Header.Get(xRateLimitRemaining))
	testza.AssertNotEqual(t, "", resp.Header.Get(xRateLimitReset))

	resp, err = app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "0", resp.Header.Get(xRateLimitRemaining))

	// Remaining saturates at zero rather than going negative once over quota.
	resp, err = app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
	testza.AssertEqual(t, "0", resp.Header.Get(xRateLimitRemaining))
	testza.AssertNotEqual(t, "", resp.Header.Get(fiber.HeaderRetryAfter))
}

// The forward-auth password path keeps its original response shape.
func TestPasswordHeaderFailureLimitDisabledByZero(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "0", "5", "1m")

	for i := 0; i < 20; i++ {
		ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
			"Stargate-Password": "wrong-password",
		}, "")
		err := rateLimitPasswordHeaderFailure(ctx)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, "", string(ctx.Response().Header.Peek(xRateLimitLimit)))
		app.ReleaseCtx(ctx)
	}
}
