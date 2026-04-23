# Implementation Plan: Automated E2E Test Suite

**Branch**: `002-e2e-tests` | **Date**: 2026-01-15 | **Spec**: [specs/002-e2e-tests/spec.md](../spec.md)
**Input**: Feature specification from `specs/002-e2e-tests/spec.md`

## Summary

Implement a robust, standalone E2E test suite in `tests/e2e/` that verifies the `aps` CLI binary against all specified user stories. The suite will handle binary compilation, isolated environment setup (temp HOME), and assertions on command outputs and side effects.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: 
- Standard Library (`testing`, `os/exec`, `path/filepath`)
- `github.com/stretchr/testify` (optional, for assertions if complexity warrants, otherwise stdlib)
**Storage**: Temporary directories (`t.TempDir()`)
**Testing**: `go test -v ./tests/e2e/...`
**Target Platform**: Cross-platform (path handling must be OS-agnostic)
**Performance Goals**: Test suite < 10s execution.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Test-First**: This entire feature IS testing. YES.
- **Simplicity**: Using standard `go test` runner. YES.
- **Library-First**: N/A (Test suite).

## Project Structure

### Documentation (this feature)

```text
specs/002-e2e-tests/
├── plan.md
├── research.md
└── tasks.md
```

### Source Code (repository root)

```text
tests/
└── e2e/
    ├── main_test.go       # TestMain (compile binary) & Helpers
    ├── profile_test.go    # US1: Profile management
    ├── run_test.go        # US2: Command execution
    ├── action_test.go     # US3: Actions
    └── webhook_test.go    # US4: Webhooks
```

**Structure Decision**: Separate test package `e2e_test` or package `e2e` inside `tests/`. Using `package e2e` prevents circular deps.

## Implementation Strategy

1. **Harness**: `TestMain` compiles `cmd/aps` to a temp location.
2. **Isolation**: Helper `runAPS(t, homeDir, args...)` executes the binary setting `HOME=homeDir`.
3. **Coverage**: One test file per User Story.