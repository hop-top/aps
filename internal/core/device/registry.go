package device

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

func GetDevicesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aps", "devices"), nil
}

func GetGlobalDevicePath(name string) (string, error) {
	devicesDir, err := GetDevicesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(devicesDir, name), nil
}

func GetProfileDevicePath(profileID, name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aps", "profiles", profileID, "devices", name), nil
}

func GetDeviceManifestPath(devicePath string) string {
	return filepath.Join(devicePath, ManifestFileName)
}

func LoadDevice(name string) (*Device, error) {
	globalPath, err := GetGlobalDevicePath(name)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(globalPath); err == nil {
		return loadDeviceFromPath(globalPath, ScopeGlobal, "")
	}

	profileIDs, err := listProfilesWithDevices()
	if err != nil {
		return nil, ErrDeviceNotFound(name)
	}

	for _, profileID := range profileIDs {
		profilePath, err := GetProfileDevicePath(profileID, name)
		if err != nil {
			continue
		}
		if _, err := os.Stat(profilePath); err == nil {
			return loadDeviceFromPath(profilePath, ScopeProfile, profileID)
		}
	}

	return nil, ErrDeviceNotFound(name)
}

func LoadDeviceByPath(path string) (*Device, error) {
	return loadDeviceFromPath(path, ScopeGlobal, "")
}

func loadDeviceFromPath(path string, scope DeviceScope, profileID string) (*Device, error) {
	manifestPath := GetDeviceManifestPath(path)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, ErrManifestInvalid(path, err)
	}

	var manifest DeviceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, ErrManifestInvalid(path, err)
	}

	device := &Device{
		Name:         manifest.Name,
		Type:         manifest.Type,
		Scope:        scope,
		ProfileID:    profileID,
		Strategy:     manifest.Strategy,
		Description:  manifest.Description,
		Config:       manifest.Config,
		Path:         path,
		ManifestPath: manifestPath,
	}

	if device.Strategy == "" {
		device.Strategy = DefaultStrategyForType(device.Type)
	}

	return device, nil
}

func SaveDevice(device *Device) error {
	var basePath string
	var err error

	if device.IsGlobal() {
		basePath, err = GetGlobalDevicePath(device.Name)
	} else {
		basePath, err = GetProfileDevicePath(device.ProfileID, device.Name)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create device directory: %w", err)
	}

	now := time.Now()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	manifest := DeviceManifest{
		APIVersion:  "device.aps.dev/v1",
		Kind:        "Device",
		Name:        device.Name,
		Type:        device.Type,
		Strategy:    device.Strategy,
		Description: device.Description,
		Config:      device.Config,
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal device manifest: %w", err)
	}

	manifestPath := GetDeviceManifestPath(basePath)
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write device manifest: %w", err)
	}

	device.Path = basePath
	device.ManifestPath = manifestPath

	return nil
}

func DeleteDevice(name string) error {
	device, err := LoadDevice(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(device.Path); err != nil {
		return fmt.Errorf("failed to delete device directory: %w", err)
	}

	return nil
}

func ListDevices(filter *DeviceFilter) ([]*Device, error) {
	var devices []*Device

	devicesDir, err := GetDevicesDir()
	if err != nil {
		return nil, err
	}

	globalDevices, err := listDevicesInDir(devicesDir, ScopeGlobal, "", filter)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	devices = append(devices, globalDevices...)

	profileIDs, err := listProfilesWithDevices()
	if err != nil {
		return nil, err
	}

	for _, profileID := range profileIDs {
		profileDevicesDir, err := GetProfileDevicePath(profileID, "")
		if err != nil {
			continue
		}
		profileDevices, err := listDevicesInDir(profileDevicesDir, ScopeProfile, profileID, filter)
		if err != nil && !os.IsNotExist(err) {
			continue
		}
		devices = append(devices, profileDevices...)
	}

	return devices, nil
}

func listDevicesInDir(dir string, scope DeviceScope, profileID string, filter *DeviceFilter) ([]*Device, error) {
	var devices []*Device

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		devicePath := filepath.Join(dir, entry.Name())
		device, err := loadDeviceFromPath(devicePath, scope, profileID)
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

func matchesFilter(device *Device, filter *DeviceFilter) bool {
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

func listProfilesWithDevices() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	profilesDir := filepath.Join(home, ".aps", "profiles")
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

func DeviceExists(name string) (bool, error) {
	device, err := LoadDevice(name)
	if err != nil {
		if IsDeviceNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return device != nil, nil
}

func GetDevicesForProfile(profileID string) ([]*Device, error) {
	return ListDevices(&DeviceFilter{Profile: profileID})
}
