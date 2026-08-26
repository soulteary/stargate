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
	resetPasswordFailureBucketsForTesting()
	t.Cleanup(resetPasswordFailureBucketsForTesting)

	for i := 0; i < 10; i++ {
		ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
			"Stargate-Password": "wrong-password",
		}, "")
		err := rateLimitPasswordHeaderFailure(ctx)
		app.ReleaseCtx(ctx)
		testza.AssertNoError(t, err)
	}

	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
		"Stargate-Password": "wrong-password",
	}, "")
	defer app.ReleaseCtx(ctx)
	err := rateLimitPasswordHeaderFailure(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusTooManyRequests, ctx.Response().StatusCode())
	testza.AssertNotEqual(t, "", string(ctx.Response().Header.Peek(fiber.HeaderRetryAfter)))
}
