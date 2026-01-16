# APS (Agent Profile System)

APS is a local-first Agent Profile System that enables running commands and agent workflows under isolated profiles.

## Build Instructions

### Prerequisites
- Go 1.22+

### Build

Build for all platforms:
```bash
make build
```

Build only for your current platform:
```bash
make build-local
```

### Run

```bash
./bin/aps
```

## Testing

To run the E2E test suite:

```bash
go test -v ./tests/e2e
```

## Shell Integration

### Shorthands
APS supports executing commands directly using the profile name:
```bash
aps <profile-id> [command]
```
If no command is provided, it starts an interactive shell session for that profile.

### Auto-completion
Add this to your `~/.zshrc` (or equivalent):
```bash
source <(aps completion zsh)
```

### Profile Aliases
To invoke profiles directly by their ID, add this to your shell config:
```bash
eval "$(aps alias)"
```
This generates aliases like `alias <profile-id>='aps <profile-id>'`.

## Configuration

APS supports a global configuration file located at `$XDG_CONFIG_HOME/aps/config.yaml` (defaults to `~/.config/aps/config.yaml` on Linux/macOS).

### Environment Variable Prefix

You can customize the prefix used for profile environment variables:

```yaml
prefix: MYTOOL
```

This will inject variables like `MYTOOL_PROFILE_ID` instead of the default `APS_PROFILE_ID`.