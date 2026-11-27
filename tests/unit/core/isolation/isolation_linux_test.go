//go:build linux

package isolation_test

import (
	"os/exec"
	"testing"

	isolation "hop.top/aps/internal/core/isolation"
)

// TestLinuxSandbox_IsAvailable checks IsAvailable agrees with an independent
// probe of its prerequisites (unshare, setfacl, sudo on PATH), resolved via
// the shell rather than exec.LookPath.
func TestLinuxSandbox_IsAvailable(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found; cannot probe prerequisites independently")
	}

	want := true
	for _, bin := range []string{"unshare", "setfacl", "sudo"} {
		if exec.Command(sh, "-c", "command -v "+bin).Run() != nil {
			want = false
		}
	}

	got := isolation.NewLinuxSandbox().IsAvailable()
	if got != want {
		t.Errorf("IsAvailable() = %v, want %v (unshare+setfacl+sudo on PATH)", got, want)
	}
}

func TestLinuxSandbox_PrepareContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test requiring profile")
	}

	tempDir := setupTestProfile(t)
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	profileID := "test-profile"

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
	tempDir := setupTestProfile(t)
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	linux := isolation.NewLinuxSandbox()
	err := linux.Validate()

	if err == nil {
		t.Error("expected error when context not prepared")
	}
}
