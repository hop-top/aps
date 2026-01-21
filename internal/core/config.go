package core

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the global configuration for APS
type Config struct {
	Prefix    string                `yaml:"prefix"`
	Isolation GlobalIsolationConfig `yaml:"isolation,omitempty"`
}

// GlobalIsolationConfig represents global isolation settings
type GlobalIsolationConfig struct {
	DefaultLevel    IsolationLevel `yaml:"default_level"`
	FallbackEnabled bool           `yaml:"fallback_enabled"`
}

// DefaultPrefix is the default prefix for environment variables
const DefaultPrefix = "APS"

// GetConfigDir returns the directory for APS configuration
func GetConfigDir() (string, error) {
	// 1. Check XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "aps"), nil
	}

	// 2. Fallback to os.UserConfigDir()
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "aps"), nil
}

// LoadConfig loads the global configuration
func LoadConfig() (*Config, error) {
	config := &Config{
		Prefix: DefaultPrefix,
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationProcess,
			FallbackEnabled: true,
		},
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return config, nil // Return default on error finding config dir
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, nil // Return default if file missing or unreadable
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, nil // Return default on malformed YAML (as per spec recommendation)
	}

	if config.Prefix == "" {
		config.Prefix = DefaultPrefix
	}

	if config.Isolation.DefaultLevel == "" {
		config.Isolation.DefaultLevel = IsolationProcess
	} else {
		switch config.Isolation.DefaultLevel {
		case IsolationProcess:
		case IsolationPlatform:
		case IsolationContainer:
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
