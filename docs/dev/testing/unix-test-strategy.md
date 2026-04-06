# Unix Test Strategy

## Overview

This document defines the testing strategy for Unix platforms (macOS and Linux), including E2E tests, CI runner configuration, and Unix-specific test considerations.

## Test Architecture

### Test Hierarchy

```
┌─────────────────────────────────────┐
│      Integration Tests             │
│  (process_isolation_test.go)     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│      Platform Tests               │
│  (macos_test.go, linux_test.go) │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         E2E Tests                │
│  (e2e_test.go, e2e_macos.go,   │
│   e2e_linux.go)                 │
└─────────────────────────────────────┘
```

## macOS E2E Tests

### Test Structure

```go
// tests/e2e/e2e_macos_test.go

package e2e

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "oss-aps-cli/internal/core"
    "oss-aps-cli/internal/core/isolation"
    "oss-aps-cli/internal/core/session"
)

type MacOSE2ETest struct {
    profileID      string
    adapter        *isolation.ProcessIsolation
    tempDir        string
}

func setupMacOSE2E(t *testing.T) *MacOSE2ETest {
    tempDir := t.TempDir()
    profileID := "test-macos-e2e"

    // Create profile
    config := core.Profile{
        ID:          profileID,
        DisplayName: "Test macOS E2E",
    }

    // Override profile directory to temp directory
    os.Setenv("HOME", tempDir)

    return &MacOSE2ETest{
        profileID: profileID,
        adapter:   isolation.NewProcessIsolation(),
        tempDir:   tempDir,
    }
}

func (t *MacOSE2ETest) teardown() {
    // Cleanup profile
    profileDir, _ := core.GetProfileDir(t.profileID)
    os.RemoveAll(profileDir)
}
```

### Test Cases

#### 1. Profile Creation and Deletion

```go
func TestMacOSE2E_ProfileCreation(t *testing.T) {
    test := setupMacOSE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test macOS Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Verify profile exists
    profile, err := core.LoadProfile(test.profileID)
    if err != nil {
        t.Fatalf("Failed to load profile: %v", err)
    }

    if profile.ID != test.profileID {
        t.Errorf("Profile ID mismatch: expected %s, got %s", test.profileID, profile.ID)
    }

    // Verify profile directory structure
    profileDir, _ := core.GetProfileDir(test.profileID)
    requiredFiles := []string{
        "profile.yaml",
        "secrets.env",
        "notes.md",
    }

    for _, file := range requiredFiles {
        path := filepath.Join(profileDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("Required file not created: %s", file)
        }
    }
}
```

#### 2. Command Execution with tmux

```go
func TestMacOSE2E_TmuxSessionExecution(t *testing.T) {
    test := setupMacOSE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test macOS Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Prepare context
    ctx, err := test.adapter.PrepareContext(test.profileID)
    if err != nil {
        t.Fatalf("Failed to prepare context: %v", err)
    }

    // Execute simple command in tmux
    err = test.adapter.Execute("echo", []string{"Hello, macOS!"})
    if err != nil {
        t.Fatalf("Failed to execute command: %v", err)
    }

    // Wait for tmux session to start
    time.Sleep(1 * time.Second)

    // Verify session was created
    registry := session.GetRegistry()
    sessions := registry.ListByProfile(test.profileID)
    if len(sessions) != 1 {
        t.Errorf("Expected 1 session, got %d", len(sessions))
    }

    // Verify session has tmux socket
    sess := sessions[0]
    if sess.TmuxSocket == "" {
        t.Error("Session missing tmux socket")
    }

    // Cleanup
    test.adapter.Cleanup()
}
```

#### 3. Session Attach and Detach

```go
func TestMacOSE2E_SessionAttachDetach(t *testing.T) {
    test := setupMacOSE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test macOS Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Start long-running command
    go func() {
        _ = test.adapter.Execute("sleep", []string{"30"})
    }()

    time.Sleep(1 * time.Second)

    // Get session
    registry := session.GetRegistry()
    sessions := registry.ListByProfile(test.profileID)
    if len(sessions) != 1 {
        t.Fatalf("Expected 1 session, got %d", len(sessions))
    }

    sessionID := sessions[0].ID

    // Attach to session (non-interactive for testing)
    // In real scenario, this would attach user's terminal
    t.Logf("Session ID: %s", sessionID)
    t.Logf("Tmux Socket: %s", sessions[0].TmuxSocket)

    // Detach from session
    // In real scenario, user would press Ctrl+B then D
    // For testing, we can simulate detach
}
```

