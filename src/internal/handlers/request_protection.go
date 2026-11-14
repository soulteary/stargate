package handlers

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
)

type requestOrigin struct {
	scheme   string
	hostname string
	port     string
}

func canonicalRequestOrigin(raw string) (requestOrigin, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return requestOrigin{}, false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return requestOrigin{}, false
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" {
		return requestOrigin{}, false
	}
	port := parsed.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return requestOrigin{scheme: scheme, hostname: hostname, port: port}, true
}

// RequireSameOrigin blocks browser cross-site writes while preserving clients
// such as curl that do not send browser Origin or Fetch Metadata headers.
func RequireSameOrigin() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if ctx.Method() == fiber.MethodGet || ctx.Method() == fiber.MethodHead || ctx.Method() == fiber.MethodOptions {
			return ctx.Next()
		}
		if strings.EqualFold(strings.TrimSpace(ctx.Get("Sec-Fetch-Site")), "cross-site") {
			return SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected")
		}
		origin := strings.TrimSpace(ctx.Get("Origin"))
		if origin == "" {
			return ctx.Next()
		}
		suppliedOrigin, suppliedOK := canonicalRequestOrigin(origin)
		requestOrigin, requestOK := canonicalRequestOrigin(GetForwardedProto(ctx) + "://" + GetForwardedHost(ctx))
		if !suppliedOK || !requestOK || suppliedOrigin != requestOrigin {
			return SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected")
		}
		return ctx.Next()
	}
}

// forwardingHeaders are the headers a reverse proxy adds when it relays a
// request. A request that carries any of them did not originate on this host.
var forwardingHeaders = []string{
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Proto",
	"Forwarded",
	"X-Real-IP",
}

// RejectForwardedRequests protects loopback-only operator endpoints. A
// same-host reverse proxy or sidecar connects from 127.0.0.1, so the peer
// address alone cannot tell a local operator from a relayed remote client.
// Relayed requests carry forwarding headers and direct local ones do not, so
// any such header hides the endpoint with 404 Not Found.
func RejectForwardedRequests() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		for _, name := range forwardingHeaders {
			if len(ctx.Request().Header.Peek(name)) > 0 {
				return ctx.SendStatus(fiber.StatusNotFound)
			}
		}
		return ctx.Next()
	}
}

// Rate-limit response headers, kept byte-for-byte compatible with the Fiber
// limiter this package previously used so existing clients keep working.
const (
	xRateLimitLimit     = "X-RateLimit-Limit"
	xRateLimitRemaining = "X-RateLimit-Remaining"
	xRateLimitReset     = "X-RateLimit-Reset"
)

// untrustedForwardedWarning reports a proxied deployment that never declared
// its proxy: once per process, rather than on every request.
var untrustedForwardedWarning sync.Once

// warnUntrustedForwardedHeaders reports the misconfiguration that silently
// collapses per-client rate limiting: a reverse proxy sits in front of
// Stargate but its address is absent from TRUSTED_PROXIES, so every client is
// attributed to the proxy and the whole deployment shares one quota.
func warnUntrustedForwardedHeaders(ctx fiber.Ctx) {
	if log == nil || trustForwardedHeaders(ctx) {
		return
	}
	if ctx.Get("X-Forwarded-For") == "" && ctx.Get("X-Forwarded-Host") == "" {
		return
	}
	untrustedForwardedWarning.Do(func() {
		log.Warn().
			Str("peer", ctx.IP()).
			Msg("Received forwarded headers from an untrusted peer: rate limits are keyed by the proxy address, so every client shares one quota. Set TRUSTED_PROXIES to the reverse-proxy source IPs or CIDRs.")
	})
}

// resetRateLimitStateForTesting restores a pristine process-local store and
// re-arms the one-shot proxy warning so ordering between tests cannot leak.
func resetRateLimitStateForTesting() {
	SetRateLimitStore(newMemoryRateLimitStore())
	untrustedForwardedWarning = sync.Once{}
}

// rateLimitWindow returns the configured fixed window. Startup validation
// rejects a non-positive duration, so the fallback only covers callers that
// construct handlers without running Initialize, such as unit tests.
func rateLimitWindow() time.Duration {
	if window := config.RateLimitWindow.ToDuration(); window > 0 {
		return window
	}
	return config.DefaultRateLimitWindow
}

func retryAfterSeconds(retryAfter time.Duration) string {
	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// rateLimitIPv6PrefixBits is the prefix length that identifies one IPv6 client.
// A /64 is the smallest allocation a network normally hands to a single
// subscriber or host, and that client can pick any address inside it.
const rateLimitIPv6PrefixBits = 64

// rateLimitClientKey returns the rate-limit identity for a client address.
// IPv4 addresses are used as they are. IPv6 addresses are reduced to their /64
// prefix, so a client cannot obtain a fresh quota by moving to another address
// in its own prefix. An unparsable value is returned unchanged.
func rateLimitClientKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap().WithZone("")
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(rateLimitIPv6PrefixBits)
	if err != nil {
		return ip
	}
	return prefix.String()
}

