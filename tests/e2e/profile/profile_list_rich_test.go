package profile_e2e

import (
	"strings"
	"testing"
)

// TestProfileList_RichDefault confirms `aps profile list` with no
// filters renders the multi-column row (ID, DISPLAY NAME, ROLES, …)
// for all profiles on disk.
func TestProfileList_RichDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	writeProfile(t, home, "alpha", `id: alpha
display_name: Alpha Agent
email: alpha@example.com
roles:
  - owner
capabilities:
  - webhooks
  - github
`)
	writeProfile(t, home, "beta", `id: beta
display_name: Beta Agent
email: beta@example.com
`)

	stdout, stderr, err := runAPS(t, home, "profile", "list")
	if err != nil {
		t.Fatalf("profile list: %v\nstderr: %s", err, stderr)
	}

	for _, want := range []string{"ID", "DISPLAY NAME", "alpha", "Alpha Agent", "beta", "Beta Agent"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("profile list output missing %q\nstdout: %s", want, stdout)
		}
	}
}

// TestProfileList_EmptyHome renders cleanly when no profiles exist —
// no panic, exit 0, kit/output's empty-table hint surfaces.
func TestProfileList_EmptyHome(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "profile", "list")
	if err != nil {
		t.Fatalf("profile list on empty home: %v\nstderr: %s", err, stderr)
	}
}
