package testing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/session"
)

// CreateTestProfile creates a test profile with the given name and isolation level
// Example:
//   profile := CreateTestProfile(t, "test-agent", core.IsolationProcess)
//   assert.Equal(t, "test-agent", profile.ID)
func CreateTestProfile(t *testing.T, name string, isolationLevel core.IsolationLevel) *core.Profile {
	t.Helper()

	if isolationLevel == "" {
		isolationLevel = core.IsolationProcess
	}

	profile := &core.Profile{
		ID:          name,
		DisplayName: "Test " + name,
		Persona: core.Persona{
			Tone:  "professional",
			Style: "concise",
			Risk:  "low",
		},
		Capabilities: []string{"a2a", "execute", "query", "analyze"},
		Accounts: map[string]core.Account{
			"default": {
				Username: "test-user",
			},
		},
		Preferences: core.Preferences{
			Language: "en",
			Timezone: "UTC",
			Shell:    "/bin/sh",
		},
		Limits: core.Limits{
			MaxConcurrency:    10,
			MaxRuntimeMinutes: 60,
		},
		Git: core.GitConfig{
			Enabled: true,
		},
		SSH: core.SSHConfig{
			Enabled: true,
			KeyPath: "/home/user/.ssh/id_rsa",
		},
		Webhooks: core.WebhookConfig{
			AllowedEvents: []string{"task.completed", "task.failed"},
		},
		Isolation: core.IsolationConfig{
			Level:    isolationLevel,
			Strict:   false,
			Fallback: true,
			Platform: core.PlatformConfig{
				SandboxID: "sandbox-" + uuid.New().String()[:8],
				Name:      "default-platform",
			},
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Network: "bridge",
				Volumes: []string{"/tmp"},
				Resources: core.ContainerResources{
					MemoryMB: 512,
					CPUQuota: 1000,
				},
			},
		},
		A2A: &core.A2AConfig{
			ProtocolBinding: "grpc",
			ListenAddr:      "127.0.0.1:8081",
			PublicEndpoint:  "http://127.0.0.1:8081",
			SecurityScheme:  "none",
			IsolationTier:   string(isolationLevel),
		},
		ACP: &core.ACPConfig{
			Enabled:    true,
			Transport:  "stdio",
			ListenAddr: "127.0.0.1:3000",
			Port:       3000,
		},
	}

	return profile
}

// MockAPSCore creates a mock core interface with all methods implemented
// Example:
//   mockCore := MockAPSCore(t)
//   mockCore.GetAgent("test-agent")
func MockAPSCore(t *testing.T) *MockCore {
	t.Helper()

	return &MockCore{
		profiles:    make(map[string]*core.Profile),
		sessions:    make(map[string]*session.SessionInfo),
		store:       make(map[string]map[string][]byte),
		runs:        make(map[string]*RunState),
		testLogger:  t,
	}
}

// AssertEventually asserts that a condition is true within a timeout period
// Polls the condition every 10ms
// Example:
//   AssertEventually(t, func() bool {
//     return service.IsReady()
//   }, 5*time.Second)
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("condition not met within %v", timeout)
			}
		}
	}
}

// WithTestContext creates a context with a deadline for testing
// Example:
//   ctx := WithTestContext(t, 30*time.Second)
//   // Use ctx for operations with timeout
func WithTestContext(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(func() {
		cancel()
	})

	return ctx
}

// CreateTestSession creates a test session with the given manager and session ID
// Example:
//   session := CreateTestSession(t, registry, "session-123", session.TierBasic)
func CreateTestSession(t *testing.T, manager *session.SessionRegistry, sessionID string, tier session.SessionTier) *session.SessionInfo {
	t.Helper()

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	if tier == "" {
		tier = session.TierBasic
	}

	sessionInfo := &session.SessionInfo{
		ID:          sessionID,
		ProfileID:   "test-profile",
		Command:     "bash",
		PID:         1234,
		Status:      session.SessionActive,
		Tier:        tier,
		TmuxSocket:  "/tmp/tmux-1000/default",
		TmuxSession: "test-session",
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
		Environment: map[string]string{
			"PATH":   "/usr/local/bin:/usr/bin",
			"SHELL":  "/bin/bash",
			"TERM":   "xterm-256color",
		},
	}

	err := manager.Register(sessionInfo)
	require.NoError(t, err, "failed to register test session")

	t.Cleanup(func() {
		_ = manager.Unregister(sessionID)
	})

	return sessionInfo
}

// CreateTestTerminal creates a test terminal session with command and arguments
// Example:
//   terminal := CreateTestTerminal(t, registry, "bash", []string{"-l"})
func CreateTestTerminal(t *testing.T, manager *session.SessionRegistry, command string, args []string) *session.SessionInfo {
	t.Helper()

	if command == "" {
		command = "bash"
	}

	sessionID := uuid.New().String()

	sessionInfo := &session.SessionInfo{
		ID:        sessionID,
		ProfileID: "test-profile",
		Command:   command,
		PID:       1235,
		Status:    session.SessionActive,
		Tier:      session.TierStandard,
		CreatedAt: time.Now(),
		LastSeenAt: time.Now(),
		Environment: map[string]string{
			"PATH":   "/usr/local/bin:/usr/bin",
			"SHELL":  "/bin/" + command,
			"TERM":   "xterm-256color",
		},
	}

	err := manager.Register(sessionInfo)
	require.NoError(t, err, "failed to register test terminal")

	t.Cleanup(func() {
		_ = manager.Unregister(sessionID)
	})

	return sessionInfo
}

// AssertNilError asserts that an error is nil
// Example:
//   AssertNilError(t, err)
func AssertNilError(t *testing.T, err error) {
	t.Helper()

	assert.NoError(t, err)
}

// RequireNilError asserts that an error is nil and fails the test if it's not
// Example:
//   RequireNilError(t, err)
func RequireNilError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}

// TestContextWithCancel creates a context that can be cancelled for testing
// Example:
//   ctx, cancel := TestContextWithCancel(t, 30*time.Second)
//   defer cancel()
func TestContextWithCancel(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()

	return context.WithTimeout(context.Background(), timeout)
}

// WaitFor blocks until the condition returns true or timeout is reached
// Example:
//   err := WaitFor(t, func() bool {
//     return len(messages) > 0
//   }, 5*time.Second)
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration) error {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return nil
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return ErrTimeoutWaitingForCondition
			}
		}
	}
}
