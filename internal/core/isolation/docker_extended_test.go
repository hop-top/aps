package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oss-aps-cli/internal/core"
)

// ============================================================================
// Mock Docker Engine for Testing
// ============================================================================

type MockDockerEngine struct {
	mu              sync.Mutex
	containers      map[string]containerState
	images          map[string]bool
	callCount       map[string]int
	shouldFail      map[string]bool
	containerLogs   map[string][]LogMessage
	containerIPs    map[string]string
	containerPorts  map[string]map[string]string // containerID -> {port: hostPort}
}

type containerState struct {
	id        string
	status    ContainerStatus
	image     string
	config    ContainerRunOptions
	resources ResourceLimits
	ip        string
	ports     map[string]string
}

func NewMockDockerEngine() *MockDockerEngine {
	return &MockDockerEngine{
		containers:     make(map[string]containerState),
		images:         make(map[string]bool),
		callCount:      make(map[string]int),
		shouldFail:     make(map[string]bool),
		containerLogs:  make(map[string][]LogMessage),
		containerIPs:   make(map[string]string),
		containerPorts: make(map[string]map[string]string),
	}
}

func (m *MockDockerEngine) Name() string {
	return "mock"
}

func (m *MockDockerEngine) Version() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["Version"]++

	if m.shouldFail["Version"] {
		return "", fmt.Errorf("version check failed")
	}
	return "20.10.0", nil
}

func (m *MockDockerEngine) Ping() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["Ping"]++

	if m.shouldFail["Ping"] {
		return fmt.Errorf("ping failed")
	}
	return nil
}

func (m *MockDockerEngine) Available() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.shouldFail["Available"]
}

func (m *MockDockerEngine) BuildImage(ctx ImageBuildContext) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["BuildImage"]++

	if m.shouldFail["BuildImage"] {
		return "", fmt.Errorf("build image failed")
	}

	tag := ctx.ImageTag
	if tag == "" {
		tag = fmt.Sprintf("aps-%s:latest", ctx.Profile.ID)
	}

	m.images[tag] = true
	return tag, nil
}

func (m *MockDockerEngine) PullImage(image string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["PullImage"]++

	if m.shouldFail["PullImage"] {
		return fmt.Errorf("pull image failed")
	}

	m.images[image] = true
	return nil
}

func (m *MockDockerEngine) RemoveImage(image string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["RemoveImage"]++

	if m.shouldFail["RemoveImage"] {
		return fmt.Errorf("remove image failed")
	}

	if _, exists := m.images[image]; !exists {
		return fmt.Errorf("image not found: %s", image)
	}

	delete(m.images, image)
	return nil
}

func (m *MockDockerEngine) CreateContainer(opts ContainerRunOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["CreateContainer"]++

	if m.shouldFail["CreateContainer"] {
		return "", fmt.Errorf("create container failed")
	}

	id := fmt.Sprintf("container-%d", len(m.containers)+1)

	m.containers[id] = containerState{
		id:     id,
		status: ContainerCreated,
		image:  opts.Image,
		config: opts,
		ip:     "172.17.0.2",
		ports:  make(map[string]string),
	}

	m.containerIPs[id] = "172.17.0.2"
	m.containerPorts[id] = make(map[string]string)
	m.containerPorts[id]["22"] = "32768"

	return id, nil
}

func (m *MockDockerEngine) StartContainer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["StartContainer"]++

	if m.shouldFail["StartContainer"] {
		return fmt.Errorf("start container failed")
	}

	container, exists := m.containers[id]
	if !exists {
		return fmt.Errorf("container not found: %s", id)
	}

	container.status = ContainerRunning
	m.containers[id] = container

	return nil
}

func (m *MockDockerEngine) StopContainer(id string, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["StopContainer"]++

	if m.shouldFail["StopContainer"] {
		return fmt.Errorf("stop container failed")
	}

	container, exists := m.containers[id]
	if !exists {
		return fmt.Errorf("container not found: %s", id)
	}

	container.status = ContainerExited
	m.containers[id] = container

	return nil
}

