# Research & Decisions

## Technology Stack

### CLI Framework
- **Decision**: `spf13/cobra`
- **Rationale**: Industry standard for Go CLIs, robust flag parsing, automatic help generation, supports subcommands nested structure required by spec.
- **Alternatives**: `urfave/cli` (good but Cobra is more ubiquitous), `flag` (too simple).

### TUI Framework
- **Decision**: `charmbracelet/bubbletea` ecosystem
- **Rationale**: Mandated by spec clarifications. Best-in-class for modern, interactive terminal UIs in Go. `lipgloss` for styling, `bubbles` for components.

### Configuration & Secrets
- **Decision**: `gopkg.in/yaml.v3` for `profile.yaml`, `joho/godotenv` for `secrets.env`.
- **Rationale**: YAML is standard for human-readable config. Dotenv is standard for secrets injection (requested in spec).
- **Security**: File permissions (0600) check must be implemented manually via `os.Stat`.

### Webhook Server
- **Decision**: Go `net/http` standard library
- **Rationale**: Sufficient for simple webhook handling. No need for heavy frameworks like Gin/Echo for a few endpoints.
- **HMAC**: `crypto/hmac` and `crypto/sha256` standard libraries.

## Architecture Patterns

### Internal Core
- **Pattern**: `internal/core` package acts as the "Domain Layer".
- **Separation**: CLI (`internal/cli`) and TUI (`internal/tui`) are "Presentation Layers" that consume `core`.
- **Benefit**: Ensures TUI and CLI behave identically regarding business logic (execution, profiles).

### Environment Injection
- **Strategy**: `os.Environ()` + `secrets.env` + `APS_*` vars.
- **Safety**: `exec.Cmd.Env` explicitly set to this combined slice.
