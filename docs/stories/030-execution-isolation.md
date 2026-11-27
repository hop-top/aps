---
status: partial
---

# Execution Isolation

**ID**: 030
**Feature**: Execution Isolation
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want to configure isolation levels (process, platform, container) for my profiles so that commands run in sandboxed environments with controlled access to the host system.

## Acceptance Scenarios

1. **Given** a profile with `isolation.level: process`, **When** I run a command, **Then** it executes in an isolated process with restricted environment variables.
2. **Given** a profile with `isolation.level: container`, **When** I run a command, **Then** it executes inside a Docker container built from the profile's container config.
3. **Given** no isolation config, **When** I run a command, **Then** the default isolation level is used.
4. **Given** a profile with strict mode disabled, **When** the requested isolation level is unavailable, **Then** execution falls back to the next available level.

## Status

Partial. Only `process` runs. `aps run`, `aps action run`, and webhook-triggered actions
fail for `platform` and `container` with `<level> isolation not yet implemented`.

| Scenario | Met |
|---|---|
| 1 | Partly. The command runs as a child process with the profile's variables added; the parent environment is inherited, not restricted. |
| 2 | No. A container profile is refused before anything starts; no Docker container is built or run. |
| 3 | Yes, with `process`. A profile with no level runs at `process`; the global `isolation.default_level` is not read by the run path. |
| 4 | No. `strict` and `fallback` are stored but not read on the run path, and there is no fallback to a lower level. |

`aps serve` and messenger and ticket routing refuse `platform` and `container` profiles the
same way. Current behaviour: [Isolation levels](../user/isolation.md).

## Tests

### Unit
- `tests/unit/core/isolation_test.go` — `TestIsolationFoundation_InterfaceCompliance`, `TestIsolationFoundation_FallbackLogic`, `TestIsolationFoundation_ProcessIsolationIntegration`, `TestIsolationFoundation_ConfigIntegration`, `TestIsolationFoundation_ErrorHandling`, `TestIsolationFoundation_BackwardCompatibility`, `TestIsolationFoundation_AllExistingTests`
- `tests/unit/core/isolation_config_test.go` — `TestIsolationConfig_DefaultLevel`, `TestIsolationConfig_ProcessLevel`, `TestIsolationConfig_PlatformLevel`, `TestIsolationConfig_ContainerLevel`, `TestIsolationConfig_InvalidLevel`, `TestIsolationConfig_ContainerWithoutImage`, `TestIsolationConfig_ValidateMethod`, `TestIsolationConfig_ValidateInvalid`
- `tests/unit/core/isolation_global_config_test.go` — `TestSaveProfileWithIsolation`, `TestConfigDefaultIsolation`, `TestConfigLoadWithIsolation`, `TestConfigSaveWithIsolation`, `TestConfigMigrate`, `TestConfigMigrateNoExisting`, `TestConfigInvalidIsolationLevel`, `TestConfigPartialIsolation`

### E2E
- `tests/e2e/isolation_test.go` — `TestIsolationManager_ProcessLevel`, `TestIsolationManager_InvalidProfile`, `TestIsolationManager_Sequence`, `TestIsolationManager_ActionExecution`, `TestIsolationManager_WithSecrets`
- `tests/e2e/isolation_fallback_test.go` — `TestIsolationFallbackProcessToDefault`, `TestIsolationStrictModeNoFallback`
- `tests/e2e/profile_isolation_test.go` — `TestIsolationProfileFallbackDisabled`, `TestIsolationGlobalFallbackDisabled`, `TestIsolationDefaultLevelUsed`, `TestProfileIsolationConfig`, `TestProfileWithContainerIsolation`, `TestProfileWithPlatformIsolation`, `TestProfileInvalidIsolationLevel`, `TestProfileContainerWithoutImage`
