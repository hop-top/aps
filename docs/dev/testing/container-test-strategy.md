# Container Isolation Test Strategy

**Date**: 2026-01-21
**Status**: Draft
**Related**: Task T-0002

## Testing Philosophy

Container isolation requires a multi-layered testing approach:

1. **Unit Tests**: Test individual components in isolation (mocked dependencies)
2. **Integration Tests**: Test interactions between components (real container engine)
3. **E2E Tests**: Test full workflows (profile creation → container execution → cleanup)
4. **Performance Tests**: Measure resource usage and startup times
5. **Security Tests**: Verify isolation boundaries and privilege restrictions

## Test Matrix

| Test Type | Local | CI (Docker) | CI (Podman) | Notes |
|-----------|-------|-------------|--------------|-------|
| Unit      | ✅    | ✅          | ✅           | No external deps |
| Integration | ✅  | ✅          | ✅           | Requires Docker/Podman |
| E2E       | ✅    | ✅          | ❌           | Docker only in CI |
| Performance| ❌    | ✅          | ❌           | Consistent env needed |
| Security  | ✅    | ✅          | ❌           | Docker only in CI |

## Unit Tests

### Test Structure

```
tests/unit/core/isolation/
├── container_docker_test.go       # Docker engine tests
├── container_engine_test.go        # ContainerEngine interface tests
├── container_image_builder_test.go # Dockerfile generation tests
├── container_limits_test.go        # Resource limits validation tests
├── container_registry_test.go     # Session registry tests
└── container_session_test.go      # Container session lifecycle tests
```

### ContainerEngine Tests

```go
// tests/unit/core/isolation/container_docker_test.go

package isolation

import (
    "context"
    "testing"
    "time"

    "github.com/docker/docker/api/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock Docker Client
type MockDockerClient struct {
    mock.Mock
}

func (m *MockDockerClient) ImageBuild(...) (types.ImageBuildResponse, error) {
    args := m.Called()
    return args.Get(0).(types.ImageBuildResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerCreate(...) (container.ContainerCreateCreatedBody, error) {
    args := m.Called()
    return args.Get(0).(container.ContainerCreateCreatedBody), args.Error(1)
}

func (m *MockDockerClient) ContainerStart(...) error {
    args := m.Called()
    return args.Error(0)
}

// Tests
func TestDockerEngine_Name(t *testing.T) {
    engine := &DockerEngine{}
    assert.Equal(t, "docker", engine.Name())
}

func TestDockerEngine_Ping_Success(t *testing.T) {
    mockClient := &MockDockerClient{}
    mockClient.On("Ping").Return(nil)

    engine := &DockerEngine{client: mockClient}
    err := engine.Ping()

    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}

func TestDockerEngine_Ping_Failure(t *testing.T) {
    mockClient := &MockDockerClient{}
    mockClient.On("Ping").Return(assert.AnError)

    engine := &DockerEngine{client: mockClient}
    err := engine.Ping()

    assert.Error(t, err)
    mockClient.AssertExpectations(t)
}

func TestDockerEngine_Available_True(t *testing.T) {
    mockClient := &MockDockerClient{}
    mockClient.On("Ping").Return(nil)

    engine := &DockerEngine{client: mockClient}
    available := engine.Available()

    assert.True(t, available)
    mockClient.AssertExpectations(t)
}

func TestDockerEngine_Available_False(t *testing.T) {
    mockClient := &MockDockerClient{}
    mockClient.On("Ping").Return(assert.AnError)

    engine := &DockerEngine{client: mockClient}
    available := engine.Available()

    assert.False(t, available)
    mockClient.AssertExpectations(t)
}
```

### ImageBuilder Tests

