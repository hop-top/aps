package squad

import "fmt"

// ContextLoad measures the context overhead for a squad.
// Spec ref §157-170.
type ContextLoad struct {
	SquadID           string  `json:"squad_id" yaml:"squad_id"`
	ToolSchemas       int     `json:"tool_schemas" yaml:"tool_schemas"`
	DomainKnowledgeKB float64 `json:"domain_knowledge_kb" yaml:"domain_knowledge_kb"`
	InteractionProtos int     `json:"interaction_protos" yaml:"interaction_protos"`
	SessionMemoryKB   float64 `json:"session_memory_kb" yaml:"session_memory_kb"`
}

// Validate checks required fields.
func (c *ContextLoad) Validate() error {
	if c.SquadID == "" {
		return fmt.Errorf("squad_id is required")
	}
	return nil
}

// TotalKB returns the total estimated context size in KB.
func (c *ContextLoad) TotalKB() float64 {
	return float64(c.ToolSchemas)*2.0 + c.DomainKnowledgeKB +
		float64(c.InteractionProtos)*1.0 + c.SessionMemoryKB
}

// CoordinationKB returns KB consumed by coordination overhead.
func (c *ContextLoad) CoordinationKB() float64 {
	return float64(c.InteractionProtos)*1.0 + c.SessionMemoryKB
}

// CoordinationRatio returns coordination / total. >= 0.5 means topology is wrong.
func (c *ContextLoad) CoordinationRatio() float64 {
	total := c.TotalKB()
	if total == 0 {
		return 0
	}
	return c.CoordinationKB() / total
}

// IsWellScoped returns true if coordination ratio < 0.5.
func (c *ContextLoad) IsWellScoped() bool {
	return c.CoordinationRatio() < 0.5
}
