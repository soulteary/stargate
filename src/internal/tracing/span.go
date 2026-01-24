package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetTracer()
	return tracer.Start(ctx, name, opts...)
}

// SetSpanAttributes sets attributes on a span
func SetSpanAttributes(span trace.Span, attrs map[string]string) {
	for k, v := range attrs {
		span.SetAttributes(attribute.String(k, v))
	}
}

// SetSpanAttributesFromMap sets attributes from a map with various value types
func SetSpanAttributesFromMap(span trace.Span, attrs map[string]interface{}) {
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			span.SetAttributes(attribute.String(k, val))
		case int:
			span.SetAttributes(attribute.Int(k, val))
		case int64:
			span.SetAttributes(attribute.Int64(k, val))
		case float64:
			span.SetAttributes(attribute.Float64(k, val))
		case bool:
			span.SetAttributes(attribute.Bool(k, val))
		default:
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
}

// RecordError records an error on a span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanStatus sets the status of a span
func SetSpanStatus(span trace.Span, code codes.Code, description string) {
	span.SetStatus(code, description)
}

// GetSpanFromContext retrieves a span from context
func GetSpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
