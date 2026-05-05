// Package clinote centralises the --note|-n flag pattern used by every
// state-changing aps subcommand (T-1291). Subpackages (identity,
// session, workspace, …) import this; the top-level cli package
// re-exports the same helpers via thin wrappers in note.go for
// convenience.
//
// Contract:
//
//   - AddFlag(cmd) registers --note|-n on the cobra command.
//   - FromCmd(cmd) extracts the value at run time.
//   - WithContext(ctx, note) stuffs it into context.Context via
//     policy.ContextAttrsKey BEFORE the SessionManager / WorkspaceContext
//     / domain.Service / core mutator call so that kit's policy engine
//     can read `context.note` from CEL during pre_persisted /
//     pre_transitioned veto evaluation (T-1292 will exploit this).
package clinote

import (
	"context"

	"github.com/spf13/cobra"

	"hop.top/kit/go/runtime/policy"
)

const flagName = "note"

// AddFlag registers --note|-n on cmd. Idempotent.
//
// If -n is already taken by another flag on this command (a few legacy
// commands use it for --dry-run / --limit), the long form --note is
// registered without a shorthand. The long form is the load-bearing
// contract; -n is a convenience.
func AddFlag(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup(flagName) != nil {
		return
	}
	usage := "Audit note (recorded against the bus event payload and exposed to policy engine via context.note)"
	if cmd.Flags().ShorthandLookup("n") != nil {
		cmd.Flags().String(flagName, "", usage)
		return
	}
	cmd.Flags().StringP(flagName, "n", "", usage)
}

// FromCmd returns the --note value (empty when unset).
func FromCmd(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	v, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return ""
	}
	return v
}

// WithContext returns ctx with note attached under
// policy.ContextAttrsKey so kit's policy engine can read it as
// `context.note` from CEL. Empty note returns ctx unchanged.
func WithContext(ctx context.Context, note string) context.Context {
	if note == "" {
		return ctx
	}
	return WithContextAttrs(ctx, map[string]any{"note": note})
}

// WithContextAttrs merges attrs into the policy context map on ctx.
// Returns ctx unchanged when attrs is empty.
func WithContextAttrs(ctx context.Context, attrs map[string]any) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	merged := make(map[string]any, len(attrs)+1)
	if existing, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any); ok {
		for k, v := range existing {
			merged[k] = v
		}
	}
	for k, v := range attrs {
		merged[k] = v
	}
	return context.WithValue(ctx, policy.ContextAttrsKey, merged)
}
