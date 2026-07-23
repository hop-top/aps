package core

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInjectEnvironmentKnowledgeSubscriptions verifies the knowledge
// subscriptions reference is injected as an env var when set.
func TestInjectEnvironmentKnowledgeSubscriptions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-knowledge",
		DisplayName: "Test",
		Knowledge: &KnowledgeConfig{
			Subscriptions: "https://registry.example.com/subscriptions.yaml",
		},
	}
	require.NoError(t, CreateProfile("env-knowledge", profile))

	loaded, err := LoadProfile("env-knowledge")
	require.NoError(t, err)

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, loaded))

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "APS_KNOWLEDGE_SUBSCRIPTIONS=https://registry.example.com/subscriptions.yaml")
}

// TestInjectEnvironmentKnowledgeAbsent verifies no injection when the
// field is not set — behavior identical to profiles predating the field.
func TestInjectEnvironmentKnowledgeAbsent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "env-no-knowledge", DisplayName: "Test"}
	require.NoError(t, CreateProfile("env-no-knowledge", profile))

	loaded, err := LoadProfile("env-no-knowledge")
	require.NoError(t, err)

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, loaded))

	envStr := strings.Join(cmd.Env, "\n")
	assert.NotContains(t, envStr, "APS_KNOWLEDGE_SUBSCRIPTIONS=")
}

// TestInjectEnvironmentKnowledgeEmptySubscriptions verifies an empty
// subscriptions value injects nothing.
func TestInjectEnvironmentKnowledgeEmptySubscriptions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-empty-knowledge",
		DisplayName: "Test",
		Knowledge:   &KnowledgeConfig{},
	}
	require.NoError(t, CreateProfile("env-empty-knowledge", profile))

	loaded, err := LoadProfile("env-empty-knowledge")
	require.NoError(t, err)

	cmd := exec.Command("env")
	require.NoError(t, InjectEnvironment(cmd, loaded))

	envStr := strings.Join(cmd.Env, "\n")
	assert.NotContains(t, envStr, "APS_KNOWLEDGE_SUBSCRIPTIONS=")
}

// TestProfileKnowledgeRoundTrip verifies load/save preserves the field.
func TestProfileKnowledgeRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "knowledge-roundtrip",
		DisplayName: "Test",
		Knowledge: &KnowledgeConfig{
			Subscriptions: "./config/registries.yaml",
		},
	}
	require.NoError(t, CreateProfile("knowledge-roundtrip", profile))

	loaded, err := LoadProfile("knowledge-roundtrip")
	require.NoError(t, err)
	require.NotNil(t, loaded.Knowledge)
	assert.Equal(t, "./config/registries.yaml", loaded.Knowledge.Subscriptions)

	// Save again and reload — field must survive a second cycle.
	loaded.DisplayName = "Updated"
	require.NoError(t, SaveProfile(loaded))

	reloaded, err := LoadProfile("knowledge-roundtrip")
	require.NoError(t, err)
	require.NotNil(t, reloaded.Knowledge)
	assert.Equal(t, "./config/registries.yaml", reloaded.Knowledge.Subscriptions)
	assert.Equal(t, "Updated", reloaded.DisplayName)
}

// TestProfileKnowledgeAbsentRoundTrip verifies profiles without the field
// stay without it (no yaml key emitted, nil after load).
func TestProfileKnowledgeAbsentRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "no-knowledge-roundtrip", DisplayName: "Test"}
	require.NoError(t, CreateProfile("no-knowledge-roundtrip", profile))

	loaded, err := LoadProfile("no-knowledge-roundtrip")
	require.NoError(t, err)
	assert.Nil(t, loaded.Knowledge)
}
