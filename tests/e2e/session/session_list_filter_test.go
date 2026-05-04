package session_e2e

import (
	"strings"
	"testing"
)

// fixtureRegistry is a minimal JSON registry covering every
// dimension the `aps session list` filter flags target.
const fixtureRegistry = `{
  "s-alpha": {
    "id": "s-alpha",
    "profile_id": "alice",
    "command": "shell",
    "pid": 100,
    "status": "active",
    "tier": "premium",
    "type": "voice",
    "workspace_id": "team-a",
    "created_at": "2026-01-01T10:00:00Z",
    "last_seen_at": "2026-01-01T10:05:00Z"
  },
  "s-beta": {
    "id": "s-beta",
    "profile_id": "bob",
    "command": "shell",
    "pid": 200,
    "status": "inactive",
    "tier": "basic",
    "type": "",
    "workspace_id": "team-b",
    "created_at": "2026-01-01T11:00:00Z",
    "last_seen_at": "2026-01-01T11:05:00Z"
  },
  "s-gamma": {
    "id": "s-gamma",
    "profile_id": "alice",
    "command": "shell",
    "pid": 300,
    "status": "errored",
    "tier": "standard",
    "type": "",
    "workspace_id": "",
    "created_at": "2026-01-01T12:00:00Z",
    "last_seen_at": "2026-01-01T12:05:00Z"
  }
}
`

func TestSessionList_RichDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, fixtureRegistry)

	stdout, stderr, err := runAPS(t, home, "session", "list")
	if err != nil {
		t.Fatalf("session list: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"ID", "PROFILE", "STATUS", "WORKSPACE", "TYPE", "TIER", "s-alpha", "s-beta", "s-gamma"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("session list output missing %q\nstdout: %s", want, stdout)
		}
	}
}

func TestSessionList_FilterStatus(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, fixtureRegistry)

	stdout, _, err := runAPS(t, home, "session", "list", "--status", "active")
	if err != nil {
		t.Fatalf("session list --status: %v", err)
	}
	if !strings.Contains(stdout, "s-alpha") {
		t.Errorf("expected s-alpha (status=active): %s", stdout)
	}
	if strings.Contains(stdout, "s-beta") || strings.Contains(stdout, "s-gamma") {
		t.Errorf("did not expect non-active sessions: %s", stdout)
	}

	stdout, _, err = runAPS(t, home, "session", "list", "--status", "no-such-status")
	if err != nil {
		t.Fatalf("session list --status bogus: %v", err)
	}
	for _, id := range []string{"s-alpha", "s-beta", "s-gamma"} {
		if strings.Contains(stdout, id) {
			t.Errorf("expected zero rows for bogus status, found %q in: %s", id, stdout)
		}
	}
}

func TestSessionList_FilterProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, fixtureRegistry)

	stdout, _, err := runAPS(t, home, "session", "list", "--profile", "bob")
	if err != nil {
		t.Fatalf("session list --profile: %v", err)
	}
	if !strings.Contains(stdout, "s-beta") {
		t.Errorf("expected s-beta (profile=bob): %s", stdout)
	}
	if strings.Contains(stdout, "s-alpha") || strings.Contains(stdout, "s-gamma") {
		t.Errorf("did not expect alice's sessions: %s", stdout)
	}
}

func TestSessionList_FilterWorkspace(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, fixtureRegistry)

	stdout, _, err := runAPS(t, home, "session", "list", "--workspace", "team-a")
	if err != nil {
		t.Fatalf("session list --workspace: %v", err)
	}
	if !strings.Contains(stdout, "s-alpha") {
		t.Errorf("expected s-alpha (workspace=team-a): %s", stdout)
	}
	if strings.Contains(stdout, "s-beta") {
		t.Errorf("did not expect s-beta (workspace=team-b): %s", stdout)
	}
}

func TestSessionList_FilterTier(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, fixtureRegistry)

	stdout, _, err := runAPS(t, home, "session", "list", "--tier", "premium")
	if err != nil {
		t.Fatalf("session list --tier: %v", err)
	}
	if !strings.Contains(stdout, "s-alpha") {
		t.Errorf("expected s-alpha (tier=premium): %s", stdout)
	}
	if strings.Contains(stdout, "s-beta") || strings.Contains(stdout, "s-gamma") {
		t.Errorf("did not expect non-premium sessions: %s", stdout)
	}
}
