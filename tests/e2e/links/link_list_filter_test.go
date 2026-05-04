package links_e2e

import (
	"strings"
	"testing"
)

// seedFilterLinks populates a home with two profiles and three
// messenger-profile links across two messenger devices.
//
//   - profile alpha:
//       link to messenger "tg-bot" (enabled, 2 mappings, default=route)
//       link to messenger "slack-bot" (disabled, 0 mappings)
//   - profile beta:
//       link to messenger "tg-bot" (enabled, 1 mapping, no default)
func seedFilterLinks(t *testing.T, home string) {
	t.Helper()

	writeProfile(t, home, "alpha")
	writeProfile(t, home, "beta")

	writeMessengerLinks(t, home, "alpha", `[
  {
    "profile_id": "alpha",
    "messenger_name": "tg-bot",
    "messenger_scope": "global",
    "enabled": true,
    "mappings": {
      "-100123": "alpha=triage",
      "-100124": "alpha=route"
    },
    "default_action": "route",
    "created_at": "2026-04-01T10:00:00Z",
    "updated_at": "2026-04-01T10:00:00Z"
  },
  {
    "profile_id": "alpha",
    "messenger_name": "slack-bot",
    "messenger_scope": "global",
    "enabled": false,
    "created_at": "2026-04-01T10:00:00Z",
    "updated_at": "2026-04-01T10:00:00Z"
  }
]`)

	writeMessengerLinks(t, home, "beta", `[
  {
    "profile_id": "beta",
    "messenger_name": "tg-bot",
    "messenger_scope": "global",
    "enabled": true,
    "mappings": {
      "-100200": "beta=route"
    },
    "created_at": "2026-04-02T11:00:00Z",
    "updated_at": "2026-04-02T11:00:00Z"
  }
]`)
}

func TestLinkList_DefaultShowsAllLinks(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, stderr, err := runAPS(t, home, "adapter", "link", "list")
	if err != nil {
		t.Fatalf("adapter link list: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"alpha", "beta", "tg-bot", "slack-bot"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in default link list, got: %s", want, stdout)
		}
	}
	for _, header := range []string{"PROFILE", "DEVICE", "PERMISSIONS", "LINKED AT"} {
		if !strings.Contains(stdout, header) {
			t.Errorf("expected %q header in table output: %s", header, stdout)
		}
	}
}

func TestLinkList_FilterProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "link", "list", "--profile", "alpha")
	if err != nil {
		t.Fatalf("adapter link list --profile=alpha: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha rows in output: %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta rows under --profile=alpha: %s", stdout)
	}

	// Bogus profile → zero rows.
	stdout, _, err = runAPS(t, home, "adapter", "link", "list", "--profile", "does-not-exist")
	if err != nil {
		t.Fatalf("adapter link list --profile=bogus: %v", err)
	}
	for _, name := range []string{"alpha", "beta"} {
		if strings.Contains(stdout, name) {
			t.Errorf("expected zero rows for bogus profile, found %q in: %s", name, stdout)
		}
	}
}

func TestLinkList_FilterMessenger(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "link", "list", "--messenger", "tg-bot")
	if err != nil {
		t.Fatalf("adapter link list --messenger=tg-bot: %v", err)
	}
	if !strings.Contains(stdout, "tg-bot") {
		t.Errorf("expected tg-bot under --messenger=tg-bot: %s", stdout)
	}
	if strings.Contains(stdout, "slack-bot") {
		t.Errorf("did not expect slack-bot under --messenger=tg-bot: %s", stdout)
	}
	// alpha + beta both have tg-bot links → two rows.
	if !strings.Contains(stdout, "alpha") || !strings.Contains(stdout, "beta") {
		t.Errorf("expected both profiles linked to tg-bot, got: %s", stdout)
	}
}

func TestLinkList_FilterProfileAndMessenger(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "link", "list",
		"--profile", "alpha", "--messenger", "slack-bot")
	if err != nil {
		t.Fatalf("adapter link list --profile=alpha --messenger=slack-bot: %v", err)
	}
	if !strings.Contains(stdout, "alpha") || !strings.Contains(stdout, "slack-bot") {
		t.Errorf("expected alpha+slack-bot row in output: %s", stdout)
	}
	if strings.Contains(stdout, "tg-bot") {
		t.Errorf("did not expect tg-bot under --messenger=slack-bot: %s", stdout)
	}
}

// TestLinkList_PermissionsSummary asserts the Permissions column carries
// the state + mapping count + default-action summary the row builder
// emits for each link.
func TestLinkList_PermissionsSummary(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "link", "list", "--profile", "alpha")
	if err != nil {
		t.Fatalf("adapter link list --profile=alpha: %v", err)
	}
	// alpha+tg-bot link: enabled, routes=2, default=route
	if !strings.Contains(stdout, "enabled,routes=2,default=route") {
		t.Errorf("expected enabled,routes=2,default=route in alpha summary: %s", stdout)
	}
	// alpha+slack-bot link: disabled, routes=0, default=-
	if !strings.Contains(stdout, "disabled,routes=0,default=-") {
		t.Errorf("expected disabled,routes=0,default=- in alpha summary: %s", stdout)
	}
}

func TestLinkList_JSONFormat(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterLinks(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "link", "list", "--format", "json")
	if err != nil {
		t.Fatalf("adapter link list --format=json: %v", err)
	}
	for _, key := range []string{`"profile"`, `"device"`, `"permissions"`, `"linked_at"`} {
		if !strings.Contains(stdout, key) {
			t.Errorf("expected JSON key %s in output: %s", key, stdout)
		}
	}
}
