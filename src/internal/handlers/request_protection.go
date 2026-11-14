package handlers

import (
	"context"
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
			if suppliedOK {
				warnSameOriginIgnoredForwardedHeaders(ctx, suppliedOrigin)
			}
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

// warnUntrustedForwardedHeaders reports a reverse proxy that sits in front of
// Stargate while its address is absent from TRUSTED_PROXIES. Its forwarded
// headers are ignored, which silently breaks two things: the login redirect
// from /_auth is built without the original host and scheme, and every client
// is attributed to the proxy, so the whole deployment shares one rate-limit
// quota.
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
			Msg("Received forwarded headers from an untrusted peer and ignored them: the login redirect from /_auth loses its callback and uses http://, so users are not returned to the application after sign-in, and rate limits are keyed by the proxy address, so every client shares one quota. Set TRUSTED_PROXIES to the reverse-proxy source IPs or CIDRs.")
	})
}

// sameOriginForwardedWarning reports, once per process, a same-origin check
// that failed only because forwarded headers were ignored.
var sameOriginForwardedWarning sync.Once

// warnSameOriginIgnoredForwardedHeaders explains a rejected write that the
// browser sent to its own origin. Behind a proxy that terminates HTTPS and is
// absent from TRUSTED_PROXIES, Stargate ignores X-Forwarded-Proto and
// X-Forwarded-Host and compares the browser's https:// Origin with the
// http:// request it received, so every sign-in is answered 403. The response
// does not say why; the log has to.
//
// The warning is limited to requests whose Origin equals the origin the
// forwarded headers describe, which is the case this configuration produces.
// Other rejected requests are cross-site and are not reported.
func warnSameOriginIgnoredForwardedHeaders(ctx fiber.Ctx, supplied requestOrigin) {
	if log == nil || trustForwardedHeaders(ctx) {
		return
	}
	proto := strings.ToLower(strings.TrimSpace(strings.Split(ctx.Get("X-Forwarded-Proto"), ",")[0]))
	host := strings.TrimSpace(strings.Split(ctx.Get("X-Forwarded-Host"), ",")[0])
	if proto == "" && host == "" {
		return
	}
	if proto != "http" && proto != "https" {
		proto = GetForwardedProto(ctx)
	}
	if host == "" {
		host = GetForwardedHost(ctx)
	}
	forwarded, ok := canonicalRequestOrigin(proto + "://" + host)
	if !ok || forwarded != supplied {
		return
	}
	sameOriginForwardedWarning.Do(func() {
		log.Warn().
			Str("peer", ctx.IP()).
			Str("path", ctx.Path()).
			Msg("Rejected a same-origin request as cross-site because forwarded headers from an untrusted peer were ignored: the Origin header matches X-Forwarded-Proto and X-Forwarded-Host, but not the request Stargate received from the proxy. Sign-in and every other POST from the browser are answered 403 until TRUSTED_PROXIES lists the reverse-proxy source IPs or CIDRs.")
	})
}

