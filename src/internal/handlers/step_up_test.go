package handlers

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

func TestRedirectToStepUpPreservesOnlyLocalCallback(t *testing.T) {
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": "/admin/users?tab=roles"}, "")
	defer app.ReleaseCtx(ctx)

	err := redirectToStepUp(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "/_step_up?callback=%2Fadmin%2Fusers%3Ftab%3Droles", string(ctx.Response().Header.Peek("Location")))
}

func TestStepUpRouteRequiresAuthentication(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_step_up", nil, "")
	defer app.ReleaseCtx(ctx)

	err := StepUpRoute(store)(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestStepUpRouteEscapesCallbackInForm(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	app := fiber.New()
	app.Get("/_step_up", StepUpRoute(store))

	seedCtx, seedApp := createTestContext("GET", "/", nil, "")
	sess, err := store.Get(seedCtx)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, auth.Authenticate(sess))
	testza.AssertNoError(t, sess.Save())
	cookie := auth.SessionCookieName + "=" + sess.ID()
	seedApp.ReleaseCtx(seedCtx)

	req := httptest.NewRequest("GET", "/_step_up?callback=%2Fadmin%22%3E%3Cscript%3E", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)
	testza.AssertContains(t, string(body), "/admin&#34;&gt;&lt;script&gt;")
	testza.AssertFalse(t, strings.Contains(string(body), `<script>`))
}

func TestStepUpAPIRejectsWrongPassword(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("POST", "/_step_up", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=wrong&callback=/admin")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, auth.Authenticate(sess))

	err = StepUpAPI(store)(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func setupStepUpTest(t *testing.T) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin/*")
	testza.AssertNoError(t, config.Initialize(testLogger()))
}

func TestRequiresStepUpUsesForwardedBusinessPath(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": "/admin/users"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	testza.AssertTrue(t, requiresStepUp(ctx, sess))
	sess.Set("step_up_verified_at", time.Now().Unix())
	testza.AssertFalse(t, requiresStepUp(ctx, sess))
}

func TestRequiresStepUpIgnoresForwardedQuery(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	testza.AssertNoError(t, config.Initialize(testLogger()))

	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": "/admin?view=users"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	testza.AssertTrue(t, requiresStepUp(ctx, sess))
}

func TestStepUpRequestPathRejectsInvalidURI(t *testing.T) {
	testza.AssertEqual(t, "/", stepUpRequestPath("https://evil.example/admin"))
	testza.AssertEqual(t, "/", stepUpRequestPath("not-a-path"))
	testza.AssertEqual(t, "/admin", stepUpRequestPath("/admin?x=1"))
}

func TestStepUpAPIRecordsRecentVerification(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("POST", "/_step_up", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=test123&callback=/admin/users")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sess.ID())

	err = StepUpAPI(store)(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "/admin/users", string(ctx.Response().Header.Peek("Location")))
	verifiedSession, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	verified, ok := verifiedSession.Get("step_up_verified_at").(int64)
	testza.AssertTrue(t, ok)
	testza.AssertTrue(t, verified > 0)
}

func TestSafeStepUpCallbackRejectsExternalURLs(t *testing.T) {
	testza.AssertEqual(t, "/", safeStepUpCallback("https://evil.example"))
	testza.AssertEqual(t, "/", safeStepUpCallback("//evil.example"))
	testza.AssertEqual(t, "/admin", safeStepUpCallback("/admin"))
}
