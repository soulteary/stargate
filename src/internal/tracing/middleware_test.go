package tracing

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestTracingMiddleware(t *testing.T) {
	// Initialize tracer
	InitTracer("test-service", "1.0.0", "")

	app := fiber.New()
	app.Use(TracingMiddleware("test-service"))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "test error")
	})

	t.Run("Success Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Check if trace headers are injected
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Error Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		// Check if trace headers are injected
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Bad Request", func(t *testing.T) {
		app.Get("/bad", func(c *fiber.Ctx) error {
			return c.SendStatus(400)
		})

		req := httptest.NewRequest("GET", "/bad", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Root Path", func(t *testing.T) {
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("root")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
