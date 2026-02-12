# Custom Prefix Configuration

**ID**: 014
**Feature**: Profile Env Prefix
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to define a custom prefix in my `config.yaml` file so that I can integrate the tool into environments with specific naming conventions or avoid collisions with other tools.

## Acceptance Scenarios

1. **Given** a configuration file at `$XDG_CONFIG_HOME/aps/config.yaml` with `prefix: CUSTOM`, **When** I run an action, **Then** the process environment should contain `CUSTOM_PROFILE_ID`, `CUSTOM_PROFILE_DIR`, etc.
2. **Given** a custom prefix is configured, **When** I run an action, **Then** the default `APS_` prefixed variables should not be automatically injected (only the custom ones).

## Tests

### E2E
- `tests/e2e/config_test.go` — `TestCustomPrefixConfig`

### Unit
- `tests/unit/core/execution_test.go` — `TestInjectEnvironmentCustomPrefix`
- `tests/unit/core/config_test.go` — `TestLoadConfig`
