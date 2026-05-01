package adapter

import (
	"context"
	"time"
)

type AdapterType string

const (
	AdapterTypeMobile    AdapterType = "mobile"
	AdapterTypeDesktop   AdapterType = "desktop"
	AdapterTypeMessenger AdapterType = "messenger"
	AdapterTypeProtocol  AdapterType = "protocol"
	AdapterTypeSense     AdapterType = "sense"
	AdapterTypeActuator  AdapterType = "actuator"
)

type LoadingStrategy string

const (
	StrategySubprocess LoadingStrategy = "subprocess"
	StrategyScript     LoadingStrategy = "script"
	StrategyBuiltin    LoadingStrategy = "builtin"
)

type AdapterScope string

const (
	ScopeGlobal  AdapterScope = "global"
	ScopeProfile AdapterScope = "profile"
)

type AdapterState string

const (
	StateStopped  AdapterState = "stopped"
	StateStarting AdapterState = "starting"
	StateRunning  AdapterState = "running"
	StateFailed   AdapterState = "failed"
	StateUnknown  AdapterState = "unknown"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

type AdapterTypeMeta struct {
	Type        AdapterType
	Description string
	Implemented bool
}

var AdapterTypes = map[AdapterType]AdapterTypeMeta{
	AdapterTypeMessenger: {Type: AdapterTypeMessenger, Description: "Telegram, Slack, etc.", Implemented: true},
	AdapterTypeProtocol:  {Type: AdapterTypeProtocol, Description: "A2A, ACP, WebSocket", Implemented: true},
	AdapterTypeDesktop:   {Type: AdapterTypeDesktop, Description: "Desktop applications", Implemented: false},
	AdapterTypeMobile:    {Type: AdapterTypeMobile, Description: "Mobile devices (via QR linking)", Implemented: true},
	AdapterTypeSense:     {Type: AdapterTypeSense, Description: "Camera, microphone", Implemented: false},
	AdapterTypeActuator:  {Type: AdapterTypeActuator, Description: "Robotics, hardware", Implemented: false},
}

func ImplementedAdapterTypes() []AdapterType {
	var implemented []AdapterType
	for _, meta := range AdapterTypes {
		if meta.Implemented {
			implemented = append(implemented, meta.Type)
		}
	}
	return implemented
}

func IsAdapterTypeImplemented(t AdapterType) bool {
	meta, ok := AdapterTypes[t]
	return ok && meta.Implemented
}

func IsAdapterTypeValid(t AdapterType) bool {
	_, ok := AdapterTypes[t]
	return ok
}

type Adapter struct {
	Name         string          `json:"name" yaml:"name"`
	Type         AdapterType      `json:"type" yaml:"type"`
	Scope        AdapterScope     `json:"scope" yaml:"scope"`
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

type AdapterRuntime struct {
	Name      string        `json:"name"`
	State     AdapterState   `json:"state"`
	Health    HealthStatus  `json:"health"`
	PID       int           `json:"pid,omitempty"`
	StartedAt *time.Time    `json:"started_at,omitempty"`
	LastError string        `json:"last_error,omitempty"`
	Restarts  int           `json:"restarts"`
	LastCheck *time.Time    `json:"last_check,omitempty"`
	Uptime    time.Duration `json:"uptime,omitempty"`
}

type AdapterManifest struct {
	APIVersion  string          `json:"api_version" yaml:"api_version"`
	Kind        string          `json:"kind" yaml:"kind"`
	Name        string          `json:"name" yaml:"name"`
	Type        AdapterType      `json:"type" yaml:"type"`
	Strategy    LoadingStrategy `json:"strategy" yaml:"strategy"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Config      map[string]any  `json:"config,omitempty" yaml:"config,omitempty"`
	LinkedTo    []string        `json:"linked_to,omitempty" yaml:"linked_to,omitempty"`
}

type AdapterCapability interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetState() AdapterState
	HealthCheck(ctx context.Context) error
}

type ProfileAdapterLink struct {
	ProfileID string            `json:"profile_id" yaml:"profile_id"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	Mappings  map[string]string `json:"mappings,omitempty" yaml:"mappings,omitempty"`
}

type AdapterFilter struct {
	Type    AdapterType
	Scope   AdapterScope
	Profile string
	State   AdapterState
}

func DefaultStrategyForType(t AdapterType) LoadingStrategy {
	switch t {
	case AdapterTypeMessenger:
		return StrategySubprocess
	case AdapterTypeProtocol:
		return StrategyBuiltin
	default:
		return StrategySubprocess
	}
}

func (d *Adapter) IsGlobal() bool {
	return d.Scope == ScopeGlobal
}

func (d *Adapter) IsProfileScoped() bool {
	return d.Scope == ScopeProfile
}

func (d *Adapter) IsLinkedToProfile(profileID string) bool {
	for _, p := range d.LinkedTo {
		if p == profileID {
			return true
		}
	}
	return false
}

const ManifestFileName = "manifest.yaml"
