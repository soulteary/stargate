package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/tracing-kit/v2/tracingtest"
	"github.com/stretchr/testify/require"
)

// With tracing enabled the health probes and the metrics scrape are left out
// of traces, as they are left out of request logs.
func TestProbeAndMetricsRoutesAreNotTraced(t *testing.T) {
	initLogger()
	setupTestConfig(t)
	original := config.OTLPEnabled.Value
	t.Cleanup(func() { config.OTLPEnabled.Value = original })
	config.OTLPEnabled.Value = "true"
	_, exporter := tracingtest.Setup(t)

	app := fiber.New()
	setupMiddleware(app)
	for _, path := range append([]string{"/_auth"}, unobservedPaths...) {
		app.Get(path, func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	}

	require.ElementsMatch(t, []string{"/healthz", "/readyz", "/health", "/metrics"}, unobservedPaths)
	for _, path := range unobservedPaths {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
		require.Empty(t, resp.Header.Get("traceparent"), path)
	}
	require.Empty(t, exporter.GetSpans())

	resp, err := app.Test(httptest.NewRequest("GET", "/_auth", nil))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Header.Get("traceparent"))
	require.Len(t, exporter.GetSpans(), 1)
}
