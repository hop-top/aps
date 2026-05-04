package adapter_e2e

import (
	"strings"
	"testing"
)

// seedFilterAdapters populates a home with adapters that exercise every
// filter dimension exposed by `aps adapter list` (T-0435).
//
//   - tg-global  : type=messenger, scope=global,  workspace=global
//   - http-proto : type=protocol,  scope=global,  workspace=global
//   - tg-alpha   : type=messenger, scope=profile, workspace=alpha
func seedFilterAdapters(t *testing.T, home string) {
	t.Helper()

	writeGlobalAdapter(t, home, "tg-global", `api_version: adapter.aps.dev/v1
kind: Adapter
name: tg-global
type: messenger
strategy: subprocess
config:
  platform: telegram
`)

	writeGlobalAdapter(t, home, "http-proto", `api_version: adapter.aps.dev/v1
kind: Adapter
name: http-proto
type: protocol
strategy: builtin
`)

	writeProfileAdapter(t, home, "alpha", "tg-alpha", `api_version: adapter.aps.dev/v1
kind: Adapter
name: tg-alpha
type: messenger
strategy: subprocess
config:
  platform: telegram
linked_to:
  - alpha
`)
}

func TestAdapterList_ShowsAllByDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterAdapters(t, home)

	stdout, stderr, err := runAPS(t, home, "adapter", "list")
	if err != nil {
		t.Fatalf("adapter list: %v\nstderr: %s", err, stderr)
	}
	for _, name := range []string{"tg-global", "http-proto", "tg-alpha"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %q in default list, got: %s", name, stdout)
		}
	}
	// Header sanity — kit/output table renderer surfaces the column tags.
	for _, header := range []string{"NAME", "TYPE", "STATUS", "WORKSPACE"} {
		if !strings.Contains(stdout, header) {
			t.Errorf("expected %q header in table output: %s", header, stdout)
		}
	}
}

func TestAdapterList_FilterType(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterAdapters(t, home)

	stdout, stderr, err := runAPS(t, home, "adapter", "list", "--type", "messenger")
	if err != nil {
		t.Fatalf("adapter list --type: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "tg-global") || !strings.Contains(stdout, "tg-alpha") {
		t.Errorf("expected messenger adapters in output: %s", stdout)
	}
	if strings.Contains(stdout, "http-proto") {
		t.Errorf("did not expect protocol adapter under --type=messenger: %s", stdout)
	}

	// Bogus type → empty rows (kit/output may print headers + hint).
	stdout, _, err = runAPS(t, home, "adapter", "list", "--type", "does-not-exist")
	if err != nil {
		t.Fatalf("adapter list --type bogus: %v", err)
	}
	for _, name := range []string{"tg-global", "http-proto", "tg-alpha"} {
		if strings.Contains(stdout, name) {
			t.Errorf("expected zero rows for bogus type, found %q in: %s", name, stdout)
		}
	}
}

func TestAdapterList_FilterStatus(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterAdapters(t, home)

	// Newly seeded adapters have no runtime state; default state is "stopped".
	stdout, _, err := runAPS(t, home, "adapter", "list", "--status", "stopped")
	if err != nil {
		t.Fatalf("adapter list --status=stopped: %v", err)
	}
	for _, name := range []string{"tg-global", "http-proto", "tg-alpha"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %q under --status=stopped: %s", name, stdout)
		}
	}

	// No adapters in "running" state at test time.
	stdout, _, err = runAPS(t, home, "adapter", "list", "--status", "running")
	if err != nil {
		t.Fatalf("adapter list --status=running: %v", err)
	}
	for _, name := range []string{"tg-global", "http-proto", "tg-alpha"} {
		if strings.Contains(stdout, name) {
			t.Errorf("did not expect %q under --status=running: %s", name, stdout)
		}
	}
}

func TestAdapterList_FilterWorkspace(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterAdapters(t, home)

	// scope=global adapters surface under workspace=global.
	stdout, _, err := runAPS(t, home, "adapter", "list", "--workspace", "global")
	if err != nil {
		t.Fatalf("adapter list --workspace=global: %v", err)
	}
	if !strings.Contains(stdout, "tg-global") || !strings.Contains(stdout, "http-proto") {
		t.Errorf("expected global adapters under --workspace=global: %s", stdout)
	}
	if strings.Contains(stdout, "tg-alpha") {
		t.Errorf("did not expect profile-scoped adapter under --workspace=global: %s", stdout)
	}

	// scope=profile adapters surface under workspace=<profile-id>.
	stdout, _, err = runAPS(t, home, "adapter", "list", "--workspace", "alpha")
	if err != nil {
		t.Fatalf("adapter list --workspace=alpha: %v", err)
	}
	if !strings.Contains(stdout, "tg-alpha") {
		t.Errorf("expected tg-alpha under --workspace=alpha: %s", stdout)
	}
	if strings.Contains(stdout, "tg-global") || strings.Contains(stdout, "http-proto") {
		t.Errorf("did not expect global adapters under --workspace=alpha: %s", stdout)
	}
}

// TestAdapterList_JSONFormat asserts --format=json emits the row schema
// (fields reachable via the json tags on adapterSummaryRow).
func TestAdapterList_JSONFormat(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterAdapters(t, home)

	stdout, _, err := runAPS(t, home, "adapter", "list", "--format", "json")
	if err != nil {
		t.Fatalf("adapter list --format=json: %v", err)
	}
	// JSON output must carry every json: tag declared on adapterSummaryRow.
	for _, key := range []string{`"name"`, `"type"`, `"status"`, `"workspace"`, `"paired_devices"`, `"last_seen_at"`} {
		if !strings.Contains(stdout, key) {
			t.Errorf("expected JSON key %s in output: %s", key, stdout)
		}
	}
}
