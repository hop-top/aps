package squad

import (
	"fmt"
	"strings"
	"time"
)

// SquadType represents the team topology type of a squad.
type SquadType string

const (
	SquadTypeStream    SquadType = "stream-aligned"
	SquadTypeEnabling  SquadType = "enabling"
	SquadTypeSubsystem SquadType = "complicated-subsystem"
	SquadTypePlatform  SquadType = "platform"
)

// ValidSquadTypes enumerates all valid squad types.
var ValidSquadTypes = []SquadType{SquadTypeStream, SquadTypeEnabling, SquadTypeSubsystem, SquadTypePlatform}

// Validate checks if the squad type is valid.
func (t SquadType) Validate() error {
	switch t {
	case SquadTypeStream, SquadTypeEnabling, SquadTypeSubsystem, SquadTypePlatform:
		return nil
	default:
		return fmt.Errorf("invalid squad type: %q", t)
	}
}

// InteractionMode represents how squads interact with each other.
type InteractionMode string

const (
	ModeXaaS          InteractionMode = "x-as-a-service"
	ModeCollaboration InteractionMode = "collaboration"
	ModeFacilitating  InteractionMode = "facilitating"
)

// ValidInteractionModes enumerates all valid interaction modes.
var ValidInteractionModes = []InteractionMode{ModeXaaS, ModeCollaboration, ModeFacilitating}

// Validate checks if the interaction mode is valid.
func (m InteractionMode) Validate() error {
	switch m {
	case ModeXaaS, ModeCollaboration, ModeFacilitating:
		return nil
	default:
		return fmt.Errorf("invalid interaction mode: %q", m)
	}
}

// Squad represents a team topology squad with its members and domain.
type Squad struct {
	ID                string    `json:"id" yaml:"id"`
	Name              string    `json:"name" yaml:"name"`
	Type              SquadType `json:"type" yaml:"type"`
	Domain            string    `json:"domain" yaml:"domain"`
	Description       string    `json:"description,omitempty" yaml:"description,omitempty"`
	Members           []string  `json:"members" yaml:"members"` // profile IDs
	GoldenPathDefined bool       `json:"golden_path_defined,omitempty" yaml:"golden_path_defined,omitempty"`
	Scope             *ScopeRule `json:"scope,omitempty" yaml:"scope,omitempty"`
	CreatedAt         time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" yaml:"updated_at"`
}

// Validate checks required fields and type validity.
func (s *Squad) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("squad ID is required")
	}
	if s.Name == "" {
		return fmt.Errorf("squad name is required")
	}
	if err := s.Type.Validate(); err != nil {
		return err
	}
	if len(s.Members) == 0 {
		return fmt.Errorf("squad must have at least one member")
	}
	return nil
}

// OwnsDomain returns true for squad types that own a domain boundary.
func (s *Squad) OwnsDomain() bool {
	return s.Type == SquadTypeStream || s.Type == SquadTypeSubsystem || s.Type == SquadTypePlatform
}

// IsTemporary returns true for squad types that are inherently temporary.
func (s *Squad) IsTemporary() bool {
	return s.Type == SquadTypeEnabling
}

// RequiresExitCondition returns true for squad types that need a defined exit condition.
func (s *Squad) RequiresExitCondition() bool {
	return s.Type == SquadTypeEnabling
}

// matchesDomain performs a case-insensitive contains check on the squad's domain.
func (s *Squad) matchesDomain(domain string) bool {
	return strings.Contains(strings.ToLower(s.Domain), strings.ToLower(domain))
}

// Topology holds a collection of squads.
type Topology struct {
	Squads []Squad `json:"squads" yaml:"squads"`
}
