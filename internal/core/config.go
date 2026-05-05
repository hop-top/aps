package core

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	kitconfig "hop.top/kit/go/core/config"
	"hop.top/kit/go/core/xdg"
)

// Config represents the global configuration for APS
type Config struct {
	Prefix            string                `yaml:"prefix"`
	Isolation         GlobalIsolationConfig `yaml:"isolation,omitempty"`
	CapabilitySources []string              `yaml:"capability_sources,omitempty"`
	Secrets           SecretsConfig         `yaml:"secrets,omitempty"`
	Profile           ProfileDefaultsConfig `yaml:"profile,omitempty"`
}

// ProfileDefaultsConfig controls default behaviour when creating profiles.
// Color is tri-state ("true"|"false"|"auto"); unset == false. Avatar is a
// nested block — see ProfileAvatarConfig.
type ProfileDefaultsConfig struct {
	Color  AutoMode            `yaml:"color,omitempty"`
	Avatar ProfileAvatarConfig `yaml:"avatar,omitempty"`
}

// ProfileAvatarConfig configures the avatar provider used when
// auto-generating avatars on profile create. Enabled is the tri-state
// toggle; the remaining fields are passed through to the kit/avatar
// provider. Empty Provider falls back to kit/avatar's default.
type ProfileAvatarConfig struct {
	Enabled  AutoMode `yaml:"enabled,omitempty"`
	Provider string   `yaml:"provider,omitempty"`
	Style    string   `yaml:"style,omitempty"`
	Size     int      `yaml:"size,omitempty"`
	Format   string   `yaml:"format,omitempty"`
}

// AutoMode is a tri-state toggle ("true" | "false" | "auto").
// Yaml accepts native booleans for ergonomics (true/false) plus the
// string "auto"; anything else is treated as the zero value (false).
type AutoMode string

const (
	AutoModeFalse AutoMode = "false"
	AutoModeTrue  AutoMode = "true"
	AutoModeAuto  AutoMode = "auto"
)

// UnmarshalYAML accepts bool or string ("auto") for AutoMode.
func (m *AutoMode) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!bool":
		if value.Value == "true" {
			*m = AutoModeTrue
		} else {
			*m = AutoModeFalse
		}
	case "!!str":
		switch value.Value {
		case "true":
			*m = AutoModeTrue
		case "auto":
			*m = AutoModeAuto
		default:
			*m = AutoModeFalse
		}
	default:
		*m = AutoModeFalse
	}
	return nil
}

// ShouldAutoAssign reports whether auto-assignment should run given that
// no explicit value was supplied. interactive=true means the user was
// (or could have been) prompted; in that case "auto" defers to the
// (currently empty) value rather than overriding silently.
func (m AutoMode) ShouldAutoAssign(interactive bool) bool {
	switch m {
	case AutoModeTrue:
		return true
	case AutoModeAuto:
		return !interactive
	default:
		return false
	}
}

// SecretsConfig selects the kit/storage/secret backend used for profile secrets.
// Backend "file" (default) keeps the legacy secrets.env per-profile layout; "env"
// reads from process environment with optional Prefix; "keyring" delegates to
// the OS keychain (Service defaults to "aps").
type SecretsConfig struct {
	Backend string `yaml:"backend,omitempty"`
	Service string `yaml:"service,omitempty"`
	Prefix  string `yaml:"prefix,omitempty"`
}

// GlobalIsolationConfig represents global isolation settings
type GlobalIsolationConfig struct {
	DefaultLevel    IsolationLevel `yaml:"default_level"`
	FallbackEnabled bool           `yaml:"fallback_enabled"`
}

// DefaultPrefix is the default prefix for environment variables
const DefaultPrefix = "APS"

// GetConfigDir returns the directory for APS configuration, resolved via
// kit/go/core/xdg (XDG Base Directory Specification with OS-native fallbacks).
func GetConfigDir() (string, error) {
	return xdg.ConfigDir("aps")
}

// LoadConfig loads the global configuration via kit/go/core/config.Load,
// merging system → user → project layers. Malformed YAML returns defaults
// (per spec recommendation).
func LoadConfig() (*Config, error) {
	config := &Config{
		Prefix: DefaultPrefix,
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationProcess,
			FallbackEnabled: true,
		},
	}

	userPath := ""
	if configDir, err := GetConfigDir(); err == nil {
		userPath = filepath.Join(configDir, "config.yaml")
	}

	projectPath := ""
	if cwd, err := os.Getwd(); err == nil {
		projectPath = filepath.Join(cwd, ".aps.yaml")
	}

	if err := kitconfig.Load(config, kitconfig.Options{
		UserConfigPath:    userPath,
		ProjectConfigPath: projectPath,
	}); err != nil {
		// Malformed YAML → fall back to defaults rather than surface an error.
		return &Config{
			Prefix: DefaultPrefix,
			Isolation: GlobalIsolationConfig{
				DefaultLevel:    IsolationProcess,
				FallbackEnabled: true,
			},
			Secrets: SecretsConfig{Backend: SecretsBackendFile},
		}, nil
	}

	if config.Prefix == "" {
		config.Prefix = DefaultPrefix
	}

	if config.Isolation.DefaultLevel == "" {
		config.Isolation.DefaultLevel = IsolationProcess
	} else {
		switch config.Isolation.DefaultLevel {
		case IsolationProcess, IsolationPlatform, IsolationContainer:
		default:
			config.Isolation.DefaultLevel = IsolationProcess
		}
	}

	if config.Secrets.Backend == "" {
		config.Secrets.Backend = SecretsBackendFile
	}

	return config, nil
}

// SaveConfig saves the global configuration to disk
func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// MigrateConfig updates an existing config file to include isolation settings
// Returns true if migration was performed, false if no migration needed
func MigrateConfig() (bool, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return false, err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No existing config to migrate
		}
		return false, err
	}

	var oldConfig struct {
		Prefix string `yaml:"prefix"`
	}

	if err := yaml.Unmarshal(data, &oldConfig); err != nil {
		return false, err
	}

	var newConfig Config
	newConfig.Prefix = oldConfig.Prefix
	if newConfig.Prefix == "" {
		newConfig.Prefix = DefaultPrefix
	}
	newConfig.Isolation = GlobalIsolationConfig{
		DefaultLevel:    IsolationProcess,
		FallbackEnabled: true,
	}

	if err := SaveConfig(&newConfig); err != nil {
		return false, err
	}

	return true, nil
}
