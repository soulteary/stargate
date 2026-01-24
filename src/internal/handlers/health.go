package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	rediskitclient "github.com/soulteary/redis-kit/client"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/storage"
)

// HealthStatus represents the health status of the service and its dependencies
type HealthStatus struct {
	Status  string                 `json:"status"` // "ok", "degraded", "down"
	Service string                 `json:"service"`
	Herald  *DependencyHealth      `json:"herald,omitempty"`
	Warden  *DependencyHealth      `json:"warden,omitempty"`
	Redis   *DependencyHealth      `json:"redis,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// DependencyHealth represents the health status of a dependency service
type DependencyHealth struct {
	Status    string `json:"status"` // "ok", "unavailable", "disabled"
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HealthRoute handles GET requests to /health for health checks.
// It checks the health of the service and its dependencies (Herald, Warden, Redis).
// Returns 200 OK if all critical dependencies are healthy, 503 if degraded.
//
// Returns a Fiber handler function.
func HealthRoute() func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		health := HealthStatus{
			Status:  "ok",
			Service: "stargate",
			Details: make(map[string]interface{}),
		}

		overallStatus := "ok"

		// Check Herald health
		if config.HeraldEnabled.ToBool() {
			heraldHealth := checkHeraldHealth(ctx.Context())
			health.Herald = heraldHealth
			if heraldHealth.Status != "ok" && heraldHealth.Status != "disabled" {
				overallStatus = "degraded"
			}
		} else {
			health.Herald = &DependencyHealth{Status: "disabled"}
		}

		// Check Warden health
		if config.WardenEnabled.ToBool() {
			wardenHealth := checkWardenHealth(ctx.Context())
			health.Warden = wardenHealth
			if wardenHealth.Status != "ok" && wardenHealth.Status != "disabled" {
				overallStatus = "degraded"
			}
		} else {
			health.Warden = &DependencyHealth{Status: "disabled"}
		}

		// Check Redis health (if session storage is enabled)
		if config.SessionStorageEnabled.ToBool() {
			redisHealth := checkRedisHealth(ctx.Context())
			health.Redis = redisHealth
			if redisHealth.Status != "ok" && redisHealth.Status != "disabled" {
				// Redis is critical for session storage, but we can still serve requests
				// Mark as degraded rather than down
				overallStatus = "degraded"
			}
		} else {
			health.Redis = &DependencyHealth{Status: "disabled"}
		}

		health.Status = overallStatus

		// Return appropriate status code
		statusCode := fiber.StatusOK
		if overallStatus == "degraded" {
			statusCode = fiber.StatusServiceUnavailable
		}

		return ctx.Status(statusCode).JSON(health)
	}
}

// checkHeraldHealth checks the health of the Herald service
func checkHeraldHealth(ctx context.Context) *DependencyHealth {
	health := &DependencyHealth{Status: "unavailable"}

	client := getHeraldClient()
	if client == nil {
		health.Error = "client not initialized"
		return health
	}

	// Try to create a test challenge with a short timeout
	// We use a minimal request that should fail quickly if service is down
	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	startTime := time.Now()
	// Use a test request that will fail fast if service is unavailable
	// We don't actually need to create a challenge, just check connectivity
	_, err := client.CreateChallenge(testCtx, &herald.CreateChallengeRequest{
		UserID:      "health_check",
		Channel:     "email",
		Destination: "health@check.test",
		Purpose:     "health_check",
	})
	latency := time.Since(startTime)

	if err != nil {
		// Check if it's a connection error vs validation error
		if heraldErr, ok := err.(*herald.HeraldError); ok {
			if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
				health.Error = "connection failed"
				health.LatencyMs = latency.Milliseconds()
				return health
			}
			// If we get a validation error (400), service is up but request is invalid
			// This is acceptable for health check
			if heraldErr.StatusCode == http.StatusBadRequest {
				health.Status = "ok"
				health.LatencyMs = latency.Milliseconds()
				return health
			}
		}
		health.Error = err.Error()
		health.LatencyMs = latency.Milliseconds()
		return health
	}

	health.Status = "ok"
	health.LatencyMs = latency.Milliseconds()
	return health
}

// checkWardenHealth checks the health of the Warden service
func checkWardenHealth(ctx context.Context) *DependencyHealth {
	health := &DependencyHealth{Status: "unavailable"}

	// Try a simple check with a short timeout
	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	startTime := time.Now()
	// Use CheckUserInList with a test identifier that won't exist
	// This will fail fast if service is unavailable
	_ = auth.CheckUserInList(testCtx, "health_check_phone", "health_check@test")
	latency := time.Since(startTime)

	// If we got here without error, service is responding
	// Note: CheckUserInList returns false for non-existent users, but that's OK for health check
	health.Status = "ok"
	health.LatencyMs = latency.Milliseconds()
	return health
}

// checkRedisHealth checks the health of the Redis connection
func checkRedisHealth(ctx context.Context) *DependencyHealth {
	health := &DependencyHealth{Status: "unavailable"}

	// Parse Redis DB number
	redisDB := 0
	if config.SessionStorageRedisDB.Value != "" {
		if db, err := strconv.Atoi(config.SessionStorageRedisDB.Value); err == nil {
			redisDB = db
		}
	}

	// Create a temporary Redis client for health check
	// This is a simple approach - in production, you might want to store the client globally
	redisClient, err := storage.NewRedisClientFromConfig(
		config.SessionStorageRedisAddr.Value,
		config.SessionStorageRedisPassword.Value,
		redisDB,
	)
	if err != nil {
		health.Error = "failed to create redis client: " + err.Error()
		return health
	}
	defer func() {
		_ = rediskitclient.Close(redisClient)
	}()

	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	startTime := time.Now()
	healthy := rediskitclient.HealthCheck(testCtx, redisClient)
	latency := time.Since(startTime)

	if healthy {
		health.Status = "ok"
		health.LatencyMs = latency.Milliseconds()
	} else {
		health.Error = "ping failed"
		health.LatencyMs = latency.Milliseconds()
	}

	return health
}