#### 4. Environment Injection

```go
func TestMacOSE2E_EnvironmentInjection(t *testing.T) {
    test := setupMacOSE2E(t)
    defer test.teardown()

    // Create profile with secrets
    profileConfig := core.Profile{
        ID:          test.profileID,
        DisplayName: "Test macOS Profile",
    }

    err := core.CreateProfile(test.profileID, profileConfig)
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Add secrets to secrets.env
    profileDir, _ := core.GetProfileDir(test.profileID)
    secretsPath := filepath.Join(profileDir, "secrets.env")
    secretsContent := "TEST_SECRET=secret_value\n"
    err = os.WriteFile(secretsPath, []byte(secretsContent), 0600)
    if err != nil {
        t.Fatalf("Failed to write secrets: %v", err)
    }

    // Execute command to verify environment
    test.adapter.Execute("env", []string{})

    // Verify environment was injected
    // (In real scenario, we would capture and parse output)
}
```

#### 5. Git Integration

```go
func TestMacOSE2E_GitIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping git integration test in short mode")
    }

    test := setupMacOSE2E(t)
    defer test.teardown()

    // Create profile with git enabled
    profileConfig := core.Profile{
        ID:          test.profileID,
        DisplayName: "Test macOS Profile",
        Git: core.GitConfig{
            Enabled: true,
        },
    }

    err := core.CreateProfile(test.profileID, profileConfig)
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Execute git command
    err = test.adapter.Execute("git", []string{"config", "--global", "user.name"})
    if err != nil {
        t.Fatalf("Failed to execute git command: %v", err)
    }

    // Verify git config was injected
}
```

## Linux E2E Tests

### Test Structure

```go
// tests/e2e/e2e_linux_test.go

package e2e

import (
    "os"
    "path/filepath"
    "syscall"
    "testing"

    "oss-aps-cli/internal/core"
    "oss-aps-cli/internal/core/isolation"
    "oss-aps-cli/internal/core/session"
)

type LinuxE2ETest struct {
    profileID string
    adapter   *isolation.ProcessIsolation
    tempDir   string
    uid       int
    gid       int
}

func setupLinuxE2E(t *testing.T) *LinuxE2ETest {
    tempDir := t.TempDir()
    profileID := "test-linux-e2e"

    uid := os.Getuid()
    gid := os.Getgid()

    // Create profile
    os.Setenv("HOME", tempDir)

    return &LinuxE2ETest{
        profileID: profileID,
        adapter:   isolation.NewProcessIsolation(),
        tempDir:   tempDir,
        uid:       uid,
        gid:       gid,
    }
}

func (t *LinuxE2ETest) teardown() {
    profileDir, _ := core.GetProfileDir(t.profileID)
    os.RemoveAll(profileDir)
}
```

### Test Cases

#### 1. Namespace Detection

```go
func TestLinuxE2E_NamespaceDetection(t *testing.T) {
    if !isLinux() {
        t.Skip("Skipping Linux-specific test on non-Linux platform")
    }

    test := setupLinuxE2E(t)
    defer test.teardown()

    // Check if user namespace is available
    if _, err := os.Stat("/proc/self/ns/user"); err == nil {
        t.Log("User namespace is available")
    } else {
        t.Error("User namespace not available")
    }

    // Check if PID namespace is available
    if _, err := os.Stat("/proc/self/ns/pid"); err == nil {
        t.Log("PID namespace is available")
    } else {
        t.Error("PID namespace not available")
    }

    // Check if mount namespace is available
    if _, err := os.Stat("/proc/self/ns/mnt"); err == nil {
        t.Log("Mount namespace is available")
    } else {
        t.Error("Mount namespace not available")
    }
}

func isLinux() bool {
    return syscall.OS == "linux"
}
```

#### 2. Cgroup Detection

