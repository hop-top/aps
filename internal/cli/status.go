// Package cli — `aps status` reserved subcommand.
//
// T-0681 — kit 0.4's signature validator (Layer-A H6, MissingStatusSubcommand
// bucket) requires the root tool to mount a subcommand literally named
// `status`. The validator check is presence-by-name only
// (kit/console/cli/cli.go:checkReservedStatus); shape, flags, and output
// schema are left to the adopter.
//
// Scope is intentionally minimal: render a small key-value snapshot of
// the active profile, active workspace, build version, and event-bus
// connection state. Read-only, idempotent, no network calls. The
// command's purpose is twofold:
//
//  1. Satisfy kit's reserved-name contract so MissingStatusSubcommand
//     drops to zero once Config.DisableValidate flips to false in
//     T-0662.
//  2. Give users + agents a non-trivial "is aps wired up" probe that
//     does not require already knowing about `aps profile status` or
//     `aps version`.
//
// Anything richer (per-subsystem health, deep config introspection,
// kit's full StatusOutput schema) is deferred. We deliberately do NOT
// adopt kit's built-in `cli.WithStatus(...)` constructor here because
// it pulls in the full six-section StatusOutput surface (profile, env,
// workspace, auth, effective-config, kit-annotations) plus a
// --show-sensitive flag — overkill for the current scope, and the
// adopter contract explicitly permits a hand-rolled `Use: "status"`.
package cli

import (
	"io"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/version"
)

// statusRow is a single key/value pair in the status report. kit/output
// derives table headers from the struct tags; JSON/YAML use the json tag
// so structured callers get a stable shape.
type statusRow struct {
	Key   string `json:"key" yaml:"key" table:"Key,priority=1"`
	Value string `json:"value" yaml:"value" table:"Value,priority=2"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show aps configuration and runtime status",
	Long: `Show a snapshot of the current aps configuration and runtime state.

Reports the active profile, active workspace, build version, and event-bus
connection state. Read-only: no network calls, no state mutation. Intended
as a quick "is aps wired up" probe for users and agents.

Output respects the --format global (table|json|yaml). Empty profile or
workspace values render as "(none)" to distinguish unset from blank.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runStatus(cmd.OutOrStdout())
	},
}

func runStatus(w io.Writer) error {
	rows := []statusRow{
		{Key: "profile", Value: orNone(globals.Profile())},
		{Key: "workspace", Value: orNone(globals.Workspace())},
		{Key: "version", Value: version.Short()},
		{Key: "bus", Value: busState()},
	}
	return listing.RenderList(w, globals.Format(), rows)
}

// orNone renders empty globals as a literal "(none)" so a missing
// profile/workspace is unambiguous in both the table and the JSON/YAML
// emissions. We don't fall back to defaults here — `aps status` reports
// what the user/env actually set, not what the resolver would pick.
func orNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

// busState reports the event-bus wiring as one of "configured" (token
// present + adapter constructed) or "disabled" (no token in env). The
// init path in bus.go creates eventBus and netAdapter together when
// the token is set, so there is no observable "local bus only" state.
// The network adapter retries on its own, so "configured" here means
// "wired", not "currently linked to the hub" — a real liveness probe
// would require a synchronous round-trip we don't want to add to a
// read-only command.
func busState() string {
	if netAdapter != nil {
		return "configured"
	}
	return "disabled"
}

func init() {
	// T-0681 — kit signature annotations. Status is a pure read with no
	// state mutation; multiple invocations return identical output up to
	// runtime drift (build metadata is constant for a given binary;
	// profile/workspace/bus reflect process state that does not change
	// mid-run).
	kitcli.SetSideEffect(statusCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(statusCmd, kitcli.IdempotencyYes)
	rootCmd.AddCommand(statusCmd)
}
