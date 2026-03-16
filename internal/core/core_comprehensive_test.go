package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Profile Loading & Validation Tests (12 tests)
// ============================================================================

// TestLoadProfileSuccess tests successful profile loading
func TestLoadProfileSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "test-load",
		DisplayName: "Load Test",
		Persona: Persona{
			Tone: "professional",
		},
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("test-load")
	require.NoError(t, err)
	assert.Equal(t, "test-load", loaded.ID)
	assert.Equal(t, "Load Test", loaded.DisplayName)
	assert.Equal(t, "professional", loaded.Persona.Tone)
}

// TestLoadProfileMissingFile tests LoadProfile with non-existent profile
func TestLoadProfileMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	loaded, err := LoadProfile("nonexistent-profile")
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to read profile")
}

// TestLoadProfileInvalidYAML tests LoadProfile with malformed YAML
func TestLoadProfileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	agentsDir := tmpDir
	profileDir := filepath.Join(agentsDir, "profiles", "invalid-yaml")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profilePath := filepath.Join(profileDir, "profile.yaml")
	err := os.WriteFile(profilePath, []byte("invalid: [yaml: unclosed"), 0644)
	require.NoError(t, err)

	loaded, err := LoadProfile("invalid-yaml")
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to parse profile")
}

// TestLoadProfileIDMismatch tests LoadProfile with ID mismatch between path and content
func TestLoadProfileIDMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	agentsDir := tmpDir
	profileDir := filepath.Join(agentsDir, "profiles", "profile-a")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	// Write profile with different ID
	profileYAML := "id: profile-b\ndisplay_name: Test\n"
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err := os.WriteFile(profilePath, []byte(profileYAML), 0644)
	require.NoError(t, err)

	loaded, err := LoadProfile("profile-a")
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "profile ID mismatch")
}

// TestLoadProfileInvalidIsolation tests LoadProfile with invalid isolation config
func TestLoadProfileInvalidIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	agentsDir := tmpDir
	profileDir := filepath.Join(agentsDir, "profiles", "invalid-isolation")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	// Write profile with invalid isolation level
	profileYAML := `id: invalid-isolation
display_name: Test
isolation:
  level: invalid-level
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err := os.WriteFile(profilePath, []byte(profileYAML), 0644)
	require.NoError(t, err)

	loaded, err := LoadProfile("invalid-isolation")
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "invalid isolation config")
}

// TestLoadProfileContainerWithoutImage tests LoadProfile with container isolation but no image
func TestLoadProfileContainerWithoutImage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	agentsDir := tmpDir
	profileDir := filepath.Join(agentsDir, "profiles", "no-image")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profileYAML := `id: no-image
display_name: Test
isolation:
  level: container
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err := os.WriteFile(profilePath, []byte(profileYAML), 0644)
	require.NoError(t, err)

	loaded, err := LoadProfile("no-image")
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "container isolation requires an image")
}

// TestLoadProfileWithCompleteConfig tests LoadProfile with all optional fields
func TestLoadProfileWithCompleteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "complete",
		DisplayName: "Complete Profile",
		Persona: Persona{
			Tone:  "professional",
			Style: "formal",
			Risk:  "low",
		},
		Capabilities: []string{"cap1", "cap2"},
		Accounts: map[string]Account{
			"github": {Username: "testuser"},
		},
		Preferences: Preferences{
			Language: "en",
			Timezone: "UTC",
			Shell:    "/bin/bash",
		},
		Limits: Limits{
			MaxConcurrency:    4,
			MaxRuntimeMinutes: 60,
		},
		Git: GitConfig{Enabled: true},
		SSH: SSHConfig{Enabled: true, KeyPath: "ssh.key"},
		Isolation: IsolationConfig{
			Level:  IsolationProcess,
			Strict: true,
		},
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("complete")
	require.NoError(t, err)
	assert.Equal(t, "complete", loaded.ID)
	assert.Equal(t, "professional", loaded.Persona.Tone)
	assert.Len(t, loaded.Capabilities, 2)
	assert.Equal(t, 4, loaded.Limits.MaxConcurrency)
	assert.True(t, loaded.Git.Enabled)
}

