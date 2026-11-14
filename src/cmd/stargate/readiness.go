package main

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	health "github.com/soulteary/health-kit/v4"
	healthfiber "github.com/soulteary/health-kit/v4/fiberadapter"
)

// readinessCacheTTL is how long one readiness result answers later requests.
// The readiness routes need no credentials and every evaluation calls Herald,
// Warden and Redis, so without a cache each request to them becomes three
// outbound requests.
const readinessCacheTTL = 5 * time.Second

// readinessCache evaluates the dependency checks at most once per TTL.
// Requests that arrive while an evaluation is running wait for it and share
// its result instead of starting their own.
//
// The cache holds one result for all clients. Stargate configures no health
// IP allow list (see setupHealthChecker), so no per-client rule is applied
// here. An allow list added later must be checked for every request before
// the cached result is returned.
type readinessCache struct {
	aggregator *health.Aggregator
	ttl        time.Duration
	now        func() time.Time

	mu       sync.Mutex
	decision health.Decision
	expires  time.Time
}

func newReadinessCache(aggregator *health.Aggregator, ttl time.Duration) *readinessCache {
	return &readinessCache{aggregator: aggregator, ttl: ttl, now: time.Now}
}

func (r *readinessCache) decide(c fiber.Ctx) health.Decision {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.now().Before(r.expires) {
		return r.decision
	}
	// The result is shared, so it must not depend on whether the request that
	// triggered it is cancelled. The aggregator applies its own timeout.
	decision := health.Decide(context.WithoutCancel(c.Context()), r.aggregator, healthfiber.Source{C: c})
	if decision.Forbidden {
		return decision
	}
	r.decision = decision
	r.expires = r.now().Add(r.ttl)
	return decision
}

// handler serves the readiness result in the format of healthfiber.Handler.
func (r *readinessCache) handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		decision := r.decide(c)
		if decision.Forbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
		}
		return c.Status(decision.StatusCode).JSON(decision.Body)
	}
}