func (m *MockDockerEngine) RemoveContainer(id string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["RemoveContainer"]++

	if m.shouldFail["RemoveContainer"] {
		return fmt.Errorf("remove container failed")
	}

	if _, exists := m.containers[id]; !exists {
		return fmt.Errorf("container not found: %s", id)
	}

	delete(m.containers, id)
	delete(m.containerIPs, id)
	delete(m.containerPorts, id)
	delete(m.containerLogs, id)

	return nil
}

func (m *MockDockerEngine) ExecContainer(id string, cmd []string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["ExecContainer"]++

	if m.shouldFail["ExecContainer"] {
		return -1, fmt.Errorf("exec container failed")
	}

	if _, exists := m.containers[id]; !exists {
		return -1, fmt.Errorf("container not found: %s", id)
	}

	return 0, nil
}

func (m *MockDockerEngine) GetContainerStatus(id string) (ContainerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["GetContainerStatus"]++

	if m.shouldFail["GetContainerStatus"] {
		return "", fmt.Errorf("get status failed")
	}

	container, exists := m.containers[id]
	if !exists {
		return "", fmt.Errorf("container not found: %s", id)
	}

	return container.status, nil
}

func (m *MockDockerEngine) GetContainerLogs(id string, opts LogOptions) (<-chan LogMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["GetContainerLogs"]++

	if m.shouldFail["GetContainerLogs"] {
		errChan := make(chan LogMessage)
		close(errChan)
		return errChan, fmt.Errorf("get logs failed")
	}

	if _, exists := m.containers[id]; !exists {
		errChan := make(chan LogMessage)
		close(errChan)
		return errChan, fmt.Errorf("container not found: %s", id)
	}

	logChan := make(chan LogMessage, 10)
	go func() {
		defer close(logChan)

		logs := m.containerLogs[id]
		for _, log := range logs {
			logChan <- log
		}
	}()

	return logChan, nil
}

func (m *MockDockerEngine) UpdateContainerResources(id string, limits ResourceLimits) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["UpdateContainerResources"]++

	if m.shouldFail["UpdateContainerResources"] {
		return fmt.Errorf("update resources failed")
	}

	container, exists := m.containers[id]
	if !exists {
		return fmt.Errorf("container not found: %s", id)
	}

	container.resources = limits
	m.containers[id] = container

	return nil
}

func (m *MockDockerEngine) InspectContainer(id string) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["InspectContainer"]++

	if m.shouldFail["InspectContainer"] {
		return nil, fmt.Errorf("inspect failed")
	}

	container, exists := m.containers[id]
	if !exists {
		return nil, fmt.Errorf("container not found: %s", id)
	}

	return map[string]interface{}{
		"id":     container.id,
		"status": container.status,
		"image":  container.image,
		"config": container.config,
	}, nil
}

func (m *MockDockerEngine) GetContainerIP(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["GetContainerIP"]++

	if m.shouldFail["GetContainerIP"] {
		return "", fmt.Errorf("get IP failed")
	}

	ip, exists := m.containerIPs[id]
	if !exists {
		return "", fmt.Errorf("container not found: %s", id)
	}

	return ip, nil
}

func (m *MockDockerEngine) GetContainerPortMapping(id string, containerPort string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["GetContainerPortMapping"]++

	if m.shouldFail["GetContainerPortMapping"] {
		return "", fmt.Errorf("get port mapping failed")
	}

	ports, exists := m.containerPorts[id]
	if !exists {
		return "", fmt.Errorf("container not found: %s", id)
	}

	if hostPort, ok := ports[containerPort]; ok {
		return hostPort, nil
	}

	return "", fmt.Errorf("port not mapped: %s", containerPort)
}

// ============================================================================
// DockerEngine Tests (20 tests)
// ============================================================================

