# Platform Sandbox

**ID**: 031
**Feature**: Execution Isolation
**Persona**: [Maintainer](../personas/maintainer.md)
**Priority**: P1

## Story

As a maintainer, I want platform-specific sandbox adapters (process, container, macOS sandbox) with a fallback chain so that isolation works reliably across operating systems and gracefully degrades when an adapter is unavailable.

## Acceptance Scenarios

1. **Given** a registered isolation adapter, **When** the manager resolves it, **Then** it returns the exact match.
2. **Given** strict mode enabled, **When** the requested adapter is unavailable, **Then** execution fails without fallback.
3. **Given** graceful degradation enabled, **When** the requested adapter is unavailable, **Then** the manager falls back through available levels.
4. **Given** macOS, **When** platform isolation is requested, **Then** the Darwin sandbox adapter is used.

## Tests

### Unit
- `tests/unit/core/isolation/manager_test.go` — `TestNewManager`, `TestRegisterAndGet`, `TestGetNotRegistered`, `TestIsolationLevels`
- `tests/unit/core/isolation/process_test.go` — `TestProcessIsolation_PrepareContext`, `TestProcessIsolation_SetupEnvironment`, `TestProcessIsolation_Validate`, `TestProcessIsolation_Cleanup`
- `tests/unit/core/isolation/container_test.go` — `TestDockerfileBuilder_Generate_Basic`, `TestDockerfileBuilder_Generate_WithPackages`, `TestDockerfileBuilder_Generate_WithBuildSteps`, `TestDockerfileBuilder_ParseVolumes`, `TestDockerEngine_Available`, `TestDockerEngine_Version`
- `tests/unit/core/isolation/fallback_test.go` — `TestGetIsolationManager_ExactMatch`, `TestGetIsolationManager_StrictModeFailure`, `TestGetIsolationManager_FallbackDisabled`, `TestGetIsolationManager_GlobalFallbackDisabled`, `TestGetIsolationManager_GracefulDegradation`, `TestGetIsolationManager_MultipleFallbackLevels`, `TestGetIsolationManager_InvalidAdapter`, `TestGetIsolationManager_UseDefaultLevel`, `TestGetIsolationManager_NoAdaptersAvailable`
- `tests/unit/core/isolation_darwin_test.go` — `TestDarwinSandbox_InterfaceCompliance`, `TestDarwinSandbox_IsAvailable`, `TestDarwinSandbox_PrepareContext`, `TestDarwinSandbox_Validate`, `TestDarwinSandbox_ValidateErrors`, `TestDarwinSandbox_PasswordGeneration`, `TestDarwinSandbox_Cleanup`, `TestDarwinSandbox_ManagerIntegration`
- `tests/unit/core/platform_ssh_test.go` — `TestPlatformSSH_AttachToPlatformSandbox`, `TestPlatformSSH_EnvironmentValidation`, `TestPlatformSSH_AdminKeyPath`
- `tests/unit/core/ssh_keys_test.go` — `TestSSHKeyGeneration`
