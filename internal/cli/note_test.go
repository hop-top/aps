package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"

	"hop.top/kit/go/runtime/policy"
)

// TestAddNoteFlag_RegistersLongAndShort guards the cross-tool contract:
// every state-changing aps subcommand exposes --note|-n with the same
// usage shape (T-1291). If this test breaks, every command that calls
// AddNoteFlag is also broken.
func TestAddNoteFlag_RegistersLongAndShort(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "x"}
	AddNoteFlag(cmd)

	long := cmd.Flags().Lookup("note")
	if long == nil {
		t.Fatal("--note flag not registered")
	}
	if long.Shorthand != "n" {
		t.Errorf("shorthand = %q, want %q", long.Shorthand, "n")
	}
	if long.DefValue != "" {
		t.Errorf("default = %q, want empty", long.DefValue)
	}
}

// TestAddNoteFlag_Idempotent guards against double-registration when a
// subcommand transitively re-uses the helper. cobra panics on duplicate
// flag names, which is the error we're avoiding.
func TestAddNoteFlag_Idempotent(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "x"}
	AddNoteFlag(cmd)
	AddNoteFlag(cmd) // must not panic
}

// TestNoteFromCmd_RoundTrip verifies the runtime extraction path used by
// every state-changing subcommand to populate ctx + event payload.
func TestNoteFromCmd_RoundTrip(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "x"}
	AddNoteFlag(cmd)
	if err := cmd.ParseFlags([]string{"--note", "explained why"}); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := NoteFromCmd(cmd); got != "explained why" {
		t.Errorf("note = %q, want %q", got, "explained why")
	}
}

// TestNoteFromCmd_ShorthandRoundTrip verifies -n carries the same value
// as --note.
func TestNoteFromCmd_ShorthandRoundTrip(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "x"}
	AddNoteFlag(cmd)
	if err := cmd.ParseFlags([]string{"-n", "shorthand"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := NoteFromCmd(cmd); got != "shorthand" {
		t.Errorf("note = %q, want %q", got, "shorthand")
	}
}

// TestNoteFromCmd_MissingFlag returns empty (not an error). Some
// commands that delegate to a shared helper may not declare the flag
// (e.g. session list) and the helper must tolerate it.
func TestNoteFromCmd_MissingFlag(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "x"}
	if got := NoteFromCmd(cmd); got != "" {
		t.Errorf("note = %q, want empty", got)
	}
}

// TestWithNote_AttachesToContextAttrs verifies the round-trip used by
// the kit policy engine: WithNote(ctx, "x") → ctx.Value(ContextAttrsKey)["note"]
// must be readable as the same string. This is the load-bearing
// contract for T-1292's CEL-based policy enforcement.
func TestWithNote_AttachesToContextAttrs(t *testing.T) {
	t.Parallel()

	ctx := WithNote(context.Background(), "audit reason")
	attrs, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	if !ok {
		t.Fatalf("ContextAttrsKey value type = %T, want map[string]any", ctx.Value(policy.ContextAttrsKey))
	}
	if attrs["note"] != "audit reason" {
		t.Errorf("attrs[note] = %v, want %q", attrs["note"], "audit reason")
	}
}

// TestWithNote_EmptyNoteSkipsAttribution verifies we don't pollute ctx
// with an empty-string note, which would shadow a meaningful note set
// by an outer caller (defensive — no current call site does this, but
// the contract is "absent" not "present-but-empty").
func TestWithNote_EmptyNoteSkipsAttribution(t *testing.T) {
	t.Parallel()

	ctx := WithNote(context.Background(), "")
	if ctx.Value(policy.ContextAttrsKey) != nil {
		t.Errorf("empty note still attached attrs to ctx")
	}
}

// TestWithNote_PreservesPreExistingAttrs verifies that callers who have
// already stuffed attrs into ctx (e.g. request_attrs from an upstream
// adapter) don't lose them when we merge in --note.
func TestWithNote_PreservesPreExistingAttrs(t *testing.T) {
	t.Parallel()

	base := context.WithValue(context.Background(), policy.ContextAttrsKey, map[string]any{
		"request_id": "req-1",
	})
	ctx := WithNote(base, "audit")
	attrs, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	if !ok {
		t.Fatalf("ContextAttrsKey type = %T", ctx.Value(policy.ContextAttrsKey))
	}
	if attrs["request_id"] != "req-1" {
		t.Errorf("request_id lost: attrs = %v", attrs)
	}
	if attrs["note"] != "audit" {
		t.Errorf("note lost: attrs = %v", attrs)
	}
}
