package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
)

func TestRequireSameOriginRejectsCrossSiteBrowserWrite(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := httptest.NewRequest(fiber.MethodPost, "http://auth.example.com/write", nil)
	req.Header.Set("Origin", "https://evil.example")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireSameOriginAllowsMatchingOrigin(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := httptest.NewRequest(fiber.MethodPost, "http://auth.example.com/write", nil)
	req.Header.Set("Origin", "https://auth.example.com")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestLoginRateLimitRejectsBurst(t *testing.T) {
	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	for i := 0; i < 10; i++ {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "http://auth.example.com/_login", nil))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	}
	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "http://auth.example.com/_login", nil))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
}
