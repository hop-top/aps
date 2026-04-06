package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/isolation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDataDir sets APS_DATA_PATH to a temp directory and returns the profiles dir.
func setupTestDataDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)
	return tmpDir
}

// createTestProfile creates a profile directory with the given YAML content.
func createTestProfile(t *testing.T, dataDir, profileID, profileYAML string) string {
	t.Helper()
	profileDir := filepath.Join(dataDir, "profiles", profileID)
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileYAML), 0o644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0o600)
	require.NoError(t, err)

	return profileDir
}

func TestIsolationFoundation_InterfaceCompliance(t *testing.T) {
	t.Run("ProcessIsolation implements IsolationManager", func(t *testing.T) {
		var _ isolation.IsolationManager = isolation.NewProcessIsolation()
	})

	t.Run("PlatformSandbox implements IsolationManager", func(t *testing.T) {
		var _ isolation.IsolationManager = isolation.NewPlatformSandbox()
	})

	t.Run("ProcessIsolation has all required methods", func(t *testing.T) {
		adapter := isolation.NewProcessIsolation()

		assert.Implements(t, (*isolation.IsolationManager)(nil), adapter)

		var (
			_ func(string) (*isolation.ExecutionContext, error) = adapter.PrepareContext
			_ func(interface{}) error                           = adapter.SetupEnvironment
			_ func(string, []string) error                      = adapter.Execute
			_ func(string, []byte) error                        = adapter.ExecuteAction
			_ func() error                                      = adapter.Cleanup
			_ func() error                                      = adapter.Validate
			_ func() bool                                       = adapter.IsAvailable
		)
	})
}

