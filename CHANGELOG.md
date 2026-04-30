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

### Added

- Persistent `--format` and `--no-hints` flags on the root command, wired by
  `hop.top/kit/go/console/cli` via `output.RegisterFlags` and
  `output.RegisterHintFlags` (T-0344).
