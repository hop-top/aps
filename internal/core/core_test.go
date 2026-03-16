package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Profile Management Tests (15 tests)
// ============================================================================

// TestValidateIsolationProcessLevel tests process isolation level validation
func TestValidateIsolationProcessLevel(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationProcess,
		},
	}

	err := profile.ValidateIsolation()
	require.NoError(t, err)
	assert.Equal(t, IsolationProcess, profile.Isolation.Level)
}

// TestValidateIsolationPlatformLevel tests platform isolation level validation
func TestValidateIsolationPlatformLevel(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationPlatform,
		},
	}

	err := profile.ValidateIsolation()
	require.NoError(t, err)
	assert.Equal(t, IsolationPlatform, profile.Isolation.Level)
}

// TestValidateIsolationContainerLevelWithImage tests container level with image
func TestValidateIsolationContainerLevelWithImage(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationContainer,
			Container: ContainerConfig{
				Image: "test-image:latest",
			},
		},
	}

	err := profile.ValidateIsolation()
	require.NoError(t, err)
	assert.Equal(t, IsolationContainer, profile.Isolation.Level)
}

// TestValidateIsolationContainerNoImage tests container isolation without image fails
func TestValidateIsolationContainerNoImage(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationContainer,
		},
	}

	err := profile.ValidateIsolation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "container isolation requires an image")
}

// TestValidateIsolationInvalidLevel tests invalid isolation level
func TestValidateIsolationInvalidLevel(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationLevel("invalid-level"),
		},
	}

	err := profile.ValidateIsolation()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid isolation level")
}

// TestValidateIsolationDefaultLevel tests default level assignment
func TestValidateIsolationDefaultLevel(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: "",
		},
	}

	err := profile.ValidateIsolation()
	require.NoError(t, err)
	assert.Equal(t, IsolationProcess, profile.Isolation.Level)
}

// TestSaveAndLoadProfile tests saving and loading a profile
func TestSaveAndLoadProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := &Profile{
		ID:          "test-profile",
		DisplayName: "Test Profile",
		Persona: Persona{
			Tone: "professional",
		},
		Preferences: Preferences{
			Shell: "/bin/bash",
		},
	}

	err := SaveProfile(profile)
	require.NoError(t, err)

	profileDir := filepath.Join(tmpDir, "profiles", "test-profile")
	assert.DirExists(t, profileDir)
	assert.FileExists(t, filepath.Join(profileDir, "profile.yaml"))
}

// TestSaveProfileEmptyID tests SaveProfile with empty ID fails
func TestSaveProfileEmptyID(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID:          "",
		DisplayName: "No ID Profile",
	}

	err := SaveProfile(profile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile ID cannot be empty")
}

// TestCreateProfileWithDefaults tests CreateProfile creates directory structure
func TestCreateProfileWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	config := Profile{
		DisplayName: "New Test Profile",
		Git: GitConfig{
			Enabled: false,
		},
	}

	err := CreateProfile("new-profile", config)
	require.NoError(t, err)

	profileDir := filepath.Join(tmpDir, "profiles", "new-profile")
	assert.DirExists(t, filepath.Join(profileDir, "actions"))
	assert.FileExists(t, filepath.Join(profileDir, "profile.yaml"))
	assert.FileExists(t, filepath.Join(profileDir, "secrets.env"))
	assert.FileExists(t, filepath.Join(profileDir, "notes.md"))
}

// TestCreateProfileWithGit tests CreateProfile with git config
func TestCreateProfileWithGit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	config := Profile{
		DisplayName: "Git Profile",
		Git: GitConfig{
			Enabled: true,
		},
	}

	err := CreateProfile("git-profile", config)
	require.NoError(t, err)

	profileDir := filepath.Join(tmpDir, "profiles", "git-profile")
	assert.FileExists(t, filepath.Join(profileDir, "gitconfig"))

	data, err := os.ReadFile(filepath.Join(profileDir, "gitconfig"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "Git Profile")
}

// TestCreateProfileAlreadyExists tests CreateProfile fails if profile exists
func TestCreateProfileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	config := Profile{
		DisplayName: "First Profile",
	}

	err := CreateProfile("existing-profile", config)
	require.NoError(t, err)

	err = CreateProfile("existing-profile", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestGetProfileDir returns correct path
func TestGetProfileDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	dir, err := GetProfileDir("test-id")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "profiles", "test-id")
	assert.Equal(t, expected, dir)
}

// TestGetProfilePath returns profile.yaml path
func TestGetProfilePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	path, err := GetProfilePath("test-id")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "profiles", "test-id", "profile.yaml")
	assert.Equal(t, expected, path)
}

