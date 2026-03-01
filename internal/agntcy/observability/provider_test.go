package observability

import (
	"context"
	"testing"

	"hop.top/aps/internal/core"
)

func TestInitTracer_Stdout(t *testing.T) {
	cfg := &core.ObservabilityConfig{
		Exporter:     "stdout",
		SamplingRate: 1.0,
	}

	if err := InitTracer(cfg, "test-profile"); err != nil {
		t.Fatalf("InitTracer(stdout) failed: %v", err)
	}
	defer Shutdown(context.Background())

	if tracerProvider == nil {
		t.Fatal("expected tracerProvider to be set")
	}
}

func TestInitTracer_None(t *testing.T) {
	cfg := &core.ObservabilityConfig{
		Exporter: "none",
	}

	if err := InitTracer(cfg, "test-profile"); err != nil {
		t.Fatalf("InitTracer(none) failed: %v", err)
	}

	// tracerProvider should remain nil for "none"
}

func TestInitTracer_Invalid(t *testing.T) {
	cfg := &core.ObservabilityConfig{
		Exporter: "invalid-exporter",
	}

	if err := InitTracer(cfg, "test-profile"); err == nil {
		t.Fatal("expected error for invalid exporter")
	}
}

func TestInitTracer_NilConfig(t *testing.T) {
	if err := InitTracer(nil, "test-profile"); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestInitMeter(t *testing.T) {
	cfg := &core.ObservabilityConfig{
		Exporter: "stdout",
	}

	if err := InitMeter(cfg, "test-profile"); err != nil {
		t.Fatalf("InitMeter failed: %v", err)
	}
	defer Shutdown(context.Background())

	if meterProvider == nil {
		t.Fatal("expected meterProvider to be set")
	}
}

func TestShutdown_Clean(t *testing.T) {
	// Reset global state
	tracerProvider = nil
	meterProvider = nil

	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown with nil providers failed: %v", err)
	}
}

func TestTracer(t *testing.T) {
	tr := Tracer("test")
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestMeter(t *testing.T) {
	m := Meter("test")
	if m == nil {
		t.Fatal("expected non-nil meter")
	}
}