func TestIsolationFoundation_FallbackLogic(t *testing.T) {
	t.Run("Strict mode enforces requested level", func(t *testing.T) {
		manager := isolation.NewManager()
		manager.Register(core.IsolationProcess, isolation.NewProcessIsolation())

		profile := &core.Profile{
			ID: "strict-test",
			Isolation: core.IsolationConfig{
				Level:    core.IsolationContainer,
				Strict:   true,
				Fallback: true,
			},
		}
		globalConfig := &core.Config{
			Isolation: core.GlobalIsolationConfig{
				DefaultLevel:    core.IsolationProcess,
				FallbackEnabled: true,
			},
		}

		_, err := manager.GetIsolationManager(profile, globalConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "strict mode violation")
	})

	t.Run("Profile-level fallback disabled", func(t *testing.T) {
		manager := isolation.NewManager()
		manager.Register(core.IsolationProcess, isolation.NewProcessIsolation())

		profile := &core.Profile{
			ID: "no-fallback-test",
			Isolation: core.IsolationConfig{
				Level:    core.IsolationContainer,
				Strict:   false,
				Fallback: false,
			},
		}
		globalConfig := &core.Config{
			Isolation: core.GlobalIsolationConfig{
				DefaultLevel:    core.IsolationProcess,
				FallbackEnabled: true,
			},
		}

		_, err := manager.GetIsolationManager(profile, globalConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fallback disabled")
	})

	t.Run("Global-level fallback disabled", func(t *testing.T) {
		manager := isolation.NewManager()
		manager.Register(core.IsolationProcess, isolation.NewProcessIsolation())

		profile := &core.Profile{
			ID: "global-no-fallback-test",
			Isolation: core.IsolationConfig{
				Level:    core.IsolationContainer,
				Strict:   false,
				Fallback: true,
			},
		}
		globalConfig := &core.Config{
			Isolation: core.GlobalIsolationConfig{
				DefaultLevel:    core.IsolationProcess,
				FallbackEnabled: false,
			},
		}

		_, err := manager.GetIsolationManager(profile, globalConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "global fallback disabled")
	})

	t.Run("Graceful degradation through multiple levels", func(t *testing.T) {
		manager := isolation.NewManager()
		processAdapter := isolation.NewProcessIsolation()
		manager.Register(core.IsolationProcess, processAdapter)

		platformAdapter := isolation.NewPlatformSandbox()
		manager.Register(core.IsolationPlatform, platformAdapter)

		profile := &core.Profile{
			ID: "degradation-test",
			Isolation: core.IsolationConfig{
				Level:    core.IsolationContainer,
				Strict:   false,
				Fallback: true,
			},
		}
		globalConfig := &core.Config{
			Isolation: core.GlobalIsolationConfig{
				DefaultLevel:    core.IsolationProcess,
				FallbackEnabled: true,
			},
		}

		result, err := manager.GetIsolationManager(profile, globalConfig)
		require.NoError(t, err)
		assert.Equal(t, processAdapter, result)
	})

	t.Run("Uses default level when profile level is empty", func(t *testing.T) {
		manager := isolation.NewManager()
		adapter := isolation.NewProcessIsolation()
		manager.Register(core.IsolationProcess, adapter)

		profile := &core.Profile{
			ID: "default-level-test",
			Isolation: core.IsolationConfig{
				Level:    "",
				Strict:   false,
				Fallback: true,
			},
		}
		globalConfig := &core.Config{
			Isolation: core.GlobalIsolationConfig{
				DefaultLevel:    core.IsolationProcess,
				FallbackEnabled: true,
			},
		}

		result, err := manager.GetIsolationManager(profile, globalConfig)
		require.NoError(t, err)
		assert.Equal(t, adapter, result)
	})
}

func TestIsolationFoundation_ProcessIsolationIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process isolation integration tests use sh/echo and are not supported on Windows")
	}

	t.Run("End-to-end command execution with process isolation", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "e2e-process-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: e2e-process-test
display_name: E2E Process Test
isolation:
  level: process
  strict: false
  fallback: true
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("E2E_VAR=e2e_value\n"), 0o600)
		require.NoError(t, err)

		err = core.RunCommand(profileID, "echo", []string{"integration test"})
		assert.NoError(t, err)
	})

	t.Run("End-to-end action execution with process isolation", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "e2e-action-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: e2e-action-test
display_name: E2E Action Test
isolation:
  level: process
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("E2E_ACTION_VAR=action_value\n"), 0o600)
		require.NoError(t, err)

		actionsDir := filepath.Join(profileDir, "actions")
		err = os.MkdirAll(actionsDir, 0o755)
		require.NoError(t, err)

		actionScript := `#!/bin/sh
echo "E2E Action Executed"
`
		actionPath := filepath.Join(actionsDir, "e2e-test.sh")
		err = os.WriteFile(actionPath, []byte(actionScript), 0o755)
		require.NoError(t, err)

		actionYaml := `id: e2e-test
title: E2E Test Action
type: sh
path: actions/e2e-test.sh
accepts_stdin: false
`
		actionYamlPath := filepath.Join(actionsDir, "e2e-test.yaml")
		err = os.WriteFile(actionYamlPath, []byte(actionYaml), 0o644)
		require.NoError(t, err)

		err = core.RunAction(profileID, "e2e-test", nil)
		assert.NoError(t, err)
	})

	t.Run("Process isolation environment injection", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "env-inject-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: env-inject-test
display_name: Environment Injection Test
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("INJECT_TEST=inject_value\n"), 0o600)
		require.NoError(t, err)

		cmd := exec.Command("sh", "-c", "echo $INJECT_TEST")
		profile, err := core.LoadProfile(profileID)
		require.NoError(t, err)

		err = core.InjectEnvironment(cmd, profile)
		assert.NoError(t, err)

		foundProfileID := false
		foundSecret := false
		for _, env := range cmd.Env {
			if env == "APS_PROFILE_ID=env-inject-test" {
				foundProfileID = true
			}
			if env == "INJECT_TEST=inject_value" {
				foundSecret = true
			}
		}
		assert.True(t, foundProfileID, "APS_PROFILE_ID not found in environment")
		assert.True(t, foundSecret, "INJECT_TEST not found in environment")
	})
}

func TestIsolationFoundation_ConfigIntegration(t *testing.T) {
	t.Run("Profile with isolation config loads successfully", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "config-integration-test"
		createTestProfile(t, dataDir, profileID, `id: config-integration-test
display_name: Config Integration Test
isolation:
  level: process
  strict: false
  fallback: true
`)

		profile, err := core.LoadProfile(profileID)
		require.NoError(t, err)
		assert.Equal(t, core.IsolationProcess, profile.Isolation.Level)
		assert.False(t, profile.Isolation.Strict)
		assert.True(t, profile.Isolation.Fallback)
	})

	t.Run("Global config with isolation settings", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("APS_DATA_PATH", tmpDir)
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		configDir := filepath.Join(tmpDir, "aps")
		err := os.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configContent := `prefix: FOUNDATION
isolation:
  default_level: process
  fallback_enabled: true
`
		configPath := filepath.Join(configDir, "config.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		config, err := core.LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "FOUNDATION", config.Prefix)
		assert.Equal(t, core.IsolationProcess, config.Isolation.DefaultLevel)
		assert.True(t, config.Isolation.FallbackEnabled)
	})
}