// TestListProfilesEmpty returns empty list when no profiles exist
func TestListProfilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profiles, err := ListProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

// TestListProfilesMultiple returns all profiles
func TestListProfilesMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	for _, id := range []string{"profile1", "profile2", "profile3"} {
		config := Profile{
			DisplayName: fmt.Sprintf("Profile %s", id),
		}
		err := CreateProfile(id, config)
		require.NoError(t, err)
	}

	profiles, err := ListProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 3)
	assert.Contains(t, profiles, "profile1")
	assert.Contains(t, profiles, "profile2")
	assert.Contains(t, profiles, "profile3")
}

// TestProfileWithA2AConfig tests profile with A2A configuration
func TestProfileWithA2AConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "A2A Profile",
		A2A: &A2AConfig{
			ProtocolBinding: "stdio",
			ListenAddr:      "127.0.0.1:5000",
		},
	}

	err := CreateProfile("a2a-profile", profile)
	require.NoError(t, err)

	profilePath := filepath.Join(tmpDir, "profiles", "a2a-profile", "profile.yaml")
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)

	var loaded Profile
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.NotNil(t, loaded.A2A)
}

// TestProfileWithACPConfig tests profile with ACP configuration
func TestProfileWithACPConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "ACP Profile",
		ACP: &ACPConfig{
			Enabled:    true,
			Transport:  "http",
			ListenAddr: "0.0.0.0",
			Port:       8080,
		},
	}

	err := CreateProfile("acp-profile", profile)
	require.NoError(t, err)

	profilePath := filepath.Join(tmpDir, "profiles", "acp-profile", "profile.yaml")
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)

	var loaded Profile
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.NotNil(t, loaded.ACP)
	assert.True(t, loaded.ACP.Enabled)
	assert.Equal(t, 8080, loaded.ACP.Port)
}

// TestConcurrentProfileCreation tests concurrent profile operations
func TestConcurrentProfileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			profile := Profile{
				DisplayName: fmt.Sprintf("Profile %d", idx),
			}
			errChan <- CreateProfile(fmt.Sprintf("profile-%d", idx), profile)
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}

	profiles, err := ListProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 10)
}

// ============================================================================
// Execution Engine Tests (8 tests)
// ============================================================================

// TestInjectEnvironmentVariables tests environment variable injection
func TestInjectEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	profile := Profile{
		DisplayName: "Env Test",
	}
	err := CreateProfile("env-test", profile)
	require.NoError(t, err)

	testProfile, err := LoadProfile("env-test")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, testProfile)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "APS_PROFILE_ID=env-test")
	assert.Contains(t, envStr, "APS_PROFILE_DIR=")
	assert.Contains(t, envStr, "APS_PROFILE_YAML=")
}

// TestInjectEnvironmentWithSecrets tests secrets injection
func TestInjectEnvironmentWithSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "Secret Test",
	}
	err := CreateProfile("secret-test", profile)
	require.NoError(t, err)

	secretsPath := filepath.Join(tmpDir, "profiles", "secret-test", "secrets.env")
	err = os.WriteFile(secretsPath, []byte("SECRET_KEY=secret_value\nAPI_TOKEN=token123\n"), 0600)
	require.NoError(t, err)

	testProfile, err := LoadProfile("secret-test")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, testProfile)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "SECRET_KEY=secret_value")
	assert.Contains(t, envStr, "API_TOKEN=token123")
}

// TestInjectEnvironmentWithGitConfig tests git config injection
func TestInjectEnvironmentWithGitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "Git Test",
		Git: GitConfig{
			Enabled: true,
		},
	}
	err := CreateProfile("git-test", profile)
	require.NoError(t, err)

	testProfile, err := LoadProfile("git-test")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, testProfile)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "GIT_CONFIG_GLOBAL=")
}

// TestInjectEnvironmentWithSSHConfig tests SSH config injection
func TestInjectEnvironmentWithSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "SSH Test",
		SSH: SSHConfig{
			Enabled: true,
			KeyPath: "ssh.key",
		},
	}
	err := CreateProfile("ssh-test", profile)
	require.NoError(t, err)

	sshKeyPath := filepath.Join(tmpDir, "profiles", "ssh-test", "ssh.key")
	err = os.WriteFile(sshKeyPath, []byte("mock-key"), 0600)
	require.NoError(t, err)

	testProfile, err := LoadProfile("ssh-test")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = InjectEnvironment(cmd, testProfile)
	require.NoError(t, err)

	envStr := strings.Join(cmd.Env, "\n")
	assert.Contains(t, envStr, "GIT_SSH_COMMAND=ssh -i")
}

