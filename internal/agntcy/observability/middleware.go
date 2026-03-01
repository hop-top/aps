package observability

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// A2AInterceptorHook starts a span for an A2A method call.
// Returns the enriched context and the span (caller must end the span).
func A2AInterceptorHook(ctx context.Context, method string) (context.Context, trace.Span) {
	tracer := Tracer("aps.a2a")
	ctx, span := tracer.Start(ctx, "a2a."+method,
		trace.WithAttributes(
			attribute.String("a2a.method", method),
		),
	)
	return ctx, span
}

// PropagateTraceContext injects the current trace context into HTTP headers.
func PropagateTraceContext(ctx context.Context, headers http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

// ExtractTraceContext extracts trace context from incoming HTTP headers.
func ExtractTraceContext(ctx context.Context, headers http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

// SpanFromContext returns the current span from context, or nil if none.
func SpanFromContext(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	return span
}
