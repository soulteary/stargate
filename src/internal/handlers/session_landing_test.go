package handlers

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

func TestSessionExchangeUsesConfiguredLandingAfterVerifiedTicket(t *testing.T) {
	setSessionExchangeSecret(t)
	original := config.SessionExchangeRedirectPath.Value
	t.Cleanup(func() { config.SessionExchangeRedirectPath.Value = original })
	for _, landing := range []string{"/", "/auth/login/example/"} {
		t.Run(landing, func(t *testing.T) {
			config.SessionExchangeRedirectPath.Value = landing
			store := setupTestStore()
			seed, seedApp := createTestContext("GET", "/", map[string]string{"Host": "app.example.com"}, "")
			sess, err := store.Get(seed)
			if err != nil {
				t.Fatal(err)
			}
			if err := auth.Authenticate(sess); err != nil {
				t.Fatal(err)
			}
			id := responseCookieValue(seed, auth.SessionCookieName)
			ticket, err := createSessionExchangeTicket(id, testExchangeState("app.example.com"))
			seedApp.ReleaseCtx(seed)
			if err != nil {
				t.Fatal(err)
			}
			ctx, app := createTestContext("GET", "/_session_exchange?ticket="+ticket+"&next=https://evil.example/", map[string]string{"Host": "app.example.com"}, "")
			bindTestExchangeBrowser(ctx)
			defer app.ReleaseCtx(ctx)
			if err := SessionShareRoute(store)(ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.Response().StatusCode() != fiber.StatusFound || string(ctx.Response().Header.Peek("Location")) != landing {
				t.Fatalf("expected configured same-origin redirect %q", landing)
			}
			if !strings.Contains(string(ctx.Response().Header.Peek("Set-Cookie")), auth.SessionCookieName+"=") {
				t.Fatal("verified exchange must establish the callback session")
			}
			replay, replayApp := createTestContext("GET", "/_session_exchange?ticket="+ticket, map[string]string{"Host": "app.example.com"}, "")
			bindTestExchangeBrowser(replay)
			defer replayApp.ReleaseCtx(replay)
			if err := SessionShareRoute(store)(replay); err != nil {
				t.Fatal(err)
			}
			if replay.Response().StatusCode() != fiber.StatusUnauthorized || len(replay.Response().Header.Peek("Location")) != 0 {
				t.Fatal("landing path must not bypass single-use ticket validation")
			}
		})
	}
}

func TestSessionExchangeRejectsUnsafeRuntimeLandingConfiguration(t *testing.T) {
	original := config.SessionExchangeRedirectPath.Value
	t.Cleanup(func() { config.SessionExchangeRedirectPath.Value = original })
	config.SessionExchangeRedirectPath.Value = "//evil.example/"
	ctx, app := createTestContext("GET", "/_session_exchange?ticket=invalid", map[string]string{"Host": "app.example.com"}, "")
	defer app.ReleaseCtx(ctx)
	if err := SessionShareRoute(setupTestStore())(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Response().StatusCode() != fiber.StatusInternalServerError || len(ctx.Response().Header.Peek("Location")) != 0 {
		t.Fatal("invalid runtime configuration must fail closed")
	}
}
