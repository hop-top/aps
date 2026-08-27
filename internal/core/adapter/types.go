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
	AdapterTypeScheduler AdapterType = "scheduler"
)

// DefaultEnvPrefix is the env-var prefix applied to script-strategy
// action inputs when an adapter manifest omits env_prefix.
const DefaultEnvPrefix = "ADAPTER"

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
	AdapterTypeDesktop:   {Type: AdapterTypeDesktop, Description: "Desktop applications", Implemented: true},
	AdapterTypeMobile:    {Type: AdapterTypeMobile, Description: "Mobile devices (via QR linking)", Implemented: true},
	AdapterTypeSense:     {Type: AdapterTypeSense, Description: "Camera, microphone", Implemented: true},
	AdapterTypeActuator:  {Type: AdapterTypeActuator, Description: "Robotics, hardware", Implemented: true},
	AdapterTypeScheduler: {Type: AdapterTypeScheduler, Description: "Calendars, schedulers, reminders", Implemented: true},
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

func IsLoadingStrategyValid(s LoadingStrategy) bool {
	switch s {
	case StrategySubprocess, StrategyScript, StrategyBuiltin:
		return true
	default:
		return false
	}
}

// LoadingStrategyMeta carries the presentation strings for one loading
// strategy as rendered in docs/dev/adapters.md.
type LoadingStrategyMeta struct {
	Display     string
	Description string
	Persistence string
}

// LoadingStrategies describes each loading strategy for docs rendering.
// Strings match the hand-written table in docs/dev/adapters.md.
var LoadingStrategies = map[LoadingStrategy]LoadingStrategyMeta{
	StrategySubprocess: {
		Display:     "Subprocess",
		Description: "Runs a standalone binary as a managed child process.",
		Persistence: "Persistent",
	},
	StrategyScript: {
		Display:     "Script",
		Description: "Executes a shell/python/node script on demand per action.",
		Persistence: "Ephemeral",
	},
	StrategyBuiltin: {
		Display:     "Built-in",
		Description: "Native Go implementation compiled into the APS binary.",
		Persistence: "Persistent",
	},
}

// AdapterScopes describes each adapter scope for docs rendering.
var AdapterScopes = map[AdapterScope]string{
	ScopeGlobal:  "Available to every profile.",
	ScopeProfile: "Owned by a single profile.",
}

// AdapterStates describes each runtime state for docs rendering.
var AdapterStates = map[AdapterState]string{
	StateStopped:  "Not running.",
	StateStarting: "Startup in progress.",
	StateRunning:  "Running under APS management.",
	StateFailed:   "Exited or crashed with an error.",
	StateUnknown:  "State could not be determined.",
}

// HealthStatuses describes each health status for docs rendering.
var HealthStatuses = map[HealthStatus]string{
	HealthHealthy:   "Last health check passed.",
	HealthUnhealthy: "Last health check failed.",
	HealthUnknown:   "No health check result yet.",
}

type Adapter struct {
	Name         string          `json:"name" yaml:"name"`
	Type         AdapterType     `json:"type" yaml:"type"`
	Scope        AdapterScope    `json:"scope" yaml:"scope"`
	ProfileID    string          `json:"profile_id,omitempty" yaml:"profile_id,omitempty"`
	Strategy     LoadingStrategy `json:"strategy" yaml:"strategy"`
	Description  string          `json:"description,omitempty" yaml:"description,omitempty"`
	EnvPrefix    string          `json:"env_prefix,omitempty" yaml:"env_prefix,omitempty"`
	Config       map[string]any  `json:"config,omitempty" yaml:"config,omitempty"`
	CreatedAt    time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" yaml:"updated_at"`
	LinkedTo     []string        `json:"linked_to,omitempty" yaml:"linked_to,omitempty"`
	Path         string          `json:"path" yaml:"-"`
	ManifestPath string          `json:"manifest_path" yaml:"-"`
}

type AdapterRuntime struct {
	Name      string        `json:"name"`
	State     AdapterState  `json:"state"`
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
	Type        AdapterType     `json:"type" yaml:"type"`
	Strategy    LoadingStrategy `json:"strategy" yaml:"strategy"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	EnvPrefix   string          `json:"env_prefix,omitempty" yaml:"env_prefix,omitempty"`
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
	case AdapterTypeMobile:
		return StrategyBuiltin
	case AdapterTypeDesktop:
		return StrategySubprocess
	case AdapterTypeSense, AdapterTypeActuator, AdapterTypeScheduler:
		return StrategyScript
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