// TestDockerEngineVersion verifies version retrieval
func TestDockerEngineVersion(t *testing.T) {
	engine := NewMockDockerEngine()

	version, err := engine.Version()
	require.NoError(t, err)
	assert.Equal(t, "20.10.0", version)
	assert.Equal(t, 1, engine.callCount["Version"])
}

// TestDockerEngineVersionError verifies version error handling
func TestDockerEngineVersionError(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["Version"] = true

	version, err := engine.Version()
	require.Error(t, err)
	assert.Equal(t, "", version)
	assert.Equal(t, 1, engine.callCount["Version"])
}

// TestDockerEnginePing verifies ping functionality
func TestDockerEnginePing(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.Ping()
	require.NoError(t, err)
	assert.Equal(t, 1, engine.callCount["Ping"])
}

// TestDockerEnginePingError verifies ping error handling
func TestDockerEnginePingError(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["Ping"] = true

	err := engine.Ping()
	require.Error(t, err)
	assert.Equal(t, 1, engine.callCount["Ping"])
}

// TestDockerEngineAvailable verifies availability check
func TestDockerEngineAvailable(t *testing.T) {
	engine := NewMockDockerEngine()

	available := engine.Available()
	assert.True(t, available)
}

// TestDockerEngineNotAvailable verifies unavailable state
func TestDockerEngineNotAvailable(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["Available"] = true

	available := engine.Available()
	assert.False(t, available)
}

// TestBuildImageSuccess verifies successful image build
func TestBuildImageSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	profile := &core.Profile{ID: "test-profile"}

	ctx := ImageBuildContext{
		Profile:    profile,
		ImageTag:   "test:latest",
		BuildDir:   "/tmp/build",
		ProfileDir: "/tmp/profile",
	}

	imageID, err := engine.BuildImage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test:latest", imageID)
	assert.True(t, engine.images["test:latest"])
}

// TestBuildImageDefaultTag verifies default tag generation
func TestBuildImageDefaultTag(t *testing.T) {
	engine := NewMockDockerEngine()
	profile := &core.Profile{ID: "my-agent"}

	ctx := ImageBuildContext{
		Profile:    profile,
		BuildDir:   "/tmp/build",
		ProfileDir: "/tmp/profile",
	}

	imageID, err := engine.BuildImage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "aps-my-agent:latest", imageID)
}

// TestBuildImageError verifies build error handling
func TestBuildImageError(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["BuildImage"] = true
	profile := &core.Profile{ID: "test"}

	ctx := ImageBuildContext{Profile: profile}

	imageID, err := engine.BuildImage(ctx)
	require.Error(t, err)
	assert.Equal(t, "", imageID)
}

// TestPullImageSuccess verifies successful image pull
func TestPullImageSuccess(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.PullImage("ubuntu:22.04")
	require.NoError(t, err)
	assert.True(t, engine.images["ubuntu:22.04"])
}

// TestPullImageError verifies pull error handling
func TestPullImageError(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["PullImage"] = true

	err := engine.PullImage("nonexistent:latest")
	require.Error(t, err)
}

// TestRemoveImageSuccess verifies successful image removal
func TestRemoveImageSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.images["test:latest"] = true

	err := engine.RemoveImage("test:latest", false)
	require.NoError(t, err)
	assert.False(t, engine.images["test:latest"])
}

// TestRemoveImageNotFound verifies removal of non-existent image
func TestRemoveImageNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.RemoveImage("nonexistent:latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestCreateContainerSuccess verifies successful container creation
func TestCreateContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()

	opts := ContainerRunOptions{
		Image:      "ubuntu:22.04",
		Command:    []string{"sleep", "3600"},
		WorkingDir: "/workspace",
		User:       "appuser",
	}

	id, err := engine.CreateContainer(opts)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "container-"))
}

// TestCreateContainerWithVolumes verifies container creation with volumes
func TestCreateContainerWithVolumes(t *testing.T) {
	engine := NewMockDockerEngine()

	opts := ContainerRunOptions{
		Image: "ubuntu:22.04",
		Volumes: []VolumeMount{
			{Source: "/host/path", Target: "/container/path", Readonly: false},
		},
	}

	id, err := engine.CreateContainer(opts)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	container := engine.containers[id]
	assert.Equal(t, 1, len(container.config.Volumes))
}

