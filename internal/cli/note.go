// Package cli — note flag plumbing.
//
// This file re-exports the shared --note|-n helpers from the clinote
// subpackage so the top-level cli package can call them without
// importing clinote at every call site. Subpackages (identity, session,
// adapter, …) import clinote directly.
package cli

import (
	"context"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
)

// AddNoteFlag registers --note|-n on cmd with a uniform usage string
// (T-1291). Idempotent.
func AddNoteFlag(cmd *cobra.Command) { clinote.AddFlag(cmd) }

// NoteFromCmd returns the --note value (empty string when unset or the
// flag is not declared).
func NoteFromCmd(cmd *cobra.Command) string { return clinote.FromCmd(cmd) }

// WithNote returns ctx with note attached under policy.ContextAttrsKey
// so kit's policy engine can read it as `context.note` from CEL. If
// note is empty, the original ctx is returned unchanged.
func WithNote(ctx context.Context, note string) context.Context {
	return clinote.WithContext(ctx, note)
}

// WithContextAttrs merges attrs into the policy context map on ctx.
func WithContextAttrs(ctx context.Context, attrs map[string]any) context.Context {
	return clinote.WithContextAttrs(ctx, attrs)
}
