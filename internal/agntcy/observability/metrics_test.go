package observability

import (
	"testing"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestRegisterMetrics(t *testing.T) {
	m := noop.NewMeterProvider().Meter("test")

	metrics, err := RegisterMetrics(m)
	if err != nil {
		t.Fatalf("RegisterMetrics failed: %v", err)
	}

	if metrics.TasksProcessed == nil {
		t.Fatal("expected TasksProcessed counter")
	}
	if metrics.MessagesRouted == nil {
		t.Fatal("expected MessagesRouted counter")
	}
	if metrics.WebhookDeliveries == nil {
		t.Fatal("expected WebhookDeliveries counter")
	}
	if metrics.TaskDuration == nil {
		t.Fatal("expected TaskDuration histogram")
	}
}
