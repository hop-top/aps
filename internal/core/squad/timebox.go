package squad

import (
	"fmt"
	"time"
)

// Timebox enforces time-bounded collaboration.
// Spec ref §127-138.
type Timebox struct {
	ContractID       string          `json:"contract_id" yaml:"contract_id"`
	StartedAt        time.Time       `json:"started_at" yaml:"started_at"`
	Duration         time.Duration   `json:"duration" yaml:"duration"`
	GraduationTarget InteractionMode `json:"graduation_target" yaml:"graduation_target"`
	Graduated        bool            `json:"graduated" yaml:"graduated"`
	GraduatedAt      *time.Time      `json:"graduated_at,omitempty" yaml:"graduated_at,omitempty"`
}

// Validate checks required fields.
func (t *Timebox) Validate() error {
	if t.ContractID == "" {
		return fmt.Errorf("contract_id is required")
	}
	if t.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if err := t.GraduationTarget.Validate(); err != nil {
		return fmt.Errorf("graduation_target: %w", err)
	}
	return nil
}

// Deadline returns the time when the timebox expires.
func (t *Timebox) Deadline() time.Time {
	return t.StartedAt.Add(t.Duration)
}

// IsExpired returns true if the timebox deadline has passed.
func (t *Timebox) IsExpired() bool {
	return time.Now().After(t.Deadline())
}

// IsActive returns true if not yet graduated and not expired.
func (t *Timebox) IsActive() bool {
	return !t.Graduated && !t.IsExpired()
}

// Graduate marks the timebox as successfully completed.
func (t *Timebox) Graduate() {
	t.Graduated = true
	now := time.Now()
	t.GraduatedAt = &now
}

// Remaining returns the time left in the timebox (min 0).
func (t *Timebox) Remaining() time.Duration {
	d := time.Until(t.Deadline())
	if d < 0 {
		return 0
	}
	return d
}
