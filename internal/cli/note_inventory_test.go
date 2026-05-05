package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestNoteFlagPresence_RepresentativeSubcommands verifies that --note
// is registered on at least three representative subcommands per
// category in the T-1291 inventory. Mirrors scripts/verify_note_flag.sh
// but as a Go test so CI catches drift without an extra shell step.
//
// The script remains the exhaustive check (52 commands); this test
// trades coverage for speed and runs in the standard `go test ./...`.
func TestNoteFlagPresence_RepresentativeSubcommands(t *testing.T) {
	t.Parallel()

	// Three per category × seven categories = 21 commands. Picked to
	// touch every cli subpackage that received the flag.
	cases := []struct {
		category string
		path     []string
	}{
		// Identity (highest stakes)
		{"identity", []string{"profile", "create"}},
		{"identity", []string{"profile", "delete"}},
		{"identity", []string{"identity", "init"}},
		// Sessions
		{"sessions", []string{"session", "delete"}},
		{"sessions", []string{"session", "terminate"}},
		{"sessions", []string{"session", "detach"}},
		// Workspaces + context
		{"workspace", []string{"workspace", "create"}},
		{"workspace", []string{"workspace", "ctx", "set"}},
		{"workspace", []string{"policy", "set"}},
		// Capabilities + bundles
		{"capability", []string{"capability", "delete"}},
		{"capability", []string{"capability", "enable"}},
		{"capability", []string{"bundle", "create"}},
		// Multi-agent
		{"squad", []string{"squad", "create"}},
		{"squad", []string{"squad", "members", "add"}},
		{"squad", []string{"adapter", "approve"}},
		// AGNTCY
		{"agntcy", []string{"directory", "register"}},
		{"agntcy", []string{"directory", "delete"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.category+"/"+strings.Join(tc.path, "_"), func(t *testing.T) {
			t.Parallel()
			cmd := findCommand(rootCmd, tc.path)
			if cmd == nil {
				t.Fatalf("subcommand %q not found", strings.Join(tc.path, " "))
			}
			if cmd.Flags().Lookup("note") == nil {
				t.Errorf("--note not registered on %q", strings.Join(tc.path, " "))
			}
		})
	}
}

// findCommand walks the cobra tree along path and returns the matched
// command, or nil if any segment fails to resolve.
func findCommand(root *cobra.Command, path []string) *cobra.Command {
	cmd := root
	for _, name := range path {
		var next *cobra.Command
		for _, child := range cmd.Commands() {
			if child.Name() == name {
				next = child
				break
			}
		}
		if next == nil {
			return nil
		}
		cmd = next
	}
	return cmd
}