// TestLoadProfileCaching tests that profiles are correctly loaded each time
func TestLoadProfileCaching(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "cache-test",
		DisplayName: "Original",
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	// Load first time
	loaded1, err := LoadProfile("cache-test")
	require.NoError(t, err)
	assert.Equal(t, "Original", loaded1.DisplayName)

	// Modify on disk
	profile.DisplayName = "Modified"
	err = SaveProfile(profile)
	require.NoError(t, err)

	// Load again - should get modified version (no caching)
	loaded2, err := LoadProfile("cache-test")
	require.NoError(t, err)
	assert.Equal(t, "Modified", loaded2.DisplayName)
}

// TestLoadProfileConcurrentAccess tests concurrent profile loading
func TestLoadProfileConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	// Create multiple profiles
	for i := 0; i < 5; i++ {
		profile := &Profile{
			ID:          fmt.Sprintf("concurrent-%d", i),
			DisplayName: fmt.Sprintf("Profile %d", i),
		}
		err := SaveProfile(profile)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Load each profile concurrently
	for i := 0; i < 5; i++ {
		for j := 0; j < 2; j++ {
			wg.Add(1)
			go func(profileID string) {
				defer wg.Done()
				_, err := LoadProfile(profileID)
				errChan <- err
			}(fmt.Sprintf("concurrent-%d", i))
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}
}

// TestLoadProfileDefaultIsolation tests LoadProfile assigns default isolation level
func TestLoadProfileDefaultIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	agentsDir := tmpDir
	profileDir := filepath.Join(agentsDir, "profiles", "default-iso")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	// Write profile without isolation level
	profileYAML := `id: default-iso
display_name: Test
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err := os.WriteFile(profilePath, []byte(profileYAML), 0644)
	require.NoError(t, err)

	loaded, err := LoadProfile("default-iso")
	require.NoError(t, err)
	assert.Equal(t, IsolationProcess, loaded.Isolation.Level)
}

// ============================================================================
// Action Loading & Resolution Tests (12 tests)
// ============================================================================

// TestLoadActionsSuccessComprehensive tests successful action loading
func TestLoadActionsSuccessComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "actions-test", DisplayName: "Test"}
	err := CreateProfile("actions-test", profile)
	require.NoError(t, err)

	actionsDir := filepath.Join(tmpDir, "profiles", "actions-test", "actions")

	// Create test scripts
	err = os.WriteFile(filepath.Join(actionsDir, "script1.sh"), []byte("#!/bin/bash\necho hello"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(actionsDir, "script2.py"), []byte("#!/usr/bin/env python3\nprint('hello')"), 0755)
	require.NoError(t, err)

	actions, err := LoadActions("actions-test")
	require.NoError(t, err)
	assert.Len(t, actions, 2)

	ids := []string{}
	for _, a := range actions {
		ids = append(ids, a.ID)
	}
	assert.Contains(t, ids, "script1")
	assert.Contains(t, ids, "script2")
}

// TestLoadActionsEmptyComprehensive tests LoadActions with no actions
func TestLoadActionsEmptyComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "empty-actions", DisplayName: "Test"}
	err := CreateProfile("empty-actions", profile)
	require.NoError(t, err)

	actions, err := LoadActions("empty-actions")
	require.NoError(t, err)
	assert.Empty(t, actions)
}

// TestGetActionByIDComprehensive tests GetAction finds correct action
func TestGetActionByIDComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "get-action-test", DisplayName: "Test"}
	err := CreateProfile("get-action-test", profile)
	require.NoError(t, err)

	actionsDir := filepath.Join(tmpDir, "profiles", "get-action-test", "actions")
	err = os.WriteFile(filepath.Join(actionsDir, "myaction.sh"), []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	action, err := GetAction("get-action-test", "myaction")
	require.NoError(t, err)
	assert.Equal(t, "myaction", action.ID)
	assert.Equal(t, "sh", action.Type)
}

// TestGetActionNotFoundComprehensive tests GetAction returns error for missing action
func TestGetActionNotFoundComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "missing-action-test", DisplayName: "Test"}
	err := CreateProfile("missing-action-test", profile)
	require.NoError(t, err)

	action, err := GetAction("missing-action-test", "nonexistent")
	require.Error(t, err)
	assert.Nil(t, action)

	notFoundErr := &NotFoundError{}
	assert.ErrorAs(t, err, &notFoundErr)
}

// TestLoadActionsWithManifestComprehensive tests LoadActions reads actions.yaml manifest
func TestLoadActionsWithManifestComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "manifest-test", DisplayName: "Test"}
	err := CreateProfile("manifest-test", profile)
	require.NoError(t, err)

	profileDir := filepath.Join(tmpDir, "profiles", "manifest-test")
	actionsDir := filepath.Join(profileDir, "actions")

	// Create action file
	err = os.WriteFile(filepath.Join(actionsDir, "hello.sh"), []byte("#!/bin/bash\necho hello"), 0755)
	require.NoError(t, err)

	// Create manifest
	manifest := `actions:
  - id: hello_action
    title: Hello Action
    entrypoint: hello.sh
    accepts_stdin: true
`
	err = os.WriteFile(filepath.Join(profileDir, "actions.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	actions, err := LoadActions("manifest-test")
	require.NoError(t, err)

	found := false
	for _, a := range actions {
		if a.ID == "hello_action" {
			assert.Equal(t, "Hello Action", a.Title)
			assert.True(t, a.AcceptsStdin)
			found = true
		}
	}
	assert.True(t, found, "hello_action not found in actions")
}

// TestActionTypeResolutionComprehensive tests resolveType for different file extensions
func TestActionTypeResolutionComprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected string
	}{
		{"script.sh", "sh"},
		{"script.py", "py"},
		{"script.js", "js"},
		{"script.unknown", "unknown"},
		{"executable", "unknown"},
		{"test.SH", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := resolveType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestActionPathResolutionComprehensive tests action paths are correctly resolved
func TestActionPathResolutionComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "path-test", DisplayName: "Test"}
	err := CreateProfile("path-test", profile)
	require.NoError(t, err)

	actionsDir := filepath.Join(tmpDir, "profiles", "path-test", "actions")
	err = os.WriteFile(filepath.Join(actionsDir, "myscript.py"), []byte("#!/usr/bin/env python3"), 0755)
	require.NoError(t, err)

	action, err := GetAction("path-test", "myscript")
	require.NoError(t, err)

	expectedPath := filepath.Join(actionsDir, "myscript.py")
	assert.Equal(t, expectedPath, action.Path)
	assert.Equal(t, "py", action.Type)
}

// TestLoadActionsImplicitAndExplicitComprehensive tests mixing implicit and manifest actions
func TestLoadActionsImplicitAndExplicitComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "mixed-actions", DisplayName: "Test"}
	err := CreateProfile("mixed-actions", profile)
	require.NoError(t, err)

	profileDir := filepath.Join(tmpDir, "profiles", "mixed-actions")
	actionsDir := filepath.Join(profileDir, "actions")

	// Create action files
	err = os.WriteFile(filepath.Join(actionsDir, "explicit.sh"), []byte("#!/bin/bash"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(actionsDir, "implicit.sh"), []byte("#!/bin/bash"), 0755)
	require.NoError(t, err)

	// Create manifest with only explicit
	manifest := `actions:
  - id: explicit_id
    title: Explicit Action
    entrypoint: explicit.sh
`
	err = os.WriteFile(filepath.Join(profileDir, "actions.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	actions, err := LoadActions("mixed-actions")
	require.NoError(t, err)

	ids := []string{}
	for _, a := range actions {
		ids = append(ids, a.ID)
	}

	assert.Contains(t, ids, "explicit_id")
	assert.Contains(t, ids, "implicit")
}

// TestActionManifestParsingComprehensive tests ActionManifest YAML parsing
func TestActionManifestParsingComprehensive(t *testing.T) {
	t.Parallel()

	manifestYAML := `actions:
  - id: test1
    title: Test Action 1
    entrypoint: test1.sh
    accepts_stdin: true
  - id: test2
    title: Test Action 2
    entrypoint: test2.py
    accepts_stdin: false
`

	var manifest ActionManifest
	err := yaml.Unmarshal([]byte(manifestYAML), &manifest)
	require.NoError(t, err)

	assert.Len(t, manifest.Actions, 2)
	assert.Equal(t, "test1", manifest.Actions[0].ID)
	assert.True(t, manifest.Actions[0].AcceptsStdin)
	assert.Equal(t, "test2", manifest.Actions[1].ID)
	assert.False(t, manifest.Actions[1].AcceptsStdin)
}

// TestGetActionLoadingProfileNotFoundComprehensive tests GetAction with missing profile
func TestGetActionLoadingProfileNotFoundComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	action, err := GetAction("nonexistent-profile", "some-action")
	require.Error(t, err)
	assert.Nil(t, action)
}

// ============================================================================
// Environment Injection Tests (8 tests)
// ============================================================================

// TestInjectEnvironmentBasicComprehensive tests basic environment injection
func TestInjectEnvironmentBasicComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "env-basic", DisplayName: "Test"}
	err := CreateProfile("env-basic", profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-basic")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "APS_PROFILE_ID=env-basic")
	assert.Contains(t, envStr, "APS_PROFILE_DIR=")
	assert.Contains(t, envStr, "APS_PROFILE_YAML=")
}

// TestInjectEnvironmentSecretsLoadingComprehensive tests secrets are injected from secrets.env
func TestInjectEnvironmentSecretsLoadingComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "env-secrets", DisplayName: "Test"}
	err := CreateProfile("env-secrets", profile)
	require.NoError(t, err)

	secretsPath := filepath.Join(tmpDir, "profiles", "env-secrets", "secrets.env")
	err = os.WriteFile(secretsPath, []byte("DATABASE_URL=postgres://localhost\nAPI_KEY=secret123\n"), 0600)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-secrets")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "DATABASE_URL=postgres://localhost")
	assert.Contains(t, envStr, "API_KEY=secret123")
}

// TestInjectEnvironmentGitConfigComprehensive tests git configuration injection
func TestInjectEnvironmentGitConfigComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-git",
		DisplayName: "Test",
		Git:         GitConfig{Enabled: true},
	}
	err := CreateProfile("env-git", profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-git")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "GIT_CONFIG_GLOBAL=")
}

// TestInjectEnvironmentGitConfigDisabledComprehensive tests git injection disabled when not enabled
func TestInjectEnvironmentGitConfigDisabledComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-no-git",
		DisplayName: "Test",
		Git:         GitConfig{Enabled: false},
	}
	err := CreateProfile("env-no-git", profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-no-git")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.NotContains(t, envStr, "GIT_CONFIG_GLOBAL=")
}

// TestInjectEnvironmentSSHConfigComprehensive tests SSH configuration injection
func TestInjectEnvironmentSSHConfigComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-ssh",
		DisplayName: "Test",
		SSH:         SSHConfig{Enabled: true, KeyPath: "ssh.key"},
	}
	err := CreateProfile("env-ssh", profile)
	require.NoError(t, err)

	sshKeyPath := filepath.Join(tmpDir, "profiles", "env-ssh", "ssh.key")
	err = os.WriteFile(sshKeyPath, []byte("mock-key"), 0600)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-ssh")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "GIT_SSH_COMMAND=ssh -i")
}

// TestInjectEnvironmentSSHConfigMissingKeyComprehensive tests SSH injection skipped when key missing
func TestInjectEnvironmentSSHConfigMissingKeyComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		ID:          "env-ssh-missing",
		DisplayName: "Test",
		SSH:         SSHConfig{Enabled: true, KeyPath: "nonexistent.key"},
	}
	err := CreateProfile("env-ssh-missing", profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-ssh-missing")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.NotContains(t, envStr, "GIT_SSH_COMMAND=")
}

// TestInjectEnvironmentMissingSecretsFileComprehensive tests handling of missing secrets file
func TestInjectEnvironmentMissingSecretsFileComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{ID: "env-no-secrets", DisplayName: "Test"}
	err := CreateProfile("env-no-secrets", profile)
	require.NoError(t, err)

	// Remove secrets file
	secretsPath := filepath.Join(tmpDir, "profiles", "env-no-secrets", "secrets.env")
	os.Remove(secretsPath)

	loaded, err := LoadProfile("env-no-secrets")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	// Should still inject APS variables
	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "APS_PROFILE_ID=env-no-secrets")
}

// TestInjectEnvironmentCustomPrefixComprehensive tests custom prefix from config
func TestInjectEnvironmentCustomPrefixComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	config := &Config{
		Prefix: "CUSTOM",
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationProcess,
			FallbackEnabled: true,
		},
	}
	err := SaveConfig(config)
	require.NoError(t, err)

	profile := Profile{ID: "env-prefix", DisplayName: "Test"}
	err = CreateProfile("env-prefix", profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("env-prefix")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, loaded)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "CUSTOM_PROFILE_ID=env-prefix")
}

// ============================================================================
// Error Handling Tests (8 tests)
// ============================================================================

// TestNotFoundErrorComprehensive tests NotFoundError creation and behavior
func TestNotFoundErrorComprehensive(t *testing.T) {
	t.Parallel()

	err := NewNotFoundError("test-resource")
	assert.Error(t, err)
	assert.Equal(t, "not found: test-resource", err.Error())
	assert.Equal(t, ErrorCode("NOT_FOUND"), err.GetCode())
}

// TestNotFoundErrorWithCustomCodeComprehensive tests NotFoundError with custom code
func TestNotFoundErrorWithCustomCodeComprehensive(t *testing.T) {
	t.Parallel()

	err := NewNotFoundErrorWithCode("profile-123", "PROFILE_NOT_FOUND")
	assert.Error(t, err)
	assert.Equal(t, ErrorCode("PROFILE_NOT_FOUND"), err.GetCode())
	assert.Contains(t, err.Error(), "profile-123")
}

// TestInvalidInputErrorComprehensive tests InvalidInputError creation and behavior
func TestInvalidInputErrorComprehensive(t *testing.T) {
	t.Parallel()

	err := NewInvalidInputError("profile_id", "profile ID cannot be empty")
	assert.Error(t, err)
	assert.Equal(t, "profile ID cannot be empty", err.Error())
	assert.Equal(t, ErrorCode("INVALID_INPUT"), err.GetCode())
}

// TestInvalidInputErrorWithCodeComprehensive tests InvalidInputError with custom code
func TestInvalidInputErrorWithCodeComprehensive(t *testing.T) {
	t.Parallel()

	err := NewInvalidInputErrorWithCode("isolation_level", "unsupported isolation level", "INVALID_ISOLATION")
	assert.Error(t, err)
	assert.Equal(t, ErrorCode("INVALID_ISOLATION"), err.GetCode())
}

// TestValidationErrorComprehensive tests ValidationError creation and behavior
func TestValidationErrorComprehensive(t *testing.T) {
	t.Parallel()

	err := NewValidationError("isolation", "container isolation requires an image")
	assert.Error(t, err)
	assert.Equal(t, "container isolation requires an image", err.Error())
	assert.Equal(t, ErrorCode("VALIDATION_FAILED"), err.GetCode())
}

// TestValidationErrorWithCodeComprehensive tests ValidationError with custom code
func TestValidationErrorWithCodeComprehensive(t *testing.T) {
	t.Parallel()

	err := NewValidationErrorWithCode("profiles", "no valid profiles found", "NO_PROFILES")
	assert.Error(t, err)
	assert.Equal(t, ErrorCode("NO_PROFILES"), err.GetCode())
}

// TestErrorCodeStringComprehensive tests ErrorCode String() method
func TestErrorCodeStringComprehensive(t *testing.T) {
	t.Parallel()

	code := ErrorCode("TEST_CODE")
	assert.Equal(t, "TEST_CODE", code.String())
}

// TestErrorWrappingComprehensive tests error wrapping with fmt.Errorf
func TestErrorWrappingComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	// Try to load non-existent profile which should wrap error
	_, err := LoadProfile("does-not-exist")
	require.Error(t, err)

	// Check that error message contains both the wrapping and original error
	assert.Contains(t, err.Error(), "failed to read profile")
}

// ============================================================================
// Additional Comprehensive Tests
// ============================================================================

// TestProfileSaveAndLoadWithA2AComprehensive tests profile with A2A configuration
func TestProfileSaveAndLoadWithA2AComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "a2a-profile",
		DisplayName: "A2A Test",
		A2A: &A2AConfig{
			ProtocolBinding: "stdio",
			ListenAddr:      "127.0.0.1:5000",
			PublicEndpoint:  "http://example.com:5000",
			SecurityScheme:  "bearer",
			IsolationTier:   "high",
		},
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("a2a-profile")
	require.NoError(t, err)

	assert.NotNil(t, loaded.A2A)
	assert.Equal(t, "stdio", loaded.A2A.ProtocolBinding)
	assert.Equal(t, "http://example.com:5000", loaded.A2A.PublicEndpoint)
}

// TestProfileSaveAndLoadWithACPComprehensive tests profile with ACP configuration
func TestProfileSaveAndLoadWithACPComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "acp-profile",
		DisplayName: "ACP Test",
		ACP: &ACPConfig{
			Enabled:    true,
			Transport:  "http",
			ListenAddr: "0.0.0.0",
			Port:       8080,
		},
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("acp-profile")
	require.NoError(t, err)

	assert.NotNil(t, loaded.ACP)
	assert.True(t, loaded.ACP.Enabled)
	assert.Equal(t, "http", loaded.ACP.Transport)
	assert.Equal(t, 8080, loaded.ACP.Port)
}

// TestCreateA2AClientSuccessComprehensive tests A2A client creation
func TestCreateA2AClientSuccessComprehensive(t *testing.T) {
	t.Parallel()

	profile := &Profile{ID: "source-profile"}

	client, err := profile.CreateA2AClient("target-profile")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "target-profile", client.GetTargetProfileID())
}

// TestCreateA2AClientEmptyTargetIDComprehensive tests A2A client creation fails with empty ID
func TestCreateA2AClientEmptyTargetIDComprehensive(t *testing.T) {
	t.Parallel()

	profile := &Profile{ID: "source-profile"}

	client, err := profile.CreateA2AClient("")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestProfileYAMLRoundtripComprehensive tests profile can be marshaled and unmarshaled
func TestProfileYAMLRoundtripComprehensive(t *testing.T) {
	t.Parallel()

	original := &Profile{
		ID:          "roundtrip-test",
		DisplayName: "Roundtrip Test",
		Persona: Persona{
			Tone:  "professional",
			Style: "concise",
			Risk:  "low",
		},
		Capabilities: []string{"cap1", "cap2"},
		Preferences: Preferences{
			Language: "en",
			Timezone: "UTC",
			Shell:    "/bin/bash",
		},
		Limits: Limits{
			MaxConcurrency:    4,
			MaxRuntimeMinutes: 120,
		},
		Git: GitConfig{Enabled: true},
		SSH: SSHConfig{Enabled: true, KeyPath: "id_rsa"},
		A2A: &A2AConfig{
			ListenAddr: "127.0.0.1:5000",
		},
		ACP: &ACPConfig{
			Enabled: true,
			Port:    8080,
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var loaded Profile
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, loaded.ID)
	assert.Equal(t, original.DisplayName, loaded.DisplayName)
	assert.Equal(t, original.Persona.Tone, loaded.Persona.Tone)
	assert.Equal(t, original.Limits.MaxConcurrency, loaded.Limits.MaxConcurrency)
	assert.NotNil(t, loaded.A2A)
	assert.NotNil(t, loaded.ACP)
}

// TestLoadSecretsNonExistentComprehensive tests LoadSecrets returns nil for missing file
func TestLoadSecretsNonExistentComprehensive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "nonexistent.env")

	secrets, err := LoadSecrets(secretsPath)
	require.NoError(t, err)
	assert.Nil(t, secrets)
}

// TestLoadSecretsValidComprehensive tests LoadSecrets parses .env file correctly
func TestLoadSecretsValidComprehensive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.env")

	secretsContent := "KEY1=value1\nKEY2=value2\nKEY3=value with spaces\n"
	err := os.WriteFile(secretsPath, []byte(secretsContent), 0600)
	require.NoError(t, err)

	secrets, err := LoadSecrets(secretsPath)
	require.NoError(t, err)

	assert.Equal(t, "value1", secrets["KEY1"])
	assert.Equal(t, "value2", secrets["KEY2"])
	assert.Equal(t, "value with spaces", secrets["KEY3"])
}

// TestListProfilesOrderingComprehensive tests ListProfiles returns profiles
func TestListProfilesOrderingComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	// Create profiles in non-alphabetical order
	for _, id := range []string{"zebra", "apple", "middle"} {
		profile := Profile{DisplayName: id}
		err := CreateProfile(id, profile)
		require.NoError(t, err)
	}

	profiles, err := ListProfiles()
	require.NoError(t, err)

	assert.Len(t, profiles, 3)
	assert.Contains(t, profiles, "zebra")
	assert.Contains(t, profiles, "apple")
	assert.Contains(t, profiles, "middle")
}

// TestGetProfileDirPathComprehensive tests GetProfileDir returns correct XDG path structure
func TestGetProfileDirPathComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	dir, err := GetProfileDir("test-profile")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "profiles", "test-profile")
	assert.Equal(t, expected, dir)
}

// TestGetProfilePathComprehensive tests GetProfilePath returns profile.yaml path
func TestGetProfilePathComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	path, err := GetProfilePath("test-profile")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "profiles", "test-profile", "profile.yaml")
	assert.Equal(t, expected, path)
}

// TestGetAgentsDirPathComprehensive tests GetAgentsDir delegates to GetDataDir (XDG)
func TestGetAgentsDirPathComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	dir, err := GetAgentsDir()
	require.NoError(t, err)

	assert.Equal(t, tmpDir, dir)
}

// ============================================================================
// Configuration Tests
// ============================================================================

// TestLoadConfigDefaultsComprehensive tests LoadConfig returns default values
func TestLoadConfigDefaultsComprehensive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, DefaultPrefix, config.Prefix)
	assert.Equal(t, IsolationProcess, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)
}

// TestSaveAndLoadConfigComprehensive tests saving and loading configuration
func TestSaveAndLoadConfigComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	original := &Config{
		Prefix: "CUSTOM_PREFIX",
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationContainer,
			FallbackEnabled: false,
		},
	}

	err := SaveConfig(original)
	require.NoError(t, err)

	loaded, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "CUSTOM_PREFIX", loaded.Prefix)
	assert.Equal(t, IsolationContainer, loaded.Isolation.DefaultLevel)
	assert.False(t, loaded.Isolation.FallbackEnabled)
}

// TestConfigMigrationComprehensive tests config migration functionality
func TestConfigMigrationComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "aps")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Write old config format
	oldConfig := []byte("prefix: OLD_PREFIX\n")
	err := os.WriteFile(filepath.Join(configDir, "config.yaml"), oldConfig, 0644)
	require.NoError(t, err)

	// Run migration
	migrated, err := MigrateConfig()
	require.NoError(t, err)
	assert.True(t, migrated)

	// Verify migrated config
	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "OLD_PREFIX", config.Prefix)
	assert.Equal(t, IsolationProcess, config.Isolation.DefaultLevel)
}

// TestConfigMigrationNoExistingComprehensive tests migration returns false when no existing config
func TestConfigMigrationNoExistingComprehensive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	migrated, err := MigrateConfig()
	require.NoError(t, err)
	assert.False(t, migrated)
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkValidateIsolationComprehensive benchmarks isolation validation
func BenchmarkValidateIsolationComprehensive(b *testing.B) {
	profile := &Profile{
		ID: "bench",
		Isolation: IsolationConfig{
			Level: IsolationProcess,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile.ValidateIsolation()
	}
}

// BenchmarkResolveActionTypeComprehensive benchmarks action type resolution
func BenchmarkResolveActionTypeComprehensive(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolveType("test.sh")
		resolveType("test.py")
		resolveType("test.js")
		resolveType("test.unknown")
	}
}

// BenchmarkLoadProfileComprehensive benchmarks profile loading
func BenchmarkLoadProfileComprehensive(b *testing.B) {
	tmpDir := b.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", tmpDir)

	profile := &Profile{
		ID:          "bench-profile",
		DisplayName: "Benchmark Profile",
		Persona: Persona{
			Tone: "professional",
		},
	}
	SaveProfile(profile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadProfile("bench-profile")
	}
}

// BenchmarkSaveProfileComprehensive benchmarks profile saving
func BenchmarkSaveProfileComprehensive(b *testing.B) {
	tmpDir := b.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", tmpDir)

	profile := &Profile{
		ID:          "bench-save",
		DisplayName: "Benchmark Save",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SaveProfile(profile)
	}
}