// TestRunCommandWithProcessIsolation tests basic command execution
func TestRunCommandWithProcessIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "Run Test",
		Isolation: IsolationConfig{
			Level: IsolationProcess,
		},
	}
	err := CreateProfile("run-test", profile)
	require.NoError(t, err)

	loadedProfile, err := LoadProfile("run-test")
	require.NoError(t, err)

	err = runCommandWithProcessIsolation(loadedProfile, "true", []string{})
	require.NoError(t, err)
}

// TestActionTypeResolution tests action type detection
func TestActionTypeResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected string
	}{
		{"script.sh", "sh"},
		{"script.py", "py"},
		{"script.js", "js"},
		{"script.unknown", "unknown"},
		{"script", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			actionType := resolveType(tt.filename)
			assert.Equal(t, tt.expected, actionType)
		})
	}
}

// TestRunActionPayloadHandling tests action with payload
func TestRunActionPayloadHandling(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	profile := Profile{
		DisplayName: "Action Test",
		Isolation: IsolationConfig{
			Level: IsolationProcess,
		},
	}
	err := CreateProfile("action-test", profile)
	require.NoError(t, err)

	actionsDir := filepath.Join(tmpDir, "profiles", "action-test", "actions")

	testScript := filepath.Join(actionsDir, "test.sh")
	err = os.WriteFile(testScript, []byte("#!/bin/bash\ncat"), 0755)
	require.NoError(t, err)

	loadedProfile, err := LoadProfile("action-test")
	require.NoError(t, err)

	action := Action{
		ID:         "test",
		Title:      "Test Action",
		Entrypoint: "test.sh",
		Path:       testScript,
		Type:       "sh",
	}

	payload := []byte("test input data")
	err = runActionWithProcessIsolation(loadedProfile, &action, payload)
	require.NoError(t, err)
}

// ============================================================================
// Configuration Tests (7 tests)
// ============================================================================

// TestLoadConfigDefault tests loading config returns defaults
func TestLoadConfigDefault(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "APS", config.Prefix)
	assert.Equal(t, IsolationProcess, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)
}

// TestLoadConfigFromDisk tests loading saved config
func TestLoadConfigFromDisk(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configData := &Config{
		Prefix: "CUSTOM_PREFIX",
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationContainer,
			FallbackEnabled: false,
		},
	}

	err := SaveConfig(configData)
	require.NoError(t, err)

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM_PREFIX", config.Prefix)
	assert.Equal(t, IsolationContainer, config.Isolation.DefaultLevel)
	assert.False(t, config.Isolation.FallbackEnabled)
}

// TestLoadConfigMalformed returns defaults on malformed YAML
func TestLoadConfigMalformed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "aps")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("invalid: [yaml:"), 0644)
	require.NoError(t, err)

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "APS", config.Prefix)
}

// TestSaveConfig saves config correctly
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	config := &Config{
		Prefix: "TEST_PREFIX",
		Isolation: GlobalIsolationConfig{
			DefaultLevel:    IsolationPlatform,
			FallbackEnabled: true,
		},
	}

	err := SaveConfig(config)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, "aps", "config.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loaded Config
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Equal(t, "TEST_PREFIX", loaded.Prefix)
}

// TestMigrateConfigNoExisting returns false when no config exists
func TestMigrateConfigNoExisting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	migrated, err := MigrateConfig()
	require.NoError(t, err)
	assert.False(t, migrated)
}

// TestMigrateConfigExisting migrates old config format
func TestMigrateConfigExisting(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "aps")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	oldConfig := []byte("prefix: OLD_PREFIX\n")
	err := os.WriteFile(filepath.Join(configDir, "config.yaml"), oldConfig, 0644)
	require.NoError(t, err)

	migrated, err := MigrateConfig()
	require.NoError(t, err)
	assert.True(t, migrated)

	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	require.NoError(t, err)

	var config Config
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)
	assert.Equal(t, "OLD_PREFIX", config.Prefix)
	assert.Equal(t, IsolationProcess, config.Isolation.DefaultLevel)
}

// TestConcurrentConfigOperations tests concurrent config access
func TestConcurrentConfigOperations(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", oldXDG) })

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config := &Config{
				Prefix: "CONCURRENT_TEST",
			}
			errChan <- SaveConfig(config)
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "CONCURRENT_TEST", config.Prefix)
}

// ============================================================================
// Additional Tests and Edge Cases
// ============================================================================

// TestCreateA2AClient tests A2A client creation
func TestCreateA2AClient(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test-profile",
	}

	client, err := profile.CreateA2AClient("other-profile")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "other-profile", client.GetTargetProfileID())
}

