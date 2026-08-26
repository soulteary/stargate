package handlers

import (
	"encoding/gob"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

// testLoggerCheckHeaders creates a logger instance for testing
func testLoggerCheckHeaders() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "check-headers-test",
	})
}

func setupCheckHeaderConfig(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	testLogger := testLoggerCheckHeaders()
	err := config.Initialize(testLogger)
	testza.AssertNoError(t, err)
	// Initialize ForwardAuth handler for testing
	InitForwardAuthHandler(testLogger)
}

func setTrustedHeaderAuthConfig(t *testing.T) {
	t.Helper()
	originalEnabled := config.HeaderAuthEnabled.Value
	originalSecret := config.HeaderAuthSharedSecret.Value
	originalHeader := config.HeaderAuthSecretHeader.Value
	t.Cleanup(func() {
		config.HeaderAuthEnabled.Value = originalEnabled
		config.HeaderAuthSharedSecret.Value = originalSecret
		config.HeaderAuthSecretHeader.Value = originalHeader
	})
	config.HeaderAuthEnabled.Value = "true"
	config.HeaderAuthSharedSecret.Value = "trusted-proxy-secret"
	config.HeaderAuthSecretHeader.Value = "X-Stargate-Header-Auth"
}

func TestSanitizeTrustedIdentityHeadersRejectsUnsignedHeaders(t *testing.T) {
	setTrustedHeaderAuthConfig(t)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone": "13800138000",
		"X-User-Mail":  "user@example.com",
	}, "")
	defer app.ReleaseCtx(ctx)

	sanitizeTrustedIdentityHeaders(ctx)

	testza.AssertEqual(t, "", ctx.Get("X-User-Phone"))
	testza.AssertEqual(t, "", ctx.Get("X-User-Mail"))
}

func TestSanitizeTrustedIdentityHeadersAcceptsProxyCredential(t *testing.T) {
	setTrustedHeaderAuthConfig(t)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone":           "13800138000",
		"X-Stargate-Header-Auth": "trusted-proxy-secret",
	}, "")
	defer app.ReleaseCtx(ctx)

	sanitizeTrustedIdentityHeaders(ctx)

	testza.AssertEqual(t, "13800138000", ctx.Get("X-User-Phone"))
	testza.AssertEqual(t, "", ctx.Get("X-Stargate-Header-Auth"))
}

func TestCheckRoute_SetsAuthHeadersFromSession(t *testing.T) {
	setupCheckHeaderConfig(t)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept": "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "user-123")
	sess.Set("user_mail", "user@example.com")
	sess.Set("user_name", "Test User")
	sess.Set("user_scope", []string{"read", "write"})
	sess.Set("user_role", "admin")

	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	testza.AssertEqual(t, "user-123", string(ctx.Response().Header.Peek("X-Forwarded-User")))
	testza.AssertEqual(t, "user@example.com", string(ctx.Response().Header.Peek("X-Auth-Email")))
	testza.AssertEqual(t, "user-123", string(ctx.Response().Header.Peek("X-Auth-User")))
	testza.AssertEqual(t, "Test User", string(ctx.Response().Header.Peek("X-Auth-Name")))
	testza.AssertEqual(t, "read,write", string(ctx.Response().Header.Peek("X-Auth-Scopes")))
	testza.AssertEqual(t, "admin", string(ctx.Response().Header.Peek("X-Auth-Role")))
}

func TestCheckRoute_SetsScopesFromInterfaceSlice(t *testing.T) {
	setupCheckHeaderConfig(t)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept": "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	gob.Register([]interface{}{})
	sess.Set("user_id", 123)
	sess.Set("user_mail", "user@example.com")
	sess.Set("user_scope", []interface{}{"read", 123, "write"})

	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	testza.AssertEqual(t, "authenticated", string(ctx.Response().Header.Peek("X-Forwarded-User")))
	testza.AssertEqual(t, "user@example.com", string(ctx.Response().Header.Peek("X-Auth-Email")))
	testza.AssertEqual(t, "", string(ctx.Response().Header.Peek("X-Auth-User")))
	testza.AssertEqual(t, "read,write", string(ctx.Response().Header.Peek("X-Auth-Scopes")))
}
