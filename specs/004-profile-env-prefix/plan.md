# Implementation Plan: Configurable Profile Env Var Prefix

**Branch**: `004-profile-env-prefix` | **Date**: 2026-01-15 | **Spec**: [specs/004-profile-env-prefix/spec.md]
**Input**: Feature specification from `/specs/004-profile-env-prefix/spec.md`

## Summary

This feature replaces the hardcoded `AGENT_PROFILE_` environment variable prefix with a configurable one, defaulting to `APS_`. Configuration is loaded from `$XDG_CONFIG_HOME/aps/config.yaml`.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: `os`, `path/filepath`, `gopkg.in/yaml.v3`
**Storage**: Local YAML configuration file.
**Testing**: `go test ./tests/unit/...` and `go test ./tests/e2e/...`.
**Target Platform**: Darwin, Linux, Windows.
**Project Type**: CLI tool.
**Performance Goals**: Negligible impact on startup time (minimal YAML parsing).
**Constraints**: Must follow XDG Base Directory Specification.
**Scale/Scope**: Core engine modification.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Library-First: Configuration loading will be a core library function.
- [x] CLI Interface: Behavior exposed through CLI action execution.
- [x] Test-First: New tests for config loading and prefix injection in `tests/unit/core`.
- [x] Integration Testing: E2E tests in `tests/e2e` will verify environment variable injection.

## Project Structure

### Documentation (this feature)

```text
specs/004-profile-env-prefix/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── config.schema.json
└── tasks.md             # Phase 2 output (to be created)
```

### Source Code (repository root)

```text
internal/
├── core/
│   ├── config.go        # NEW: Global configuration loading
│   ├── execution.go     # UPDATED: Use dynamic prefix
│   └── profile.go       # UPDATED: Path resolution for config

tests/
├── unit/
│   └── core/            # Unit tests for core logic
└── e2e/                 # E2E integration tests
```

**Structure Decision**: Single project structure (Option 1). We are adding a new file `config.go` to `internal/core` and updating `execution.go`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | | |