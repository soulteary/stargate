package handlers

import (
	"strings"

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
	return func(ctx fiber.Ctx) error {
		ticket := ctx.Query("ticket")
		if ticket == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.missing_session_id"))
		}
		targetHost := GetForwardedHost(ctx)
		sessionID, err := consumeSessionExchangeTicketWithStore(ctx.Context(), ticket, targetHost, replayStore)
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

		// A Domain cookie is valid only when the exchange target belongs to that
		// domain. Cross-root-domain callbacks receive a host-only cookie.
		sessionConfig := sessionkit.DefaultConfig().
			WithCookieName(auth.SessionCookieName).
			WithExpiration(config.SessionExpiration).
			WithSecure(config.CookieSecure.ToBool()).
			WithSameSite("Lax").
			WithHTTPOnly(true)
		if cookieDomain := sessionCookieDomainForHost(targetHost, config.CookieDomain.Value); cookieDomain != "" {
			sessionConfig = sessionConfig.WithCookieDomain(cookieDomain)
		}

		// Use session-kit's CreateCookie for consistent cookie creation
		cookie := sessionkit.CreateCookie(sessionConfig, sessionID)
		ctx.Cookie(cookie)

		return ctx.Redirect().Status(fiber.StatusFound).To("/")
	}
}

func sessionCookieDomainForHost(host, configuredDomain string) string {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(configuredDomain), "."), "."))
	hostname := normalizeHost(host)
	if domain == "" || hostname == "" {
		return ""
	}
	if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
		return configuredDomain
	}
	return ""
}
