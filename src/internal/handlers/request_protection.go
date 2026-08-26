package handlers

import (
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RequireSameOrigin blocks browser cross-site writes while preserving clients
// such as curl that do not send browser Origin or Fetch Metadata headers.
func RequireSameOrigin() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
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
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" || !strings.EqualFold(parsed.Hostname(), ctx.Hostname()) {
			return SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected")
		}
		return ctx.Next()
	}
}

func endpointRateLimit(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return ctx.IP() + ":" + ctx.Path()
		},
		LimitReached: func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderRetryAfter, "60")
			return SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many requests")
		},
	})
}

func LoginRateLimit() fiber.Handler {
	return endpointRateLimit(10, time.Minute)
}

func VerificationRateLimit() fiber.Handler {
	return endpointRateLimit(5, time.Minute)
}