// TestCreateA2AClientEmptyID tests A2A client fails with empty ID
func TestCreateA2AClientEmptyID(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID: "test-profile",
	}

	client, err := profile.CreateA2AClient("")
	require.Error(t, err)
	assert.Nil(t, client)
}

// TestShellDetection tests shell detection utility
func TestShellDetection(t *testing.T) {
	t.Parallel()

	shell := DetectShell()
	assert.NotEmpty(t, shell)
}

// TestIsCommandAvailable tests command availability check
func TestIsCommandAvailable(t *testing.T) {
	t.Parallel()

	sh := IsCommandAvailable("sh")
	if runtime.GOOS != "windows" {
		assert.True(t, sh)
	}

	unavailable := IsCommandAvailable("nonexistent-command-xyz-abc-123")
	assert.False(t, unavailable)
}

// TestGetShellName tests shell name extraction
func TestGetShellName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		expected string
	}{
		{"/bin/bash", "bash"},
		{"/usr/bin/zsh", "zsh"},
		{"/bin/sh", "sh"},
		{"bash", "bash"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			name := GetShellName(tt.path)
			assert.Equal(t, tt.expected, name)
		})
	}
}

// TestGetConfigDir tests config directory resolution
func TestGetConfigDir(t *testing.T) {
	t.Parallel()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	dir, err := GetConfigDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/xdg/aps", dir)

	os.Unsetenv("XDG_CONFIG_HOME")
	dir, err = GetConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "aps")
}

// TestLoadSecretsNonExistent returns nil for non-existent file
func TestLoadSecretsNonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "nonexistent.env")

	secrets, err := LoadSecrets(secretsPath)
	require.NoError(t, err)
	assert.Nil(t, secrets)
}

// TestLoadSecretsValid loads secrets correctly
func TestLoadSecretsValid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.env")

	err := os.WriteFile(secretsPath, []byte("KEY1=value1\nKEY2=value2\n"), 0600)
	require.NoError(t, err)

	secrets, err := LoadSecrets(secretsPath)
	require.NoError(t, err)
	assert.Equal(t, "value1", secrets["KEY1"])
	assert.Equal(t, "value2", secrets["KEY2"])
}

// TestProfileYAMLMarshaling tests profile can be marshaled to YAML
func TestProfileYAMLMarshaling(t *testing.T) {
	t.Parallel()

	profile := &Profile{
		ID:          "test",
		DisplayName: "Test Profile",
		Persona: Persona{
			Tone: "professional",
		},
		A2A: &A2AConfig{},
		ACP: &ACPConfig{
			Enabled: true,
			Port:    8080,
		},
	}

	data, err := yaml.Marshal(profile)
	require.NoError(t, err)

	var loaded Profile
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, profile.ID, loaded.ID)
	assert.Equal(t, profile.DisplayName, loaded.DisplayName)
	assert.Equal(t, profile.Persona.Tone, loaded.Persona.Tone)
	assert.NotNil(t, loaded.A2A)
	assert.NotNil(t, loaded.ACP)
}

func TestProfileWithWorkspaceLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".agents", "profiles", "ws-test"), 0755)

	profile := Profile{
		ID:          "ws-test",
		DisplayName: "Workspace Test",
		Workspace: &WorkspaceLink{
			Name:  "dev-project",
			Scope: "global",
		},
	}

	err := SaveProfile(&profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("ws-test")
	require.NoError(t, err)
	require.NotNil(t, loaded.Workspace)
	assert.Equal(t, "dev-project", loaded.Workspace.Name)
	assert.Equal(t, "global", loaded.Workspace.Scope)
}

func TestProfileWithoutWorkspaceLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".agents", "profiles", "no-ws-test"), 0755)

	profile := Profile{
		ID:          "no-ws-test",
		DisplayName: "No Workspace",
	}

	err := SaveProfile(&profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("no-ws-test")
	require.NoError(t, err)
	assert.Nil(t, loaded.Workspace)
}

// TestGetAgentsDir returns XDG data dir (delegates to GetDataDir)
func TestGetAgentsDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmpDir)

	dir, err := GetAgentsDir()
	require.NoError(t, err)

	assert.Equal(t, tmpDir, dir)
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkValidateIsolation benchmarks isolation validation
func BenchmarkValidateIsolation(b *testing.B) {
	profile := &Profile{
		ID: "test",
		Isolation: IsolationConfig{
			Level: IsolationProcess,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile.ValidateIsolation()
	}
}

// BenchmarkResolveActionType benchmarks action type resolution
func BenchmarkResolveActionType(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolveType("test.sh")
		resolveType("test.py")
		resolveType("test.js")
	}
}
