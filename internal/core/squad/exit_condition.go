package squad

import (
	"fmt"
	"time"
)

// ExitCondition defines when an enabling squad's engagement ends.
// Spec ref §46-64, §146-154.
type ExitCondition struct {
	SquadID          string     `json:"squad_id" yaml:"squad_id"`
	TargetSquad      string     `json:"target_squad" yaml:"target_squad"`
	Criteria         string     `json:"criteria" yaml:"criteria"`
	Deadline         time.Time  `json:"deadline" yaml:"deadline"`
	HandoffArtifacts []string   `json:"handoff_artifacts,omitempty" yaml:"handoff_artifacts,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	CompletionNote   string     `json:"completion_note,omitempty" yaml:"completion_note,omitempty"`
}

// Validate checks required fields.
func (e *ExitCondition) Validate() error {
	if e.SquadID == "" {
		return fmt.Errorf("squad_id is required")
	}
	if e.TargetSquad == "" {
		return fmt.Errorf("target_squad is required")
	}
	if e.Criteria == "" {
		return fmt.Errorf("criteria is required")
	}
	if e.Deadline.IsZero() {
		return fmt.Errorf("deadline is required")
	}
	return nil
}

// IsComplete returns true if the exit condition has been satisfied.
func (e *ExitCondition) IsComplete() bool {
	return e.CompletedAt != nil
}

// Complete marks the exit condition as done with the given note.
func (e *ExitCondition) Complete(note string) {
	now := time.Now()
	e.CompletedAt = &now
	e.CompletionNote = note
}

// IsOverdue returns true if deadline has passed and condition is not complete.
func (e *ExitCondition) IsOverdue() bool {
	return time.Now().After(e.Deadline) && !e.IsComplete()
}
