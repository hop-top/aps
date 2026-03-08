package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type MessengerConfig struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Strategy    string         `yaml:"strategy"`
	TokenEnv    string         `yaml:"token_env"`
	WebhookPath string         `yaml:"webhook_path,omitempty"`
	Profile     string         `yaml:"profile,omitempty"`
	Config      map[string]any `yaml:"config,omitempty"`
}

type MessengerManifest struct {
	APIVersion string          `yaml:"api_version"`
	Kind       string          `yaml:"kind"`
	Messenger  MessengerConfig `yaml:"messenger"`
}

func DiscoverMessengers(messengersDir string) ([]MessengerConfig, error) {
	var messengers []MessengerConfig

	entries, err := os.ReadDir(messengersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return messengers, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(messengersDir, entry.Name(), "manifest.yaml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest MessengerManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			continue
		}

		if manifest.Messenger.Name == "" {
			manifest.Messenger.Name = entry.Name()
		}

		messengers = append(messengers, manifest.Messenger)
	}

	return messengers, nil
}

func ConvertMessengerToAdapter(messenger MessengerConfig) (*Adapter, error) {
	deviceType := AdapterTypeMessenger
	if messenger.Type == "protocol" {
		deviceType = AdapterTypeProtocol
	}

	strategy := StrategySubprocess
	switch messenger.Strategy {
	case "script":
		strategy = StrategyScript
	case "builtin":
		strategy = StrategyBuiltin
	}

	scope := ScopeGlobal
	if messenger.Profile != "" {
		scope = ScopeProfile
	}

	dev := &Adapter{
		Name:      messenger.Name,
		Type:      deviceType,
		Scope:     scope,
		ProfileID: messenger.Profile,
		Strategy:  strategy,
		Config:    messenger.Config,
	}

	if dev.Name == "" {
		return nil, fmt.Errorf("messenger name is required")
	}

	return dev, nil
}

func MigrateMessenger(messengerPath string, devicePath string) error {
	manifestPath := filepath.Join(messengerPath, "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ErrMigrationFailed(messengerPath, err)
	}

	var manifest MessengerManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return ErrManifestInvalid(manifestPath, err)
	}

	dev, err := ConvertMessengerToAdapter(manifest.Messenger)
	if err != nil {
		return ErrMigrationFailed(manifest.Messenger.Name, err)
	}

	if err := SaveAdapter(dev); err != nil {
		return ErrMigrationFailed(dev.Name, err)
	}

	return nil
}
