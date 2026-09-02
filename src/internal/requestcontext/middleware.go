// Package requestcontext provides a standard Go request context for Fiber
// handlers and the upstream calls they make.
package requestcontext

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Middleware installs a request-scoped standard Go context with a deadline.
//
// Fiber's Ctx implements context.Context, but its Done channel is nil. The
// underlying fasthttp request context is also not canceled for an ordinary
// client disconnect. This middleware therefore provides a real deadline and
// guarantees cancellation when the handler returns. It does not claim to turn
// client disconnects into per-request cancellation signals.
func Middleware(timeout time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), timeout)
		c.SetContext(ctx)
		defer cancel()
		return c.Next()
	}
}
