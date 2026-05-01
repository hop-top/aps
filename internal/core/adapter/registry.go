package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// getXDGDataDir returns the XDG-compliant data directory for APS,
// inlined here to avoid a circular import with internal/core.
func getXDGDataDir() (string, error) {
	if v := os.Getenv("APS_DATA_PATH"); v != "" {
		return v, nil
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "aps"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "aps"), nil
}

func GetAdaptersDir() (string, error) {
	dataDir, err := getXDGDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "devices"), nil
}

func GetGlobalAdapterPath(name string) (string, error) {
	devicesDir, err := GetAdaptersDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(devicesDir, name), nil
}

func GetProfileAdapterPath(profileID, name string) (string, error) {
	dataDir, err := getXDGDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "profiles", profileID, "devices", name), nil
}

func GetAdapterManifestPath(devicePath string) string {
	return filepath.Join(devicePath, ManifestFileName)
}

func LoadAdapter(name string) (*Adapter, error) {
	globalPath, err := GetGlobalAdapterPath(name)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(globalPath); err == nil {
		return loadAdapterFromPath(globalPath, ScopeGlobal, "")
	}

	profileIDs, err := listProfilesWithAdapters()
	if err != nil {
		return nil, ErrAdapterNotFound(name)
	}

	for _, profileID := range profileIDs {
		profilePath, err := GetProfileAdapterPath(profileID, name)
		if err != nil {
			continue
		}
		if _, err := os.Stat(profilePath); err == nil {
			return loadAdapterFromPath(profilePath, ScopeProfile, profileID)
		}
	}

	return nil, ErrAdapterNotFound(name)
}

func LoadAdapterByPath(path string) (*Adapter, error) {
	return loadAdapterFromPath(path, ScopeGlobal, "")
}

func loadAdapterFromPath(path string, scope AdapterScope, profileID string) (*Adapter, error) {
	manifestPath := GetAdapterManifestPath(path)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, ErrManifestInvalid(path, err)
	}

	var manifest AdapterManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, ErrManifestInvalid(path, err)
	}

	device := &Adapter{
		Name:         manifest.Name,
		Type:         manifest.Type,
		Scope:        scope,
		ProfileID:    profileID,
		Strategy:     manifest.Strategy,
		Description:  manifest.Description,
		Config:       manifest.Config,
		LinkedTo:     manifest.LinkedTo,
		Path:         path,
		ManifestPath: manifestPath,
	}

	if device.Strategy == "" {
		device.Strategy = DefaultStrategyForType(device.Type)
	}

	return device, nil
}

func SaveAdapter(device *Adapter) error {
	var basePath string
	var err error

	if device.IsGlobal() {
		basePath, err = GetGlobalAdapterPath(device.Name)
	} else {
		basePath, err = GetProfileAdapterPath(device.ProfileID, device.Name)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create adapter directory: %w", err)
	}

	now := time.Now()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	manifest := AdapterManifest{
		APIVersion:  "adapter.aps.dev/v1",
		Kind:        "Adapter",
		Name:        device.Name,
		Type:        device.Type,
		Strategy:    device.Strategy,
		Description: device.Description,
		Config:      device.Config,
		LinkedTo:    device.LinkedTo,
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal adapter manifest: %w", err)
	}

	manifestPath := GetAdapterManifestPath(basePath)
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write adapter manifest: %w", err)
	}

	device.Path = basePath
	device.ManifestPath = manifestPath

	return nil
}

func DeleteAdapter(name string) error {
	device, err := LoadAdapter(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(device.Path); err != nil {
		return fmt.Errorf("failed to delete adapter directory: %w", err)
	}

	return nil
}

func ListAdapters(filter *AdapterFilter) ([]*Adapter, error) {
	var devices []*Adapter

	devicesDir, err := GetAdaptersDir()
	if err != nil {
		return nil, err
	}

	globalDevices, err := listAdaptersInDir(devicesDir, ScopeGlobal, "", filter)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	devices = append(devices, globalDevices...)

	profileIDs, err := listProfilesWithAdapters()
	if err != nil {
		return nil, err
	}

	for _, profileID := range profileIDs {
		profileDevicesDir, err := GetProfileAdapterPath(profileID, "")
		if err != nil {
			continue
		}
		profileDevices, err := listAdaptersInDir(profileDevicesDir, ScopeProfile, profileID, filter)
		if err != nil && !os.IsNotExist(err) {
			continue
		}
		devices = append(devices, profileDevices...)
	}

	return devices, nil
}

func listAdaptersInDir(dir string, scope AdapterScope, profileID string, filter *AdapterFilter) ([]*Adapter, error) {
	var devices []*Adapter

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		devicePath := filepath.Join(dir, entry.Name())
		device, err := loadAdapterFromPath(devicePath, scope, profileID)
		if err != nil {
			continue
		}

		if filter != nil && !matchesFilter(device, filter) {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func matchesFilter(device *Adapter, filter *AdapterFilter) bool {
	if filter.Type != "" && device.Type != filter.Type {
		return false
	}
	if filter.Scope != "" && device.Scope != filter.Scope {
		return false
	}
	if filter.Profile != "" && !device.IsLinkedToProfile(filter.Profile) {
		return false
	}
	return true
}

func listProfilesWithAdapters() ([]string, error) {
	dataDir, err := getXDGDataDir()
	if err != nil {
		return nil, err
	}

	profilesDir := filepath.Join(dataDir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var profileIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			devicesDir := filepath.Join(profilesDir, entry.Name(), "devices")
			if _, err := os.Stat(devicesDir); err == nil {
				profileIDs = append(profileIDs, entry.Name())
			}
		}
	}

	return profileIDs, nil
}

func AdapterExists(name string) (bool, error) {
	device, err := LoadAdapter(name)
	if err != nil {
		if IsAdapterNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return device != nil, nil
}

func GetAdaptersForProfile(profileID string) ([]*Adapter, error) {
	return ListAdapters(&AdapterFilter{Profile: profileID})
}
