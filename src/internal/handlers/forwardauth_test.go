package handlers

import (
	"os"
	"testing"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

// TestGetForwardAuthHandler_ReturnsNonNilAfterInit verifies that after InitForwardAuthHandler
// (invoked from TestMain in handlers_test.go), GetForwardAuthHandler returns a non-nil handler.
func TestGetForwardAuthHandler_ReturnsNonNilAfterInit(t *testing.T) {
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil after InitForwardAuthHandler")
	}
}

// TestInitForwardAuthHandler_WithMinimalConfig verifies InitForwardAuthHandler runs without panic
// when given minimal env and config. Can be run in isolation via -run InitForwardAuthHandler.
func TestInitForwardAuthHandler_WithMinimalConfig(t *testing.T) {
	_ = os.Setenv("AUTH_HOST", "auth.test.com")
	_ = os.Setenv("PASSWORDS", "plaintext:minimal")
	testLog := testLogger()
	if err := config.Initialize(testLog); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	InitForwardAuthHandler(testLog)
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil after InitForwardAuthHandler")
	}
}

// TestForwardAuthCheckRoute_ReturnsHandler verifies ForwardAuthCheckRoute returns a non-nil Fiber handler.
func TestForwardAuthCheckRoute_ReturnsHandler(t *testing.T) {
	store := session.New(session.Config{
		KeyLookup:    "cookie:" + auth.SessionCookieName,
		KeyGenerator: utils.UUID,
	})
	if store == nil {
		t.Fatal("session.New returned nil")
	}
	handler := ForwardAuthCheckRoute(store)
	if handler == nil {
		t.Error("ForwardAuthCheckRoute(store) must not return nil")
	}
}