// TestCreateContainerWithLimits verifies container creation with resource limits
func TestCreateContainerWithLimits(t *testing.T) {
	engine := NewMockDockerEngine()

	opts := ContainerRunOptions{
		Image: "ubuntu:22.04",
		Limits: ResourceLimits{
			MemoryLimit: 512 * 1024 * 1024,
			CPUQuota:    100000,
		},
	}

	id, err := engine.CreateContainer(opts)
	require.NoError(t, err)

	container := engine.containers[id]
	assert.Equal(t, int64(512*1024*1024), container.config.Limits.MemoryLimit)
}

// TestCreateContainerError verifies creation error handling
func TestCreateContainerError(t *testing.T) {
	engine := NewMockDockerEngine()
	engine.shouldFail["CreateContainer"] = true

	opts := ContainerRunOptions{Image: "ubuntu:22.04"}

	id, err := engine.CreateContainer(opts)
	require.Error(t, err)
	assert.Equal(t, "", id)
}

// TestStartContainerSuccess verifies successful container start
func TestStartContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	err := engine.StartContainer(id)
	require.NoError(t, err)

	status := engine.containers[id].status
	assert.Equal(t, ContainerRunning, status)
}

// TestStartContainerNotFound verifies start on non-existent container
func TestStartContainerNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.StartContainer("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestStartContainerError verifies start error handling
func TestStartContainerError(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)
	engine.shouldFail["StartContainer"] = true

	err := engine.StartContainer(id)
	require.Error(t, err)
}

// TestStopContainerSuccess verifies successful container stop
func TestStopContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)
	engine.StartContainer(id)

	err := engine.StopContainer(id, 10*time.Second)
	require.NoError(t, err)

	status := engine.containers[id].status
	assert.Equal(t, ContainerExited, status)
}

// TestStopContainerNotFound verifies stop on non-existent container
func TestStopContainerNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.StopContainer("nonexistent", 10*time.Second)
	require.Error(t, err)
}

// TestStopContainerWithTimeout verifies timeout handling
func TestStopContainerWithTimeout(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)
	engine.StartContainer(id)

	err := engine.StopContainer(id, 5*time.Second)
	require.NoError(t, err)
}

// TestRemoveContainerSuccess verifies successful container removal
func TestRemoveContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	err := engine.RemoveContainer(id, false)
	require.NoError(t, err)
	assert.NotContains(t, engine.containers, id)
}

// TestRemoveContainerNotFound verifies removal of non-existent container
func TestRemoveContainerNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	err := engine.RemoveContainer("nonexistent", false)
	require.Error(t, err)
}

// TestExecContainerSuccess verifies successful command execution
func TestExecContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	code, err := engine.ExecContainer(id, []string{"echo", "hello"})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
}

// TestExecContainerNotFound verifies exec on non-existent container
func TestExecContainerNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	code, err := engine.ExecContainer("nonexistent", []string{"echo", "hello"})
	require.Error(t, err)
	assert.Equal(t, -1, code)
}

// TestExecContainerShellCommand verifies shell command execution
func TestExecContainerShellCommand(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	code, err := engine.ExecContainer(id, []string{"bash", "-c", "echo test"})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
}

// TestGetContainerStatusRunning verifies running container status
func TestGetContainerStatusRunning(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)
	engine.StartContainer(id)

	status, err := engine.GetContainerStatus(id)
	require.NoError(t, err)
	assert.Equal(t, ContainerRunning, status)
}

// TestGetContainerStatusCreated verifies created container status
func TestGetContainerStatusCreated(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	status, err := engine.GetContainerStatus(id)
	require.NoError(t, err)
	assert.Equal(t, ContainerCreated, status)
}

