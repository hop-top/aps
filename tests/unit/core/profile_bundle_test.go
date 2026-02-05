package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"oss-aps-cli/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProfileBundle_ExportImport(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	profileID := "alpha"
	config := core.Profile{
		DisplayName: "Alpha Bot",
		Preferences: core.Preferences{Shell: "/bin/zsh"},
		Git:         core.GitConfig{Enabled: true},
	}

	require.NoError(t, core.CreateProfile(profileID, config))

	profileDir, err := core.GetProfileDir(profileID)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	require.NoError(t, os.WriteFile(secretsPath, []byte("SECRET=shh\n"), 0600))

	actionsDir := filepath.Join(profileDir, "actions")
	scriptPath := filepath.Join(actionsDir, "hello.sh")
	script := "#!/bin/sh\necho hello\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	manifest := core.ActionManifest{Actions: []core.Action{
		{
			ID:           "hello",
			Title:        "Say Hello",
			Entrypoint:   "hello.sh",
			AcceptsStdin: true,
		},
	}}
	manifestBytes, err := yaml.Marshal(&manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "actions.yaml"), manifestBytes, 0644))

	bundlePath := filepath.Join(tempDir, "alpha.aps-profile.yaml")
	bundle, err := core.ExportProfileBundle(profileID, bundlePath)
	require.NoError(t, err)
	require.Equal(t, profileID, bundle.Profile.ID)
	require.Len(t, bundle.Actions, 1)
	assert.Equal(t, "hello", bundle.Actions[0].ID)
	assert.Equal(t, script, bundle.Actions[0].Content)

	bundleBytes, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	assert.NotContains(t, string(bundleBytes), "SECRET=shh")

	importedID := "beta"
	importedProfile, importedBundle, err := core.ImportProfileBundle(bundlePath, importedID, false)
	require.NoError(t, err)
	require.Equal(t, importedID, importedProfile.ID)
	require.Equal(t, profileID, importedBundle.SourceID)

	importedDir, err := core.GetProfileDir(importedID)
	require.NoError(t, err)

	importedScript, err := os.ReadFile(filepath.Join(importedDir, "actions", "hello.sh"))
	require.NoError(t, err)
	assert.Equal(t, script, string(importedScript))

	importedManifestBytes, err := os.ReadFile(filepath.Join(importedDir, "actions.yaml"))
	require.NoError(t, err)
	var importedManifest core.ActionManifest
	require.NoError(t, yaml.Unmarshal(importedManifestBytes, &importedManifest))
	require.Len(t, importedManifest.Actions, 1)
	assert.Equal(t, "hello", importedManifest.Actions[0].ID)
	assert.True(t, importedManifest.Actions[0].AcceptsStdin)
}

func TestTrackEvent_WritesJSONL(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	err := core.TrackEvent("profile_share_created", map[string]string{
		"profile_id": "alpha",
	})
	require.NoError(t, err)

	eventsPath := filepath.Join(tempDir, ".aps", "metrics", "events.jsonl")
	data, err := os.ReadFile(eventsPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"profile_share_created\"")
	assert.Contains(t, string(data), "\"profile_id\":\"alpha\"")
}
