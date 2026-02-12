# Standardized Default Prefix

**ID**: 013
**Feature**: Profile Env Prefix (specs/004)
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want the CLI to use `APS` as the default prefix for profile environment variables instead of `AGENT_PROFILE` so that the environment variables are consistently named after the tool.

## Acceptance Scenarios

1. **Given** no configuration file exists, **When** I run an action that executes a command, **Then** the process should have `APS_PROFILE_ID` and other `APS_` prefixed variables in its environment.
2. **Given** no configuration file exists, **When** I run an action, **Then** `AGENT_PROFILE_ID` should no longer be present in the environment (unless explicitly set by the user).

## Tests

### E2E
- `tests/e2e/config_test.go` — `TestCustomPrefixConfig`

### Unit
- `tests/unit/core/execution_test.go` — `TestInjectEnvironment`
- `tests/unit/core/config_test.go` — `TestLoadConfig`, `TestGetConfigDir`
