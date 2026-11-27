package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story 037: A2A Protocol Toggle
// Test enabling/disabling A2A protocol and auto-enable on server start

func TestA2AToggle_Enable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-a2a-enable"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test A2A")
	require.NoError(t, err)

	// Enable A2A with toggle command
	stdout, _, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "A2A enabled for profile")
	assert.Contains(t, stdout, "jsonrpc")
	assert.Contains(t, stdout, "127.0.0.1:8081")

	// Verify profile has a2a capability and config
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- a2a")
	assert.Contains(t, string(content), "protocol_binding: jsonrpc")
	assert.Contains(t, string(content), "listen_addr: 127.0.0.1:8081")
}

func TestA2AToggle_Disable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-a2a-disable"

	// Create profile and enable A2A
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test A2A")
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "a2a", "toggle", "--profile", profileID)
	require.NoError(t, err)

	// Disable A2A with toggle command
	stdout, _, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "A2A disabled for profile")

	// Verify profile no longer has a2a capability or config
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "- a2a")
	assert.NotContains(t, string(content), "protocol_binding")
}

func TestA2AToggle_CustomConfig(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-a2a-custom"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test A2A Custom")
	require.NoError(t, err)

	// Enable A2A with custom protocol and port
	stdout, _, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID, "--protocol", "grpc", "--host", "0.0.0.0", "--port", "9000")
	require.NoError(t, err)
	assert.Contains(t, stdout, "grpc")
	assert.Contains(t, stdout, "0.0.0.0:9000")

	// Verify config
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "protocol_binding: grpc")
	assert.Contains(t, string(content), "listen_addr: 0.0.0.0:9000")
}

func TestA2AToggle_ForceEnable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-a2a-force"

	// Create profile and enable A2A
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test A2A Force")
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "a2a", "toggle", "--profile", profileID)
	require.NoError(t, err)

	// Force enable with custom config (should replace existing config)
	stdout, _, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID, "--enabled=on", "--port", "9001")
	require.NoError(t, err)
	assert.Contains(t, stdout, "A2A enabled")
	assert.Contains(t, stdout, "9001")

	// Verify new config
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "listen_addr: 127.0.0.1:9001")
}

// startUntilProfileHas starts cmd, waits until every want substring is
// present in the profile file, then kills it. A fixed sleep before the
// kill races process start-up under load (parallel tests, cold caches);
// polling for the effect the test asserts on removes the race. The
// kill only happens after the last write is observed, so it cannot
// truncate a write in flight. Fails with the file content on timeout.
func startUntilProfileHas(t *testing.T, cmd *exec.Cmd, profilePath string, want ...string) {
	t.Helper()

	require.NoError(t, cmd.Start())
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	var content []byte
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		content, _ = os.ReadFile(profilePath)
		all := true
		for _, w := range want {
			if !strings.Contains(string(content), w) {
				all = false
				break
			}
		}
		if all {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("profile %s never contained %q within 30s; last content:\n%s",
		profilePath, want, content)
}

func TestA2AServer_AutoEnable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-a2a-server-auto"

	// Create profile (without enabling A2A)
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test A2A Server Auto")
	require.NoError(t, err)

	// Start server with timeout (it auto-enables and starts listening)
	// We'll use a short timeout since the goal is to verify auto-enable message
	cmd := prepareAPS(t, home, nil, "a2a", "server", "--profile", profileID)

	// Run until the server has persisted the auto-enabled config, then stop it
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	startUntilProfileHas(t, cmd, profilePath, "- a2a", "protocol_binding: jsonrpc")

	// Verify profile now has A2A enabled
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- a2a")
	assert.Contains(t, string(content), "protocol_binding: jsonrpc")
}

// Story 038: ACP Protocol Toggle
// Test enabling/disabling ACP protocol and auto-enable on server start

func TestACPToggle_Enable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-acp-enable"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test ACP")
	require.NoError(t, err)

	// Enable ACP with toggle command
	stdout, _, err := runAPS(t, home, "acp", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "ACP enabled for profile")
	assert.Contains(t, stdout, "stdio")

	// Verify profile has agent-protocol capability and config
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- agent-protocol")
	assert.Contains(t, string(content), "transport: stdio")
	assert.Contains(t, string(content), "enabled: true")
}

