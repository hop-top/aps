# Implementation Plan: Shell Integration & Shorthands

**Branch**: `003-shell-integration` | **Date**: 2026-01-15 | **Spec**: [specs/003-shell-integration/spec.md](../spec.md)
**Input**: Feature specification from `specs/003-shell-integration/spec.md`

## Summary

Implement shell integration features for the `aps` CLI, including profile shorthand execution (e.g., `aps agent-a`), alias generation, shell completion, and smart shell session launching.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: 
- `spf13/cobra` (for root command args handling and completion generation)
- `os/exec` (for alias conflict detection)
- Standard library (`os`, `strings`)
**Storage**: Updates to `profile.yaml` via existing `core.Profile` struct.
**Testing**: E2E tests for shell behavior (using `tests/e2e/`).
**Target Platform**: Cross-platform (Unix-like shells primarily for completion/aliases, Windows for PowerShell completion).
**Project Type**: CLI Application extension.
**Performance Goals**: Shorthand resolution < 10ms.
**Constraints**: Must not break existing `aps run` behavior.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **CLI Interface**: Enhances CLI usability significantly. YES.
- **Simplicity**: Leveraging Cobra's built-in completion. YES.
- **Test-First**: E2E tests planned. YES.

## Project Structure

### Documentation (this feature)

```text
specs/003-shell-integration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── core/
│   ├── profile.go       # Update Profile struct
│   └── shell.go         # New: Shell detection logic
├── cli/
│   ├── root.go          # Update: Shorthand logic
│   ├── completion.go    # New: Completion command
│   └── alias.go         # New: Alias command
└── ...
```

**Structure Decision**: Add `shell.go` to `core` for shared shell logic (detection, path checking). Extend `cli` package with new commands.