package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// SessionGetter defines an interface for getting sessions from a context.
// This interface allows for easier testing by enabling mock implementations.
type SessionGetter interface {
	Get(ctx *fiber.Ctx) (*session.Session, error)
}

// SessionStoreAdapter wraps a session.Store to implement SessionGetter interface.
type SessionStoreAdapter struct {
	store *session.Store
}

// Get retrieves a session from the store.
func (a *SessionStoreAdapter) Get(ctx *fiber.Ctx) (*session.Session, error) {
	return a.store.Get(ctx)
}

// Unauthenticator defines an interface for unauthenticating sessions.
// This interface allows for easier testing by enabling mock implementations.
type Unauthenticator interface {
	Unauthenticate(sess *session.Session) error
}

// AuthUnauthenticator wraps auth.Unauthenticate to implement Unauthenticator interface.
type AuthUnauthenticator struct{}

// Unauthenticate destroys a session.
func (a *AuthUnauthenticator) Unauthenticate(sess *session.Session) error {
	return auth.Unauthenticate(sess)
}

// logoutHandler is the internal handler that can be tested with mocked dependencies.
func logoutHandler(ctx *fiber.Ctx, sessionGetter SessionGetter, unauthenticator Unauthenticator) error {
	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
	}

	err = unauthenticator.Unauthenticate(sess)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
	}

	return ctx.SendString("Logged out")
}

// LogoutRoute handles GET requests to /_logout for user logout.
// It destroys the user's session and returns a confirmation message.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LogoutRoute(store *session.Store) func(c *fiber.Ctx) error {
	sessionGetter := &SessionStoreAdapter{store: store}
	unauthenticator := &AuthUnauthenticator{}
	return func(ctx *fiber.Ctx) error {
		return logoutHandler(ctx, sessionGetter, unauthenticator)
	}
}
