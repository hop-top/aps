package squad

import (
	"fmt"
	"time"
)

// Contract defines an interaction contract between two squads.
// Spec ref §109-154.
type Contract struct {
	ID                string          `json:"id" yaml:"id"`
	ProviderSquad     string          `json:"provider_squad" yaml:"provider_squad"`
	ConsumerSquad     string          `json:"consumer_squad" yaml:"consumer_squad"`
	Mode              InteractionMode `json:"mode" yaml:"mode"`
	Version           string          `json:"version" yaml:"version"`
	InputSchema       map[string]any  `json:"input_schema,omitempty" yaml:"input_schema,omitempty"`
	OutputSchema      map[string]any  `json:"output_schema,omitempty" yaml:"output_schema,omitempty"`
	SLA               SLA             `json:"sla,omitempty" yaml:"sla,omitempty"`
	DeprecationWindow time.Duration   `json:"deprecation_window,omitempty" yaml:"deprecation_window,omitempty"`
	Timebox           *time.Duration  `json:"timebox,omitempty" yaml:"timebox,omitempty"`
	ExitCondition     string          `json:"exit_condition,omitempty" yaml:"exit_condition,omitempty"`
	CreatedAt         time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" yaml:"updated_at"`
}

// SLA defines service level expectations for a contract.
type SLA struct {
	MaxLatency   time.Duration `json:"max_latency,omitempty" yaml:"max_latency,omitempty"`
	Availability float64       `json:"availability,omitempty" yaml:"availability,omitempty"` // 0.0-1.0
}

// Validate checks contract invariants per spec §109-154.
func (c *Contract) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("contract ID is required")
	}
	if c.ProviderSquad == "" {
		return fmt.Errorf("provider squad is required")
	}
	if c.ConsumerSquad == "" {
		return fmt.Errorf("consumer squad is required")
	}
	if err := c.Mode.Validate(); err != nil {
		return err
	}
	if c.Mode == ModeCollaboration && c.Timebox == nil {
		return fmt.Errorf("collaboration contracts require a timebox")
	}
	if c.Mode == ModeFacilitating && c.ExitCondition == "" {
		return fmt.Errorf("facilitating contracts require an exit condition")
	}
	return nil
}