```go
func TestLinuxE2E_CgroupDetection(t *testing.T) {
    if !isLinux() {
        t.Skip("Skipping Linux-specific test on non-Linux platform")
    }

    // Check for cgroup v2
    if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
        t.Log("Cgroup v2 is available")
    } else {
        t.Log("Cgroup v2 not available, checking v1")
    }

    // Check for cgroup v1
    if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
        t.Log("Cgroup v1 memory controller is available")
    } else {
        t.Error("Cgroup v1 not available")
    }

    if _, err := os.Stat("/sys/fs/cgroup/cpu"); err == nil {
        t.Log("Cgroup v1 CPU controller is available")
    } else {
        t.Error("Cgroup v1 CPU controller not available")
    }
}
```

#### 3. User Information

```go
func TestLinuxE2E_UserInformation(t *testing.T) {
    if !isLinux() {
        t.Skip("Skipping Linux-specific test on non-Linux platform")
    }

    test := setupLinuxE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test Linux Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Prepare context
    ctx, err := test.adapter.PrepareContext(test.profileID)
    if err != nil {
        t.Fatalf("Failed to prepare context: %v", err)
    }

    // Verify UID/GID are set
    if ctx.UID != test.uid {
        t.Errorf("UID mismatch: expected %d, got %d", test.uid, ctx.UID)
    }

    if ctx.GID != test.gid {
        t.Errorf("GID mismatch: expected %d, got %d", test.gid, ctx.GID)
    }
}
```

#### 4. File Permissions

```go
func TestLinuxE2E_FilePermissions(t *testing.T) {
    if !isLinux() {
        t.Skip("Skipping Linux-specific test on non-Linux platform")
    }

    test := setupLinuxE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test Linux Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Check secrets.env permissions
    profileDir, _ := core.GetProfileDir(test.profileID)
    secretsPath := filepath.Join(profileDir, "secrets.env")

    info, err := os.Stat(secretsPath)
    if err != nil {
        t.Fatalf("Failed to stat secrets.env: %v", err)
    }

    perms := info.Mode().Perm()
    if perms != 0600 {
        t.Errorf("secrets.env has wrong permissions: %o (expected 0600)", perms)
    }
}
```

#### 5. Process Signals

```go
func TestLinuxE2E_ProcessSignaling(t *testing.T) {
    if !isLinux() {
        t.Skip("Skipping Linux-specific test on non-Linux platform")
    }

    test := setupLinuxE2E(t)
    defer test.teardown()

    // Create profile
    err := core.CreateProfile(test.profileID, core.Profile{
        ID:          test.profileID,
        DisplayName: "Test Linux Profile",
    })
    if err != nil {
        t.Fatalf("Failed to create profile: %v", err)
    }

    // Start long-running process
    go func() {
        _ = test.adapter.Execute("sleep", []string{"30"})
    }()

    // Wait for session to be created
    registry := session.GetRegistry()
    var sessionID string
    for i := 0; i < 10; i++ {
        sessions := registry.ListByProfile(test.profileID)
        if len(sessions) > 0 {
            sessionID = sessions[0].ID
            break
        }
        time.Sleep(100 * time.Millisecond)
    }

    if sessionID == "" {
        t.Fatal("Session was not created")
    }

    session, _ := registry.Get(sessionID)

    // Send SIGTERM to process
    process, err := os.FindProcess(session.PID)
    if err != nil {
        t.Fatalf("Failed to find process: %v", err)
    }

    err = process.Signal(syscall.SIGTERM)
    if err != nil {
        t.Errorf("Failed to send SIGTERM: %v", err)
    }

    // Wait for session to terminate
    time.Sleep(1 * time.Second)
}
```

## Unix-Specific CI Runner Configuration

### macOS Runner Configuration

#### GitHub Actions macOS Runner

```yaml
# .github/workflows/e2e-macos.yml

name: E2E Tests (macOS)

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]
  schedule:
    - cron: '0 6 * * *'  # Daily at 6 AM UTC

jobs:
  e2e-macos:
    name: E2E Tests on macOS
    runs-on: macos-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.5'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: macos-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            macos-go-

      - name: Install tmux
        run: |
          brew install tmux

      - name: Verify tmux installation
        run: |
          tmux -V
          tmux -L

      - name: Run E2E tests
        run: |
          go test -v ./tests/e2e/e2e_macos_test.go

      - name: Upload test results
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: macos-e2e-results
          path: |
            test-results/
          retention-days: 30
```

