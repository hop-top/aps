package multidevice

import (
	"fmt"
	"time"
)

// Linker manages device-workspace associations.
type Linker struct{}

// NewLinker creates a new Linker instance.
func NewLinker() *Linker {
	return &Linker{}
}

// AttachDevice links a device to a workspace with the specified role.
func (l *Linker) AttachDevice(workspaceID, deviceID string, role DeviceRole) (*WorkspaceDeviceLink, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if deviceID == "" {
		return nil, fmt.Errorf("device ID is required")
	}
	if !IsValidRole(role) {
		return nil, fmt.Errorf("invalid device role: %s", role)
	}

	// Check if already linked.
	existing, err := LoadLink(workspaceID, deviceID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("device %s is already linked to workspace %s", deviceID, workspaceID)
	}

	perms := PermissionsForRole(role)

	link := &WorkspaceDeviceLink{
		WorkspaceID: workspaceID,
		DeviceID:    deviceID,
		LinkedAt:    time.Now(),
		LinkedBy:    deviceID,
		Permissions: perms,
		Status:      PresenceLinking,
	}

	if err := SaveLink(link); err != nil {
		return nil, fmt.Errorf("failed to save device link: %w", err)
	}

	return link, nil
}

// DetachDevice unlinks a device from a workspace.
func (l *Linker) DetachDevice(workspaceID, deviceID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if deviceID == "" {
		return fmt.Errorf("device ID is required")
	}

	// Verify the link exists before deleting.
	_, err := LoadLink(workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("cannot detach: %w", err)
	}

	return DeleteLink(workspaceID, deviceID)
}

// GetLink returns the link for a device in a workspace.
func (l *Linker) GetLink(workspaceID, deviceID string) (*WorkspaceDeviceLink, error) {
	return LoadLink(workspaceID, deviceID)
}

// ListLinks returns all device links for a workspace.
func (l *Linker) ListLinks(workspaceID string) ([]*WorkspaceDeviceLink, error) {
	return ListLinks(workspaceID)
}

// UpdatePermissions updates the permissions for a device in a workspace.
func (l *Linker) UpdatePermissions(workspaceID, deviceID string, perms DevicePermissions) error {
	link, err := LoadLink(workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("cannot update permissions: %w", err)
	}

	link.Permissions = perms

	if err := SaveLink(link); err != nil {
		return fmt.Errorf("failed to save updated permissions: %w", err)
	}

	return nil
}

// SetRole sets a role on a device link and applies the default permissions for that role.
func (l *Linker) SetRole(workspaceID, deviceID string, role DeviceRole) error {
	if !IsValidRole(role) {
		return fmt.Errorf("invalid device role: %s", role)
	}

	link, err := LoadLink(workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("cannot set role: %w", err)
	}

	link.Permissions = PermissionsForRole(role)

	if err := SaveLink(link); err != nil {
		return fmt.Errorf("failed to save updated role: %w", err)
	}

	return nil
}
