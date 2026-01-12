package middleware

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
)

func TestNewLogMiddleware_Success(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	// Register a route with the middleware and a handler
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Use app.Test() to make a proper HTTP request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestNewLogMiddleware_WithError(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	// Register a route that returns an error
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "Test error")
	})

	// Use app.Test() to make a proper HTTP request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	// The middleware should handle the error and still log
	testza.AssertEqual(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestNewLogMiddleware_LogsRequest(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	// The middleware logs the request, but we can't easily verify the log output
	// Instead, we verify that the middleware executes without error
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestNewLogMiddleware_DifferentMethods(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	// Register routes for different methods
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Post("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Put("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Delete("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Patch("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			resp, err := app.Test(req)
			// Middleware should handle all methods without panicking
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
		})
	}
}

func TestNewLogMiddleware_DifferentStatusCodes(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	statusCodes := []int{
		fiber.StatusOK,
		fiber.StatusBadRequest,
		fiber.StatusUnauthorized,
		fiber.StatusNotFound,
		fiber.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(fmt.Sprintf("Status%d", statusCode), func(t *testing.T) {
			// Register a route that returns the specific status code
			app.Get(fmt.Sprintf("/test%d", statusCode), middleware, func(c *fiber.Ctx) error {
				return c.SendStatus(statusCode)
			})

			req := httptest.NewRequest("GET", fmt.Sprintf("/test%d", statusCode), nil)
			resp, err := app.Test(req)
			// Middleware should handle all status codes without panicking
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, statusCode, resp.StatusCode)
		})
	}
}

func TestNewLogMiddleware_WithIP(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	resp, err := app.Test(req)
	// Middleware should handle IP logging without panicking
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestNewLogMiddleware_WithHostname(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	resp, err := app.Test(req)
	// Middleware should handle hostname logging without panicking
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestNewLogMiddleware_WithOriginalURL(t *testing.T) {
	app := fiber.New()
	middleware := NewLogMiddleware()

	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test?param=value", nil)
	resp, err := app.Test(req)
	// Middleware should handle URL logging without panicking
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}