func TestIsolationFoundation_ErrorHandling(t *testing.T) {
	t.Run("Invalid isolation level in profile", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "invalid-level-test"
		createTestProfile(t, dataDir, profileID, `id: invalid-level-test
display_name: Invalid Level Test
isolation:
  level: invalid_level
`)

		_, err := core.LoadProfile(profileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid isolation level")
	})

	t.Run("Unsupported isolation level during execution", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("container isolation execution test not supported on Windows")
		}

		dataDir := setupTestDataDir(t)

		profileID := "unsupported-exec-test"
		createTestProfile(t, dataDir, profileID, `id: unsupported-exec-test
display_name: Unsupported Execution Test
isolation:
  level: container
  strict: false
  fallback: true
  container:
    image: ubuntu:22.04
`)

		err := core.RunCommand(profileID, "echo", []string{"test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("Non-existent profile", func(t *testing.T) {
		err := core.RunCommand("nonexistent-profile", "echo", []string{"test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load profile")
	})
}

func TestIsolationFoundation_BackwardCompatibility(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backward compatibility tests use sh and are not supported on Windows")
	}

	t.Run("Old-style profile without isolation section", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "old-style-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: old-style-test
display_name: Old Style Profile
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("OLD_VAR=old_value\n"), 0o600)
		require.NoError(t, err)

		profile, err := core.LoadProfile(profileID)
		require.NoError(t, err)
		assert.Equal(t, core.IsolationProcess, profile.Isolation.Level)

		err = core.RunCommand(profileID, "echo", []string{"backward compat"})
		assert.NoError(t, err)
	})

	t.Run("InjectEnvironment still works for old code", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "inject-compat-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: inject-compat-test
display_name: Inject Compat Test
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("COMPAT_VAR=compat_value\n"), 0o600)
		require.NoError(t, err)

		profile, err := core.LoadProfile(profileID)
		require.NoError(t, err)

		cmd := exec.Command("env")
		err = core.InjectEnvironment(cmd, profile)
		assert.NoError(t, err)

		foundID := false
		foundCompat := false
		for _, env := range cmd.Env {
			if env == "APS_PROFILE_ID=inject-compat-test" {
				foundID = true
			}
			if env == "COMPAT_VAR=compat_value" {
				foundCompat = true
			}
		}
		assert.True(t, foundID, "APS_PROFILE_ID not found")
		assert.True(t, foundCompat, "COMPAT_VAR not found")
	})
}

func TestIsolationFoundation_AllExistingTests(t *testing.T) {
	t.Run("Execution injection test still passes", func(t *testing.T) {
		setupTestDataDir(t)

		_, err := core.LoadConfig()
		assert.NoError(t, err)
	})

	t.Run("Load profile still works", func(t *testing.T) {
		dataDir := setupTestDataDir(t)

		profileID := "existing-test-profile"
		profileDir := createTestProfile(t, dataDir, profileID, `id: existing-test-profile
display_name: Existing Test Profile
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("EXISTING_VAR=existing_value\n"), 0o600)
		require.NoError(t, err)

		profile, err := core.LoadProfile(profileID)
		require.NoError(t, err)
		assert.Equal(t, profileID, profile.ID)
		assert.Equal(t, "Existing Test Profile", profile.DisplayName)
	})

	t.Run("RunAction with shell type still works", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell action tests use sh and are not supported on Windows")
		}

		dataDir := setupTestDataDir(t)

		profileID := "shell-action-test"
		profileDir := createTestProfile(t, dataDir, profileID, `id: shell-action-test
display_name: Shell Action Test
`)
		secretsPath := filepath.Join(profileDir, "secrets.env")
		err := os.WriteFile(secretsPath, []byte("SHELL_ACTION_VAR=shell_action\n"), 0o600)
		require.NoError(t, err)

		actionsDir := filepath.Join(profileDir, "actions")
		err = os.MkdirAll(actionsDir, 0o755)
		require.NoError(t, err)

		actionScript := `#!/bin/sh
echo "Shell action executed"
`
		actionPath := filepath.Join(actionsDir, "existing-test.sh")
		err = os.WriteFile(actionPath, []byte(actionScript), 0o755)
		require.NoError(t, err)

		actionYaml := `id: existing-test
title: Existing Test Action
type: sh
path: actions/existing-test.sh
accepts_stdin: false
`
		actionYamlPath := filepath.Join(actionsDir, "existing-test.yaml")
		err = os.WriteFile(actionYamlPath, []byte(actionYaml), 0o644)
		require.NoError(t, err)

		err = core.RunAction(profileID, "existing-test", nil)
		assert.NoError(t, err)
	})
}