### Linux Runner Configuration

#### GitHub Actions Linux Runner

```yaml
# .github/workflows/e2e-linux.yml

name: E2E Tests (Linux)

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]
  schedule:
    - cron: '0 6 * * *'  # Daily at 6 AM UTC

jobs:
  e2e-linux:
    name: E2E Tests on Linux
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.5'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: linux-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            linux-go-

      - name: Install tmux
        run: |
          sudo apt-get update
          sudo apt-get install -y tmux

      - name: Verify tmux installation
        run: |
          tmux -V
          tmux -L

      - name: Run E2E tests
        run: |
          go test -v ./tests/e2e/e2e_linux_test.go

      - name: Upload test results
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: linux-e2e-results
          path: |
            test-results/
          retention-days: 30
```

### Self-Hosted Unix Runners

#### Self-Hosted macOS Runner Setup

```bash
#!/bin/bash
# setup-macos-runner.sh

# Install dependencies
brew install go tmux

# Create runner user
sudo dscl . -create /Users/apsrunner
sudo dscl . -create /Users/apsrunner UserShell /bin/bash
sudo dscl . -passwd /Users/apsrunner "apsrunner"
sudo dscl . -merge /Users/apsrunner UniqueID 501
sudo dscl . -merge /Users/apsrunner PrimaryGroupID 20

# Create .ssh directory
sudo mkdir -p /Users/apsrunner/.ssh
sudo chown apsrunner:staff /Users/apsrunner/.ssh
sudo chmod 700 /Users/apsrunner/.ssh

# Configure runner
mkdir -p actions-runner
cd actions-runner
curl -o actions-runner-osx-x64.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-osx-x64-2.311.0.tar.gz
tar xzf ./actions-runner-osx-x64-2.311.0.tar.gz

./config.sh \
  --url https://github.com/IdeaCraftersLabs/oss-aps-cli \
  --token YOUR_RUNNER_TOKEN \
  --labels self-hosted,macos,x64

# Install and start runner
sudo ./svc.sh install
sudo ./svc.sh start
```

#### Self-Hosted Linux Runner Setup

```bash
#!/bin/bash
# setup-linux-runner.sh

# Install dependencies
sudo apt-get update
sudo apt-get install -y tmux

# Create runner user
sudo useradd -m -s /bin/bash -G sudo apsrunner
echo "apsrunner:apsrunner" | sudo chpasswd

# Configure sudo without password
echo "apsrunner ALL=(ALL) NOPASSWD:ALL" | sudo tee -a /etc/sudoers

# Create .ssh directory
sudo -u apsrunner mkdir -p /home/apsrunner/.ssh
sudo -u apsrunner chmod 700 /home/apsrunner/.ssh

# Configure runner
mkdir -p actions-runner
cd actions-runner
curl -o actions-runner-linux-x64.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.311.0.tar.gz

./config.sh \
  --url https://github.com/IdeaCraftersLabs/oss-aps-cli \
  --token YOUR_RUNNER_TOKEN \
  --labels self-hosted,linux,x64

# Install and start runner
sudo ./svc.sh install
sudo ./svc.sh start
```

## Test Data and Fixtures

### Test Profile Data

```go
// tests/e2e/fixtures.go

package e2e

import (
    "oss-aps-cli/internal/core"
)

var (
    TestProfile = core.Profile{
        ID:          "test-profile",
        DisplayName: "Test Profile",
        Persona: core.Persona{
            Tone:  "concise",
            Style: "technical",
            Risk:  "low",
        },
        Capabilities: []string{"git", "github", "webhooks"},
        Preferences: core.Preferences{
            Language: "en",
            Timezone: "America/New_York",
        },
        Git: core.GitConfig{
            Enabled: true,
        },
        SSH: core.SSHConfig{
            Enabled: true,
        },
    }

    TestSecrets = map[string]string{
        "TEST_SECRET":      "secret_value",
        "GITHUB_TOKEN":     "ghp_test_token",
        "WEBHOOK_SECRET":   "webhook_secret",
    }

    TestActions = []core.Action{
        {
            ID:        "test-action",
            Title:     "Test Action",
            Entrypoint: "test-action.sh",
            Type:      "sh",
        },
    }
)

func CreateTestProfile(profileID string) error {
    config := TestProfile
    config.ID = profileID
    return core.CreateProfile(profileID, config)
}

func CreateTestSecrets(profileID string) error {
    profileDir, err := core.GetProfileDir(profileID)
    if err != nil {
        return err
    }

    secretsPath := profileDir + "/secrets.env"
    var content string
    for k, v := range TestSecrets {
        content += fmt.Sprintf("%s=%s\n", k, v)
    }

    return os.WriteFile(secretsPath, []byte(content), 0600)
}
```

