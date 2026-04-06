# Release Process

This document describes the release workflow for APS CLI.

## Overview

APS uses [GoReleaser](https://goreleaser.com) for automated cross-platform releases.

**Source of Truth**: Git tags (e.g., `v1.0.0-alpha.1`)

## Release Flow

```
git tag v1.0.0  →  push  →  GitHub Actions:
  1. Updates VERSION.txt from tag
  2. Runs tests
  3. Builds binaries (linux/darwin/windows × amd64/arm64)
  4. Generates checksums + SBOMs
  5. Creates GitHub Release with changelog
  6. Commits VERSION.txt back to main
  7. Verifies artifacts on all platforms
```

## How to Release

```bash
# 1. Ensure main is clean and tested
git checkout main
git pull

# 2. Create and push tag
git tag v1.0.0-alpha.1
git push origin v1.0.0-alpha.1

# 3. Monitor release at GitHub Actions
```

## Versioning

Follows [Semantic Versioning](https://semver.org/):

| Format | Example | Use Case |
|--------|---------|----------|
| `vX.Y.Z` | `v1.0.0` | Stable release |
| `vX.Y.Z-alpha.N` | `v1.0.0-alpha.1` | Alpha pre-release |
| `vX.Y.Z-beta.N` | `v1.0.0-beta.1` | Beta pre-release |
| `vX.Y.Z-rc.N` | `v1.0.0-rc.1` | Release candidate |

## Key Files

| File | Purpose |
|------|---------|
| `.goreleaser.yaml` | GoReleaser configuration |
| `.github/workflows/release.yml` | Release workflow |
| `internal/version/version.go` | Version package (ldflags injection) |
| `internal/cli/version.go` | `aps version` command |
| `VERSION.txt` | Current version (auto-updated) |

## Build Artifacts

Each release produces:
- `aps_VERSION_linux_amd64.tar.gz`
- `aps_VERSION_linux_arm64.tar.gz`
- `aps_VERSION_darwin_amd64.tar.gz`
- `aps_VERSION_darwin_arm64.tar.gz`
- `aps_VERSION_windows_amd64.zip`
- `checksums.txt`
- SBOMs (Software Bill of Materials)

## Local Testing

Recommended: Use **[mise](https://mise.jdx.dev)** to manage development tools.

```bash
# Install all required tools (go, goreleaser, golangci-lint, act)
mise install
```

```bash
# Test GoReleaser locally (dry run)
goreleaser release --snapshot --clean
```

## Testing Workflows Locally

Use **[act](https://github.com/nektos/act)** to run GitHub Actions locally:

```bash
# Install act
brew install act

# List available workflows
act -l

# Dry run release workflow (simulate tag push)
act push --eventpath <(echo '{"ref": "refs/tags/v0.5.0-test"}') -n

# Run CI workflow
act push -W .github/workflows/ci.yml

# Run specific job
act -j test

# Use medium runner image (closer to GitHub's)
act -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest
```

> **Note**: Some actions may not work perfectly in `act` (e.g., caching, artifact uploads). Use for syntax validation and basic testing.
