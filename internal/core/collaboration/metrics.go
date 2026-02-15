package collaboration

import (
	"slices"
	"time"
)

// WorkspaceMetrics holds computed statistics for a collaboration workspace.
type WorkspaceMetrics struct {
	// Agents
	ActiveAgents   int `json:"active_agents"`
	TotalAgents    int `json:"total_agents"`
	OnlineAgents   int `json:"online_agents"`

	// Tasks
	TasksSubmitted int           `json:"tasks_submitted"`
	TasksCompleted int           `json:"tasks_completed"`
	TasksFailed    int           `json:"tasks_failed"`
	TasksCancelled int           `json:"tasks_cancelled"`
	TasksWorking   int           `json:"tasks_working"`
	AvgDuration    time.Duration `json:"avg_duration_ns"`
	P95Duration    time.Duration `json:"p95_duration_ns"`

	// Conflicts
	ConflictsDetected    int `json:"conflicts_detected"`
	ConflictsResolved    int `json:"conflicts_resolved"`
	ConflictsUnresolved  int `json:"conflicts_unresolved"`

	// Context
	ContextVariables int `json:"context_variables"`
	ContextMutations int `json:"context_mutations"`

	// Audit
	AuditEvents int `json:"audit_events"`
}

// MetricsCollector computes workspace metrics from storage.
type MetricsCollector struct {
	storage Storage
}

// NewMetricsCollector creates a metrics collector backed by storage.
func NewMetricsCollector(storage Storage) *MetricsCollector {
	return &MetricsCollector{storage: storage}
}

// Collect computes metrics for a workspace.
func (mc *MetricsCollector) Collect(workspaceID string) (*WorkspaceMetrics, error) {
	ws, err := mc.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}

	m := &WorkspaceMetrics{}

	// Agent metrics
	m.TotalAgents = len(ws.Agents)
	for _, a := range ws.Agents {
		if a.Status == "online" {
			m.OnlineAgents++
		}
	}
	m.ActiveAgents = m.OnlineAgents

	// Task metrics
	tasks, err := mc.storage.LoadTasks(workspaceID)
	if err != nil {
		tasks = []TaskInfo{}
	}
	m.TasksSubmitted = len(tasks)

	var durations []time.Duration
	for _, t := range tasks {
		switch t.Status {
		case TaskCompleted:
			m.TasksCompleted++
			if t.CompletedAt != nil {
				durations = append(durations, t.CompletedAt.Sub(t.CreatedAt))
			}
		case TaskFailed:
			m.TasksFailed++
			if t.CompletedAt != nil {
				durations = append(durations, t.CompletedAt.Sub(t.CreatedAt))
			}
		case TaskCancelled:
			m.TasksCancelled++
		case TaskWorking:
			m.TasksWorking++
		}
	}

	if len(durations) > 0 {
		m.AvgDuration = avgDuration(durations)
		m.P95Duration = percentileDuration(durations, 95)
	}

	// Conflict metrics
	conflicts, err := mc.storage.LoadConflicts(workspaceID)
	if err != nil {
		conflicts = []Conflict{}
	}
	m.ConflictsDetected = len(conflicts)
	for _, c := range conflicts {
		if c.IsResolved() {
			m.ConflictsResolved++
		} else {
			m.ConflictsUnresolved++
		}
	}

	// Context metrics
	variables, err := mc.storage.LoadContext(workspaceID)
	if err != nil {
		variables = []ContextVariable{}
	}
	m.ContextVariables = len(variables)
	if ws.Context != nil {
		m.ContextMutations = len(ws.Context.Mutations())
	}

	// Audit metrics
	events, err := mc.storage.LoadAuditEvents(workspaceID)
	if err != nil {
		events = []AuditEvent{}
	}
	m.AuditEvents = len(events)

	return m, nil
}

// avgDuration computes the average of a slice of durations.
func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// percentileDuration computes the Nth percentile of a slice of durations.
func percentileDuration(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	slices.Sort(sorted)

	idx := (percentile * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
