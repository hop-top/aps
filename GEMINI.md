# oss-aps-cli Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-15

## Active Technologies

- **Go 1.25.5**: Core application language.
- **Cobra**: CLI framework for subcommands and flags.
- **Bubble Tea / Lip Gloss**: TUI framework for interactive screens.
- **YAML v3**: Configuration parsing for `profile.yaml` and global `config.yaml`.
- **GoDotEnv**: Secrets parsing from `secrets.env`.
- **Testify**: Assertion library for unit and E2E tests.
- **XDG Base Directory**: Standardized configuration discovery via `os.UserConfigDir()`.

## Project Structure

```text
bin/                 # Compiled binaries
cmd/aps/             # CLI Entry point
internal/
  cli/               # Cobra command definitions
  core/              # Core business logic (Profile, Config, Execution, Webhooks)
  tui/               # Bubble Tea models and views
specs/               # Feature specifications and implementation plans
tests/
  e2e/               # End-to-end integration tests
  unit/              # Centralized unit tests
```

## Commands

- `make build`: Build binaries for all platforms.
- `make build-local`: Build binary for current platform.
- `go test ./tests/unit/...`: Run all unit tests.
- `go test -v ./tests/e2e`: Run full E2E test suite.
- `go test ./...`: Run all tests (unit and E2E).
- `go fmt ./...`: Format all source code.
- `go vet ./...`: Run static analysis.

## Code Style

- **Standard Go Conventions**: Follow `Effective Go`.
- **Internal Package**: Keep core logic in `internal/core` to prevent external imports.
- **TDD-First**: Write failing tests before implementing feature logic.
- **Environment Prefixes**: Use dynamic prefixes (default `APS_`) for injected variables.
- **Security**: Strictly enforce `0600` permissions for secret files.

## Recent Changes

- **004-profile-env-prefix**: Switched default prefix to `APS_` and added global `config.yaml` support with XDG compliance.
- **003-shell-integration**: Implemented shorthand execution, shell aliases, and auto-completion scripts.
- **001-build-cli-core**: Established core engine, profile contract, and TUI/CLI foundation.

<!-- MANUAL ADDITIONS START -->
- **Secrets Management**: Always redact secret values when printing to stdout/stderr.
- **TUI/CLI Parity**: Every feature exposed in the TUI must also be accessible via a scriptable CLI command.
<!-- MANUAL ADDITIONS END -->