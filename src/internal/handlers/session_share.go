package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	sessionkit "github.com/soulteary/session-kit/v2"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// SessionShareRoute handles GET requests to /_session_exchange for cross-domain session sharing.
// It sets a session cookie with the provided session ID and redirects to the root path.
// This allows sessions to be shared across different domains/subdomains.
//
// Query parameters:
//   - ticket: encrypted, short-lived, single-use session exchange ticket
//
// Returns a Fiber handler function.
func SessionShareRoute(store *session.Store, replayStores ...SessionExchangeReplayStore) func(c fiber.Ctx) error {
	replayStore := defaultSessionExchangeReplayStore
	if len(replayStores) > 0 && replayStores[0] != nil {
		replayStore = replayStores[0]
	}
	// Create session config for cookie creation
	sessionConfig := sessionkit.DefaultConfig().
		WithCookieName(auth.SessionCookieName).
		WithExpiration(config.SessionExpiration).
		WithCookieDomain(config.CookieDomain.Value).
		WithSecure(config.CookieSecure.ToBool()).
		WithSameSite("Lax").
		WithHTTPOnly(true)

	return func(ctx fiber.Ctx) error {
		ticket := ctx.Query("ticket")
		if ticket == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.missing_session_id"))
		}
		sessionID, err := consumeSessionExchangeTicketWithStore(ctx.Context(), ticket, GetForwardedHost(ctx), replayStore)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "invalid session exchange ticket")
		}

		ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "invalid session exchange ticket")
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "invalid session exchange ticket")
		}

		// Use session-kit's CreateCookie for consistent cookie creation
		cookie := sessionkit.CreateCookie(sessionConfig, sessionID)
		ctx.Cookie(cookie)

		return ctx.Redirect().To("/")
	}
}
