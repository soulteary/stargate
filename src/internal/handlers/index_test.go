package handlers

import (
	"testing"

	"github.com/gofiber/fiber/v2/middleware/session"
)

// Ensure *session.Store satisfies IndexSessionStore at compile time.
var _ IndexSessionStore = (*session.Store)(nil)

func TestIndexSessionStore_StoreImplementsInterface(t *testing.T) {
	// Compile-time check above; this test just ensures the test file is built.
	t.Helper()
}
