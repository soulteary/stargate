package main

import (
	"context"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	health "github.com/soulteary/health-kit/v4"
	"github.com/soulteary/stargate/src/internal/handlers"
	"github.com/stretchr/testify/require"
)

// countingAggregator returns an aggregator with one dependency check that
// counts its runs and reports the status held in healthy.
func countingAggregator(cfg health.Config, runs *atomic.Int64, healthy *atomic.Bool) *health.Aggregator {
	aggregator := health.NewAggregator(cfg)
	aggregator.AddChecker(health.NewCheckerFunc("dependency", func(context.Context) health.CheckResult {
		runs.Add(1)
		status := health.StatusUnhealthy
		if healthy.Load() {
			status = health.StatusHealthy
		}
		return health.CheckResult{Name: "dependency", Status: status}
	}))
	return aggregator
}

// Requests to /readyz and /health must not each reach the dependencies.
func TestReadinessRoutesShareOneDependencyCheck(t *testing.T) {
	initLogger()
	setupTestConfig(t)

	var runs atomic.Int64
	var healthy atomic.Bool
	healthy.Store(true)

	app := fiber.New()
	store, _ := setupSessionStore()
	aggregator := countingAggregator(health.DefaultConfig().WithServiceName("stargate"), &runs, &healthy)
	setupRoutes(app, store, aggregator, handlers.NewSessionExchangeReplayStore(nil))

	for i := 0; i < 10; i++ {
		for _, route := range []string{RouteReadyz, RouteHealth} {
			resp, err := app.Test(httptest.NewRequest("GET", route, nil))
			require.NoError(t, err)
			require.Equal(t, fiber.StatusOK, resp.StatusCode)
		}
	}
	require.EqualValues(t, 1, runs.Load())
}

func TestReadinessCacheRefreshesAfterTTL(t *testing.T) {
	var runs atomic.Int64
	var healthy atomic.Bool
	healthy.Store(true)

	now := time.Unix(1_700_000_000, 0)
	cache := newReadinessCache(countingAggregator(health.DefaultConfig().WithServiceName("stargate"), &runs, &healthy), 5*time.Second)
	cache.now = func() time.Time { return now }

	app := fiber.New()
	app.Get("/readyz", cache.handler())
	status := func() int {
		resp, err := app.Test(httptest.NewRequest("GET", "/readyz", nil))
		require.NoError(t, err)
		return resp.StatusCode
	}

	require.Equal(t, fiber.StatusOK, status())

	// A dependency that fails inside the window is reported after it, and the
	// failure is cached like a success.
	healthy.Store(false)
	now = now.Add(5*time.Second - time.Nanosecond)
	require.Equal(t, fiber.StatusOK, status())
	require.EqualValues(t, 1, runs.Load())

	now = now.Add(time.Nanosecond)
	require.Equal(t, fiber.StatusServiceUnavailable, status())
	require.Equal(t, fiber.StatusServiceUnavailable, status())
	require.EqualValues(t, 2, runs.Load())

	healthy.Store(true)
	now = now.Add(5 * time.Second)
	require.Equal(t, fiber.StatusOK, status())
	require.EqualValues(t, 3, runs.Load())
}

func TestReadinessCacheSharesOneRunAcrossConcurrentRequests(t *testing.T) {
	var runs atomic.Int64
	release := make(chan struct{})
	aggregator := health.NewAggregator(health.DefaultConfig().WithServiceName("stargate"))
	aggregator.AddChecker(health.NewCheckerFunc("dependency", func(context.Context) health.CheckResult {
		runs.Add(1)
		<-release
		return health.CheckResult{Name: "dependency", Status: health.StatusHealthy}
	}))

	app := fiber.New()
	app.Get("/readyz", newReadinessCache(aggregator, time.Minute).handler())

	const clients = 8
	statuses := make([]int, clients)
	var wg sync.WaitGroup
	for i := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := app.Test(httptest.NewRequest("GET", "/readyz", nil), fiber.TestConfig{Timeout: 10 * time.Second})
			if err == nil {
				statuses[i] = resp.StatusCode
			}
		}()
	}
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 5*time.Second, time.Millisecond)
	close(release)
	wg.Wait()

	require.EqualValues(t, 1, runs.Load())
	for _, status := range statuses {
		require.Equal(t, fiber.StatusOK, status)
	}
}

// A refusal by address is decided per client and must never be served to, or
// replace the result for, another client.
func TestReadinessCacheDoesNotShareForbidden(t *testing.T) {
	var runs atomic.Int64
	var healthy atomic.Bool
	healthy.Store(true)

	cfg := health.DefaultConfig().WithServiceName("stargate").WithIPWhitelist([]string{"203.0.113.9"})
	app := fiber.New()
	app.Get("/readyz", newReadinessCache(countingAggregator(cfg, &runs, &healthy), time.Minute).handler())

	for i := 0; i < 2; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/readyz", nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	}
	require.Zero(t, runs.Load())
}

// A result evaluated for an allowed address must not be served to an address
// the allow list refuses while it is still cached.
func TestReadinessCacheChecksAddressBeforeServingCachedResult(t *testing.T) {
	var runs atomic.Int64
	var healthy atomic.Bool
	healthy.Store(true)

	cfg := health.DefaultConfig().
		WithServiceName("stargate").
		WithTrustedProxies([]string{"0.0.0.0"}).
		WithIPWhitelist([]string{"203.0.113.9"})
	app := fiber.New()
	app.Get("/readyz", newReadinessCache(countingAggregator(cfg, &runs, &healthy), time.Minute).handler())

	request := func(forwardedFor string) int {
		req := httptest.NewRequest("GET", "/readyz", nil)
		req.Header.Set("X-Forwarded-For", forwardedFor)
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp.StatusCode
	}

	require.Equal(t, fiber.StatusOK, request("203.0.113.9"))
	require.Equal(t, fiber.StatusForbidden, request("198.51.100.7"))
	require.Equal(t, fiber.StatusOK, request("203.0.113.9"))
	require.EqualValues(t, 1, runs.Load())
}
