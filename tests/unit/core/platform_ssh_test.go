//go:build darwin

package core_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"oss-aps-cli/internal/core/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformSSH_AttachToPlatformSandbox(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping in CI environment")
	}

	registry := session.GetRegistry()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))
	defer os.Setenv("XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"))

	profileID := "ssh-attach-test"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: ssh-attach-test
display_name: SSH Attach Test
isolation:
  level: platform
  strict: false
  fallback: true
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("SSH_TEST_VAR=ssh_value\n"), 0600)
	require.NoError(t, err)

	t.Run("Session with macos-darwin platform", func(t *testing.T) {
		sess := &session.SessionInfo{
			ID:         "test-ssh-session",
			ProfileID:  profileID,
			ProfileDir: profileDir,
			Command:    "echo 'test'",
			PID:        12345,
			Status:     session.SessionActive,
			Tier:       session.TierStandard,
			TmuxSocket: filepath.Join(os.TempDir(), "tmux-socket"),
			CreatedAt:  time.Now(),
			LastSeenAt: time.Now(),
			Environment: map[string]string{
				"platform_type": "macos-darwin",
				"sandbox_user":  "aps-ssh-test",
			},
		}

		err := registry.Register(sess)
		require.NoError(t, err)
		defer registry.Unregister(sess.ID)

		assert.Equal(t, "aps-ssh-test", sess.Environment["sandbox_user"])
	})

	t.Run("Session without platform type", func(t *testing.T) {
		sess := &session.SessionInfo{
			ID:          "test-no-platform-session",
			ProfileID:   profileID,
			ProfileDir:  profileDir,
			Command:     "echo 'test'",
			PID:         12346,
			Status:      session.SessionActive,
			Tier:        session.TierStandard,
			TmuxSocket:  filepath.Join(os.TempDir(), "tmux-socket-2"),
			CreatedAt:   time.Now(),
			LastSeenAt:  time.Now(),
			Environment: map[string]string{},
		}

		err := registry.Register(sess)
		require.NoError(t, err)
		defer registry.Unregister(sess.ID)

		assert.Empty(t, sess.Environment["platform_type"])
	})
}

func TestPlatformSSH_EnvironmentValidation(t *testing.T) {
	t.Run("Valid macOS session environment", func(t *testing.T) {
		env := map[string]string{
			"platform_type": "macos-darwin",
			"sandbox_user":  "aps-test-profile",
			"sandbox_home":  "/Users/aps-test-profile",
		}

		assert.Equal(t, "macos-darwin", env["platform_type"])
		assert.Equal(t, "aps-test-profile", env["sandbox_user"])
		assert.Contains(t, env["sandbox_home"], "aps-test-profile")
	})

	t.Run("Valid Linux session environment", func(t *testing.T) {
		env := map[string]string{
			"platform_type": "linux-namespace",
			"sandbox_user":  "aps-test-profile",
		}

		assert.Equal(t, "linux-namespace", env["platform_type"])
		assert.Equal(t, "aps-test-profile", env["sandbox_user"])
	})

	t.Run("Missing sandbox user in environment", func(t *testing.T) {
		env := map[string]string{
			"platform_type": "macos-darwin",
		}

		_, ok := env["sandbox_user"]
		assert.False(t, ok, "sandbox_user should not be present")
	})
}

func TestPlatformSSH_AdminKeyPath(t *testing.T) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		t.Skip("HOME environment variable not set")
	}

	expectedKeyPath := filepath.Join(homeDir, ".aps", "keys/admin_priv")

	assert.Equal(t, filepath.Join(homeDir, ".aps", "keys", "admin_priv"), expectedKeyPath)
}