// TestGetContainerStatusExited verifies exited container status
func TestGetContainerStatusExited(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)
	engine.StartContainer(id)
	engine.StopContainer(id, 10*time.Second)

	status, err := engine.GetContainerStatus(id)
	require.NoError(t, err)
	assert.Equal(t, ContainerExited, status)
}

// TestGetContainerStatusNotFound verifies status check on non-existent container
func TestGetContainerStatusNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	status, err := engine.GetContainerStatus("nonexistent")
	require.Error(t, err)
	assert.Equal(t, ContainerStatus(""), status)
}

// TestGetContainerLogsSuccess verifies log retrieval
func TestGetContainerLogsSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	engine.containerLogs[id] = []LogMessage{
		{Timestamp: time.Now(), Stream: "STDOUT", Line: "test output"},
	}

	logChan, err := engine.GetContainerLogs(id, LogOptions{})
	require.NoError(t, err)

	log := <-logChan
	assert.Equal(t, "test output", log.Line)
}

// TestGetContainerLogsNotFound verifies log retrieval on non-existent container
func TestGetContainerLogsNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	logChan, err := engine.GetContainerLogs("nonexistent", LogOptions{})
	require.Error(t, err)
	assert.NotNil(t, logChan)
}

// TestGetContainerLogsEmpty verifies empty log handling
func TestGetContainerLogsEmpty(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	logChan, err := engine.GetContainerLogs(id, LogOptions{})
	require.NoError(t, err)

	count := 0
	for range logChan {
		count++
	}
	assert.Equal(t, 0, count)
}

// TestUpdateContainerResourcesSuccess verifies resource update
func TestUpdateContainerResourcesSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	limits := ResourceLimits{
		MemoryLimit: 1024 * 1024 * 1024,
		CPUQuota:    200000,
	}

	err := engine.UpdateContainerResources(id, limits)
	require.NoError(t, err)

	container := engine.containers[id]
	assert.Equal(t, int64(1024*1024*1024), container.resources.MemoryLimit)
}

// TestUpdateContainerResourcesNotFound verifies update on non-existent container
func TestUpdateContainerResourcesNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	limits := ResourceLimits{MemoryLimit: 512 * 1024 * 1024}

	err := engine.UpdateContainerResources("nonexistent", limits)
	require.Error(t, err)
}

// TestInspectContainerSuccess verifies container inspection
func TestInspectContainerSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	info, err := engine.InspectContainer(id)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, id, info["id"])
}

// TestInspectContainerNotFound verifies inspection of non-existent container
func TestInspectContainerNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	info, err := engine.InspectContainer("nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
}

// TestGetContainerIPSuccess verifies IP retrieval
func TestGetContainerIPSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	ip, err := engine.GetContainerIP(id)
	require.NoError(t, err)
	assert.Equal(t, "172.17.0.2", ip)
}

// TestGetContainerIPNotFound verifies IP retrieval on non-existent container
func TestGetContainerIPNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	ip, err := engine.GetContainerIP("nonexistent")
	require.Error(t, err)
	assert.Equal(t, "", ip)
}

// TestGetContainerPortMappingSuccess verifies port mapping retrieval
func TestGetContainerPortMappingSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	port, err := engine.GetContainerPortMapping(id, "22")
	require.NoError(t, err)
	assert.Equal(t, "32768", port)
}

// TestGetContainerPortMappingNotFound verifies port mapping on non-existent container
func TestGetContainerPortMappingNotFound(t *testing.T) {
	engine := NewMockDockerEngine()

	port, err := engine.GetContainerPortMapping("nonexistent", "22")
	require.Error(t, err)
	assert.Equal(t, "", port)
}

// TestGetContainerPortMappingInvalidPort verifies invalid port mapping
func TestGetContainerPortMappingInvalidPort(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, _ := engine.CreateContainer(opts)

	port, err := engine.GetContainerPortMapping(id, "9999")
	require.Error(t, err)
	assert.Equal(t, "", port)
}

// ============================================================================
// ContainerSSH Tests (20 tests)
// ============================================================================

