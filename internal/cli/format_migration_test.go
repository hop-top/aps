package cli

// TestJSONFlag_Removed asserts that the per-command --json flag has been
// removed from commands migrated to kit/go/console/output.Render in T-0345.
// Output format is now controlled by the persistent --format flag bound
// on root by kit's cli.New (see TestRoot_FormatFlag).
//
// Migration list (T-0345 spec):
//   - aps version
//   - aps profile list
//   - aps action list
//
// Out of scope (still --json bool): profile trust, collab helpers,
// session inspect — tracked by T-0347 follow-ups.

import (
	"testing"

	"github.com/spf13/cobra"
)

func findSubcommand(parent *cobra.Command, names ...string) *cobra.Command {
	cur := parent
	for _, n := range names {
		var next *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == n {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

func TestJSONFlag_Removed_Version(t *testing.T) {
	cmd := findSubcommand(rootCmd, "version")
	if cmd == nil {
		t.Fatal("version command not registered on root")
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		t.Errorf("aps version still has --json flag; expected removal (T-0345)")
	}
}

func TestJSONFlag_Removed_ProfileList(t *testing.T) {
	cmd := findSubcommand(rootCmd, "profile", "list")
	if cmd == nil {
		t.Fatal("profile list command not registered")
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		t.Errorf("aps profile list still has --json flag; expected removal (T-0345)")
	}
}

func TestJSONFlag_Removed_ActionList(t *testing.T) {
	cmd := findSubcommand(rootCmd, "action", "list")
	if cmd == nil {
		t.Fatal("action list command not registered")
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		t.Errorf("aps action list still has --json flag; expected removal (T-0345)")
	}
}