func TestACPToggle_Disable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-acp-disable"

	// Create profile and enable ACP
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test ACP")
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "acp", "toggle", "--profile", profileID)
	require.NoError(t, err)

	// Disable ACP with toggle command
	stdout, _, err := runAPS(t, home, "acp", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "ACP disabled for profile")

	// Verify profile no longer has agent-protocol capability
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "- agent-protocol")
}

func TestACPToggle_RejectsUnwiredNetworkTransport(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-acp-custom"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test ACP Custom")
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "acp", "toggle", "--profile", profileID, "--transport", "http", "--port", "9000")
	require.Error(t, err)
	assert.Contains(t, stderr, "stdio")
}

func TestACPServer_AutoEnable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-acp-server-auto"

	// Create profile (without enabling ACP)
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test ACP Server Auto")
	require.NoError(t, err)

	// Start server - it should auto-enable ACP
	cmd := prepareAPS(t, home, nil, "acp", "server", profileID)

	// Run until the server has persisted the auto-enabled config, then stop it
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	startUntilProfileHas(t, cmd, profilePath, "- agent-protocol", "enabled: true")

	// Verify profile now has ACP enabled
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- agent-protocol")
	assert.Contains(t, string(content), "enabled: true")
}

// Story 039: Webhook Protocol Toggle
// Test enabling/disabling Webhook protocol and auto-enable on server start

func TestWebhookToggle_Enable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-webhook-enable"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test Webhook")
	require.NoError(t, err)

	// Enable Webhook with toggle command
	stdout, _, err := runAPS(t, home, "webhook", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Webhook enabled for profile")

	// Verify profile has webhooks capability
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- webhooks")
}

func TestWebhookToggle_Disable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-webhook-disable"

	// Create profile and enable Webhook
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test Webhook")
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "webhook", "toggle", "--profile", profileID)
	require.NoError(t, err)

	// Disable Webhook with toggle command
	stdout, _, err := runAPS(t, home, "webhook", "toggle", "--profile", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Webhook disabled for profile")

	// Verify profile no longer has webhooks capability
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "- webhooks")
}

func TestWebhookToggle_ForceDisable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-webhook-force-off"

	// Create profile and enable Webhook
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test Webhook Force")
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "webhook", "toggle", "--profile", profileID, "--enabled=on")
	require.NoError(t, err)

	// Force disable
	stdout, _, err := runAPS(t, home, "webhook", "toggle", "--profile", profileID, "--enabled=off")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Webhook disabled")

	// Verify disabled
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "- webhooks")
}

func TestWebhookServer_AutoEnable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-webhook-server-auto"

	// Create profile (without enabling webhooks)
	_, _, err := runAPS(t, home, "profile", "create", profileID, "--display-name", "Test Webhook Server Auto")
	require.NoError(t, err)

	// Start server with --profile - it should auto-enable webhooks
	cmd := prepareAPS(t, home, nil, "webhook", "server", "--profile", profileID, "--addr", "127.0.0.1:0")

	// Run until the server has persisted the auto-enabled config, then stop it
	profilePath := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "profile.yaml")
	startUntilProfileHas(t, cmd, profilePath, "- webhooks")

	// Verify profile now has webhooks enabled
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- webhooks")
}

// Test invalid flag values are properly rejected

func TestToggle_InvalidEnabledValue(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-invalid"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Try A2A toggle with invalid value
	_, stderr, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID, "--enabled=invalid")
	require.Error(t, err)
	assert.Contains(t, stderr, "invalid value for --enabled")
}

func TestToggle_InvalidProtocol(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-invalid-proto"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Try A2A toggle with invalid protocol
	_, stderr, err := runAPS(t, home, "a2a", "toggle", "--profile", profileID, "--protocol", "invalid")
	require.Error(t, err)
	assert.Contains(t, stderr, "invalid protocol")
}

func TestToggle_InvalidTransport(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "test-invalid-transport"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Try ACP toggle with invalid transport
	_, stderr, err := runAPS(t, home, "acp", "toggle", "--profile", profileID, "--transport", "invalid")
	require.Error(t, err)
	assert.Contains(t, stderr, "invalid transport")
}
