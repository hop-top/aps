# Changelog

All notable changes to `aps` are documented in this file.

## Unreleased

### Improvements — list-commands-uplift (track: `list-commands-uplift`)

All `aps <noun> list` (and `aps <noun> <subnoun> list`) commands now
emit rich tabular output via kit/output.Render with consistent filter
flag conventions. 16 commands migrated; 1 new shared helper package.

**New** — `internal/cli/listing/`: `RenderList[T]` generic dispatch
to kit/output.Render, `Predicate[T]` combinators (All, Any, Not,
Filter), CLI-flag-shaped helpers (MatchString, MatchSlice, BoolFlag).
24 tests; codifies the pattern in package doc.go.

**Migrated commands** (each now: rich row struct with `priority=N`
tags, json/yaml struct tags, filter flags below, render via
listing.RenderList):

  aps profile list           --capability, --role, --squad,
                             --workspace, --has-identity,
                             --has-secrets, --tone
  aps session list           --type (existing) + --status,
                             --profile, --workspace, --tier
  aps workspace list         --member, --owner, --archived
  aps squad list             --member, --role
  aps bundle list            --tag, --builtin, --user
  aps capability list        --tag, --builtin, --external,
                             --enabled-on
  aps capability patterns    same treatment as parent
  aps contact list           --org, --has-email
                             (existing --addressbook preserved)
  aps adapter list           --type, --status, --workspace
  aps adapter messenger list --platform, --status
  aps adapter links          --profile, --messenger
  aps a2a tasks list         --profile (global), --status
  aps action list            --type (sh|py|js)
  aps skill list             --profile (global), --source
  aps workspace conflicts list  --workspace (global), --unresolved
  aps workspace ctx list     --workspace (global), --key-prefix
  aps workspace policy list  (workspace stays positional;
                             rich row only)

**Removed flags** (canonicalization via shared listing helper):

- `aps skill list --verbose` — kit/output's table priority tags
  drop low-value columns on narrow terminals automatically; full
  fields available via `--format json|yaml`.
- `aps skill list --profile` (local) — superseded by the kit-owned
  global `--profile` (T-0376) bound on root.

**Backing data additions**:

- `core.ListProfilesFull() ([]Profile, error)` — loads each profile
  YAML once; cheap `ListProfiles() []string` retained for callers
  that only need IDs.
- `core/bundle.Bundle.Tags []string` (yaml-tagged; existing assets
  unaffected).
- `core/capability.Capability.Tags`, `BuiltinCapability.Tags`.
- `internal/cli/globals.Profile()` and `globals.Format()` accessors —
  let non-`internal/cli` subpackages read kit-owned globals without
  forming an import cycle.
- `aps skill` now wired to rootCmd (was previously defined but
  unreachable). Lives in PIPELINES group alongside a2a/acp/etc.

**Internal**: `core/skills.Registry.SourceLabel(...)` exported for
the cli row builder (was internal `getSourceLabel`).

Closes the §6 partial-compliance finding in
`~/.ops/reviews/aps-cli-review-2026-04-30.md` (kit/output not fully
adopted across the surface). Convention doc
`~/.ops/docs/cli-conventions-with-kit.md` §3.3 now cites listing.
RenderList as the canonical aps pattern.

### Breaking changes — cli-surface-refactor (track: `cli-surface-refactor`)

This wave consolidates the aps command surface from 30 top-level commands
to 26, drops asymmetric verb-noun-flat commands in favor of noun-verb
subtrees, and adds kit/cli help-section grouping. Aps is pre-release;
no deprecation aliases ship.

**Command merges** (T-0362, T-0363, T-0364):

  aps collab     → aps workspace             (full surface)
  aps audit log  → aps workspace audit
  aps conflict   → aps workspace conflicts
  aps messenger  → aps adapter messenger     (subcommand of adapter)
  aps voice session list  → aps session list --type voice
                            (voice sessions unified into core
                            session registry; SessionType field
                            added to SessionInfo)

**Verb-noun-flat → noun-verb renames** (T-0365, T-0373):

  aps profile add-capability     → aps profile capability add
  aps profile remove-capability  → aps profile capability remove
  aps profile set-workspace      → aps profile workspace set
  aps squad add-member           → aps squad members add
  aps squad remove-member        → aps squad members remove
  aps adapter set-permissions    → aps adapter permissions set
  aps a2a list-tasks             → aps a2a tasks list
  aps a2a get-task               → aps a2a tasks show
  aps a2a send-task              → aps a2a tasks send
  aps a2a cancel-task            → aps a2a tasks cancel
  aps a2a subscribe-task         → aps a2a tasks subscribe
  aps a2a send-stream            → aps a2a tasks stream
  aps a2a fetch-card             → aps a2a card fetch
  aps a2a show-card              → aps a2a card show

**Naming-convention renames** (T-0374, T-0375):

  aps profile new                → aps profile create
  aps directory deregister       → aps directory delete

**Help-section grouping** (T-0366, T-0367) — `aps --help` now organizes
commands into 5 visible groups (INTERACT, ORGANIZE, PIPELINES, SECURITY,
INSTANCE) plus a hidden MANAGEMENT group (alias, docs, env, migrate,
upgrade, toolspec, version, completion). Per-group help via
`--help-<id>`; reveal hidden groups with `--help-all` or
`--help-management`.

**New persistent globals** (T-0376) — `--config`, `--profile`, `--workspace`
declared on the root command. Subcommand-local duplicates removed; reads
fall through cobra's persistent flag set.

### Breaking changes — kit-reorg-adoption (track: `kit-reorg-adoption`)

- **Removed per-command `--json` flags** on `aps version`, `aps profile list`,
  and `aps action list`. Use the persistent `--format` flag (now provided by
  `hop.top/kit/go/console/cli`) to select output mode:
  - `--format table` (default) — human-readable table output
  - `--format json`              — JSON
  - `--format yaml`              — YAML

  Migration: replace `aps version --json` with `aps version --format json`,
  `aps profile list --json` with `aps profile list --format json`, etc.
  Tracked in T-0345 (track: `kit-reorg-adoption`).

- **Flag shortname realignment** (T-0347, audit at
  `docs/plans/2026-04-29-kit-reorg-adoption/flag-audit.md`):
  - Dropped `-f` shortname on `--force` for `profile delete`,
    `session delete`, `session terminate`, and collab helpers (`-f` is
    reserved for a future `--format` short alias). Long form `--force`
    still works.
  - Dropped `-v` shortname on `--verbose` for `profile status`,
    `skill list`, `adapter status`, `adapter links`. Use kit's persistent
    `-V` (count) flag instead, or the long `--verbose` form locally.
  - Removed `aps upgrade -q --quiet`. The flag is now provided by kit as a
    persistent root flag — `aps upgrade --quiet` still works, but reads
    from `root.Viper.GetBool("quiet")` instead of a local flag.

### Added

- `-n` short alias for `--dry-run` on: `action run`, `adapter link`,
  `adapter unlink`, `adapter stop`, `adapter revoke`, `conflict resolve`,
  `migrate messengers`, `webhook server` (POSIX `make -n` convention).

### Added

- Persistent `--format` and `--no-hints` flags on the root command, wired by
  `hop.top/kit/go/console/cli` via `output.RegisterFlags` and
  `output.RegisterHintFlags` (T-0344).
