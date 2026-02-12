# XDG Configuration Discovery

**ID**: 015
**Feature**: Profile Env Prefix
**Persona**: [User](../personas/user.md)
**Priority**: P3

## Story

As a user, I want the tool to automatically find my configuration file in the standard XDG location for my operating system so that I don't have to manage extra environment variables just to point to a config file.

## Acceptance Scenarios

1. **Given** `$XDG_CONFIG_HOME` is set to `/tmp/myconfig`, **When** I have a config file at `/tmp/myconfig/aps/config.yaml`, **Then** the tool should use the settings from that file.
2. **Given** `$XDG_CONFIG_HOME` is not set, **When** I have a config file at `~/.config/aps/config.yaml`, **Then** the tool should use the settings from that file as the default fallback.

## Tests

### Unit
- `tests/unit/core/config_test.go` — `TestGetConfigDir`

### E2E
- `tests/e2e/global_config_isolation_test.go` — `TestGlobalConfigIsolationDefaults`, `TestGlobalConfigIsolationCustom`, `TestGlobalConfigInvalidIsolation`