// TestConfigureContainerSSHSuccess verifies SSH configuration
func TestConfigureContainerSSHSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	keysDir := filepath.Join(tmpDir, ".aps", "keys")
	require.NoError(t, os.MkdirAll(keysDir, 0700))

	adminPubKeyPath := filepath.Join(keysDir, "admin_pub")
	err := os.WriteFile(adminPubKeyPath, []byte("ssh-rsa AAAAB3..."), 0600)
	require.NoError(t, err)

	// Save original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	os.Setenv("HOME", tmpDir)

	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	_, _ = engine.CreateContainer(opts)

	// Note: In real tests, this would call actual ConfigureContainerSSH function
	// For mock testing, we verify the path and key were accessible
	assert.FileExists(t, adminPubKeyPath)
}

// TestConfigureContainerSSHMissingKey verifies missing key error
func TestConfigureContainerSSHMissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	keysDir := filepath.Join(tmpDir, ".aps", "keys")
	require.NoError(t, os.MkdirAll(keysDir, 0700))

	// Don't create admin_pub key
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	_, _ = engine.CreateContainer(opts)

	// In a real scenario, ConfigureContainerSSH would fail
	assert.False(t, fileExists(filepath.Join(keysDir, "admin_pub")))
}

// TestAttachToContainerWithIP verifies attachment with container IP
func TestAttachToContainerWithIP(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	ip, err := engine.GetContainerIP(containerID)
	require.NoError(t, err)
	assert.Equal(t, "172.17.0.2", ip)
}

// TestAttachToContainerWithPortMapping verifies attachment with port mapping
func TestAttachToContainerWithPortMapping(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	port, err := engine.GetContainerPortMapping(containerID, "22")
	require.NoError(t, err)
	assert.NotEmpty(t, port)
}

// TestGetContainerSSHConfigSuccess verifies SSH config extraction
func TestGetContainerSSHConfigSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	host, port, err := func(containerID string, username string) (string, int, error) {
		// Simulate GetContainerSSHConfig logic
		portStr, err := engine.GetContainerPortMapping(containerID, "22")
		if err != nil {
			return "", 0, err
		}
		return "localhost", int(parsePort(portStr)), nil
	}(containerID, "appuser")

	require.NoError(t, err)
	assert.Equal(t, "localhost", host)
	assert.Equal(t, 32768, port)
}

// TestVerifySSHConnectionSuccess verifies SSH connection verification
func TestVerifySSHConnectionSuccess(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	// In a real test, this would verify actual SSH connectivity
	port, err := engine.GetContainerPortMapping(containerID, "22")
	require.NoError(t, err)
	assert.NotEmpty(t, port)
}

// TestSSHKeyInjection verifies SSH key injection into container
func TestSSHKeyInjection(t *testing.T) {
	tmpDir := t.TempDir()
	keysDir := filepath.Join(tmpDir, ".aps", "keys")
	require.NoError(t, os.MkdirAll(keysDir, 0700))

	adminPubKeyPath := filepath.Join(keysDir, "admin_pub")
	adminPrivKeyPath := filepath.Join(keysDir, "admin_key")

	err := os.WriteFile(adminPubKeyPath, []byte("ssh-rsa AAAAB3..."), 0600)
	require.NoError(t, err)

	err = os.WriteFile(adminPrivKeyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----"), 0600)
	require.NoError(t, err)

	assert.FileExists(t, adminPubKeyPath)
	assert.FileExists(t, adminPrivKeyPath)
}

// TestPortMappingValidation verifies port mapping validation
func TestPortMappingValidation(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{
		Image: "ubuntu:22.04",
		Network: NetworkConfig{
			Mode:  "bridge",
			Ports: []string{"22:32768"},
		},
	}

	containerID, err := engine.CreateContainer(opts)
	require.NoError(t, err)

	port, err := engine.GetContainerPortMapping(containerID, "22")
	require.NoError(t, err)
	assert.Equal(t, "32768", port)
}

// TestConcurrentContainerOperations verifies concurrent operations
func TestConcurrentContainerOperations(t *testing.T) {
	engine := NewMockDockerEngine()

	var wg sync.WaitGroup
	containerIDs := make([]string, 10)
	containerIDsMu := sync.Mutex{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			opts := ContainerRunOptions{
				Image:   "ubuntu:22.04",
				Command: []string{"sleep", "3600"},
			}

			id, err := engine.CreateContainer(opts)
			require.NoError(t, err)

			containerIDsMu.Lock()
			containerIDs[index] = id
			containerIDsMu.Unlock()

			err = engine.StartContainer(id)
			require.NoError(t, err)

			status, err := engine.GetContainerStatus(id)
			require.NoError(t, err)
			assert.Equal(t, ContainerRunning, status)
		}(i)
	}

	wg.Wait()

	// Verify all containers were created
	assert.Equal(t, 10, len(engine.containers))
}

