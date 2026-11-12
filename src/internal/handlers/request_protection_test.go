package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
)

func newRequestProtectionTestRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Host = "auth.example.com"
	return req
}

func TestRequireSameOriginRejectsCrossSiteBrowserWrite(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := newRequestProtectionTestRequest(fiber.MethodPost, "/write")
	req.Header.Set("Origin", "https://evil.example")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	testza.AssertEqual(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireSameOriginAllowsMatchingOrigin(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := newRequestProtectionTestRequest(fiber.MethodPost, "/write")
	req.Header.Set("Origin", "http://auth.example.com")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestRequireSameOriginRejectsSchemeAndPortMismatch(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	for _, origin := range []string{
		"https://auth.example.com",
		"http://auth.example.com:8080",
		"http://auth.example.com/path",
	} {
		req := newRequestProtectionTestRequest(fiber.MethodPost, "/write")
		req.Header.Set("Origin", origin)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("execute request with origin %q: %v", origin, err)
		}
		testza.AssertEqual(t, fiber.StatusForbidden, resp.StatusCode)
	}
}

func TestRequireSameOriginNormalizesDefaultPort(t *testing.T) {
	app := fiber.New()
	app.Post("/write", RequireSameOrigin(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := newRequestProtectionTestRequest(fiber.MethodPost, "/write")
	req.Host = "auth.example.com:80"
	req.Header.Set("Origin", "http://auth.example.com")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestLoginRateLimitRejectsBurst(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	app := fiber.New()
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	for i := 0; i < 10; i++ {
		resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
		if err != nil {
			t.Fatalf("execute request %d: %v", i+1, err)
		}
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	}
	resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodPost, "/_login"))
	if err != nil {
		t.Fatalf("execute rate-limited request: %v", err)
	}
	testza.AssertEqual(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

func TestPasswordHeaderFailureRateLimit(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	for i := 0; i < 10; i++ {
		ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
			"Stargate-Password": "wrong-password",
		}, "")
		limited, err := rateLimitPasswordHeaderFailure(ctx)
		app.ReleaseCtx(ctx)
		testza.AssertFalse(t, limited)
		testza.AssertNoError(t, err)
	}

	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
		"Stargate-Password": "wrong-password",
	}, "")
	defer app.ReleaseCtx(ctx)
	limited, err := rateLimitPasswordHeaderFailure(ctx)
	testza.AssertTrue(t, limited)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, ctx.Response().StatusCode())
	testza.AssertNotEqual(t, "", string(ctx.Response().Header.Peek(fiber.HeaderRetryAfter)))
}

func TestRejectForwardedRequests(t *testing.T) {
	app := fiber.New()
	app.Get("/local", RejectForwardedRequests(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	t.Run("direct request passes", func(t *testing.T) {
		resp, err := app.Test(newRequestProtectionTestRequest(fiber.MethodGet, "/local"))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusNoContent, resp.StatusCode)
	})

	for name, value := range map[string]string{
		"X-Forwarded-For":   "203.0.113.7",
		"X-Forwarded-Host":  "auth.example.com",
		"X-Forwarded-Proto": "https",
		"Forwarded":         "for=203.0.113.7",
		"X-Real-IP":         "203.0.113.7",
	} {
		t.Run(name, func(t *testing.T) {
			req := newRequestProtectionTestRequest(fiber.MethodGet, "/local")
			req.Header.Set(name, value)
			resp, err := app.Test(req)
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, fiber.StatusNotFound, resp.StatusCode)
		})
	}
}
