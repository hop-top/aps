package mobile

import (
	"time"
)

// PairingState represents the state of a mobile device pairing
type PairingState string

const (
	PairingStatePending  PairingState = "pending"
	PairingStateActive   PairingState = "active"
	PairingStateRevoked  PairingState = "revoked"
	PairingStateExpired  PairingState = "expired"
	PairingStateRejected PairingState = "rejected"
)

// DeviceCapabilityType represents capabilities a mobile device can have
type DeviceCapabilityType string

const (
	CapRunStateless     DeviceCapabilityType = "run:stateless"
	CapRunStreaming      DeviceCapabilityType = "run:streaming"
	CapMonitorSessions  DeviceCapabilityType = "monitor:sessions"
	CapMonitorLogs      DeviceCapabilityType = "monitor:logs"
)

// AllCapabilities returns all valid device capabilities
func AllCapabilities() []DeviceCapabilityType {
	return []DeviceCapabilityType{
		CapRunStateless,
		CapRunStreaming,
		CapMonitorSessions,
		CapMonitorLogs,
	}
}

// DefaultCapabilities returns the default set of capabilities for a new device
func DefaultCapabilities() []DeviceCapabilityType {
	return []DeviceCapabilityType{
		CapRunStateless,
		CapRunStreaming,
		CapMonitorSessions,
	}
}

// IsValidCapability checks if a capability string is valid
func IsValidCapability(cap string) bool {
	for _, c := range AllCapabilities() {
		if string(c) == cap {
			return true
		}
	}
	return false
}

// QRPayload is the JSON structure encoded in the QR code
type QRPayload struct {
	Version         string   `json:"version"`
	ProfileID       string   `json:"profile_id"`
	Endpoint        string   `json:"endpoint"`
	PairingCode     string   `json:"pairing_code"`
	ExpiresAt       string   `json:"expires_at"`
	CertFingerprint string   `json:"cert_fingerprint,omitempty"`
	Capabilities    []string `json:"capabilities"`
}

// MobileDevice represents a registered mobile device
type MobileDevice struct {
	DeviceID         string       `json:"device_id"`
	ProfileID        string       `json:"profile_id"`
	DeviceName       string       `json:"device_name"`
	DeviceOS         string       `json:"device_os"`
	DeviceVersion    string       `json:"device_version,omitempty"`
	DeviceModel      string       `json:"device_model,omitempty"`
	RegisteredAt     time.Time    `json:"registered_at"`
	LastSeenAt       time.Time    `json:"last_seen_at"`
	ExpiresAt        time.Time    `json:"expires_at"`
	TokenHash        string       `json:"token_hash"`
	Status           PairingState `json:"status"`
	Capabilities     []string     `json:"capabilities"`
	ApprovalRequired bool         `json:"approval_required"`
	ApprovedAt       *time.Time   `json:"approved_at,omitempty"`
}

// MobileDeviceRegistry is the on-disk format for the mobile device registry
type MobileDeviceRegistryData struct {
	Version string          `json:"version"`
	Devices []*MobileDevice `json:"devices"`
}

// PairingRequest is sent by the mobile client to pair
type PairingRequest struct {
	PairingCode   string `json:"pairing_code"`
	DeviceName    string `json:"device_name"`
	DeviceOS      string `json:"device_os"`
	DeviceVersion string `json:"device_version,omitempty"`
	DeviceModel   string `json:"device_model,omitempty"`
}

// PairingResponse is returned after successful pairing
type PairingResponse struct {
	DeviceID    string `json:"device_id"`
	Token       string `json:"token"`
	WSEndpoint  string `json:"ws_endpoint"`
	ExpiresAt   string `json:"expires_at"`
	ProfileID   string `json:"profile_id"`
	Status      string `json:"status"`
}

// WSMessage is the WebSocket message format
type WSMessage struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Action  string      `json:"action,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// WSCommandPayload is the payload for a command execution request
type WSCommandPayload struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
	Stream  bool     `json:"stream,omitempty"`
}

// WSStatusPayload is the payload for a status update
type WSStatusPayload struct {
	Status     string `json:"status"`
	StartedAt  string `json:"started_at,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// WSOutputPayload is the payload for command output
type WSOutputPayload struct {
	Stream string `json:"stream"` // "stdout" or "stderr"
	Data   string `json:"data"`
}

// IsExpired returns true if the device token has expired
func (d *MobileDevice) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// IsActive returns true if the device is active and not expired
func (d *MobileDevice) IsActive() bool {
	return d.Status == PairingStateActive && !d.IsExpired()
}
