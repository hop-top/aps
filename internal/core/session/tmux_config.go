package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oss-aps-cli/internal/core"
)

const (
	TmuxConfigFile = "tmux.conf"
)

type TmuxConfig struct {
	StatusBar    string `yaml:"status_bar,omitempty"`
	NoIdleTimout bool   `yaml:"no_idle_timeout,omitempty"`
	SessionDir   string `yaml:"session_dir,omitempty"`
	KeysDir      string `yaml:"keys_dir,omitempty"`
}

func NewTmuxConfigManager() *TmuxConfigManager {
	return &TmuxConfigManager{}
}

type TmuxConfigManager struct{}

func (m *TmuxConfigManager) GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	apsDir := filepath.Join(home, core.ApsHomeDir)
	return apsDir, nil
}

func (m *TmuxConfigManager) GetConfigPath() (string, error) {
	configDir, err := m.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, TmuxConfigFile), nil
}

func (m *TmuxConfigManager) LoadConfig() (*TmuxConfig, error) {
	configPath, err := m.GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return m.getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tmux config: %w", err)
	}

	if len(data) == 0 {
		return m.getDefaultConfig(), nil
	}

	lines := strings.Split(string(data), "\n")
	config := &TmuxConfig{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		option := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch option {
		case "status-bar":
			config.StatusBar = value
		case "no-idle-timeout":
			config.NoIdleTimout = (value == "on" || value == "1" || strings.ToLower(value) == "true")
		case "session-dir":
			config.SessionDir = value
		case "keys-dir":
			config.KeysDir = value
		}
	}

	return config, nil
}

func (m *TmuxConfigManager) SaveConfig(config *TmuxConfig) error {
	configPath, err := m.GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var content string
	if config.StatusBar != "" {
		content += fmt.Sprintf("status-bar %s\n", config.StatusBar)
	}
	if config.NoIdleTimout {
		content += "no-idle-timeout on\n"
	}
	if config.SessionDir != "" {
		content += fmt.Sprintf("session-dir %s\n", config.SessionDir)
	}
	if config.KeysDir != "" {
		content += fmt.Sprintf("keys-dir %s\n", config.KeysDir)
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write tmux config: %w", err)
	}

	return nil
}

func (m *TmuxConfigManager) getDefaultConfig() *TmuxConfig {
	return &TmuxConfig{
		StatusBar:    "top",
		NoIdleTimout: true,
		SessionDir:   filepath.Join(core.ApsHomeDir, "sessions"),
		KeysDir:      filepath.Join(core.ApsHomeDir, "keys"),
	}
}
