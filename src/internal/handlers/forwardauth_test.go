package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/utils/v2"
	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	fafiber "github.com/soulteary/forwardauth-kit/v3/fiberadapter"
	fahttp "github.com/soulteary/forwardauth-kit/v3/httpadapter"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// TestGetForwardAuthHandler_ReturnsNonNilAfterInit verifies that after InitForwardAuthHandler
// (invoked from TestMain in handlers_test.go), GetForwardAuthHandler returns a non-nil handler.
func TestGetForwardAuthHandler_ReturnsNonNilAfterInit(t *testing.T) {
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil after InitForwardAuthHandler")
	}
}

// TestInitForwardAuthHandler_WithMinimalConfig verifies InitForwardAuthHandler runs without panic
// when given minimal env and config. Can be run in isolation via -run InitForwardAuthHandler.
func TestInitForwardAuthHandler_WithMinimalConfig(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.test.com")
	t.Setenv("PASSWORDS", "plaintext:minimal")
	t.Setenv("STEP_UP_ENABLED", "false")
	t.Setenv("STEP_UP_PATHS", "")
	testLog := testLogger()
	if err := config.Initialize(testLog); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	InitForwardAuthHandler(testLog)
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil after InitForwardAuthHandler")
	}
}

// TestForwardAuthCheckRoute_ReturnsHandler verifies ForwardAuthCheckRoute returns a non-nil Fiber handler.
func TestForwardAuthCheckRoute_ReturnsHandler(t *testing.T) {
	store := session.NewStore(session.Config{
		Extractor:    extractors.FromCookie(auth.SessionCookieName),
		KeyGenerator: utils.UUID,
	})
	if store == nil {
		t.Fatal("session.New returned nil")
	}
	handler := ForwardAuthCheckRoute(store)
	if handler == nil {
		t.Error("ForwardAuthCheckRoute(store) must not return nil")
	}
}

// TestInitForwardAuthHandler_WithStepUpPaths verifies InitForwardAuthHandler runs with step-up paths
// (covers parseStepUpPaths: comma-separated, trimmed).
func TestInitForwardAuthHandler_WithStepUpPaths(t *testing.T) {
	setupStepUpTestWithPaths(t, " /admin*, /api/secret*, ")
	testLog := testLogger()
	InitForwardAuthHandler(testLog)
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil")
	}
}

// TestForwardAuthLogger_InfoWarnErrorAndFields exercises the forwardAuthLogger wrapper so that
// Info(), Warn(), Error() and Bool(), Int(), Int64(), Dur() are covered (zerolog adapter for forwardauth-kit).
func TestForwardAuthLogger_InfoWarnErrorAndFields(t *testing.T) {
	l := &forwardAuthLogger{log: testLogger()}

	l.Info().Str("key", "val").Msg("info message")
	l.Warn().Bool("enabled", true).Msg("warn message")
	l.Error().Err(errors.New("test err")).Int("code", 400).Int64("count", 1).Dur("latency", 10*time.Millisecond).Msg("error message")
}

// TestTranslateForwardAuthUsesRequestLanguage pins the type assertion in
// translateForwardAuth. forwardauth-kit v3 renamed FiberContext.Underlying to
// CtxSource.Unwrap; asserting the wrong type there still compiles and turns
// every ForwardAuth message back into its raw key, which is only visible in the
// response body.
func TestTranslateForwardAuthUsesRequestLanguage(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang i18n.Language
		want string
	}{
		{name: "chinese", lang: i18n.LangZH, want: "需要身份验证"},
		{name: "english", lang: i18n.LangEN, want: "Authentication required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("GET", "/_auth", map[string]string{
				"Accept-Language": string(tc.lang),
			}, "")
			defer app.ReleaseCtx(ctx)
			ctx.Locals("i18n-language", tc.lang)

			got := translateForwardAuth(fafiber.NewContext(ctx), "error.auth_required")
			testza.AssertEqual(t, tc.want, got)
			testza.AssertNotEqual(t, "error.auth_required", got,
				"the key leaked through untranslated, so the Fiber context was not unwrapped")
		})
	}
}

// TestTranslateForwardAuthFallsBackToKey covers the other side of the
// assertion. forwardauth-kit's net/http adapter is a forwardauth.Context too,
// but it carries no Fiber request, so the key is the only honest answer.
func TestTranslateForwardAuthFallsBackToKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/_auth", nil)
	req.Header.Set("Accept-Language", "zh")
	var faCtx forwardauth.Context = fahttp.NewContext(httptest.NewRecorder(), req)

	testza.AssertEqual(t, "error.auth_required",
		translateForwardAuth(faCtx, "error.auth_required"))
}
