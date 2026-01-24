package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer(t *testing.T) {
	// Test initialization
	tp, err := InitTracer("test-service", "1.0.0", "")
	assert.NoError(t, err)
	assert.NotNil(t, tp)
	assert.True(t, IsEnabled())

	// Test GetTracer
	tracer := GetTracer()
	assert.NotNil(t, tracer)

	// Test Shutdown
	err = Shutdown(context.Background())
	assert.NoError(t, err)

	// Test InitTracer with endpoint
	// This might fail to connect but should execute the code path
	// We expect an error or success depending on whether it tries to connect immediately
	// otlptracehttp usually connects lazily or we can ignore the error if it's about connection
	_, _ = InitTracer("test-service", "1.0.0", "http://localhost:4318")
}

func TestPropagation(t *testing.T) {
	// Setup tracer provider for this test to ensure we have a working tracer
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Setup propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create a context with a span
	ctx := context.Background()
	tracer := otel.Tracer("test-tracer")
	ctx, span := tracer.Start(ctx, "test-span")

	// Inject
	headers := make(map[string]string)
	InjectTraceContext(ctx, headers)
	// Even if not sampled, TraceContext should inject traceparent
	assert.NotEmpty(t, headers["traceparent"])

	// Extract
	newCtx := ExtractTraceContext(context.Background(), headers)
	remoteSpan := trace.SpanFromContext(newCtx)
	assert.True(t, remoteSpan.SpanContext().IsValid())
	assert.Equal(t, span.SpanContext().TraceID(), remoteSpan.SpanContext().TraceID())

	span.End()
}

func TestSpanOperations(t *testing.T) {
	// Setup tracer
	_, err := InitTracer("test-service", "1.0.0", "")
	assert.NoError(t, err)
	ctx := context.Background()

	t.Run("StartSpan", func(t *testing.T) {
		newCtx, span := StartSpan(ctx, "test-span")
		assert.NotNil(t, span)
		assert.NotNil(t, newCtx)
		span.End()
	})

	t.Run("SetSpanAttributes", func(t *testing.T) {
		_, span := StartSpan(ctx, "test-span")
		defer span.End()

		attrs := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		SetSpanAttributes(span, attrs)
	})

	t.Run("SetSpanAttributesFromMap", func(t *testing.T) {
		_, span := StartSpan(ctx, "test-span")
		defer span.End()

		attrs := map[string]interface{}{
			"string": "value",
			"int":    123,
			"int64":  int64(456),
			"float":  12.34,
			"bool":   true,
			"other":  []string{"a", "b"},
		}
		SetSpanAttributesFromMap(span, attrs)
	})

	t.Run("RecordError", func(t *testing.T) {
		_, span := StartSpan(ctx, "test-span")
		defer span.End()

		RecordError(span, assert.AnError)
	})

	t.Run("SetSpanStatus", func(t *testing.T) {
		_, span := StartSpan(ctx, "test-span")
		defer span.End()

		SetSpanStatus(span, codes.Error, "something went wrong")
	})

	t.Run("GetSpanFromContext", func(t *testing.T) {
		newCtx, span := StartSpan(ctx, "test-span")
		defer span.End()

		retrievedSpan := GetSpanFromContext(newCtx)
		assert.Equal(t, span, retrievedSpan)
	})
}