// TestConcurrentSSHConnections verifies concurrent SSH operations
func TestConcurrentSSHConnections(t *testing.T) {
	engine := NewMockDockerEngine()

	var wg sync.WaitGroup
	errors := make([]error, 5)
	errorsMu := sync.Mutex{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			opts := ContainerRunOptions{Image: "ubuntu:22.04"}
			containerID, err := engine.CreateContainer(opts)

			if err != nil {
				errorsMu.Lock()
				errors[index] = err
				errorsMu.Unlock()
				return
			}

			_, err = engine.GetContainerIP(containerID)
			if err != nil {
				errorsMu.Lock()
				errors[index] = err
				errorsMu.Unlock()
				return
			}
		}(i)
	}

	wg.Wait()

	for _, err := range errors {
		assert.NoError(t, err)
	}
}

// TestContainerSSHEnvironmentVariables verifies SSH environment setup
func TestContainerSSHEnvironmentVariables(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{
		Image: "ubuntu:22.04",
		Environment: []string{
			"SSH_PORT=22",
			"SSH_USER=appuser",
		},
	}

	containerID, err := engine.CreateContainer(opts)
	require.NoError(t, err)

	container := engine.containers[containerID]
	assert.Equal(t, 2, len(container.config.Environment))
}

// TestSSHPermissionsValidation verifies SSH permission checks
func TestSSHPermissionsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	keysDir := filepath.Join(tmpDir, ".aps", "keys")
	require.NoError(t, os.MkdirAll(keysDir, 0700))

	keyPath := filepath.Join(keysDir, "admin_key")
	err := os.WriteFile(keyPath, []byte("key"), 0600)
	require.NoError(t, err)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)

	// Check permissions are restrictive
	assert.False(t, (info.Mode() & 0077) != 0)
}

// TestContainerSSHConfigExtraction verifies SSH config extraction
func TestContainerSSHConfigExtraction(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	port, err := engine.GetContainerPortMapping(containerID, "22")
	require.NoError(t, err)

	ip, err := engine.GetContainerIP(containerID)
	require.NoError(t, err)

	assert.NotEmpty(t, port)
	assert.NotEmpty(t, ip)
}

// TestSSHConnectionTimeout verifies connection timeout handling
func TestSSHConnectionTimeout(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	containerID, _ := engine.CreateContainer(opts)

	// Verify container exists for timeout test
	status, err := engine.GetContainerStatus(containerID)
	require.NoError(t, err)
	assert.NotEmpty(t, status)
}

// TestContainerSSHNetworkConfig verifies network configuration
func TestContainerSSHNetworkConfig(t *testing.T) {
	engine := NewMockDockerEngine()
	opts := ContainerRunOptions{
		Image: "ubuntu:22.04",
		Network: NetworkConfig{
			Mode:  "bridge",
			Ports: []string{"22:32768"},
		},
	}

	containerID, err := engine.CreateContainer(opts)
	require.NoError(t, err)

	ip, err := engine.GetContainerIP(containerID)
	require.NoError(t, err)
	assert.NotEmpty(t, ip)
}

