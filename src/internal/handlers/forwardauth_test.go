package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/utils/v2"
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
	t.Setenv("AUTH_HOST", "auth.test.com")
	t.Setenv("PASSWORDS", "plaintext:minimal")
	t.Setenv("STEP_UP_ENABLED", "false")
	t.Setenv("STEP_UP_PATHS", "")
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
	store := session.NewStore(session.Config{
		Extractor:    extractors.FromCookie(auth.SessionCookieName),
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

// TestInitForwardAuthHandler_WithStepUpPaths verifies InitForwardAuthHandler runs with step-up paths
// (covers parseStepUpPaths: comma-separated, trimmed).
func TestInitForwardAuthHandler_WithStepUpPaths(t *testing.T) {
	setupStepUpTestWithPaths(t, " /admin*, /api/secret*, ")
	testLog := testLogger()
	InitForwardAuthHandler(testLog)
	h := GetForwardAuthHandler()
	if h == nil {
		t.Error("GetForwardAuthHandler() must not be nil")
	}
}

// TestForwardAuthLogger_InfoWarnErrorAndFields exercises the forwardAuthLogger wrapper so that
// Info(), Warn(), Error() and Bool(), Int(), Int64(), Dur() are covered (zerolog adapter for forwardauth-kit).
func TestForwardAuthLogger_InfoWarnErrorAndFields(t *testing.T) {
	l := &forwardAuthLogger{log: testLogger()}

	l.Info().Str("key", "val").Msg("info message")
	l.Warn().Bool("enabled", true).Msg("warn message")
	l.Error().Err(errors.New("test err")).Int("code", 400).Int64("count", 1).Dur("latency", 10*time.Millisecond).Msg("error message")
}
