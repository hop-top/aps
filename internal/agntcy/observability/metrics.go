package observability

import (
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds registered instrument handles.
type Metrics struct {
	TasksProcessed    metric.Int64Counter
	MessagesRouted    metric.Int64Counter
	WebhookDeliveries metric.Int64Counter
	TaskDuration      metric.Float64Histogram
}

// RegisterMetrics creates and registers all APS metrics on the given meter.
func RegisterMetrics(m metric.Meter) (*Metrics, error) {
	tasksProcessed, err := m.Int64Counter("aps.tasks.processed",
		metric.WithDescription("Number of A2A tasks processed"),
	)
	if err != nil {
		return nil, err
	}

	messagesRouted, err := m.Int64Counter("aps.messages.routed",
		metric.WithDescription("Number of messages routed between agents"),
	)
	if err != nil {
		return nil, err
	}

	webhookDeliveries, err := m.Int64Counter("aps.webhooks.deliveries",
		metric.WithDescription("Number of webhook events delivered"),
	)
	if err != nil {
		return nil, err
	}

	taskDuration, err := m.Float64Histogram("aps.tasks.duration",
		metric.WithDescription("Duration of task execution in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		TasksProcessed:    tasksProcessed,
		MessagesRouted:    messagesRouted,
		WebhookDeliveries: webhookDeliveries,
		TaskDuration:      taskDuration,
	}, nil
}
