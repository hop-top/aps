package cli

// T-0363 — drop top-level `aps messenger`; keep as `aps adapter messenger`.
// Local `--json` on `aps adapter messenger list` removed (kit's --format
// from T-0345 supersedes it).

import "testing"

// TestMessenger_NotRegisteredOnRoot asserts the top-level `aps messenger`
// command no longer exists. The `messenger_alias.go` command lives under
// `aps adapter` only.
func TestMessenger_NotRegisteredOnRoot(t *testing.T) {
	if cmd := findSubcommand(rootCmd, "messenger"); cmd != nil {
		t.Errorf("aps messenger still registered at root; expected drop (T-0363)")
	}
}

// TestMessenger_RegisteredUnderAdapter asserts the messenger alias is
// available as `aps adapter messenger`.
func TestMessenger_RegisteredUnderAdapter(t *testing.T) {
	if cmd := findSubcommand(rootCmd, "adapter", "messenger"); cmd == nil {
		t.Fatal("aps adapter messenger not registered (T-0363)")
	}
}

// TestMessenger_AdapterListHasMessengerSubcommands asserts the messenger
// subcommand exposes the expected operations under adapter.
func TestMessenger_AdapterListHasMessengerSubcommands(t *testing.T) {
	for _, sub := range []string{"list", "link", "unlink", "channels", "links"} {
		if cmd := findSubcommand(rootCmd, "adapter", "messenger", sub); cmd == nil {
			t.Errorf("aps adapter messenger %s not registered", sub)
		}
	}
}

// TestMessenger_ListNoLocalJSONFlag asserts the per-command --json flag is
// removed from `aps adapter messenger list` (kit's --format covers it).
func TestMessenger_ListNoLocalJSONFlag(t *testing.T) {
	cmd := findSubcommand(rootCmd, "adapter", "messenger", "list")
	if cmd == nil {
		t.Fatal("aps adapter messenger list not registered")
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		t.Errorf("aps adapter messenger list still has --json flag; expected removal (T-0363)")
	}
}
