package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/metric"

	"hop.top/aps/internal/core"
)

var (
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
)

// InitTracer initializes the OpenTelemetry tracer provider based on config.
func InitTracer(cfg *core.ObservabilityConfig, profileID string) error {
	if cfg == nil {
		return fmt.Errorf("observability config is nil")
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("aps"),
			semconv.ServiceInstanceID(profileID),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	switch cfg.Exporter {
	case "otlp":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "localhost:4317"
		}
		exporter, err = otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	case "none", "":
		return nil
	default:
		return fmt.Errorf("unsupported exporter: %s (use: otlp, stdout, none)", cfg.Exporter)
	}

	samplingRate := cfg.SamplingRate
	if samplingRate <= 0 {
		samplingRate = 1.0
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRate))),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// InitMeter initializes the OpenTelemetry meter provider.
func InitMeter(cfg *core.ObservabilityConfig, profileID string) error {
	if cfg == nil {
		return fmt.Errorf("observability config is nil")
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("aps"),
			semconv.ServiceInstanceID(profileID),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)

	return nil
}

// Shutdown gracefully shuts down both providers.
func Shutdown(ctx context.Context) error {
	var errs []error
	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}
	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Meter returns a named meter from the global provider.
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}