```go
// tests/unit/core/isolation/container_image_builder_test.go

func TestDockerfileBuilder_Generate_Basic(t *testing.T) {
    builder := &DockerfileBuilder{}

    profile := &core.Profile{
        ID: "test-profile",
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Image: "ubuntu:22.04",
            },
        },
    }

    dockerfile, err := builder.Generate(profile)

    assert.NoError(t, err)
    assert.Contains(t, dockerfile, "FROM ubuntu:22.04")
}

func TestDockerfileBuilder_Generate_WithPackages(t *testing.T) {
    builder := &DockerfileBuilder{}

    profile := &core.Profile{
        ID: "test-profile",
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Image: "ubuntu:22.04",
                Packages: []string{"nodejs", "python3", "git"},
            },
        },
    }

    dockerfile, err := builder.Generate(profile)

    assert.NoError(t, err)
    assert.Contains(t, dockerfile, "RUN apt-get update && apt-get install -y")
    assert.Contains(t, dockerfile, "nodejs")
    assert.Contains(t, dockerfile, "python3")
    assert.Contains(t, dockerfile, "git")
}

func TestDockerfileBuilder_Generate_WithBuildSteps(t *testing.T) {
    builder := &DockerfileBuilder{}

    profile := &core.Profile{
        ID: "test-profile",
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Image: "ubuntu:22.04",
                BuildSteps: []core.BuildStep{
                    {Type: "shell", Run: "npm install -g @anthropic-ai/claude-code@latest"},
                    {Type: "shell", Run: "pip install -q google-generativeai"},
                },
            },
        },
    }

    dockerfile, err := builder.Generate(profile)

    assert.NoError(t, err)
    assert.Contains(t, dockerfile, "RUN npm install -g @anthropic-ai/claude-code@latest")
    assert.Contains(t, dockerfile, "RUN pip install -q google-generativeai")
}

func TestDockerfileBuilder_Generate_VolumeMounts(t *testing.T) {
    builder := &DockerfileBuilder{}

    profile := &core.Profile{
        ID: "test-profile",
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Image: "ubuntu:22.04",
                Volumes: []string{
                    "/host/path:/container/path",
                },
            },
        },
    }

    dockerfile, err := builder.Generate(profile)

    assert.NoError(t, err)
    assert.Contains(t, dockerfile, "VOLUME [\"/container/path\"]")
}
```

### Resource Limits Validation Tests

```go
// tests/unit/core/isolation/container_limits_test.go

func TestValidateContainerResources_Success(t *testing.T) {
    profile := &core.Profile{
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Resources: core.ContainerResources{
                    CPUQuota:   100000,
                    CPUPeriod:  100000,
                    MemoryMB:   1024,
                    MemorySwapMB: 2048,
                },
            },
        },
    }

    err := profile.ValidateContainerResources()

    assert.NoError(t, err)
}

func TestValidateContainerResources_MissingPeriod(t *testing.T) {
    profile := &core.Profile{
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Resources: core.ContainerResources{
                    CPUQuota: 100000,
                    // CPUPeriod missing
                },
            },
        },
    }

    err := profile.ValidateContainerResources()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cpu_period must be specified")
}

func TestValidateContainerResources_MemorySwapLessThanMemory(t *testing.T) {
    profile := &core.Profile{
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Resources: core.ContainerResources{
                    MemoryMB:   1024,
                    MemorySwapMB: 512, // Less than memory
                },
            },
        },
    }

    err := profile.ValidateContainerResources()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "memory_swap_mb must be >= memory_mb")
}

func TestValidateContainerResources_NegativeCPUQuota(t *testing.T) {
    profile := &core.Profile{
        Isolation: core.IsolationConfig{
            Container: core.ContainerConfig{
                Resources: core.ContainerResources{
                    CPUQuota: -100,
                },
            },
        },
    }

    err := profile.ValidateContainerResources()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "must be positive")
}
```

### Session Registry Tests

