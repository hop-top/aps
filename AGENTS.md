# oss-aps-cli Development Guidelines

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

For the most accurate representation of the codebase, see @.xray map.

Use `xray --help` for more.

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


<!-- MANUAL ADDITIONS START -->
- **Secrets Management**: Always redact secret values when printing to stdout/stderr.
- **TUI/CLI Parity**: Every feature exposed in the TUI must also be accessible via a scriptable CLI command.
- **Git Commit Messages**: Stick to conventional commit messages and NEVER include co-author.
<!-- MANUAL ADDITIONS END -->
