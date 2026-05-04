// Package listing centralizes the rendering and filtering pattern used
// by every `aps <noun> list` (and `aps <noun> <subnoun> list`) command.
// It exists so the 16 list commands aps ships do not each hand-roll
// table output, filter flag parsing, or json/yaml format dispatch.
//
// # Convention
//
// Every list subcommand:
//
//  1. Defines a `<noun>SummaryRow` struct whose fields carry
//     `table:"NAME[,priority=N]"`, `json:"name"`, `yaml:"name"` tags.
//     Higher priority columns survive narrow terminals; the kit/output
//     table renderer drops low-priority columns first when width is
//     constrained.
//
//  2. Builds rows from the loaded backing data (profiles, sessions,
//     adapters, etc.). The backing source typically has its own
//     "Full" loader (e.g. core.ListProfilesFull) so the cheap
//     ID-only path stays available for callers that don't need rows.
//
//  3. Applies a Predicate composed from CLI flags via the helpers in
//     this package: All, Any, Not, MatchString, MatchSlice, BoolFlag.
//     Filters that are unset (zero value) match every row.
//
//  4. Calls listing.RenderList(w, format, rows) — this dispatches to
//     kit/output.Render with the format string from root.Viper.
//     The list subcommand reads format via cmd.Flags().GetString or
//     viper directly; the global --format flag is wired by kit/cli.New.
//
// # Filter flag conventions
//
// Per ~/.ops/docs/cli-conventions-with-kit.md §3.3:
//
//   - --<dimension> <value>  set membership ("--capability webhooks")
//   - --has-<feature>        boolean ("--has-identity")
//   - --no-color, --quiet, --no-hints, --format are kit-owned globals
//   - Avoid --json (removed in T-0345 across the surface)
//
// # Why a wrapper around kit/output.Render
//
// One indirection point so the day kit ships a richer table renderer,
// or aps wants per-command default-format overrides, all 16 callsites
// pick it up via a single edit. RenderList today is a thin pass-through
// to output.Render; that is intentional.
package listing