```go
// tests/unit/core/isolation/container_registry_test.go

func TestSessionRegistry_RegisterContainerSession(t *testing.T) {
    registry := session.GetRegistry()

    sess := &session.SessionInfo{
        ID:              "test-session",
        ProfileID:       "test-profile",
        Command:         "echo hello",
        Status:          session.SessionActive,
        Tier:            session.TierPremium,
        IsolationType:   core.IsolationContainer,
        ContainerID:     "abc123",
        ContainerImage:  "ubuntu:22.04",
        ContainerStatus: ContainerRunning,
        CreatedAt:       time.Now(),
        LastSeenAt:      time.Now(),
    }

    err := registry.Register(sess)

    assert.NoError(t, err)

    // Verify registration
    retrieved, err := registry.Get("test-session")
    assert.NoError(t, err)
    assert.Equal(t, "abc123", retrieved.ContainerID)
    assert.Equal(t, "ubuntu:22.04", retrieved.ContainerImage)

    // Cleanup
    _ = registry.Unregister("test-session")
}

func TestSessionRegistry_GetContainerSessions(t *testing.T) {
    registry := session.GetRegistry()

    // Register container session
    containerSess := &session.SessionInfo{
        ID:              "container-session",
        ProfileID:       "test-profile",
        Command:         "echo hello",
        Status:          session.SessionActive,
        Tier:            session.TierPremium,
        IsolationType:   core.IsolationContainer,
        ContainerID:     "abc123",
        CreatedAt:       time.Now(),
        LastSeenAt:      time.Now(),
    }

    // Register process session
    processSess := &session.SessionInfo{
        ID:              "process-session",
        ProfileID:       "test-profile",
        Command:         "echo hello",
        Status:          session.SessionActive,
        Tier:            session.TierStandard,
        IsolationType:   core.IsolationProcess,
        CreatedAt:       time.Now(),
        LastSeenAt:      time.Now(),
    }

    _ = registry.Register(containerSess)
    _ = registry.Register(processSess)

    // Get container sessions only
    containerSessions := registry.GetContainerSessions()

    assert.Len(t, containerSessions, 1)
    assert.Equal(t, "container-session", containerSessions[0].ID)

    // Cleanup
    _ = registry.Unregister("container-session")
    _ = registry.Unregister("process-session")
}

func TestSessionRegistry_UpdateContainerStatus(t *testing.T) {
    registry := session.GetRegistry()

    sess := &session.SessionInfo{
        ID:              "test-session",
        ProfileID:       "test-profile",
        Command:         "echo hello",
        Status:          session.SessionActive,
        ContainerStatus: ContainerRunning,
        CreatedAt:       time.Now(),
        LastSeenAt:      time.Now(),
    }

    _ = registry.Register(sess)

    // Update status to exited
    err := registry.UpdateContainerStatus("test-session", ContainerExited)
    assert.NoError(t, err)

    // Verify update
    updated, _ := registry.Get("test-session")
    assert.Equal(t, ContainerExited, updated.ContainerStatus)
    assert.Equal(t, session.SessionInactive, updated.Status)

    // Cleanup
    _ = registry.Unregister("test-session")
}
```

## Integration Tests

### Test Structure

```
tests/integration/isolation/
├── container_lifecycle_test.go    # Container create/start/stop/remove
├── container_exec_test.go        # Command execution in containers
├── container_volumes_test.go     # Volume mounting
├── container_network_test.go     # Network configuration
├── container_resources_test.go   # Resource limits
└── container_fallback_test.go    # Fallback behavior
```

### Container Lifecycle Tests

