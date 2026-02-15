package multidevice

import "time"

// DeviceRole defines predefined permission sets for devices in a workspace.
type DeviceRole string

const (
	// RoleOwner grants full access: read, write, execute, manage, sync.
	RoleOwner DeviceRole = "owner"
	// RoleCollaborator grants operational access: read, write, execute, sync.
	RoleCollaborator DeviceRole = "collaborator"
	// RoleViewer grants read-only access: read, sync.
	RoleViewer DeviceRole = "viewer"
)

// ValidRoles returns all valid device roles.
func ValidRoles() []DeviceRole {
	return []DeviceRole{RoleOwner, RoleCollaborator, RoleViewer}
}

// IsValidRole checks whether a role string is a recognized role.
func IsValidRole(role DeviceRole) bool {
	switch role {
	case RoleOwner, RoleCollaborator, RoleViewer:
		return true
	}
	return false
}

// PresenceState represents a device's presence in a workspace.
type PresenceState string

const (
	PresenceLinking  PresenceState = "linking"
	PresenceOnline   PresenceState = "online"
	PresenceAway     PresenceState = "away"
	PresenceOffline  PresenceState = "offline"
	PresenceUnlinked PresenceState = "unlinked"
)

// WorkspaceDeviceLink associates a device with a workspace.
type WorkspaceDeviceLink struct {
	WorkspaceID string            `json:"workspace_id" yaml:"workspace_id"`
	DeviceID    string            `json:"device_id" yaml:"device_id"`
	LinkedAt    time.Time         `json:"linked_at" yaml:"linked_at"`
	LinkedBy    string            `json:"linked_by" yaml:"linked_by"`
	Permissions DevicePermissions `json:"permissions" yaml:"permissions"`
	Status      PresenceState     `json:"status" yaml:"status"`
	LastEventID string            `json:"last_event_id,omitempty" yaml:"last_event_id,omitempty"`
	LastSyncAt  time.Time         `json:"last_sync_at,omitempty" yaml:"last_sync_at,omitempty"`
}

// DevicePermissions defines what a device can do in a workspace.
type DevicePermissions struct {
	Role             DeviceRole      `json:"role" yaml:"role"`
	CanRead          bool            `json:"can_read" yaml:"can_read"`
	CanWrite         bool            `json:"can_write" yaml:"can_write"`
	CanExecute       bool            `json:"can_execute" yaml:"can_execute"`
	CanManage        bool            `json:"can_manage" yaml:"can_manage"`
	CanSync          bool            `json:"can_sync" yaml:"can_sync"`
	MaxIsolationTier string          `json:"max_isolation_tier,omitempty" yaml:"max_isolation_tier,omitempty"`
	AllowedActions   []string        `json:"allowed_actions,omitempty" yaml:"allowed_actions,omitempty"`
	DeniedActions    []string        `json:"denied_actions,omitempty" yaml:"denied_actions,omitempty"`
	RateLimitPerMin  int             `json:"rate_limit_per_min,omitempty" yaml:"rate_limit_per_min,omitempty"`
	AccessSchedule   *AccessSchedule `json:"access_schedule,omitempty" yaml:"access_schedule,omitempty"`
}

// PermissionsForRole returns the default permissions for a given role.
func PermissionsForRole(role DeviceRole) DevicePermissions {
	switch role {
	case RoleOwner:
		return DevicePermissions{
			Role:       RoleOwner,
			CanRead:    true,
			CanWrite:   true,
			CanExecute: true,
			CanManage:  true,
			CanSync:    true,
		}
	case RoleCollaborator:
		return DevicePermissions{
			Role:       RoleCollaborator,
			CanRead:    true,
			CanWrite:   true,
			CanExecute: true,
			CanManage:  false,
			CanSync:    true,
		}
	case RoleViewer:
		return DevicePermissions{
			Role:       RoleViewer,
			CanRead:    true,
			CanWrite:   false,
			CanExecute: false,
			CanManage:  false,
			CanSync:    true,
		}
	default:
		return DevicePermissions{
			Role:    RoleViewer,
			CanRead: true,
			CanSync: true,
		}
	}
}

// AccessSchedule defines time-based access constraints for a device.
type AccessSchedule struct {
	StartTime  string `json:"start_time" yaml:"start_time"`     // "09:00"
	EndTime    string `json:"end_time" yaml:"end_time"`         // "17:00"
	DaysOfWeek []int  `json:"days_of_week" yaml:"days_of_week"` // 0=Monday
	Timezone   string `json:"timezone" yaml:"timezone"`
}

// DevicePresence tracks the current status of a device in a workspace.
type DevicePresence struct {
	DeviceID      string        `json:"device_id" yaml:"device_id"`
	WorkspaceID   string        `json:"workspace_id" yaml:"workspace_id"`
	State         PresenceState `json:"state" yaml:"state"`
	LastHeartbeat time.Time     `json:"last_heartbeat" yaml:"last_heartbeat"`
	LastActivity  time.Time     `json:"last_activity" yaml:"last_activity"`
	ClientVersion string        `json:"client_version,omitempty" yaml:"client_version,omitempty"`
	SyncLag       int           `json:"sync_lag" yaml:"sync_lag"`           // events behind
	OfflineQueue  int           `json:"offline_queue" yaml:"offline_queue"` // events waiting
}

// PresenceConfig holds configurable timeout values for presence tracking.
type PresenceConfig struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`
	AwayTimeout       time.Duration `json:"away_timeout" yaml:"away_timeout"`
	OfflineTimeout    time.Duration `json:"offline_timeout" yaml:"offline_timeout"`
}

// DefaultPresenceConfig returns sensible default presence timeouts.
func DefaultPresenceConfig() PresenceConfig {
	return PresenceConfig{
		HeartbeatInterval: 10 * time.Second,
		AwayTimeout:       30 * time.Second,
		OfflineTimeout:    120 * time.Second,
	}
}
