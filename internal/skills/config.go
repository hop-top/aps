package skills

// Config represents skill-related configuration
type Config struct {
	// Enabled controls whether skills are enabled
	Enabled bool `yaml:"enabled"`

	// SkillSources are additional paths to search for skills
	SkillSources []string `yaml:"skill_sources,omitempty"`

	// AutoDetectIDEPaths automatically discovers IDE skill directories
	AutoDetectIDEPaths bool `yaml:"auto_detect_ide_paths"`

	// SecretReplacement configures how secrets are injected into skill executions
	SecretReplacement SecretReplacementConfig `yaml:"secret_replacement,omitempty"`

	// Telemetry configures skill usage tracking
	Telemetry TelemetryConfig `yaml:"telemetry,omitempty"`
}

// SecretReplacementConfig controls secret injection behavior
type SecretReplacementConfig struct {
	// Enabled controls whether secret replacement is active
	Enabled bool `yaml:"enabled"`

	// LocalModels are Ollama models to use for intelligent secret replacement
	// If empty and local_only is false, falls back to remote model
	LocalModels []string `yaml:"local_models,omitempty"`

	// LocalOnly forces using only local models (fails if unavailable)
	LocalOnly bool `yaml:"local_only"`

	// PlaceholderPattern is the regex pattern to detect secret placeholders
	// Default: \$\{SECRET:([A-Z_]+)\}
	PlaceholderPattern string `yaml:"placeholder_pattern,omitempty"`
}

// TelemetryConfig controls skill usage tracking
type TelemetryConfig struct {
	// Enabled controls whether telemetry is active
	Enabled bool `yaml:"enabled"`

	// EventLog is the file path to write usage events
	// Default: ~/.agents/skills/usage.jsonl
	EventLog string `yaml:"event_log,omitempty"`

	// IncludeMetadata includes skill metadata in events
	IncludeMetadata bool `yaml:"include_metadata"`
}

// DefaultConfig returns default skill configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		SkillSources:       []string{},
		AutoDetectIDEPaths: false, // Opt-in for privacy
		SecretReplacement: SecretReplacementConfig{
			Enabled:            true,
			LocalModels:        []string{"llama3.2:3b", "qwen2.5:3b"},
			LocalOnly:          false,
			PlaceholderPattern: `\$\{SECRET:([A-Z_]+)\}`,
		},
		Telemetry: TelemetryConfig{
			Enabled:         true,
			EventLog:        "", // Will default to ~/.agents/skills/usage.jsonl
			IncludeMetadata: false,
		},
	}
}

// Merge merges another config into this one (other takes precedence)
func (c *Config) Merge(other *Config) {
	if other.Enabled {
		c.Enabled = other.Enabled
	}

	if len(other.SkillSources) > 0 {
		c.SkillSources = append(c.SkillSources, other.SkillSources...)
	}

	if other.AutoDetectIDEPaths {
		c.AutoDetectIDEPaths = other.AutoDetectIDEPaths
	}

	// Merge secret replacement config
	if other.SecretReplacement.Enabled {
		c.SecretReplacement.Enabled = other.SecretReplacement.Enabled
	}
	if len(other.SecretReplacement.LocalModels) > 0 {
		c.SecretReplacement.LocalModels = other.SecretReplacement.LocalModels
	}
	if other.SecretReplacement.LocalOnly {
		c.SecretReplacement.LocalOnly = other.SecretReplacement.LocalOnly
	}
	if other.SecretReplacement.PlaceholderPattern != "" {
		c.SecretReplacement.PlaceholderPattern = other.SecretReplacement.PlaceholderPattern
	}

	// Merge telemetry config
	if other.Telemetry.Enabled {
		c.Telemetry.Enabled = other.Telemetry.Enabled
	}
	if other.Telemetry.EventLog != "" {
		c.Telemetry.EventLog = other.Telemetry.EventLog
	}
	if other.Telemetry.IncludeMetadata {
		c.Telemetry.IncludeMetadata = other.Telemetry.IncludeMetadata
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validation can be added here as needed
	return nil
}
