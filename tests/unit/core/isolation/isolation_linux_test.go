//go:build linux

package isolation_test

import (
	"os/exec"
	"testing"

	isolation "oss-aps-cli/internal/core/isolation"
)

func TestLinuxSandbox_IsAvailable(t *testing.T) {
	linux := isolation.NewLinuxSandbox()
	available := linux.IsAvailable()

	if _, err := exec.LookPath("unshare"); err != nil {
		if available {
			t.Error("expected LinuxSandbox to be unavailable without unshare")
		}
		return
	}

	if !available {
		t.Error("expected LinuxSandbox to be available on Linux with unshare")
	}
}

func TestLinuxSandbox_PrepareContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test requiring profile")
	}

	profileID := "test-linux-profile"

	linux := isolation.NewLinuxSandbox()
	ctx, err := linux.PrepareContext(profileID)

	if err != nil {
		t.Fatalf("failed to prepare context: %v", err)
	}

	if ctx.ProfileID != profileID {
		t.Errorf("expected ProfileID %s, got %s", profileID, ctx.ProfileID)
	}

	if ctx.WorkingDir == "" {
		t.Error("expected WorkingDir to be set")
	}
}

func TestLinuxSandbox_Validate_NotPrepared(t *testing.T) {
	linux := isolation.NewLinuxSandbox()
	err := linux.Validate()

	if err == nil {
		t.Error("expected error when context not prepared")
	}
}
