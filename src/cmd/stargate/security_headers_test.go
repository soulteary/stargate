package main

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
)

const (
	headerCSP  = "Content-Security-Policy"
	headerHSTS = "Strict-Transport-Security"
)

func TestLoginPageSendsContentSecurityPolicy(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)
	app := createApp()

	req := httptest.NewRequest(fiber.MethodGet, RouteLogin, nil)
	req.Host = "auth.example.com"
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)
	testza.AssertContains(t, string(body), "<form")

	policy := resp.Header.Get(headerCSP)
	testza.AssertEqual(t, contentSecurityPolicy, policy)
	for _, directive := range []string{"default-src 'none'", "script-src 'self'", "style-src 'self'", "base-uri 'none'", "frame-ancestors 'none'"} {
		testza.AssertContains(t, policy, directive)
	}
	testza.AssertFalse(t, strings.Contains(policy, "unsafe-inline"))
	testza.AssertFalse(t, strings.Contains(policy, "unsafe-eval"))

	// The headers that were already sent stay in place.
	testza.AssertEqual(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	testza.AssertEqual(t, "DENY", resp.Header.Get("X-Frame-Options"))
}

// The policy has no 'unsafe-inline', so the served pages must not depend on
// inline script, inline style or inline event handlers.
func TestLoginPageIsCompatibleWithContentSecurityPolicy(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)
	app := createApp()

	req := httptest.NewRequest(fiber.MethodGet, RouteLogin, nil)
	req.Host = "auth.example.com"
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	raw, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)
	page := strings.ToLower(string(raw))

	testza.AssertFalse(t, strings.Contains(page, "<style"))
	testza.AssertFalse(t, strings.Contains(page, " style="))
	testza.AssertFalse(t, strings.Contains(page, " onclick="))
	testza.AssertFalse(t, strings.Contains(page, " onsubmit="))
	testza.AssertFalse(t, strings.Contains(page, " onload="))
	for _, script := range strings.Split(page, "<script")[1:] {
		tag, content, _ := strings.Cut(script, ">")
		testza.AssertContains(t, tag, `src="/assets/`)
		testza.AssertTrue(t, strings.HasPrefix(content, "</script"))
	}
}

func TestStrictTransportSecurityFollowsCookieSecure(t *testing.T) {
	for _, tc := range []struct {
		name         string
		cookieSecure string
		want         string
	}{
		{name: "default is HTTPS", cookieSecure: "", want: strictTransportSecurity},
		{name: "explicit HTTPS", cookieSecure: "true", want: strictTransportSecurity},
		{name: "local HTTP", cookieSecure: "false", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ensureTestWorkingDir(t)
			if tc.cookieSecure != "" {
				t.Setenv("COOKIE_SECURE", tc.cookieSecure)
			}
			setupTestConfig(t)
			app := createApp()

			for _, route := range []string{RouteLogin, RouteHealthz} {
				req := httptest.NewRequest(fiber.MethodGet, route, nil)
				req.Host = "auth.example.com"
				resp, err := app.Test(req)
				testza.AssertNoError(t, err)
				testza.AssertEqual(t, tc.want, resp.Header.Get(headerHSTS))
			}
		})
	}
	testza.AssertFalse(t, strings.Contains(strictTransportSecurity, "includeSubDomains"))
}

// A denied forward-auth response can be relayed to the browser under the
// protected application's origin, so it must not carry Stargate's page policy
// or pin that origin to HTTPS.
func TestForwardAuthResponseKeepsPreviousSecurityHeaders(t *testing.T) {
	ensureTestWorkingDir(t)
	t.Setenv("COOKIE_SECURE", "true")
	setupTestConfig(t)
	app := createApp()

	for _, path := range []string{RouteAuth, RouteAuth + "/", "/_AUTH"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodGet, path, nil)
			req.Host = "auth.example.com"
			resp, err := app.Test(req)
			testza.AssertNoError(t, err)
			testza.AssertNotEqual(t, fiber.StatusNotFound, resp.StatusCode)
			testza.AssertEqual(t, "", resp.Header.Get(headerCSP))
			testza.AssertEqual(t, "", resp.Header.Get(headerHSTS))
			testza.AssertEqual(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			testza.AssertEqual(t, "DENY", resp.Header.Get("X-Frame-Options"))
		})
	}
}
