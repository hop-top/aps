package cli

import (
	"errors"
	"strings"
	"testing"

	"hop.top/kit/go/console/output"
)

// envelopeOf unwraps err into the structured *output.Error the process
// renders, failing the test when err is not envelope-shaped.
func envelopeOf(t *testing.T, err error) *output.Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ce interface{ AsCLIError() *output.Error }
	if !errors.As(err, &ce) {
		t.Fatalf("error %v is not a structured envelope", err)
	}
	e := ce.AsCLIError()
	if e == nil {
		t.Fatal("envelope is nil")
	}
	return e
}

// TestUnknownSubcommandIsUsageError pins the corrective contract for a
// mistyped subcommand on a group node. Stock cobra renders help and
// exits 0 here, which tells an agent branching on $? that the
// invocation succeeded; aps must instead produce a USAGE envelope
// (exit 2) that echoes the rejected token and points at the group's
// own --help.
func TestUnknownSubcommandIsUsageError(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{"top-level group", []string{"profile", "frobnicate"}, "aps profile"},
		{"config group", []string{"config", "bogus"}, "aps config"},
		{"adapter group", []string{"adapter", "nope"}, "aps adapter"},
		{"nested group", []string{"profile", "capability", "nope"}, "aps profile capability"},
		{"token after flags", []string{"profile", "--format", "json", "frobnicate"}, "aps profile"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rejectUnknownSubcommand(rootCmd, tc.args)
			env := envelopeOf(t, err)

			if env.Code != output.CodeUsage || env.ExitCode != 2 {
				t.Fatalf("envelope = %+v, want USAGE/2", env)
			}
			token := tc.args[len(tc.args)-1]
			if !strings.Contains(env.Message, token) {
				t.Errorf("message %q must echo the rejected token %q", env.Message, token)
			}
			if !strings.Contains(env.Message, tc.wantPath+" --help") {
				t.Errorf("message %q must point at %q --help", env.Message, tc.wantPath)
			}
			if env.SuggestedFix == "" {
				t.Error("envelope lacks a suggested fix")
			}
		})
	}
}

// TestWellFormedInvocationsAreNotRejected is the other half of the
// contract: the pre-dispatch check must stay invisible to every valid
// invocation. A false positive here would break working commands, so
// each shape that legitimately reaches cobra is pinned.
func TestWellFormedInvocationsAreNotRejected(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"bare group renders help", []string{"profile"}},
		{"bare nested group", []string{"profile", "capability"}},
		{"valid leaf", []string{"profile", "list"}},
		{"valid leaf with flags", []string{"profile", "list", "--format", "json"}},
		{"group with help flag", []string{"profile", "--help"}},
		{"top-level leaf", []string{"status"}},
		{"no args at all", nil},
		{"root positional dispatches to profile runner", []string{"some-profile-id"}},
		{"root positional with command", []string{"some-profile-id", "ls"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := rejectUnknownSubcommand(rootCmd, tc.args); err != nil {
				t.Fatalf("well-formed args %v rejected: %v", tc.args, err)
			}
		})
	}
}

// TestPassthroughArgsAreNotInspected pins that anything after `--`
// belongs to the invoked command. Treating a passthrough token as a
// subcommand name would break `aps <group> -- <argv>` forwarding.
func TestPassthroughArgsAreNotInspected(t *testing.T) {
	if err := rejectUnknownSubcommand(rootCmd, []string{"profile", "--", "frobnicate"}); err != nil {
		t.Fatalf("passthrough token treated as subcommand: %v", err)
	}
}

// TestScanArgsForFormat pins the pre-parse --format resolution the
// envelope renderer depends on. The check answers before cobra parses
// flags, so the bound flag value is unavailable and argv must be read
// directly.
func TestScanArgsForFormat(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unset", []string{"profile", "frobnicate"}, ""},
		{"separate value", []string{"profile", "--format", "json", "x"}, "json"},
		{"inline value", []string{"profile", "--format=yaml", "x"}, "yaml"},
		{"shorthand", []string{"profile", "-f", "json"}, "json"},
		{"dangling flag", []string{"profile", "--format"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scanArgsForFormat(tc.args); got != tc.want {
				t.Fatalf("scanArgsForFormat(%v) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

// TestUnknownSubcommandLeavesTreeUnmodified guards the design
// constraint that made the pre-dispatch check necessary: kit gates
// MissingLong, kit/top-level-verb, and MaxTopLevelVerbs on
// cmd.Runnable(), so any fix that flips group nodes runnable (or swaps
// their Args validator) fails aps's strict gates. The check must be
// purely read-only over the command tree.
func TestUnknownSubcommandLeavesTreeUnmodified(t *testing.T) {
	type shape struct {
		runnable bool
		hasArgs  bool
	}
	snapshot := func() map[string]shape {
		out := map[string]shape{}
		for _, c := range rootCmd.Commands() {
			out[c.Name()] = shape{runnable: c.Runnable(), hasArgs: c.Args != nil}
		}
		return out
	}

	before := snapshot()
	_ = rejectUnknownSubcommand(rootCmd, []string{"profile", "frobnicate"})
	_ = rejectUnknownSubcommand(rootCmd, []string{"profile", "list"})
	after := snapshot()

	for name, b := range before {
		a := after[name]
		if a != b {
			t.Errorf("command %q shape changed %+v -> %+v: the check must not mutate the tree",
				name, b, a)
		}
	}

	// The tree must still satisfy the gates kit re-runs at boot.
	// ValidateSignature returns a *SignatureReport, never an error:
	// an empty report is success, so HasViolations is the predicate.
	if err := root.Validate(); err != nil {
		t.Fatalf("tree fails kit validation after unknown-command check: %v", err)
	}
	if report := root.ValidateSignature(); report.HasViolations() {
		t.Fatalf("tree fails kit signature validation: %+v", report.Violations)
	}
}
