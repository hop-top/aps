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
