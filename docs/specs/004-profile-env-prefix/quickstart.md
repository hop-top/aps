# Quickstart: Configuring Profile Environment Prefixes

This guide shows how to change the default prefix used for environment variables injected by the `aps` CLI.

## Default Behavior

By default, `aps` injects variables with the `APS_` prefix:

- `APS_PROFILE_ID`
- `APS_PROFILE_DIR`
- ...

## Customizing the Prefix

1. Create a configuration file at the standard location for your OS:
   - **Linux**: `~/.config/aps/config.yaml`
   - **macOS**: `~/Library/Application Support/aps/config.yaml`
   - **Windows**: `%AppData%\aps\config.yaml`

2. Add the `prefix` setting:

```yaml
prefix: MYTOOL
```

3. Run any action. The environment variables will now use your custom prefix:

```bash
aps run my-profile -- env | grep MYTOOL_
# Output:
# MYTOOL_PROFILE_ID=my-profile
# MYTOOL_PROFILE_DIR=/path/to/profile
```

## Verifying Configuration

If the configuration file is invalid or unreadable, `aps` will silently fallback to the default `APS_` prefix.
