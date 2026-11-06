package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	sessionkit "github.com/soulteary/session-kit/v3"
	sessionfiber "github.com/soulteary/session-kit/v3/fiberadapter"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

// SessionShareRoute handles GET requests to /_session_exchange for cross-domain session sharing.
// It sets a session cookie and redirects to the configured same-origin landing path.
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
		landingPath := config.SessionExchangeRedirectPath.String()
		if !config.ValidateSessionExchangeRedirectPath(config.EnvVariable{Value: landingPath}) {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, "invalid session exchange redirect configuration")
		}
		ticket := ctx.Query("ticket")
		if ticket == "" {
			return startSessionExchange(ctx)
		}
		if exchangeNonceCookie(ctx) == "" {
			return startSessionExchange(ctx)
		}
		targetHost := GetForwardedHost(ctx)
		sessionID, err := consumeSessionExchangeTicketWithStore(ctx.Context(), ticket, targetHost, exchangeNonceCookie(ctx), GetForwardedProto(ctx), landingPath, replayStore)
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

		// session-kit v3's root CreateCookie returns a *http.Cookie; the Fiber
		// adapter builds the *fiber.Cookie that ctx.Cookie needs.
		cookie := sessionfiber.Cookie(sessionConfig, sessionID)
		ctx.Cookie(cookie)
		setExchangeCookie(ctx, false, "", true)

		return ctx.Redirect().Status(fiber.StatusFound).To(landingPath)
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
