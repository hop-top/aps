package core

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the global configuration for APS
type Config struct {
	Prefix string `yaml:"prefix"`
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

	return config, nil
}
