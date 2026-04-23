# Data Model: Global Configuration

## Config Entity

Represents the global settings for the `aps` CLI tool.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `prefix` | string | The prefix used for profile environment variables. | `APS` |

## Validation Rules

- `prefix`: Must be a valid environment variable name component (alphanumeric and underscores).
- `prefix`: If empty, it defaults to `APS`.

## Persistence

- **Location**: `$XDG_CONFIG_HOME/aps/config.yaml`
- **Format**: YAML
- **Search Order**:
    1. `$XDG_CONFIG_HOME/aps/config.yaml` (if `$XDG_CONFIG_HOME` is set)
    2. `[UserConfigDir]/aps/config.yaml` (fallback via `os.UserConfigDir()`)
