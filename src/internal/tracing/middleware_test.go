package tracing

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/requestcontext"
	common_tracing "github.com/soulteary/tracing-kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingMiddleware(t *testing.T) {
	defer common_tracing.TeardownTestTracer()
	_, _ = common_tracing.SetupTestTracer(t)

	app := fiber.New()
	app.Use(TracingMiddleware("test-service"))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Get("/error", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "test error")
	})

	t.Run("Success Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Check if trace headers are injected
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Error Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		// Check if trace headers are injected
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Bad Request", func(t *testing.T) {
		app.Get("/bad", func(c fiber.Ctx) error {
			return c.SendStatus(400)
		})

		req := httptest.NewRequest("GET", "/bad", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("traceparent"))
	})

	t.Run("Root Path", func(t *testing.T) {
		app.Get("/", func(c fiber.Ctx) error {
			return c.SendString("root")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

type tracingContextKey struct{}

func TestTracingMiddlewarePreservesDeadlineCancellationAndValues(t *testing.T) {
	defer common_tracing.TeardownTestTracer()
	_, _ = common_tracing.SetupTestTracer(t)

	var captured context.Context
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.SetContext(context.WithValue(c.Context(), tracingContextKey{}, "preserved"))
		return c.Next()
	})
	app.Use(requestcontext.Middleware(time.Second))
	app.Use(TracingMiddleware("test-service"))
	app.Get("/context", func(c fiber.Ctx) error {
		captured = c.Context()
		deadline, ok := captured.Deadline()
		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(time.Second), deadline, 200*time.Millisecond)
		assert.Equal(t, "preserved", captured.Value(tracingContextKey{}))
		assert.True(t, trace.SpanFromContext(captured).SpanContext().IsValid())
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/context", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	require.NotNil(t, captured)
	select {
	case <-captured.Done():
		assert.ErrorIs(t, captured.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("traced request context was not canceled after handler return")
	}
}

// TestHeaderCarrier_Keys covers headerCarrier.Keys() (OpenTelemetry TextMapCarrier) for coverage.
func TestHeaderCarrier_Keys(t *testing.T) {
	carrier := &headerCarrier{headers: map[string]string{"a": "1", "b": "2"}}
	assert.Empty(t, carrier.Get("x"))
	assert.Equal(t, "1", carrier.Get("a"))
	carrier.Set("c", "3")
	keys := carrier.Keys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
	assert.Contains(t, keys, "c")
}

func TestSanitizeTraceURLRemovesSensitiveQueryValues(t *testing.T) {
	raw := "/_session/exchange?ticket=secret-ticket&callback=app.example.com"
	sanitized := sanitizeTraceURL(raw)

	assert.Equal(t, "/_session/exchange", sanitized)
	assert.NotContains(t, sanitized, "secret-ticket")
	assert.Equal(t, "/", sanitizeTraceURL("?ticket=secret-ticket"))
}
