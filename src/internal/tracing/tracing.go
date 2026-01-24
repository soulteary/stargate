package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
)

// InitTracer initializes OpenTelemetry tracer
func InitTracer(serviceName, serviceVersion, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
		resource.WithFromEnv(), // Automatically detect resource attributes from environment
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider options
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // For production, use TraceIDRatioBased
	}

	if otlpEndpoint != "" {
		// Use OTLP HTTP exporter
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(otlpEndpoint),
			otlptracehttp.WithInsecure(), // For development, use WithTLSClientConfig in production
		)
		otlpExporter, err := otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(otlpExporter))
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(opts...)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracerProvider = tp
	tracer = tp.Tracer(serviceName)

	return tp, nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	if tracer == nil {
		// Return noop tracer if not initialized
		return noop.NewTracerProvider().Tracer("stargate")
	}
	return tracer
}

// IsEnabled returns whether tracing is enabled
func IsEnabled() bool {
	return tracerProvider != nil && tracer != nil
}