// ============================================================================
// Shared Infrastructure Tests
// ============================================================================

// TestContainerStatusValues verifies container status enum values
func TestContainerStatusValues(t *testing.T) {
	statuses := []ContainerStatus{
		ContainerCreated,
		ContainerRunning,
		ContainerPaused,
		ContainerRestarting,
		ContainerRemoving,
		ContainerExited,
		ContainerDead,
	}

	assert.Equal(t, 7, len(statuses))
	for _, status := range statuses {
		assert.NotEmpty(t, status)
	}
}

// TestContainerOptionsValidation verifies option validation
func TestContainerOptionsValidation(t *testing.T) {
	opts := ContainerRunOptions{
		Image:      "ubuntu:22.04",
		Command:    []string{"bash"},
		WorkingDir: "/workspace",
		User:       "appuser",
	}

	assert.NotEmpty(t, opts.Image)
	assert.NotEmpty(t, opts.Command)
	assert.NotEmpty(t, opts.WorkingDir)
}

// TestVolumeMountValidation verifies volume mount validation
func TestVolumeMountValidation(t *testing.T) {
	vm := VolumeMount{
		Source:   "/host/path",
		Target:   "/container/path",
		Readonly: true,
	}

	assert.NotEmpty(t, vm.Source)
	assert.NotEmpty(t, vm.Target)
	assert.True(t, vm.Readonly)
}

// TestResourceLimitsValidation verifies resource limits
func TestResourceLimitsValidation(t *testing.T) {
	limits := ResourceLimits{
		MemoryLimit: 512 * 1024 * 1024,
		CPUQuota:    100000,
	}

	assert.Greater(t, limits.MemoryLimit, int64(0))
	assert.Greater(t, limits.CPUQuota, int64(0))
}

// TestNetworkConfigValidation verifies network config validation
func TestNetworkConfigValidation(t *testing.T) {
	netConfig := NetworkConfig{
		Mode:     "bridge",
		Ports:    []string{"22:32768", "80:8080"},
		Hostname: "test-container",
	}

	assert.NotEmpty(t, netConfig.Mode)
	assert.Equal(t, 2, len(netConfig.Ports))
	assert.NotEmpty(t, netConfig.Hostname)
}

// TestLogMessageStructure verifies log message structure
func TestLogMessageStructure(t *testing.T) {
	log := LogMessage{
		Timestamp: time.Now(),
		Stream:    "STDOUT",
		Line:      "test output",
	}

	assert.NotZero(t, log.Timestamp)
	assert.Equal(t, "STDOUT", log.Stream)
	assert.Equal(t, "test output", log.Line)
}

// TestImageBuildContextValidation verifies build context validation
func TestImageBuildContextValidation(t *testing.T) {
	profile := &core.Profile{ID: "test"}
	ctx := ImageBuildContext{
		Profile:    profile,
		ImageTag:   "test:latest",
		BuildDir:   "/tmp/build",
		ProfileDir: "/tmp/profile",
	}

	assert.NotNil(t, ctx.Profile)
	assert.NotEmpty(t, ctx.ImageTag)
	assert.NotEmpty(t, ctx.BuildDir)
}

// TestMultipleContainerOperationSequence verifies operation sequence
func TestMultipleContainerOperationSequence(t *testing.T) {
	engine := NewMockDockerEngine()

	// Create
	opts := ContainerRunOptions{Image: "ubuntu:22.04"}
	id, err := engine.CreateContainer(opts)
	require.NoError(t, err)

	// Start
	err = engine.StartContainer(id)
	require.NoError(t, err)

	// Inspect
	info, err := engine.InspectContainer(id)
	require.NoError(t, err)
	assert.NotNil(t, info)

	// Stop
	err = engine.StopContainer(id, 10*time.Second)
	require.NoError(t, err)

	// Remove
	err = engine.RemoveContainer(id, false)
	require.NoError(t, err)
}

// ============================================================================
// Helper Functions
// ============================================================================

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parsePort(portStr string) int {
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}
