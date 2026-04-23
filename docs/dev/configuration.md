# Configuration

APS follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) for all file locations.

## Directory Layout

| Purpose | Priority chain |
|---------|---------------|
| **Data** (profiles, adapters) | `$APS_DATA_PATH` → `$XDG_DATA_HOME/aps` → `~/.local/share/aps` |
| **Config** (settings) | `$XDG_CONFIG_HOME/aps` → `~/.config/aps` |
| **State** (runtime state) | `$XDG_STATE_HOME/aps` → `~/.local/state/aps` |
| **Cache** | `$XDG_CACHE_HOME/aps` → `~/.cache/aps` |

Set `$APS_DATA_PATH` to override all data storage to a custom path (useful for multi-profile setups or portable installs).

## Profile Storage

Profiles are stored under the data directory:

```
~/.local/share/aps/profiles/{profile-id}/
├── profile.yaml       # config, email, squad memberships, scope
├── secrets.env        # secret key-value pairs (mode 0600)
├── notes.md           # profile notes
├── gitconfig          # git configuration (if git enabled)
└── actions/           # profile actions
```

### profile.yaml Fields

```yaml
id: noor
display_name: Noor
email: noor@example.com          # used by adapters, bundles
persona: {}
capabilities: []
preferences:
  shell: /bin/zsh
git:
  enabled: true
isolation:
  level: process
```

The `email` field is used by:
- Script adapter exec (`APS_EMAIL_FROM` env var)
- Bundle template variable (`${PROFILE_EMAIL}`)
- Set via `aps profile new --email <addr>`

## Global Config File

`~/.config/aps/config.yaml`:

```yaml
prefix: APS                        # environment variable prefix
isolation:
  default_level: process           # process | platform | container
  fallback_enabled: true           # fall back to process isolation if preferred level fails
capability_sources:
  - /path/to/additional/capabilities
```

**Fields:**

| Field | Default | Description |
|-------|---------|-------------|
| `prefix` | `APS` | Prefix for environment variables injected into sessions |
| `isolation.default_level` | `process` | Default isolation level for profiles that don't specify one |
| `isolation.fallback_enabled` | `true` | Allow degraded-mode operation if preferred isolation is unavailable |
| `capability_sources` | `[]` | Additional directories to search for capability definitions |

## Migration from Legacy Paths

If you previously used APS before XDG support (profiles stored in `~/.agents/profiles/`), run:

```bash
aps migrate
```

This moves profiles from `~/.agents/profiles/` to `~/.local/share/aps/profiles/` (or your configured data path). The migration:

- Copies each profile directory
- Preserves all files including secrets
- Skips if source and destination are the same path
- Does not delete the original (remove manually after verifying)

Config migration (adding `isolation` block to existing config) runs automatically on startup.

## Adapter Path Sanitization

When exporting adapters or profiles, absolute paths containing the home directory are automatically replaced with `~` placeholders for portability:

```yaml
# On export
ssh_key_path: ~/.ssh/id_rsa

# On import — expanded back to absolute path
ssh_key_path: /Users/alice/.ssh/id_rsa
```

This applies to SSH key paths and DID identity key paths.

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/paths.go` | `DataDir()`, `ConfigDir()`, `StateDir()`, `CacheDir()` |
| `internal/core/config.go` | `LoadConfig()`, `SaveConfig()`, `MigrateConfig()` |
| `internal/core/profile.go` | `MigrateProfilesFromLegacy()` |
| `internal/core/profile_export.go` | `SanitizePathsForExport()`, `RestorePathsFromImport()` |