// consumeRateLimit records one request against key and reports whether the
// caller is over quota, how much of the quota is left, and how long the current
// window still runs.
func consumeRateLimit(ctx fiber.Ctx, key string, max int) (exceeded bool, remaining int, retryAfter time.Duration) {
	warnUntrustedForwardedHeaders(ctx)

	count, window, err := getRateLimitStore().Incr(ctx.Context(), key, rateLimitWindow())
	if err != nil && log != nil {
		// The store already fell back to process-local counting, so the request
		// still carries a bound; only cross-replica sharing is degraded.
		log.Warn().Err(err).Msg("Shared rate-limit store is unavailable, falling back to per-replica limits")
	}

	remaining = max - count
	if remaining < 0 {
		remaining = 0
	}
	return count > max, remaining, window
}

// endpointRateLimit enforces a fixed-window limit through the shared rate-limit
// store. The limit is read per request so operators get the value that startup
// validation accepted, and a limit of zero disables the endpoint quota.
func endpointRateLimit(limit func() int) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		max := limit()
		if max <= 0 {
			return ctx.Next()
		}

		// Route identity is fixed by registration. The raw request spelling can
		// vary by case or trailing slash while Fiber still matches this endpoint.
		exceeded, remaining, retryAfter := consumeRateLimit(ctx, rateLimitClientKey(ctx.IP())+":"+ctx.Route().Method+":"+ctx.Route().Path, max)
		ctx.Set(xRateLimitLimit, strconv.Itoa(max))
		ctx.Set(xRateLimitRemaining, strconv.Itoa(remaining))
		ctx.Set(xRateLimitReset, retryAfterSeconds(retryAfter))
		if !exceeded {
			return ctx.Next()
		}

		ctx.Set(fiber.HeaderRetryAfter, retryAfterSeconds(retryAfter))
		return SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many requests")
	}
}

func LoginRateLimit() fiber.Handler {
	return endpointRateLimit(func() int {
		return config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	})
}

func VerificationRateLimit() fiber.Handler {
	return endpointRateLimit(func() int {
		return config.RateLimitVerificationMax.ToInt(config.DefaultRateLimitVerificationMax)
	})
}

// passwordHeaderFailureKey is the quota that counts wrong Stargate-Password
// values for one client.
func passwordHeaderFailureKey(ctx fiber.Ctx) string {
	return "password-failure:" + rateLimitClientKey(ctx.IP())
}

// passwordHeaderQuotaExhausted reports whether the client has already used up
// its failed Stargate-Password attempts for the current window. It is checked
// before the password is evaluated, so a client that is over quota learns
// nothing about further guesses, including a correct one. It records nothing:
// only failures are counted, by recordPasswordHeaderFailure.
func passwordHeaderQuotaExhausted(ctx fiber.Ctx) (bool, error) {
	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	if max <= 0 {
		return false, nil
	}

	count, retryAfter, err := getRateLimitStore().Count(ctx.Context(), passwordHeaderFailureKey(ctx))
	if err != nil && log != nil {
		log.Warn().Err(err).Msg("Shared rate-limit store is unavailable, falling back to per-replica limits")
	}
	if count < max {
		return false, nil
	}

	ctx.Set(fiber.HeaderRetryAfter, retryAfterSeconds(retryAfter))
	return true, SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many failed password attempts")
}

// recordPasswordHeaderFailure counts one wrong Stargate-Password value against
// the login quota. It shares that quota's store, so a credential-stuffing
// client cannot simply move to another replica.
func recordPasswordHeaderFailure(ctx fiber.Ctx) {
	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	if max <= 0 {
		return
	}
	consumeRateLimit(ctx, passwordHeaderFailureKey(ctx), max)
}

// rateLimitPasswordHeaderFailure counts one Stargate-Password failure and
// answers 429 once the count exceeds the login quota. CheckRoute counts
// failures itself; this covers a password error that reaches the failure
// handler directly.
func rateLimitPasswordHeaderFailure(ctx fiber.Ctx) (bool, error) {
	if strings.TrimSpace(ctx.Get("Stargate-Password")) == "" {
		return false, nil
	}

	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	if max <= 0 {
		return false, nil
	}

	// The forward-auth path deliberately keeps its original response shape and
	// does not advertise quota state through X-RateLimit-* headers.
	exceeded, _, retryAfter := consumeRateLimit(ctx, passwordHeaderFailureKey(ctx), max)
	if !exceeded {
		return false, nil
	}

	ctx.Set(fiber.HeaderRetryAfter, retryAfterSeconds(retryAfter))
	return true, SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many failed password attempts")
}
