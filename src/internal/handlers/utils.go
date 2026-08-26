// Package handlers provides HTTP request handlers for authentication and authorization.
//
// Note: Some utility functions in this file (GetForwardedHost, GetForwardedProto, etc.)
// have counterparts in forwardauth-kit (ForwardedHeaders type). These Fiber-specific
// implementations are kept for use in login.go and other handlers that work directly
// with fiber.Ctx. The forwardauth-kit equivalents are used in the ForwardAuth check
// handler through the Context interface abstraction.
package handlers

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
)

// CallbackCookieName stores the cookie name for origin domain
const CallbackCookieName = "stargate_callback"

func trustForwardedHeaders(ctx fiber.Ctx) bool {
	return strings.TrimSpace(config.TrustedProxies.String()) != "" && ctx.IsProxyTrusted()
}

// GetForwardedHost returns the forwarded hostname from the request.
// It prioritizes the X-Forwarded-Host header if present, otherwise falls back to the request's Hostname.
//
// This is useful when the application is behind a reverse proxy (like Traefik)
// that forwards the original hostname via headers.
func GetForwardedHost(ctx fiber.Ctx) string {
	if trustForwardedHeaders(ctx) {
		forwardedHost := strings.TrimSpace(strings.Split(ctx.Get("X-Forwarded-Host"), ",")[0])
		if forwardedHost != "" {
			return forwardedHost
		}
	}
	return string(ctx.Request().URI().Host())
}

// GetForwardedURI returns the forwarded URI from the request.
// It prioritizes the X-Forwarded-Uri header if present, otherwise falls back to the request's Path.
func GetForwardedURI(ctx fiber.Ctx) string {
	if trustForwardedHeaders(ctx) {
		forwardedURI := ctx.Get("X-Forwarded-Uri")
		if forwardedURI != "" {
			return forwardedURI
		}
	}
	return ctx.Path()
}

// GetForwardedProto returns the forwarded protocol from the request.
// It prioritizes the X-Forwarded-Proto header if present, otherwise falls back to the request's scheme.
//
// This is useful for determining whether the original request was HTTP or HTTPS
// when behind a reverse proxy.
func GetForwardedProto(ctx fiber.Ctx) string {
	if trustForwardedHeaders(ctx) {
		forwardedProto := strings.ToLower(strings.TrimSpace(strings.Split(ctx.Get("X-Forwarded-Proto"), ",")[0]))
		if forwardedProto == "http" || forwardedProto == "https" {
			return forwardedProto
		}
	}
	// Fiber's Scheme also consults forwarded headers when the application-level
	// proxy configuration trusts the peer. Avoid that path here when Stargate's
	// own TRUSTED_PROXIES setting disables forwarded headers.
	if ctx.RequestCtx().IsTLS() {
		return "https"
	}
	return "http"
}

// IsDifferentDomain checks if the origin host is different from the auth host.
// This is used to determine if we need to store the callback in a cookie.
func IsDifferentDomain(ctx fiber.Ctx) bool {
	originHost := GetForwardedHost(ctx)
	authHost := config.AuthHost.String()

	// Normalize domain (remove port number)
	originHost = normalizeHost(originHost)
	authHost = normalizeHost(authHost)

	return originHost != authHost
}

// normalizeHost removes port number from hostname for comparison.
func normalizeHost(host string) string {
	parsed, err := parseCallbackHost(host)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix((&url.URL{Host: parsed}).Hostname(), "."))
}

func parseCallbackHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\\\r\n\t") {
		return "", fmt.Errorf("invalid callback host")
	}

	value := raw
	if !strings.Contains(value, "://") {
		value = "//" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid callback host")
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid callback scheme")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("callback paths are not allowed")
	}

	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" || net.ParseIP(hostname) != nil {
		return "", fmt.Errorf("callback host must be a DNS name")
	}
	if strings.Contains(parsed.Host, ":") {
		if _, _, err := net.SplitHostPort(parsed.Host); err != nil {
			return "", fmt.Errorf("invalid callback port")
		}
	}
	return parsed.Host, nil
}

// ValidateCallbackHost returns a canonical callback host only when it is explicitly
// allowed or belongs to the configured cookie domain.
func ValidateCallbackHost(raw string) (string, error) {
	host, err := parseCallbackHost(raw)
	if err != nil {
		return "", err
	}
	hostname := strings.ToLower(strings.TrimSuffix((&url.URL{Host: host}).Hostname(), "."))

	authHost, err := parseCallbackHost(config.AuthHost.String())
	if err == nil && strings.EqualFold(host, authHost) {
		return host, nil
	}

	for _, allowed := range strings.Split(config.CallbackAllowedHosts.String(), ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		canonical, parseErr := parseCallbackHost(allowed)
		if parseErr == nil && strings.EqualFold(host, canonical) {
			return host, nil
		}
	}

	cookieDomain := strings.ToLower(strings.Trim(strings.TrimSpace(config.CookieDomain.String()), "."))
	if cookieDomain != "" && (hostname == cookieDomain || strings.HasSuffix(hostname, "."+cookieDomain)) {
		return host, nil
	}

	return "", fmt.Errorf("callback host is not allowed")
}

