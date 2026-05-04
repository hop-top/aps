package messenger_e2e

import (
	"strings"
	"testing"
)

// seedFilterMessengers populates a home with messenger adapters that
// exercise every filter dimension exposed by `aps adapter messenger
// list` (T-0436).
//
//   - tg-global    : platform=telegram, scope=global,  no profile
//   - slack-bot    : platform=slack,    scope=global,  no profile
//   - tg-alpha     : platform=telegram, scope=profile, profile=alpha
func seedFilterMessengers(t *testing.T, home string) {
	t.Helper()

	writeGlobalAdapter(t, home, "tg-global", `api_version: adapter.aps.dev/v1
kind: Adapter
name: tg-global
type: messenger
strategy: subprocess
config:
  platform: telegram
`)

	writeGlobalAdapter(t, home, "slack-bot", `api_version: adapter.aps.dev/v1
kind: Adapter
name: slack-bot
type: messenger
strategy: subprocess
config:
  platform: slack
`)

	writeProfileAdapter(t, home, "alpha", "tg-alpha", `api_version: adapter.aps.dev/v1
kind: Adapter
name: tg-alpha
type: messenger
strategy: subprocess
config:
  platform: telegram
`)
}

func TestMessengerList_DefaultShowsAllMessengers(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterMessengers(t, home)

	stdout, stderr, err := runAPS(t, home, "adapter", "messenger", "list")
	if err != nil {
		t.Fatalf("messenger list: %v\nstderr: %s", err, stderr)
	}
	for _, name := range []string{"tg-global", "slack-bot", "tg-alpha"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %q in default list, got: %s", name, stdout)
		}
	}
	for _, header := range []string{"NAME", "PLATFORM", "STATUS", "PROFILE"} {
		if !strings.Contains(stdout, header) {
			t.Errorf("expected %q header in table output: %s", header, stdout)
		}
	}
}

func TestMessengerList_FilterPlatform(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterMessengers(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "messenger", "list", "--platform", "telegram")
	if err != nil {
		t.Fatalf("messenger list --platform=telegram: %v", err)
	}
	if !strings.Contains(stdout, "tg-global") || !strings.Contains(stdout, "tg-alpha") {
		t.Errorf("expected telegram messengers in output: %s", stdout)
	}
	if strings.Contains(stdout, "slack-bot") {
		t.Errorf("did not expect slack messenger under --platform=telegram: %s", stdout)
	}

	stdout, _, err = runAPS(t, home, "adapter", "messenger", "list", "--platform", "slack")
	if err != nil {
		t.Fatalf("messenger list --platform=slack: %v", err)
	}
	if !strings.Contains(stdout, "slack-bot") {
		t.Errorf("expected slack-bot under --platform=slack: %s", stdout)
	}
	for _, name := range []string{"tg-global", "tg-alpha"} {
		if strings.Contains(stdout, name) {
			t.Errorf("did not expect %q under --platform=slack: %s", name, stdout)
		}
	}
}

func TestMessengerList_FilterStatus(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterMessengers(t, home)

	// All messengers default to "stopped" without a runtime in scope.
	stdout, _, err := runAPS(t, home, "adapter", "messenger", "list", "--status", "running")
	if err != nil {
		t.Fatalf("messenger list --status=running: %v", err)
	}
	for _, name := range []string{"tg-global", "slack-bot", "tg-alpha"} {
		if strings.Contains(stdout, name) {
			t.Errorf("did not expect %q under --status=running: %s", name, stdout)
		}
	}

	stdout, _, err = runAPS(t, home, "adapter", "messenger", "list", "--status", "stopped")
	if err != nil {
		t.Fatalf("messenger list --status=stopped: %v", err)
	}
	for _, name := range []string{"tg-global", "slack-bot", "tg-alpha"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %q under --status=stopped: %s", name, stdout)
		}
	}
}

// TestMessengerList_ProfileInheritedFromGlobal asserts the messenger
// list reads the profile filter from the global --profile flag (T-0376),
// not from a per-command flag.
func TestMessengerList_ProfileInheritedFromGlobal(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterMessengers(t, home)

	stdout, _, err := runAPS(t, home, "--profile", "alpha", "adapter", "messenger", "list")
	if err != nil {
		t.Fatalf("messenger list --profile=alpha: %v", err)
	}
	if !strings.Contains(stdout, "tg-alpha") {
		t.Errorf("expected tg-alpha under --profile=alpha: %s", stdout)
	}
	for _, name := range []string{"tg-global", "slack-bot"} {
		if strings.Contains(stdout, name) {
			t.Errorf("did not expect %q under --profile=alpha: %s", name, stdout)
		}
	}
}

// TestMessengerList_GlobalProfileAlsoWorksAfterSubcommand asserts that
// because --profile is registered as a tool-level global (T-0376),
// cobra accepts it at any position — the user can drop it after the
// subcommand and still get profile-scoped results.
func TestMessengerList_GlobalProfileAlsoWorksAfterSubcommand(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterMessengers(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "messenger", "list", "--profile", "alpha")
	if err != nil {
		t.Fatalf("messenger list ... --profile=alpha: %v", err)
	}
	if !strings.Contains(stdout, "tg-alpha") {
		t.Errorf("expected tg-alpha under --profile=alpha (post-sub): %s", stdout)
	}
	for _, name := range []string{"tg-global", "slack-bot"} {
		if strings.Contains(stdout, name) {
			t.Errorf("did not expect %q under --profile=alpha: %s", name, stdout)
		}
	}
}
