package squad

import "time"

// EvolutionSignal indicates a topology health concern.
// Spec ref §186-200.
type EvolutionSignal string

const (
	SignalStaleCollaboration EvolutionSignal = "stale_collaboration"
	SignalStaleEnabling      EvolutionSignal = "stale_enabling"
	SignalNarrowInterface    EvolutionSignal = "narrow_interface"
	SignalInterfaceChurn     EvolutionSignal = "interface_churn"
)

// EvolutionThresholds configures when signals fire.
type EvolutionThresholds struct {
	MaxCollabCycles     int `json:"max_collab_cycles" yaml:"max_collab_cycles"`
	MaxEnableCycles     int `json:"max_enable_cycles" yaml:"max_enable_cycles"`
	MinSubsystemCalls   int `json:"min_subsystem_calls" yaml:"min_subsystem_calls"`
	MaxInterfaceChanges int `json:"max_interface_changes" yaml:"max_interface_changes"`
}

// DefaultThresholds returns sensible defaults per spec.
func DefaultThresholds() EvolutionThresholds {
	return EvolutionThresholds{
		MaxCollabCycles:     4,
		MaxEnableCycles:     6,
		MinSubsystemCalls:   5,
		MaxInterfaceChanges: 3,
	}
}

// EvolutionObservation records a detected signal.
type EvolutionObservation struct {
	SquadID           string          `json:"squad_id"`
	Signal            EvolutionSignal `json:"signal"`
	Value             int             `json:"value"`
	Threshold         int             `json:"threshold"`
	ObservedAt        time.Time       `json:"observed_at"`
	RecommendedAction string          `json:"recommended_action"`
}

// EvolutionMonitor checks for topology health signals.
type EvolutionMonitor struct {
	thresholds EvolutionThresholds
}

// NewEvolutionMonitor creates a monitor with the given thresholds.
func NewEvolutionMonitor(t EvolutionThresholds) *EvolutionMonitor {
	return &EvolutionMonitor{thresholds: t}
}

func (m *EvolutionMonitor) observe(squadID string, signal EvolutionSignal, value, threshold int, action string) *EvolutionObservation {
	return &EvolutionObservation{
		SquadID:           squadID,
		Signal:            signal,
		Value:             value,
		Threshold:         threshold,
		ObservedAt:        time.Now(),
		RecommendedAction: action,
	}
}

// CheckCollaboration fires if collaboration cycles exceed threshold.
func (m *EvolutionMonitor) CheckCollaboration(squadID string, cycles int) *EvolutionObservation {
	if cycles > m.thresholds.MaxCollabCycles {
		return m.observe(squadID, SignalStaleCollaboration, cycles, m.thresholds.MaxCollabCycles,
			"evaluate merge or graduate to x-as-a-service")
	}
	return nil
}

// CheckEnabling fires if enabling attachment cycles exceed threshold.
func (m *EvolutionMonitor) CheckEnabling(squadID string, cycles int) *EvolutionObservation {
	if cycles > m.thresholds.MaxEnableCycles {
		return m.observe(squadID, SignalStaleEnabling, cycles, m.thresholds.MaxEnableCycles,
			"redefine exit condition and enforce")
	}
	return nil
}

// CheckSubsystemCalls fires if calls per task exceed threshold (interface too narrow).
func (m *EvolutionMonitor) CheckSubsystemCalls(squadID string, callsPerTask int) *EvolutionObservation {
	if callsPerTask > m.thresholds.MinSubsystemCalls {
		return m.observe(squadID, SignalNarrowInterface, callsPerTask, m.thresholds.MinSubsystemCalls,
			"consider enriching subsystem output")
	}
	return nil
}

// CheckInterfaceChurn fires if interface changes exceed threshold.
func (m *EvolutionMonitor) CheckInterfaceChurn(squadID string, changes int) *EvolutionObservation {
	if changes > m.thresholds.MaxInterfaceChanges {
		return m.observe(squadID, SignalInterfaceChurn, changes, m.thresholds.MaxInterfaceChanges,
			"decouple versioning from internal evolution")
	}
	return nil
}
