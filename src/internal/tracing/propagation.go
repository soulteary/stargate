package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
)

// ExtractTraceContext extracts trace context from headers
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := &mapCarrier{headers: headers}
	return propagator.Extract(ctx, carrier)
}

// InjectTraceContext injects trace context into headers
func InjectTraceContext(ctx context.Context, headers map[string]string) {
	propagator := otel.GetTextMapPropagator()
	carrier := &mapCarrier{headers: headers}
	propagator.Inject(ctx, carrier)
}

// mapCarrier implements the TextMapCarrier interface for map[string]string
type mapCarrier struct {
	headers map[string]string
}

func (c *mapCarrier) Get(key string) string {
	return c.headers[key]
}

func (c *mapCarrier) Set(key, value string) {
	c.headers[key] = value
}

func (c *mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}
