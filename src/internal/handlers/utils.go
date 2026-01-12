// Package handlers provides HTTP request handlers for authentication and authorization.
package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/soulteary/stargate/src/internal/config"
)

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

// BuildCallbackURL constructs a callback URL for authentication redirects.
// It uses X-Forwarded-* headers to build the correct URL with protocol and host.
//
// The URL format is: {protocol}://{authHost}/_login?callback={originalHost}
func BuildCallbackURL(ctx *fiber.Ctx) string {
	callbackHost := GetForwardedHost(ctx)
	proto := GetForwardedProto(ctx)
	authHost := config.AuthHost.String()
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
