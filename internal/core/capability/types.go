package capability

import (
	"time"
)

// Capability represents an installed tool or configuration set
type Capability struct {
	Name        string            `json:"name" yaml:"name"`
	Source      string            `json:"source,omitempty" yaml:"source,omitempty"`
	Path        string            `json:"path" yaml:"path"` // Absolute path in ~/.aps/capabilities
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	InstalledAt time.Time         `json:"installed_at" yaml:"installed_at"`
	Links       map[string]string `json:"links,omitempty" yaml:"links,omitempty"` // Map of Target -> Source
	Type        CapabilityType    `json:"type" yaml:"type"`
}

type CapabilityType string

const (
	TypeManaged   CapabilityType = "managed"   // Installed/Adopted (APS owns it)
	TypeReference CapabilityType = "reference" // Watched (System owns it)
)

// SmartPattern defines a known tool configuration pattern
type SmartPattern struct {
	ToolName    string
	DefaultPath string // Relative path from worksapce root (e.g., .github/agents/...)
	Description string
}