```go
// tests/integration/isolation/container_lifecycle_test.go

//go:build integration && !windows
// +build integration,!windows

package isolation

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestContainerLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create engine
    engine, err := NewDockerEngine()
    require.NoError(t, err)
    require.True(t, engine.Available())

    // Create container
    opts := ContainerRunOptions{
        Image:    "alpine:latest",
        Command:  []string{"echo", "hello"},
        Network:  NetworkConfig{Mode: "none"},
    }

    containerID, err := engine.CreateContainer(opts)
    require.NoError(t, err)
    assert.NotEmpty(t, containerID)

    // Verify status
    status, err := engine.GetContainerStatus(containerID)
    require.NoError(t, err)
    assert.Equal(t, ContainerCreated, status)

    // Start container
    err = engine.StartContainer(containerID)
    require.NoError(t, err)

    // Verify running
    status, err = engine.GetContainerStatus(containerID)
    require.NoError(t, err)
    assert.Equal(t, ContainerRunning, status)

    // Wait for exit
    time.Sleep(1 * time.Second)

    // Stop container
    timeout := 5 * time.Second
    err = engine.StopContainer(containerID, timeout)
    require.NoError(t, err)

    // Verify stopped
    status, err = engine.GetContainerStatus(containerID)
    require.NoError(t, err)
    assert.Equal(t, ContainerStopped, status)

    // Remove container
    err = engine.RemoveContainer(containerID, false)
    require.NoError(t, err)
}

func TestContainerExec(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    engine, err := NewDockerEngine()
    require.NoError(t, err)

    // Create and start container
    opts := ContainerRunOptions{
        Image:    "alpine:latest",
        Command:  []string{"sleep", "30"},
        Network:  NetworkConfig{Mode: "none"},
    }

    containerID, err := engine.CreateContainer(opts)
    require.NoError(t, err)
    defer engine.RemoveContainer(containerID, true)

    err = engine.StartContainer(containerID)
    require.NoError(t, err)
    defer engine.StopContainer(containerID, 5*time.Second)

    // Exec command
    cmd := []string{"echo", "test output"}
    exitCode, err := engine.ExecContainer(containerID, cmd)
    require.NoError(t, err)
    assert.Equal(t, 0, exitCode)
}

func TestContainerVolumeMounts(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create test file in temp directory
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    err := os.WriteFile(testFile, []byte("hello"), 0644)
    require.NoError(t, err)

    engine, err := NewDockerEngine()
    require.NoError(t, err)

    // Create container with volume mount
    opts := ContainerRunOptions{
        Image: "alpine:latest",
        Command: []string{"cat", "/data/test.txt"},
        Volumes: []string{
            fmt.Sprintf("%s:/data:ro", tmpDir),
        },
        Network: NetworkConfig{Mode: "none"},
    }

    containerID, err := engine.CreateContainer(opts)
    require.NoError(t, err)
    defer engine.RemoveContainer(containerID, true)

    err = engine.StartContainer(containerID)
    require.NoError(t, err)

    // Wait for execution
    time.Sleep(1 * time.Second)

    // Verify output
    logs := make(chan LogMessage, 10)
    go func() {
        opts := LogOptions{ShowStdout: true, ShowStderr: true}
        logs, err = engine.GetContainerLogs(containerID, opts)
    }()

    // Read logs and verify content
    found := false
    for msg := range logs {
        if strings.Contains(msg.Line, "hello") {
            found = true
            break
        }
    }
    assert.True(t, found, "expected to find 'hello' in container output")
}
```

## E2E Tests

### Test Structure

```
tests/e2e/container/
├── profile_test.go              # Profile creation with container isolation
├── execution_test.go            # Full command execution flow
├── session_test.go              # Session lifecycle management
├── ssh_test.go                 # SSH key mounting and agent forwarding
├── tmux_test.go                # Tmux session with containers
└── fallback_test.go            # Fallback from container to platform/process
```

### Profile Creation E2E Test

