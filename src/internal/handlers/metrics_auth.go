package handlers

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
)

// MetricsAuth keeps scrape credentials independent of login and directory keys.
// An absent key disables scraping; operators must explicitly provision one.
func MetricsAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		expected := config.MetricsAPIKey.String()
		if expected == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		supplied, bearer := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
		if !bearer || supplied == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) != 1 {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Next()
	}
}
