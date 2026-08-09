package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
)

// newFormatProbe builds a throwaway command carrying a --format flag,
// standing in for any leaf that inherits the root's persistent flag.
func newFormatProbe(t *testing.T, value string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "probe"}
	cmd.Flags().String("format", "table", "output format")
	if value != "" {
		if err := cmd.Flags().Set("format", value); err != nil {
			t.Fatalf("set --format=%q: %v", value, err)
		}
	}
	return cmd
}

// TestUndeclaredFormatIsUsageError pins Factor 11: a bad flag VALUE is
// a usage error, not a generic failure.
//
// kit's dispatch returns a plain error for a format outside the
// registry, which its middleware then wraps GENERIC/exit 1 — the same
// class as an internal failure. An agent could not tell "I asked for a
// format you do not serve" (fix the invocation) from "the command
// broke" (retry or escalate).
func TestUndeclaredFormatIsUsageError(t *testing.T) {
	err := validateFormatFlag(newFormatProbe(t, "xml"))

	env := envelopeOf(t, err)
	if env.Code != output.CodeUsage || env.ExitCode != 2 {
		t.Fatalf("envelope = %+v, want USAGE/2", env)
	}
	if !strings.Contains(env.Message, "xml") {
		t.Errorf("message %q must echo the rejected format", env.Message)
	}
	// The accepted set has to be named, or the agent is left guessing.
	if !strings.Contains(env.Message, "json") {
		t.Errorf("message %q must name the accepted formats", env.Message)
	}
	if env.SuggestedFix == "" {
		t.Error("envelope lacks a suggested fix")
	}
}

// TestDeclaredFormatsAccepted guards against the validator rejecting
// formats the registry actually serves.
func TestDeclaredFormatsAccepted(t *testing.T) {
	for _, f := range output.Default.Keys() {
		t.Run(f, func(t *testing.T) {
			if err := validateFormatFlag(newFormatProbe(t, f)); err != nil {
				t.Fatalf("declared format %q rejected: %v", f, err)
			}
		})
	}
}

// TestUnsetFormatIsAccepted pins that a command left at its default is
// untouched: only an explicitly set flag is validated, so per-leaf
// defaults keep working.
func TestUnsetFormatIsAccepted(t *testing.T) {
	if err := validateFormatFlag(newFormatProbe(t, "")); err != nil {
		t.Fatalf("unset --format rejected: %v", err)
	}
	// A command with no --format flag at all must not trip the check.
	if err := validateFormatFlag(&cobra.Command{Use: "bare"}); err != nil {
		t.Fatalf("command without --format rejected: %v", err)
	}
}
