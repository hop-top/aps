package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"oss-aps-cli/internal/core"

	"github.com/stretchr/testify/assert"
)

func TestInjectEnvironment(t *testing.T) {
	profile := &core.Profile{
		ID: "test-profile",
	}

	cmd := exec.Command("env")
	err := core.InjectEnvironment(cmd, profile)
	assert.NoError(t, err)

	// Check for default prefix APS_
	foundID := false
	for _, env := range cmd.Env {
		if env == "APS_PROFILE_ID=test-profile" {
			foundID = true
			break
		}
	}
	assert.True(t, foundID, "Expected APS_PROFILE_ID=test-profile in environment")
}

func TestInjectEnvironmentCustomPrefix(t *testing.T) {
	tempHome := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempHome)

	configDir := filepath.Join(tempHome, "aps")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("prefix: CUSTOM"), 0644)

	profile := &core.Profile{
		ID: "test-profile",
	}

	cmd := exec.Command("env")
	err := core.InjectEnvironment(cmd, profile)
	assert.NoError(t, err)

	// Check for custom prefix CUSTOM_
	foundID := false
	for _, env := range cmd.Env {
		if env == "CUSTOM_PROFILE_ID=test-profile" {
			foundID = true
			break
		}
	}
	assert.True(t, foundID, "Expected CUSTOM_PROFILE_ID=test-profile in environment")
}