package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// HealthRoute handles GET requests to /health for health checks.
// It returns a 200 OK status to indicate the service is running.
//
// Returns a Fiber handler function.
func HealthRoute() func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	}
}
