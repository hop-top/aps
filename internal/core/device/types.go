package device

import (
	"context"
	"time"
)

type DeviceType string

const (
	DeviceTypeMobile    DeviceType = "mobile"
	DeviceTypeDesktop   DeviceType = "desktop"
	DeviceTypeMessenger DeviceType = "messenger"
	DeviceTypeProtocol  DeviceType = "protocol"
	DeviceTypeSense     DeviceType = "sense"
	DeviceTypeActuator  DeviceType = "actuator"
)

type LoadingStrategy string

const (
	StrategySubprocess LoadingStrategy = "subprocess"
	StrategyScript     LoadingStrategy = "script"
	StrategyBuiltin    LoadingStrategy = "builtin"
)

type DeviceScope string

const (
	ScopeGlobal  DeviceScope = "global"
	ScopeProfile DeviceScope = "profile"
)

type DeviceState string

const (
	StateStopped  DeviceState = "stopped"
	StateStarting DeviceState = "starting"
	StateRunning  DeviceState = "running"
	StateFailed   DeviceState = "failed"
	StateUnknown  DeviceState = "unknown"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

type DeviceTypeMeta struct {
	Type        DeviceType
	Description string
	Implemented bool
}

var DeviceTypes = map[DeviceType]DeviceTypeMeta{
	DeviceTypeMessenger: {Type: DeviceTypeMessenger, Description: "Telegram, Slack, etc.", Implemented: true},
	DeviceTypeProtocol:  {Type: DeviceTypeProtocol, Description: "A2A, ACP, WebSocket", Implemented: true},
	DeviceTypeDesktop:   {Type: DeviceTypeDesktop, Description: "Desktop applications", Implemented: false},
	DeviceTypeMobile:    {Type: DeviceTypeMobile, Description: "Mobile devices (via QR linking)", Implemented: false},
	DeviceTypeSense:     {Type: DeviceTypeSense, Description: "Camera, microphone", Implemented: false},
	DeviceTypeActuator:  {Type: DeviceTypeActuator, Description: "Robotics, hardware", Implemented: false},
}

func ImplementedDeviceTypes() []DeviceType {
	var implemented []DeviceType
	for _, meta := range DeviceTypes {
		if meta.Implemented {
			implemented = append(implemented, meta.Type)
		}
	}
	return implemented
}

func IsDeviceTypeImplemented(t DeviceType) bool {
	meta, ok := DeviceTypes[t]
	return ok && meta.Implemented
}

func IsDeviceTypeValid(t DeviceType) bool {
	_, ok := DeviceTypes[t]
	return ok
}

type Device struct {
	Name         string          `json:"name" yaml:"name"`
	Type         DeviceType      `json:"type" yaml:"type"`
	Scope        DeviceScope     `json:"scope" yaml:"scope"`
	ProfileID    string          `json:"profile_id,omitempty" yaml:"profile_id,omitempty"`
	Strategy     LoadingStrategy `json:"strategy" yaml:"strategy"`
	Description  string          `json:"description,omitempty" yaml:"description,omitempty"`
	Config       map[string]any  `json:"config,omitempty" yaml:"config,omitempty"`
	CreatedAt    time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" yaml:"updated_at"`
	LinkedTo     []string        `json:"linked_to,omitempty" yaml:"linked_to,omitempty"`
	Path         string          `json:"path" yaml:"-"`
	ManifestPath string          `json:"manifest_path" yaml:"-"`
}

type DeviceRuntime struct {
	Name      string        `json:"name"`
	State     DeviceState   `json:"state"`
	Health    HealthStatus  `json:"health"`
	PID       int           `json:"pid,omitempty"`
	StartedAt *time.Time    `json:"started_at,omitempty"`
	LastError string        `json:"last_error,omitempty"`
	Restarts  int           `json:"restarts"`
	LastCheck *time.Time    `json:"last_check,omitempty"`
	Uptime    time.Duration `json:"uptime,omitempty"`
}

type DeviceManifest struct {
	APIVersion  string          `json:"api_version" yaml:"api_version"`
	Kind        string          `json:"kind" yaml:"kind"`
	Name        string          `json:"name" yaml:"name"`
	Type        DeviceType      `json:"type" yaml:"type"`
	Strategy    LoadingStrategy `json:"strategy" yaml:"strategy"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Config      map[string]any  `json:"config,omitempty" yaml:"config,omitempty"`
}

type DeviceCapability interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetState() DeviceState
	HealthCheck(ctx context.Context) error
}

type ProfileDeviceLink struct {
	ProfileID string            `json:"profile_id" yaml:"profile_id"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	Mappings  map[string]string `json:"mappings,omitempty" yaml:"mappings,omitempty"`
}

type DeviceFilter struct {
	Type    DeviceType
	Scope   DeviceScope
	Profile string
	State   DeviceState
}

func DefaultStrategyForType(t DeviceType) LoadingStrategy {
	switch t {
	case DeviceTypeMessenger:
		return StrategySubprocess
	case DeviceTypeProtocol:
		return StrategyBuiltin
	default:
		return StrategySubprocess
	}
}

func (d *Device) IsGlobal() bool {
	return d.Scope == ScopeGlobal
}

func (d *Device) IsProfileScoped() bool {
	return d.Scope == ScopeProfile
}

func (d *Device) IsLinkedToProfile(profileID string) bool {
	for _, p := range d.LinkedTo {
		if p == profileID {
			return true
		}
	}
	return false
}

const ManifestFileName = "manifest.yaml"
