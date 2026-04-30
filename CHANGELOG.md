# Changelog

All notable changes to `aps` are documented in this file.

## Unreleased

### Breaking changes

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
