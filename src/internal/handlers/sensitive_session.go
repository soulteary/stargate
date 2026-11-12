package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
)

// authorizeSensitiveSession rechecks directory identity before factor-management
// and step-up operations, even when background refresh is disabled or not due.
func authorizeSensitiveSession(ctx fiber.Ctx, store *session.Store, sess *session.Session) (bool, error) {
	if !auth.IsAuthenticated(sess) {
		return false, SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
	}
	refreshed, err := refreshDirectoryAuthorization(ctx.Context(), sess, true)
	if err != nil {
		if errors.Is(err, errAuthorizationUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
		}
		return false, SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
	}
	if refreshed {
		if err := saveLiveSession(ctx.Context(), store, sess); err != nil {
			return false, SendErrorResponse(ctx, fiber.StatusInternalServerError, "failed to save directory authorization")
		}
	}
	if wardenSessionSourceExpired(sess) {
		return false, SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
	}
	return true, nil
}
