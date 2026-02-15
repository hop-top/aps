package multidevice

// IsDeviceAllowed checks if a device is permitted by the workspace policy.
func (p *Policy) IsDeviceAllowed(deviceID string) bool {
	switch p.Mode {
	case PolicyAllowAll:
		return true

	case PolicyAllowList:
		for _, allowed := range p.AllowDevices {
			if allowed == deviceID {
				return true
			}
		}
		return false

	case PolicyDenyList:
		for _, denied := range p.DenyDevices {
			if denied == deviceID {
				return false
			}
		}
		return true

	default:
		// Unknown policy mode; default to allow for safety.
		return true
	}
}

// DefaultPolicy returns a workspace policy that allows all devices.
func DefaultPolicy(workspaceID string) *Policy {
	return &Policy{
		WorkspaceID: workspaceID,
		Mode:        PolicyAllowAll,
	}
}
