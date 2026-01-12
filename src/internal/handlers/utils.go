// Package handlers provides HTTP request handlers for authentication and authorization.
package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/soulteary/stargate/src/internal/config"
)

// CallbackCookieName 存储来源域名的 cookie 名称
const CallbackCookieName = "stargate_callback"

// GetForwardedHost returns the forwarded hostname from the request.
// It prioritizes the X-Forwarded-Host header if present, otherwise falls back to the request's Hostname.
//
// This is useful when the application is behind a reverse proxy (like Traefik)
// that forwards the original hostname via headers.
func GetForwardedHost(ctx *fiber.Ctx) string {
	forwardedHost := ctx.Get("X-Forwarded-Host")
	if forwardedHost != "" {
		return forwardedHost
	}
	return ctx.Hostname()
}

// GetForwardedURI returns the forwarded URI from the request.
// It prioritizes the X-Forwarded-Uri header if present, otherwise falls back to the request's Path.
func GetForwardedURI(ctx *fiber.Ctx) string {
	forwardedURI := ctx.Get("X-Forwarded-Uri")
	if forwardedURI != "" {
		return forwardedURI
	}
	return ctx.Path()
}

// GetForwardedProto returns the forwarded protocol from the request.
// It prioritizes the X-Forwarded-Proto header if present, otherwise falls back to the request's Protocol.
//
// This is useful for determining whether the original request was HTTP or HTTPS
// when behind a reverse proxy.
func GetForwardedProto(ctx *fiber.Ctx) string {
	forwardedProto := ctx.Get("X-Forwarded-Proto")
	if forwardedProto != "" {
		return forwardedProto
	}
	return ctx.Protocol()
}

// IsDifferentDomain checks if the origin host is different from the auth host.
// This is used to determine if we need to store the callback in a cookie.
func IsDifferentDomain(ctx *fiber.Ctx) bool {
	originHost := GetForwardedHost(ctx)
	authHost := config.AuthHost.String()

	// 规范化域名（去除端口号）
	originHost = normalizeHost(originHost)
	authHost = normalizeHost(authHost)

	return originHost != authHost
}

// normalizeHost removes port number from hostname for comparison.
func normalizeHost(host string) string {
	// 如果包含端口号，只取主机名部分
	if idx := strings.Index(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// SetCallbackCookie stores the origin host in a cookie if it's different from the auth host.
// This allows the callback to persist even if the user refreshes the login page.
func SetCallbackCookie(ctx *fiber.Ctx, callbackHost string) {
	if callbackHost == "" {
		return
	}

	// 规范化域名
	callbackHost = normalizeHost(callbackHost)
	authHost := normalizeHost(config.AuthHost.String())

	// 只有当域名不一致时才设置 cookie
	if callbackHost != authHost {
		cookie := &fiber.Cookie{
			Name:     CallbackCookieName,
			Value:    callbackHost,
			Expires:  time.Now().Add(10 * time.Minute), // 10 分钟过期，足够完成登录流程
			SameSite: fiber.CookieSameSiteLaxMode,
			HTTPOnly: true,
			Secure:   GetForwardedProto(ctx) == "https",
		}

		// 如果配置了 Cookie 域名，则设置
		if config.CookieDomain.Value != "" {
			cookie.Domain = config.CookieDomain.Value
		}

		ctx.Cookie(cookie)
	}
}

// GetCallbackFromCookie retrieves the callback host from cookie.
func GetCallbackFromCookie(ctx *fiber.Ctx) string {
	return ctx.Cookies(CallbackCookieName)
}

// ClearCallbackCookie removes the callback cookie.
func ClearCallbackCookie(ctx *fiber.Ctx) {
	cookie := &fiber.Cookie{
		Name:     CallbackCookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // 设置为过去的时间以删除 cookie
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
	}

	// 如果配置了 Cookie 域名，则设置
	if config.CookieDomain.Value != "" {
		cookie.Domain = config.CookieDomain.Value
	}

	ctx.Cookie(cookie)
}

// BuildCallbackURL constructs a callback URL for authentication redirects.
// It uses X-Forwarded-* headers to build the correct URL with protocol and host.
//
// The URL format is: {protocol}://{authHost}/_login?callback={originalHost}
func BuildCallbackURL(ctx *fiber.Ctx) string {
	callbackHost := GetForwardedHost(ctx)
	proto := GetForwardedProto(ctx)
	authHost := config.AuthHost.String()

	// 如果来源域名与认证服务域名不一致，在 cookie 中存储来源域名
	if IsDifferentDomain(ctx) {
		SetCallbackCookie(ctx, callbackHost)
	}

	return fmt.Sprintf("%s://%s/_login?callback=%s", proto, authHost, callbackHost)
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
func IsHTMLRequest(ctx *fiber.Ctx) bool {
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
func SendErrorResponse(ctx *fiber.Ctx, statusCode int, message string) error {
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
		})
	case "xml":
		ctx.Set("Content-Type", "application/xml")
		return ctx.Status(statusCode).SendString(`<errors><error code="` + fmt.Sprintf("%d", statusCode) + `">` + message + `</error></errors>`)
	default:
		ctx.Set("Content-Type", "text/plain")
		return ctx.Status(statusCode).SendString(message)
	}
}
