package isolation

import (
	"errors"
	"time"

	"oss-aps-cli/internal/core"
)

type ContainerStatus string

const (
	ContainerCreated    ContainerStatus = "created"
	ContainerRunning    ContainerStatus = "running"
	ContainerPaused     ContainerStatus = "paused"
	ContainerRestarting ContainerStatus = "restarting"
	ContainerRemoving   ContainerStatus = "removing"
	ContainerExited     ContainerStatus = "exited"
	ContainerDead       ContainerStatus = "dead"
)

type ContainerHealthStatus string

const (
	HealthStarting  ContainerHealthStatus = "starting"
	HealthHealthy   ContainerHealthStatus = "healthy"
	HealthUnhealthy ContainerHealthStatus = "unhealthy"
	HealthUnknown   ContainerHealthStatus = "unknown"
)

type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Readonly bool   `json:"readonly"`
	Options  string `json:"options,omitempty"`
}

type NetworkConfig struct {
	Mode     string   `json:"mode"`
	Network  string   `json:"network,omitempty"`
	Ports    []string `json:"ports,omitempty"`
	DNS      []string `json:"dns,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
}

type ResourceLimits struct {
	CPUQuota         int64  `json:"cpu_quota,omitempty"`
	CPUPeriod        int64  `json:"cpu_period,omitempty"`
	CPUShares        int64  `json:"cpu_shares,omitempty"`
	CPUSetCPUs       string `json:"cpu_set_cpus,omitempty"`
	CPUSetMems       string `json:"cpu_set_mems,omitempty"`
	MemoryLimit      int64  `json:"memory_limit,omitempty"`
	MemorySwap       int64  `json:"memory_swap,omitempty"`
	MemorySwappiness int64  `json:"memory_swappiness,omitempty"`
	DiskQuota        int64  `json:"disk_quota,omitempty"`
	DiskIOPS         int64  `json:"disk_iops,omitempty"`
}

type LogOptions struct {
	Since      time.Time
	Until      time.Time
	Follow     bool
	Tail       string
	ShowStdout bool
	ShowStderr bool
	Timestamps bool
}

type LogMessage struct {
	Timestamp time.Time
	Stream    string
	Line      string
}

type ContainerRunOptions struct {
	Image       string
	Command     []string
	Environment []string
	WorkingDir  string
	Volumes     []VolumeMount
	Network     NetworkConfig
	Limits      ResourceLimits
	User        string
}

type ImageBuildContext struct {
	Profile    *core.Profile
	ImageTag   string
	BuildDir   string
	ProfileDir string
}

type ImageBuilder interface {
	Generate(profile *core.Profile) (string, error)
	BuildOptions(profile *core.Profile) ContainerRunOptions
}

type ContainerEngine interface {
	Name() string
	Version() (string, error)
	Ping() error
	Available() bool
	BuildImage(ctx ImageBuildContext) (string, error)
	PullImage(image string) error
	RemoveImage(image string, force bool) error
	CreateContainer(opts ContainerRunOptions) (string, error)
	StartContainer(id string) error
	StopContainer(id string, timeout time.Duration) error
	RemoveContainer(id string, force bool) error
	ExecContainer(id string, cmd []string) (int, error)
	GetContainerStatus(id string) (ContainerStatus, error)
	GetContainerLogs(id string, opts LogOptions) (<-chan LogMessage, error)
	UpdateContainerResources(id string, limits ResourceLimits) error
	InspectContainer(id string) (map[string]interface{}, error)
	GetContainerIP(id string) (string, error)
	GetContainerPortMapping(id string, containerPort string) (string, error)
}

type ContainerIsolation struct {
	context      *ExecutionContext
	engine       ContainerEngine
	imageBuilder ImageBuilder
	containerID  string
	status       ContainerStatus
	config       ContainerConfig
	imageTag     string
	limits       ResourceLimits
	tmuxSocket   string
	tmuxSession  string
	useTmux      bool
	sessionPID   int
	configured   bool
}

type ContainerConfig struct {
	Image      string
	Volumes    []VolumeMount
	Network    NetworkConfig
	Resources  ResourceLimits
	BuildSteps []core.BuildStep
	Packages   []string
	User       string
}

var (
	ErrNotSupported = errors.New("operation not supported by this isolation level")
)
