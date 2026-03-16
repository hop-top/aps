package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTempHome(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
	t.Setenv("APS_DATA_PATH", "")
	return tmpHome
}

func TestAddCapabilityToProfile(t *testing.T) {
	setTempHome(t)

	err := core.CreateProfile("test-add", core.Profile{DisplayName: "Test"})
	require.NoError(t, err)

	// Add capability
	err = core.AddCapabilityToProfile("test-add", "a2a")
	require.NoError(t, err)

	profile, err := core.LoadProfile("test-add")
	require.NoError(t, err)
	assert.Contains(t, profile.Capabilities, "a2a")

	// Deduplicate: adding again should be a no-op
	err = core.AddCapabilityToProfile("test-add", "a2a")
	require.NoError(t, err)

	profile, err = core.LoadProfile("test-add")
	require.NoError(t, err)
	count := 0
	for _, c := range profile.Capabilities {
		if c == "a2a" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should not duplicate capability")
}

func TestRemoveCapabilityFromProfile(t *testing.T) {
	setTempHome(t)

	profile := core.Profile{
		DisplayName:  "Test",
		Capabilities: []string{"a2a", "webhooks"},
	}
	err := core.CreateProfile("test-remove", profile)
	require.NoError(t, err)

	// Remove existing
	err = core.RemoveCapabilityFromProfile("test-remove", "a2a")
	require.NoError(t, err)

	p, err := core.LoadProfile("test-remove")
	require.NoError(t, err)
	assert.NotContains(t, p.Capabilities, "a2a")
	assert.Contains(t, p.Capabilities, "webhooks")

	// Remove non-existent should error
	err = core.RemoveCapabilityFromProfile("test-remove", "nonexistent")
	assert.Error(t, err)
}

func TestProfileHasCapability(t *testing.T) {
	profile := &core.Profile{
		Capabilities: []string{"a2a", "my-cap"},
	}

	assert.True(t, core.ProfileHasCapability(profile, "a2a"))
	assert.True(t, core.ProfileHasCapability(profile, "my-cap"))
	assert.False(t, core.ProfileHasCapability(profile, "absent"))

	// Nil/empty capabilities
	empty := &core.Profile{}
	assert.False(t, core.ProfileHasCapability(empty, "a2a"))
}

func TestProfilesUsingCapability(t *testing.T) {
	setTempHome(t)

	// Create two profiles
	err := core.CreateProfile("prof-a", core.Profile{
		DisplayName:  "A",
		Capabilities: []string{"a2a", "webhooks"},
	})
	require.NoError(t, err)

	err = core.CreateProfile("prof-b", core.Profile{
		DisplayName:  "B",
		Capabilities: []string{"a2a"},
	})
	require.NoError(t, err)

	err = core.CreateProfile("prof-c", core.Profile{
		DisplayName: "C",
	})
	require.NoError(t, err)

	// a2a is on prof-a and prof-b
	profs, err := core.ProfilesUsingCapability("a2a")
	require.NoError(t, err)
	assert.Len(t, profs, 2)
	assert.Contains(t, profs, "prof-a")
	assert.Contains(t, profs, "prof-b")

	// webhooks is only on prof-a
	profs, err = core.ProfilesUsingCapability("webhooks")
	require.NoError(t, err)
	assert.Len(t, profs, 1)
	assert.Contains(t, profs, "prof-a")

	// unknown capability
	profs, err = core.ProfilesUsingCapability("nonexistent")
	require.NoError(t, err)
	assert.Len(t, profs, 0)
}

func TestInjectEnvironment_PerProfileCaps(t *testing.T) {
	tmpHome := setTempHome(t)

	// Create capability dir at the XDG data path (XDG_DATA_HOME/aps/capabilities/...)
	capDir := filepath.Join(tmpHome, ".local", "share", "aps", "capabilities", "my-tool")
	err := os.MkdirAll(capDir, 0755)
	require.NoError(t, err)

	// Create profile with the external capability
	err = core.CreateProfile("cap-inject", core.Profile{
		DisplayName:  "Cap Inject",
		Capabilities: []string{"a2a", "my-tool"},
	})
	require.NoError(t, err)

	profile, err := core.LoadProfile("cap-inject")
	require.NoError(t, err)

	cmd := exec.Command("echo", "test")
	err = core.InjectEnvironment(cmd, profile)
	require.NoError(t, err)

	// Check that APS_MY_TOOL_PATH is injected
	found := false
	for _, e := range cmd.Env {
		if e == "APS_MY_TOOL_PATH="+capDir {
			found = true
		}
	}
	assert.True(t, found, "expected APS_MY_TOOL_PATH in env")
}

func TestInjectEnvironment_NonEnabledCapsNotInjected(t *testing.T) {
	tmpHome := setTempHome(t)

	// Create capability dir for a cap NOT in the profile
	capDir := filepath.Join(tmpHome, ".local", "share", "aps", "capabilities", "other-tool")
	err := os.MkdirAll(capDir, 0755)
	require.NoError(t, err)

	// Profile does NOT have "other-tool" in capabilities
	err = core.CreateProfile("no-cap", core.Profile{
		DisplayName:  "No Cap",
		Capabilities: []string{"a2a"},
	})
	require.NoError(t, err)

	profile, err := core.LoadProfile("no-cap")
	require.NoError(t, err)

	cmd := exec.Command("echo", "test")
	err = core.InjectEnvironment(cmd, profile)
	require.NoError(t, err)

	// APS_OTHER_TOOL_PATH should NOT be present
	for _, e := range cmd.Env {
		assert.NotContains(t, e, "APS_OTHER_TOOL_PATH",
			"non-enabled cap should not be injected")
	}
}
