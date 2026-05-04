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
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	InstalledAt time.Time         `json:"installed_at" yaml:"installed_at"`
	Links       map[string]string `json:"links,omitempty" yaml:"links,omitempty"` // Map of Target -> Source
	Type        CapabilityType    `json:"type" yaml:"type"`
}

type CapabilityType string

const (
	TypeManaged   CapabilityType = "managed"   // Installed/Adopted (APS owns it)
	TypeReference CapabilityType = "reference" // Watched (System owns it)
)

// CapabilityKind distinguishes builtin vs external capabilities
type CapabilityKind string

const (
	KindBuiltin  CapabilityKind = "builtin"
	KindExternal CapabilityKind = "external"
)

// BuiltinCapability represents a built-in capability (a2a, webhooks, etc.)
type BuiltinCapability struct {
	Name        string
	Description string
	Tags        []string
}

// SmartPattern defines a known tool configuration pattern
type SmartPattern struct {
	ToolName    string
	DefaultPath string // Relative path from worksapce root (e.g., .github/agents/...)
	Description string
}
