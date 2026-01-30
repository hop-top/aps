# Capability Management Design

## Overview
Capability Management allows APS to install, manage, and link external tools, configurations, and dotfiles. This system provides a unified way to manage development environment dependencies and integrate them into the APS workspace.

## Core Concepts

### 1. Capability Structure
Capabilities are stored in `~/.aps/capabilities/` by default. Each capability is a directory containing the tool's files or configurations.

**Metadata** (`manifest.yaml`):
```yaml
name: my-tool
path: /Users/user/.aps/capabilities/my-tool
type: managed  # or reference
installed_at: 2024-01-20T10:00:00Z
links:
  /path/to/link: /path/to/target
```

### 2. Capability Types
*   **Managed**: The capability data is copied into `~/.aps/capabilities/`. APS owns the data.
*   **Reference**: The capability is a symlink to an external location. APS only manages the reference. (Used by `watch` command).

### 3. Smart Linking
Smart Linking simplifies the configuration of common tools by maintaining a registry of known tool patterns.

**Registry Example**:
*   **Tool**: `windsurf` -> **Default Path**: `.windsurf/workflows/agent.md`
*   **Tool**: `copilot` -> **Default Path**: `.github/agents/agent.agent.md`

When a user runs `aps capability link windsurf`, APS automatically resolves the target path to `.windsurf/workflows/agent.md` relative to the current working directory.

### 4. Environment Integration via `aps env`
To integrate capabilities into the shell environment without polluting the global scope permanently, APS provides the `env` command.

**Mechanism**:
1. Iterate all installed capabilities.
2. Generate export statements: `export APS_<NAME>_PATH="<capability_path>"`.
3. Sanitization: Names are uppercased and hyphens replaced with underscores (e.g., `test-tool` -> `APS_TEST_TOOL_PATH`).
4. **Usage**: `eval $(aps env)` in `.zshrc` or `.bashrc`.

## Architecture

### `internal/core/capability` Package
*   **`Manager`**: Handles lifecycle operations (`Install`, `Link`, `Adopt`, `Watch`, `Delete`).
*   **`Registry`**: Stores Smart Patterns and handles resolution logic.
*   **`Config`**: Integrates with `config.yaml` to support multiple `capability_sources`.

### Configuration
Users can define additional source roots in `~/.config/aps/config.yaml`:
```yaml
capability_sources:
  - /shared/team/capabilities
  - ~/personal/capabilities
```
APS scans all configured roots when listing or loading capabilities.
