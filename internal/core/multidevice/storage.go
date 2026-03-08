package multidevice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"hop.top/aps/internal/core"
)

// GetWorkspaceDir returns the base directory for a workspace.
func GetWorkspaceDir(workspaceID string) (string, error) {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}
	return filepath.Join(dataDir, "workspaces", workspaceID), nil
}

// GetWorkspaceDevicesDir returns the devices subdirectory for a workspace.
func GetWorkspaceDevicesDir(workspaceID string) (string, error) {
	wsDir, err := GetWorkspaceDir(workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "devices"), nil
}

// getDeviceLinkPath returns the file path for a specific device link.
func getDeviceLinkPath(workspaceID, deviceID string) (string, error) {
	devicesDir, err := GetWorkspaceDevicesDir(workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(devicesDir, deviceID+".json"), nil
}

// SaveLink persists a device-workspace link to disk.
func SaveLink(link *WorkspaceDeviceLink) error {
	if link.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if link.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}

	devicesDir, err := GetWorkspaceDevicesDir(link.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to resolve devices directory: %w", err)
	}

	if err := os.MkdirAll(devicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create devices directory: %w", err)
	}

	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal device link: %w", err)
	}

	linkPath := filepath.Join(devicesDir, link.DeviceID+".json")
	if err := os.WriteFile(linkPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write device link: %w", err)
	}

	return nil
}

// LoadLink loads a device-workspace link from disk.
func LoadLink(workspaceID, deviceID string) (*WorkspaceDeviceLink, error) {
	linkPath, err := getDeviceLinkPath(workspaceID, deviceID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("device link not found: workspace=%s device=%s", workspaceID, deviceID)
		}
		return nil, fmt.Errorf("failed to read device link: %w", err)
	}

	var link WorkspaceDeviceLink
	if err := json.Unmarshal(data, &link); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device link: %w", err)
	}

	return &link, nil
}

// DeleteLink removes a device-workspace link from disk.
func DeleteLink(workspaceID, deviceID string) error {
	linkPath, err := getDeviceLinkPath(workspaceID, deviceID)
	if err != nil {
		return err
	}

	if err := os.Remove(linkPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("device link not found: workspace=%s device=%s", workspaceID, deviceID)
		}
		return fmt.Errorf("failed to delete device link: %w", err)
	}

	return nil
}

// ListLinks returns all device links for a workspace.
func ListLinks(workspaceID string) ([]*WorkspaceDeviceLink, error) {
	devicesDir, err := GetWorkspaceDevicesDir(workspaceID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*WorkspaceDeviceLink{}, nil
		}
		return nil, fmt.Errorf("failed to read devices directory: %w", err)
	}

	var links []*WorkspaceDeviceLink
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(devicesDir, entry.Name()))
		if err != nil {
			continue
		}

		var link WorkspaceDeviceLink
		if err := json.Unmarshal(data, &link); err != nil {
			continue
		}

		links = append(links, &link)
	}

	return links, nil
}
