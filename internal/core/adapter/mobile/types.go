package mobile

import (
	"time"
)

// PairingState represents the state of a mobile adapter pairing
type PairingState string

const (
	PairingStatePending  PairingState = "pending"
	PairingStateActive   PairingState = "active"
	PairingStateRevoked  PairingState = "revoked"
	PairingStateExpired  PairingState = "expired"
	PairingStateRejected PairingState = "rejected"
)

// AdapterCapabilityType represents capabilities a mobile adapter can have
type AdapterCapabilityType string

const (
	CapRunStateless    AdapterCapabilityType = "run:stateless"
	CapRunStreaming    AdapterCapabilityType = "run:streaming"
	CapMonitorSessions AdapterCapabilityType = "monitor:sessions"
	CapMonitorLogs     AdapterCapabilityType = "monitor:logs"
)

// AllCapabilities returns all valid device capabilities
func AllCapabilities() []AdapterCapabilityType {
	return []AdapterCapabilityType{
		CapRunStateless,
		CapRunStreaming,
		CapMonitorSessions,
		CapMonitorLogs,
	}
}

// DefaultCapabilities returns the default set of capabilities for a new device
func DefaultCapabilities() []AdapterCapabilityType {
	return []AdapterCapabilityType{
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

// MobileAdapter represents a registered mobile adapter
type MobileAdapter struct {
	AdapterID        string       `json:"device_id"`
	ProfileID        string       `json:"profile_id"`
	AdapterName      string       `json:"device_name"`
	AdapterOS        string       `json:"device_os"`
	AdapterVersion   string       `json:"adapter_version,omitempty"`
	AdapterModel     string       `json:"adapter_model,omitempty"`
	RegisteredAt     time.Time    `json:"registered_at"`
	LastSeenAt       time.Time    `json:"last_seen_at"`
	ExpiresAt        time.Time    `json:"expires_at"`
	TokenHash        string       `json:"token_hash"`
	Status           PairingState `json:"status"`
	Capabilities     []string     `json:"capabilities"`
	ApprovalRequired bool         `json:"approval_required"`
	ApprovedAt       *time.Time   `json:"approved_at,omitempty"`
}

// MobileAdapterRegistry is the on-disk format for the mobile adapter registry
type MobileAdapterRegistryData struct {
	Version  string           `json:"version"`
	Adapters []*MobileAdapter `json:"adapters"`
}

// PairingRequest is sent by the mobile client to pair
type PairingRequest struct {
	PairingCode    string `json:"pairing_code"`
	AdapterName    string `json:"device_name"`
	AdapterOS      string `json:"device_os"`
	AdapterVersion string `json:"adapter_version,omitempty"`
	AdapterModel   string `json:"adapter_model,omitempty"`
}

// PairingResponse is returned after successful pairing
type PairingResponse struct {
	AdapterID  string `json:"device_id"`
	Token      string `json:"token"`
	WSEndpoint string `json:"ws_endpoint"`
	ExpiresAt  string `json:"expires_at"`
	ProfileID  string `json:"profile_id"`
	Status     string `json:"status"`
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
	Maturity   string `json:"maturity,omitempty"`
	Executes   string `json:"executes,omitempty"`
	Message    string `json:"message,omitempty"`
}

// WSOutputPayload is the payload for command output
type WSOutputPayload struct {
	Stream string `json:"stream"` // "stdout" or "stderr"
	Data   string `json:"data"`
}

// IsExpired returns true if the device token has expired
func (d *MobileAdapter) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// IsActive returns true if the device is active and not expired
func (d *MobileAdapter) IsActive() bool {
	return d.Status == PairingStateActive && !d.IsExpired()
}
