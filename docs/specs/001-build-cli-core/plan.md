# Implementation Plan: CLI and Core Engine Implementation

**Branch**: `001-build-cli-core` | **Date**: 2026-01-15 | **Spec**: [specs/001-build-cli-core/spec.md](../spec.md)
**Input**: Feature specification from `specs/001-build-cli-core/spec.md`

## Summary

Implement the core `aps` CLI, TUI, and execution engine as a single Go binary. This includes profile management, secure secret injection, action execution, and an on-demand webhook server. The system follows a standard Go project layout with shared logic in `internal/core`.

## Technical Context

**Language/Version**: Go 1.22+ (Standard generic Go version)
**Primary Dependencies**: 
- `spf13/cobra` (CLI)
- `charmbracelet/bubbletea` (TUI)
- `charmbracelet/lipgloss` (Styling)
- `joho/godotenv` (Secrets)
- `gopkg.in/yaml.v3` (Config)
**Storage**: Filesystem (`~/.agents` directory structure)
**Testing**: Go standard library `testing`
**Target Platform**: Darwin (as per user env), Linux, Windows (Cross-platform Go)
**Project Type**: CLI/TUI Application
**Performance Goals**: `aps run` overhead < 50ms, profile creation < 1s
**Constraints**: Single binary output, safe-by-default secrets handling
**Scale/Scope**: Local usage, potentially hundreds of profiles

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Library-First**: Core logic encapsulated in `internal/core`? YES.
- **CLI Interface**: `aps` CLI defined with text/JSON output? YES.
- **Test-First**: Plan includes testing strategy? YES.
- **Simplicity**: Standard Go layout used? YES.

## Project Structure

### Documentation (this feature)

```text
specs/001-build-cli-core/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
cmd/
└── aps/
    └── main.go          # Entry point

internal/
├── core/
│   ├── profile.go       # Profile entity & logic
│   ├── execution.go     # Command/Action execution
│   ├── secrets.go       # Secret loading & injection
│   └── webhook.go       # Webhook server logic
├── cli/
│   ├── root.go
│   ├── profile.go
│   ├── run.go
│   ├── action.go
│   └── webhook.go
└── tui/
    ├── model.go
    └── update.go
```

**Structure Decision**: Standard Go Layout (`cmd/`, `internal/`) as clarified in specification.