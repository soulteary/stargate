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
	setupStepUpTest(t)
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
	setupStepUpTestWithPaths(t, "/admin/*")
}

func setupStepUpTestWithPaths(t *testing.T, paths string) {
	t.Helper()
	previousEnabled := config.StepUpEnabled.Value
	previousPaths := config.StepUpPaths.Value
	previousTrustedProxies := config.TrustedProxies.Value
	t.Cleanup(func() {
		config.StepUpEnabled.Value = previousEnabled
		config.StepUpPaths.Value = previousPaths
		config.TrustedProxies.Value = previousTrustedProxies
		config.InitStepUpMatcher()
	})
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", paths)
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	testza.AssertNoError(t, config.Initialize(testLogger()))
}

func TestRequiresStepUpUsesForwardedBusinessPath(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": "/admin/users"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	testza.AssertFalse(t, requiresStepUp(ctx, sess))
	testza.AssertNoError(t, auth.Authenticate(sess))
	testza.AssertTrue(t, requiresStepUp(ctx, sess))
	sess.Set("step_up_verified_at", time.Now().Unix())
	testza.AssertFalse(t, requiresStepUp(ctx, sess))
}

func TestRequiresStepUpIgnoresForwardedQuery(t *testing.T) {
	setupStepUpTestWithPaths(t, "/admin")

	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": "/admin?view=users"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, auth.Authenticate(sess))

	testza.AssertTrue(t, requiresStepUp(ctx, sess))
}

func TestStepUpRequestPathRejectsInvalidURI(t *testing.T) {
	_, ok := stepUpRequestPath("https://evil.example/admin")
	testza.AssertFalse(t, ok)
	_, ok = stepUpRequestPath("not-a-path")
	testza.AssertFalse(t, ok)
	for _, invalid := range []string{"//evil.example/admin", "/admin/../public", "/admin/%2e%2e/public", `/admin\\public`} {
		_, ok = stepUpRequestPath(invalid)
		testza.AssertFalse(t, ok)
	}
	path, ok := stepUpRequestPath("/admin?x=1")
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "/admin", path)
}

func TestRequiresStepUpFailsClosedWithoutValidForwardedURI(t *testing.T) {
	setupStepUpTest(t)
	store := setupTestStore()

	for _, forwardedURI := range []string{"", "https://evil.example/admin", "not-a-path"} {
		ctx, app := createTestContext("GET", "/_auth", map[string]string{"X-Forwarded-Uri": forwardedURI}, "")
		sess, err := store.Get(ctx)
		testza.AssertNoError(t, err)
		testza.AssertNoError(t, auth.Authenticate(sess))
		testza.AssertTrue(t, requiresStepUp(ctx, sess))
		sess.Release()
		app.ReleaseCtx(ctx)
	}
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
	testza.AssertEqual(t, "/", safeStepUpCallback("/admin/../public"))
	testza.AssertEqual(t, "/", safeStepUpCallback("/admin/%2e%2e/public"))
	testza.AssertEqual(t, "/", safeStepUpCallback(`/admin\\public`))
	testza.AssertEqual(t, "/admin", safeStepUpCallback("/admin"))
	testza.AssertEqual(t, "/admin?tab=users", safeStepUpCallback("/admin?tab=users"))
}