### Test Helper Functions

```go
// tests/e2e/helpers_test.go

package e2e

import (
    "os"
    "testing"
    "time"
)

func WaitForSession(t *testing.T, profileID string, timeout time.Duration) string {
    registry := session.GetRegistry()
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        sessions := registry.ListByProfile(profileID)
        if len(sessions) > 0 {
            return sessions[0].ID
        }
        time.Sleep(100 * time.Millisecond)
    }

    t.Fatalf("Session not created for profile %s", profileID)
    return ""
}

func WaitForProcess(t *testing.T, pid int, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        if processExists(pid) {
            return true
        }
        time.Sleep(100 * time.Millisecond)
    }

    return false
}

func processExists(pid int) bool {
    _, err := os.FindProcess(pid)
    return err == nil
}

func CleanupSession(t *testing.T, sessionID string) {
    registry := session.GetRegistry()
    _ = registry.Unregister(sessionID)
}
```

## Test Execution

### Running macOS E2E Tests

```bash
# Run all macOS E2E tests
go test -v ./tests/e2e/e2e_macos_test.go

# Run specific test
go test -v ./tests/e2e/e2e_macos_test.go -run TestMacOSE2E_TmuxSessionExecution

# Run tests with coverage
go test -v -coverprofile=coverage_macos.out ./tests/e2e/e2e_macos_test.go

# Run tests in short mode
go test -v -short ./tests/e2e/e2e_macos_test.go
```

### Running Linux E2E Tests

```bash
# Run all Linux E2E tests
go test -v ./tests/e2e/e2e_linux_test.go

# Run specific test
go test -v ./tests/e2e/e2e_linux_test.go -run TestLinuxE2E_NamespaceDetection

# Run tests with coverage
go test -v -coverprofile=coverage_linux.out ./tests/e2e/e2e_linux_test.go

# Run tests in short mode
go test -v -short ./tests/e2e/e2e_linux_test.go
```

## Test Coverage Goals

### Coverage Targets

| Component         | Target Coverage | Current Coverage |
|------------------|----------------|-----------------|
| macOS isolation   | 80%            | N/A             |
| Linux isolation   | 80%            | N/A             |
| Session registry  | 85%            | N/A             |
| Process isolation | 75%            | N/A             |
| Tmux integration | 70%            | N/A             |

### Coverage Reporting

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./tests/e2e/...
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Acceptance Criteria

✅ **E2E Tests for macOS**
- [ ] Profile creation and deletion tests implemented
- [ ] Tmux session execution tests implemented
- [ ] Session attach and detach tests implemented
- [ ] Environment injection tests implemented
- [ ] Git integration tests implemented
- [ ] All tests pass on macOS runner

✅ **E2E Tests for Linux**
- [ ] Namespace detection tests implemented
- [ ] Cgroup detection tests implemented
- [ ] User information tests implemented
- [ ] File permissions tests implemented
- [ ] Process signaling tests implemented
- [ ] All tests pass on Linux runner

✅ **CI Runner Configuration**
- [ ] macOS runner workflow configured
- [ ] Linux runner workflow configured
- [ ] Self-hosted macOS runner setup documented
- [ ] Self-hosted Linux runner setup documented
- [ ] Runner labels configured for routing

✅ **Test Strategy**
- [ ] Test hierarchy defined
- [ ] Test fixtures and helpers implemented
- [ ] Coverage targets defined
- [ ] Test execution guidelines documented
- [ ] Test coverage reporting configured