// SetCallbackCookie stores the origin host in a cookie if it's different from the auth host.
// This allows the callback to persist even if the user refreshes the login page.
func SetCallbackCookie(ctx fiber.Ctx, callbackHost string) {
	if callbackHost == "" {
		return
	}

	// Never persist an untrusted redirect target.
	callbackHost, err := ValidateCallbackHost(callbackHost)
	if err != nil {
		return
	}
	authHost := normalizeHost(config.AuthHost.String())

	// Only set cookie if domains are different
	if callbackHost != authHost {
		cookie := &fiber.Cookie{
			Name:     CallbackCookieName,
			Value:    callbackHost,
			Expires:  time.Now().Add(10 * time.Minute), // 10 minutes expiration, enough to complete login flow
			SameSite: fiber.CookieSameSiteLaxMode,
			HTTPOnly: true,
			Secure:   GetForwardedProto(ctx) == "https",
		}

		// If Cookie domain is configured, set it
		if config.CookieDomain.Value != "" {
			cookie.Domain = config.CookieDomain.Value
		}

		ctx.Cookie(cookie)
	}
}

// GetCallbackFromCookie retrieves the callback host from cookie.
func GetCallbackFromCookie(ctx fiber.Ctx) string {
	return ctx.Cookies(CallbackCookieName)
}

// ClearCallbackCookie removes the callback cookie.
func ClearCallbackCookie(ctx fiber.Ctx) {
	cookie := &fiber.Cookie{
		Name:     CallbackCookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Set to past time to delete cookie
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
	}

	// If Cookie domain is configured, set it
	if config.CookieDomain.Value != "" {
		cookie.Domain = config.CookieDomain.Value
	}

	ctx.Cookie(cookie)
}

// BuildCallbackURL constructs a callback URL for authentication redirects.
// It uses X-Forwarded-* headers to build the correct URL with protocol and host.
//
// The URL format is: {protocol}://{authHost}/_login?callback={originalHost}
func BuildCallbackURL(ctx fiber.Ctx) string {
	callbackHost := GetForwardedHost(ctx)
	proto := GetForwardedProto(ctx)
	authHost := config.AuthHost.String()

	// If origin domain is different from auth service domain, store origin domain in cookie
	if canonical, err := ValidateCallbackHost(callbackHost); err == nil && IsDifferentDomain(ctx) {
		callbackHost = canonical
		SetCallbackCookie(ctx, callbackHost)
	} else if err != nil {
		callbackHost = ""
	}

	loginURL := url.URL{Scheme: proto, Host: authHost, Path: "/_login"}
	if callbackHost != "" {
		query := loginURL.Query()
		query.Set("callback", callbackHost)
		loginURL.RawQuery = query.Encode()
	}
	return loginURL.String()
}

// IsHTMLRequest checks if the request accepts HTML responses.
// It examines the Accept header to determine if the client expects HTML content.
//
// Returns true if:
//   - Accept header is empty (defaults to HTML)
//   - Accept header contains "text/html"
//   - Accept header starts with "*/*" (accepts all types)
//
// This is used to determine whether to redirect to a login page (HTML) or return an error response (API).
func IsHTMLRequest(ctx fiber.Ctx) bool {
	acceptHeader := ctx.Get("Accept")
	if acceptHeader == "" {
		return true // Default to HTML request
	}

	acceptParts := strings.Split(acceptHeader, ",")
	for i, acceptPart := range acceptParts {
		format := strings.Trim(strings.SplitN(acceptPart, ";", 2)[0], " ")
		if format == "text/html" || (i == 0 && format == "*/*") {
			return true
		}
	}
	return false
}

// SendErrorResponse sends an error response in the format preferred by the client.
// It automatically detects the best response format based on the Accept header:
//   - application/json -> JSON format with error object
//   - application/xml -> XML format with error element
//   - default -> plain text
//
// Parameters:
//   - ctx: Fiber context
//   - statusCode: HTTP status code (e.g., 401, 500)
//   - message: Error message to send
//
// Returns an error if the response cannot be sent.
func SendErrorResponse(ctx fiber.Ctx, statusCode int, message string) error {
	acceptHeader := ctx.Get("Accept")
	bestFormat := ""

	// Detect best response format
	acceptParts := strings.Split(acceptHeader, ",")
	for _, acceptPart := range acceptParts {
		format := strings.Trim(strings.SplitN(acceptPart, ";", 2)[0], " ")
		if strings.HasPrefix(format, "application/json") {
			bestFormat = "json"
			break
		} else if strings.HasPrefix(format, "application/xml") {
			bestFormat = "xml"
			break
		}
	}

	switch bestFormat {
	case "json":
		ctx.Set("Content-Type", "application/json")
		return ctx.Status(statusCode).JSON(fiber.Map{
			"error": message,
			"code":  statusCode,
		}, fiber.MIMEApplicationJSON)
	case "xml":
		ctx.Set("Content-Type", "application/xml")
		return ctx.Status(statusCode).SendString(`<errors><error code="` + fmt.Sprintf("%d", statusCode) + `">` + message + `</error></errors>`)
	default:
		ctx.Set("Content-Type", "text/plain")
		return ctx.Status(statusCode).SendString(message)
	}
}
