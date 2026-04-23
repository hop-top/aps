# Research: Configurable Profile Env Var Prefix

## Decision: Use `os.UserConfigDir()` for XDG Compliance
- **Decision**: Use Go's standard `os.UserConfigDir()` to locate the base configuration directory.
- **Rationale**: It is cross-platform and follows OS conventions (Darwin: `~/Library/Application Support`, Linux: `~/.config`, Windows: `%AppData%`).
- **Alternatives considered**: `adrg/xdg` library. Rejected to avoid unnecessary dependencies since `os.UserConfigDir()` provides the core functionality needed.

## Decision: Configuration File Structure
- **Decision**: Use `~/.config/aps/config.yaml` (on Linux) or equivalent on other OSs.
- **Rationale**: Follows standard CLI tool patterns.
- **Format**: 
  ```yaml
  prefix: APS
  ```

## Decision: Configuration Loading Logic
- **Decision**: Implement a `Config` struct and a `LoadConfig()` function in `internal/core/config.go`.
- **Rationale**: Centralizes configuration management and makes it reusable across the core engine.
- **Defaulting**: If the file is missing or `prefix` is empty, default to `APS`.

## Decision: Breaking Change Handling
- **Decision**: Replace `AGENT_PROFILE_` with the resolved prefix entirely.
- **Rationale**: The user requested a default change to `APS` and the ability to configure it. Maintaining `AGENT_PROFILE_` as a fallback would complicate the implementation and potentially cause confusion.
- **Implementation**: Update `internal/core/execution.go` to use the prefix from the loaded config.