// resetRateLimitStateForTesting restores a pristine process-local store and
// re-arms the one-shot proxy warnings so ordering between tests cannot leak.
func resetRateLimitStateForTesting() {
	SetRateLimitStore(newMemoryRateLimitStore())
	untrustedForwardedWarning = sync.Once{}
	sameOriginForwardedWarning = sync.Once{}
	unattributedPasswordHeaderWarning.untrustedPeer = sync.Once{}
	unattributedPasswordHeaderWarning.noClientAddress = sync.Once{}
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

// unattributedPasswordHeaderWarning reports, once per process and reason, a
// Stargate-Password request that PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY refused.
var unattributedPasswordHeaderWarning struct {
	untrustedPeer   sync.Once
	noClientAddress sync.Once
}

// refuseUnattributedPasswordHeader applies PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY.
// The failure quota is keyed by client address. A request that reaches
// Stargate from a peer outside TRUSTED_PROXIES, or from a trusted proxy that
// supplied no client address, is attributed to the peer, so every client
// behind that peer shares one quota and one of them can use it up for all.
// With the setting on, such a request is answered 403 before the password is
// compared and before it is counted: it gets no answer about the password and
// cannot use up the quota of any address.
//
// Both conditions are read from the connection: whether the peer address is
// in TRUSTED_PROXIES, and whether the resolved client address differs from the
// peer address. A request header cannot change either of them for a peer that
// is not trusted.
func refuseUnattributedPasswordHeader(ctx fiber.Ctx) (bool, error) {
	if !config.PasswordHeaderRequireTrustedProxy.ToBool() {
		return false, nil
	}
	const message = "Stargate-Password is accepted only through a trusted proxy that supplies the client address"
	if !trustForwardedHeaders(ctx) {
		unattributedPasswordHeaderWarning.untrustedPeer.Do(func() {
			if log != nil {
				log.Warn().Str("peer", peerAddress(ctx)).
					Msg("Refused a Stargate-Password request because PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY is true and the peer is not in TRUSTED_PROXIES. The password was not compared. Add the reverse-proxy source IPs or CIDRs to TRUSTED_PROXIES.")
			}
		})
		return true, SendErrorResponse(ctx, fiber.StatusForbidden, message)
	}
	if sameAddress(ctx.IP(), peerAddress(ctx)) {
		unattributedPasswordHeaderWarning.noClientAddress.Do(func() {
			if log != nil {
				log.Warn().Str("peer", peerAddress(ctx)).Str("proxy_header", config.ProxyHeader.String()).
					Msg("Refused a Stargate-Password request because PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY is true and the trusted proxy supplied no client address other than its own. The password was not compared. Make the proxy send the client address in the header named by PROXY_HEADER.")
			}
		})
		return true, SendErrorResponse(ctx, fiber.StatusForbidden, message)
	}
	return false, nil
}

// peerAddress returns the address of the connection the request arrived on.
func peerAddress(ctx fiber.Ctx) string {
	if ip := ctx.RequestCtx().RemoteIP(); ip != nil {
		return ip.String()
	}
	return ""
}

// sameAddress compares two addresses by value, so that an IPv4-mapped IPv6
// address equals its IPv4 form. Values that do not parse are compared as text.
func sameAddress(a, b string) bool {
	left, leftErr := netip.ParseAddr(a)
	right, rightErr := netip.ParseAddr(b)
	if leftErr != nil || rightErr != nil {
		return a == b
	}
	return left.Unmap().WithZone("") == right.Unmap().WithZone("")
}

// reservePasswordHeaderAttempt counts one Stargate-Password attempt against the
// login quota before the password is compared. The count and the decision are
// one step in the store, so attempts that arrive together cannot all be
// compared: at most the quota is, on every replica together when the store is
// shared.
//
// It reports limited once the quota is used up; the response is then written
// and the attempt is not kept. Otherwise the caller compares the password and
// calls release when it was correct, so only failures stay counted. A nil
// release means the quota is disabled.
func reservePasswordHeaderAttempt(ctx fiber.Ctx) (release func(), limited bool, err error) {
	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	if max <= 0 {
		return nil, false, nil
	}
	warnUntrustedForwardedHeaders(ctx)

	// A release must still reach the store when the request context has ended.
	storeCtx := context.WithoutCancel(ctx.Context())
	count, retryAfter, releaseAttempt, storeErr := getRateLimitStore().Reserve(ctx.Context(), passwordHeaderFailureKey(ctx), rateLimitWindow())
	if storeErr != nil && log != nil {
		log.Warn().Err(storeErr).Msg("Shared rate-limit store is unavailable, falling back to per-replica limits")
	}
	if count <= max {
		return func() { releaseAttempt(storeCtx) }, false, nil
	}

	// An attempt that is refused was not a guess. Taking it back keeps the
	// counter at the failures and the attempts still being compared.
	releaseAttempt(storeCtx)
	// The forward-auth path deliberately keeps its original response shape and
	// does not advertise quota state through X-RateLimit-* headers.
	ctx.Set(fiber.HeaderRetryAfter, retryAfterSeconds(retryAfter))
	return nil, true, SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many failed password attempts")
}

// rateLimitPasswordHeaderFailure counts one Stargate-Password failure and
// answers 429 once the count exceeds the login quota. CheckRoute counts
// attempts itself with reservePasswordHeaderAttempt; this covers a password
// error that reaches the failure handler directly.
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
