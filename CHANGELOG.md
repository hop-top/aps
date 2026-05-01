# Changelog

All notable changes to `aps` are documented in this file.

## Unreleased

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