```go
// tests/e2e/container/profile_test.go

package container

import (
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreateContainerProfile(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Check Docker is available
    cli, err := client.NewClientWithOpts(client.FromEnv)
    require.NoError(t, err)
    _, err = cli.Ping(context.Background())
    require.NoError(t, err)

    // Create profile with container isolation
    profileID := "test-container-profile"
    config := core.Profile{
        DisplayName: "Test Container Profile",
        Isolation: core.IsolationConfig{
            Level:    core.IsolationContainer,
            Fallback: true,
            Container: core.ContainerConfig{
                Image: "alpine:latest",
                Packages: []string{"curl", "git"},
                Network: "none",
                Resources: core.ContainerResources{
                    MemoryMB: 512,
                    CPUQuota: 50000,
                    CPUPeriod: 100000,
                },
            },
        },
    }

    err = core.CreateProfile(profileID, config)
    require.NoError(t, err)
    defer os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".agents", "profiles", profileID))

    // Verify profile was created
    profile, err := core.LoadProfile(profileID)
    require.NoError(t, err)
    assert.Equal(t, core.IsolationContainer, profile.Isolation.Level)
    assert.Equal(t, "alpine:latest", profile.Isolation.Container.Image)
    assert.Equal(t, 512, profile.Isolation.Container.Resources.MemoryMB)
}
```

### Command Execution E2E Test

```go
// tests/e2e/container/execution_test.go

func TestExecuteInContainer(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Setup profile
    profileID := setupTestContainerProfile(t)
    defer cleanupTestProfile(t, profileID)

    // Execute command in container
    cmd := exec.Command("aps", "run", profileID, "--", "echo", "hello from container")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)
    assert.Contains(t, string(output), "hello from container")

    // Verify session was created
    sessions := session.GetRegistry().ListByProfile(profileID)
    assert.Len(t, sessions, 1)

    sess := sessions[0]
    assert.Equal(t, session.SessionActive, sess.Status)
    assert.Equal(t, core.IsolationContainer, sess.IsolationType)
    assert.NotEmpty(t, sess.ContainerID)
    assert.Equal(t, ContainerRunning, sess.ContainerStatus)

    // Cleanup session
    cleanupCmd := exec.Command("aps", "session", "delete", sess.ID)
    _ = cleanupCmd.Run()
}

func TestContainerResourceLimits(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Setup profile with memory limit
    profileID := setupTestContainerProfile(t, withMemoryLimit(256))
    defer cleanupTestProfile(t, profileID)

    // Execute memory-intensive command (should fail)
    cmd := exec.Command("aps", "run", profileID, "--", "stress-ng", "--vm", "1", "--vm-bytes", "512M", "--timeout", "10s")
    output, err := cmd.CombinedOutput()

    // Should fail due to OOM
    assert.Error(t, err)
    assert.Contains(t, string(output), "killed") || assert.Contains(t, string(output), "OOM")
}

func TestContainerFallbackToPlatform(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Setup profile with fallback enabled
    profileID := "test-fallback-profile"
    config := core.Profile{
        DisplayName: "Test Fallback Profile",
        Isolation: core.IsolationConfig{
            Level:    core.IsolationContainer,
            Strict:   false,
            Fallback: true,
            Container: core.ContainerConfig{
                Image: "nonexistent:latest", // Invalid image
            },
        },
    }

    err := core.CreateProfile(profileID, config)
    require.NoError(t, err)
    defer cleanupTestProfile(t, profileID)

    // Execute command (should fall back to platform isolation)
    cmd := exec.Command("aps", "run", profileID, "--", "echo", "hello")
    output, err := cmd.CombinedOutput()

    // Should succeed with fallback
    if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
        require.NoError(t, err)
        assert.Contains(t, string(output), "hello")
    }
}
```

### SSH Key Mounting E2E Test

```go
// tests/e2e/container/ssh_test.go

func TestSSHKeyMounting(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Create test SSH key
    tmpDir := t.TempDir()
    privKeyPath := filepath.Join(tmpDir, "id_ed25519")
    pubKeyPath := filepath.Join(tmpDir, "id_ed25519.pub")

    // Generate key
    keygenCmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", privKeyPath, "-N", "")
    err := keygenCmd.Run()
    require.NoError(t, err)

    // Setup profile with SSH volumes
    profileID := setupTestContainerProfile(t,
        withSSHMounts(privKeyPath, pubKeyPath))

    defer cleanupTestProfile(t, profileID)

    // Execute command that uses SSH key
    cmd := exec.Command("aps", "run", profileID, "--", "ls", "-la", "/root/.ssh")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    // Verify SSH files are mounted
    assert.Contains(t, string(output), "id_ed25519")
    assert.Contains(t, string(output), "id_ed25519.pub")
}
```

