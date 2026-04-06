# Capability Implementation Summary

**Date**: 2026-01-30
**Status**: Complete

## Overview
Implemented the Capability Management system (Spec 007) to handle external tools and configurations within APS.

## Components

### 1. Core Logic (`internal/core/capability`)
*   **`manager.go`**: Implements the core business logic.
    *   `Install`: and `copyDir` for managed capabilities.
    *   `Link`: Creates symlinks to workspace targets.
    *   `Delete`: Removes capability artifacts.
    *   `GenerateEnvExports`: Iterates capabilities and formats `APS_<NAME>_PATH`.
*   **`registry.go`**: Defines `SmartPattern` struct and the default list of supported tools (Claude, Cursor, Windsurf, Copilot, etc.).

### 2. CLI Integration (`internal/cli`)
*   **`capability.go`**: Implements `aps capability` subcommand group.
*   **`env.go`**: Implements `aps env` command using `GenerateEnvExports`.

### 3. Configuration
*   Updated `internal/core/config.go` to support `CapabilitySources []string` in `config.yaml`.
*   Updates `List()` logic to scan multiple roots.

## Testing
*   **Unit Tests**: `tests/unit/core/capability/capability_test.go` verifies lifecycle and environmental generation.
*   **E2E Tests**: `tests/e2e/capability_test.go` verifies the CLI workflow (Install -> List -> Env -> Delete).

## Known Issues / Future Work
*   **Outbound Link Cleanup**: `Delete` warns about but does not automatically remove outbound symlinks created in workspaces.
*   **Remote Sources**: Currently only local directory sources are supported. Future support for Git URLs is planned.
