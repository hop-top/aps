package cli

// TestFlagAlignment_T0347 verifies the canonical flag naming applied
// per docs/plans/2026-04-29-kit-reorg-adoption/flag-audit.md:
//
//   - `-f` is no longer a `--force` shortname (reserved for future
//     `--format` short alias).
//   - `-v` is no longer a `--verbose` shortname (kit uses `-V`).
//   - `--dry-run` gets `-n` short (POSIX make convention).
//   - `aps upgrade` no longer owns its own `--quiet -q` (kit's
//     persistent `--quiet` covers it).

import (
	"testing"

	"github.com/spf13/cobra"
)

// shortLetter returns the empty string when the flag has no short form.
func shortLetter(cmd *cobra.Command, name string) string {
	if f := cmd.Flags().Lookup(name); f != nil {
		return f.Shorthand
	}
	if f := cmd.LocalFlags().Lookup(name); f != nil {
		return f.Shorthand
	}
	return ""
}

// hasFlag reports whether the command (or its inherited persistent flags)
// has a flag with the given long name.
func hasFlag(cmd *cobra.Command, name string) bool {
	return cmd.Flags().Lookup(name) != nil
}

func TestFlag_ProfileDelete_ForceNoShort(t *testing.T) {
	cmd := findSubcommand(rootCmd, "profile", "delete")
	if cmd == nil {
		t.Fatal("profile delete not registered")
	}
	if got := shortLetter(cmd, "force"); got != "" {
		t.Errorf("profile delete --force short = %q, want \"\"", got)
	}
}

func TestFlag_ProfileStatus_VerboseNoShort(t *testing.T) {
	cmd := findSubcommand(rootCmd, "profile", "status")
	if cmd == nil {
		t.Fatal("profile status not registered")
	}
	if got := shortLetter(cmd, "verbose"); got == "v" {
		t.Errorf("profile status --verbose still bound to -v; expected drop")
	}
}

func TestFlag_Upgrade_NoLocalQuiet(t *testing.T) {
	cmd := findSubcommand(rootCmd, "upgrade")
	if cmd == nil {
		t.Fatal("upgrade not registered")
	}
	// The kit persistent --quiet is inherited; assert there is no LOCAL
	// shadowing flag (Local-only lookup ignores persistent inherits).
	if f := cmd.LocalFlags().Lookup("quiet"); f != nil {
		t.Errorf("aps upgrade still owns local --quiet flag; expected use of kit persistent")
	}
	// Persistent --quiet should still be reachable.
	if !hasFlag(cmd, "quiet") {
		t.Error("expected kit persistent --quiet to be inherited on upgrade")
	}
}

func TestFlag_ActionRun_DryRunHasShortN(t *testing.T) {
	cmd := findSubcommand(rootCmd, "action", "run")
	if cmd == nil {
		t.Fatal("action run not registered")
	}
	if got := shortLetter(cmd, "dry-run"); got != "n" {
		t.Errorf("action run --dry-run short = %q, want \"n\"", got)
	}
}