## CI/CD Strategy

### GitHub Actions Configuration

```yaml
# .github/workflows/container-tests.yml

name: Container Isolation Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test -v ./tests/unit/core/isolation/container_*.go

  integration-tests:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
        options: --privileged
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Build APS
        run: go build -o aps ./cmd/aps
      - name: Run integration tests
        run: |
          go test -v -tags=integration \
            ./tests/integration/isolation/container_*.go
        env:
          DOCKER_HOST: tcp://docker:2375
          DOCKER_TLS_CERTDIR: ""

  e2e-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        engine: [docker] # Add podman when supported
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Install Docker
        if: matrix.engine == 'docker'
        run: |
          sudo apt-get update
          sudo apt-get install -y docker.io
          sudo systemctl start docker
          sudo usermod -aG docker $USER
      - name: Build APS
        run: go build -o aps ./cmd/aps
      - name: Run E2E tests
        run: |
          go test -v ./tests/e2e/container/*.go
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Install Docker
        run: |
          sudo apt-get update
          sudo apt-get install -y docker.io
      - name: Run performance benchmarks
        run: |
          go test -bench=. -benchmem \
            ./tests/performance/container_*.go
```

### Local Testing Commands

```bash
# Run unit tests
go test -v ./tests/unit/core/isolation/container_*.go

# Run integration tests
go test -v -tags=integration ./tests/integration/isolation/container_*.go

# Run E2E tests
go test -v ./tests/e2e/container/*.go

# Run specific test
go test -v -run TestContainerLifecycle ./tests/integration/isolation/

# Run with coverage
go test -coverprofile=coverage.out ./tests/unit/core/isolation/container_*.go
go tool cover -html=coverage.out
```

## Test Data Management

### Test Profiles

```yaml
# tests/fixtures/profiles/container-basic.yml
id: test-container-basic
display_name: "Test Container Basic Profile"
isolation:
  level: "container"
  container:
    image: "alpine:latest"
    network: "none"
```

```yaml
# tests/fixtures/profiles/container-with-packages.yml
id: test-container-packages
display_name: "Test Container Packages Profile"
isolation:
  level: "container"
  container:
    image: "ubuntu:22.04"
    packages:
      - nodejs
      - python3
      - git
    build_steps:
      - type: "shell"
        run: "npm install -g @anthropic-ai/claude-code@latest"
```

```yaml
# tests/fixtures/profiles/container-with-limits.yml
id: test-container-limits
display_name: "Test Container Limits Profile"
isolation:
  level: "container"
  container:
    image: "alpine:latest"
    resources:
      cpu_quota: 100000
      cpu_period: 100000
      memory_mb: 512
      memory_swap_mb: 1024
```

## Summary

### Testing Coverage Goals
1. ✅ Unit Tests: 80%+ code coverage for container isolation code
2. ✅ Integration Tests: All ContainerEngine methods tested
3. ✅ E2E Tests: Full workflow tested (profile → container → session → cleanup)
4. ✅ CI/CD: Automated tests on every PR
5. ✅ Performance: Benchmark container startup and execution times

### Acceptance Criteria
- [ ] Unit tests pass locally and in CI
- [ ] Integration tests pass with Docker daemon
- [ ] E2E tests cover full container lifecycle
- [ ] Fallback behavior tested and documented
- [ ] SSH key mounting tested
- [ ] Tmux integration tested
- [ ] Resource limits validated in tests
- [ ] CI pipeline runs all test suites
- [ ] Performance benchmarks established
- [ ] Test documentation complete
