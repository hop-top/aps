package bundle

// Bundle represents a named preset that groups capabilities together.
type Bundle struct {
	Name             string                     `yaml:"name"`
	Description      string                     `yaml:"description"`
	Version          string                     `yaml:"version"`
	Extends          string                     `yaml:"extends,omitempty"`
	Tags             []string                   `yaml:"tags,omitempty"`
	Capabilities     []string                   `yaml:"capabilities,omitempty"`
	Scope            BundleScope                `yaml:"scope,omitempty"`
	Env              map[string]string          `yaml:"env,omitempty"`
	Services         []ServiceEntry             `yaml:"services,omitempty"`
	Requires         []BinaryRequirement        `yaml:"requires,omitempty"`
	RuntimeOverrides map[string]RuntimeOverride `yaml:"runtime_overrides,omitempty"`
}

// BundleScope defines the access scope rules contributed by a bundle.
type BundleScope struct {
	Operations   []string `yaml:"operations,omitempty"`
	FilePatterns []string `yaml:"file_patterns,omitempty"`
	Networks     []string `yaml:"networks,omitempty"`
}

// ServiceEntry describes a service started alongside the profile.
type ServiceEntry struct {
	Name    string `yaml:"name"`
	Adapter string `yaml:"adapter"`
	Start   string `yaml:"start"` // "always" | "on-demand"
}

// BinaryRequirement describes a binary dependency and its enforcement policy.
type BinaryRequirement struct {
	Binary     string   `yaml:"binary"`
	Missing    string   `yaml:"missing,omitempty"` // "skip" | "warn" | "error"
	Blocked    bool     `yaml:"blocked,omitempty"`
	Command    string   `yaml:"command,omitempty"`
	DenyFlags  []string `yaml:"deny_flags,omitempty"`
	DenyPolicy string   `yaml:"deny_policy,omitempty"` // "strip" | "error"
	Message    string   `yaml:"message,omitempty"`
}

// RuntimeOverride holds runtime-specific configuration applied on top of the base bundle config.
type RuntimeOverride struct {
	Env map[string]string `yaml:"env,omitempty"`
}
