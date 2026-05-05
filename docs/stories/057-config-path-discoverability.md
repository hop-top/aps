---
status: shipped
---

# 057 - Config Path Discoverability

**ID**: 057
**Feature**: CLI Conventions §7.4
**Persona**: [User](../personas/user.md)
**Priority**: P1
**Status**: shipped
**Author**: jadb
**Task**: T-0457

## Story

As an aps user (or sibling tool author wiring kit), I want `aps config path`
to print the active config file aps would load, and `aps config paths` to
print the full layered resolution chain (cwd → walk-up project → user →
system → defaults), so I can audit "which config does aps actually see?"
without reading source.

Convention parity with `git config --list --show-origin`, `npm config get`,
`gh config get`, `kubectl config view`, `tlc config paths`. Per
`~/.ops/docs/cli-conventions-with-kit.md` §7.4 every kit-built CLI must
expose these subcommands. Pre-T-0457 aps shipped without them — a concrete
miss against a hard requirement (audit `~/.ops/reviews/aps-kit-integration-audit-2026-05-04.md`
§2 + §7.4).

## Acceptance Scenarios

1. **Given** a config file at `$XDG_CONFIG_HOME/aps/config.yaml` and no
   project marker, **When** I run `aps config path`, **Then** stdout is
   that exact path and exit code is 0.

2. **Given** both `$XDG_CONFIG_HOME/aps/config.yaml` and `<cwd>/.aps.yaml`
   exist, **When** I run `aps config path` from cwd, **Then** stdout is
   the project marker (`<cwd>/.aps.yaml`) — project beats user.

3. **Given** no real config files exist, **When** I run `aps config path`,
   **Then** stdout is `<defaults>` (the synthetic in-binary fallback rung)
   and exit code is 0.

4. **Given** any environment, **When** I run `aps config paths`, **Then**
   stdout lists one path per line in highest-precedence-first order:
   cwd markers → walked-up project markers → user → system → `<defaults>`.

5. **Given** `--format json`, **When** I run `aps config paths --format
   json`, **Then** stdout is a JSON array of `{path, source, scope, exists}`
   objects (kit's ResolvedPath shape) and `--no-color` is honoured (color
   handling lives in kit/output, no aps-side branching).

## Implementation Notes

- Wired in `internal/cli/config.go` via
  `kitcliconfig.RegisterPathSubcommands(configCmd, "aps", WithResolver(apsConfigPathsResolver))`.
- Resolver mirrors what `internal/core/config.LoadConfig` consumes:
  project markers `.aps/config.yaml`, `.hop/aps/config.yaml`, `.aps.yaml`;
  user `$XDG_CONFIG_HOME/aps/config.yaml` via `kit/go/core/xdg.ConfigDir`;
  system `/etc/aps/config.yaml`; synthetic `<defaults>`.
- Walk-up stops at `$HOME` so user-scope discovery never escapes ~.
- `config` registered in MANAGEMENT group (hidden from default --help;
  visible via --help-management or --help-all) per §4.1.
- Out of scope: `validate`, `set`, `get`, `list`. Kit's shared introspection
  subcommands solve §7.4 alone; aps's existing config loader is unchanged.

## Tests

### E2E
- `tests/e2e/config/config_paths_test.go`
  - `TestConfigPath_PrintsUserConfigWhenPresent`
  - `TestConfigPath_PrefersProjectOverUser`
  - `TestConfigPath_NoConfigFallsBackToDefaults`
  - `TestConfigPaths_ListsLayeredChainText`
  - `TestConfigPaths_JSONHonoursFormatFlag`

### Unit
- Existing `tests/unit/core/config_test.go` covers `LoadConfig` /
  `GetConfigDir`. Resolver behaviour piggybacks on kit's
  `kit/go/console/cli/config/paths_cmd_test.go`.

## Dependencies

- `kit/go/console/cli/config.RegisterPathSubcommands` (kit ≥ 0.3.2)
- `kit/go/core/xdg.ConfigDir`
- CLI conventions §4.1 (groups), §7.4 (config introspection)
