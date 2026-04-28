# Research & Decisions

## Test Framework
- **Decision**: Go Standard `testing` package + `github.com/stretchr/testify/assert`.
- **Rationale**: Standard library is sufficient for running, but `testify` dramatically improves assertion readability and error reporting for E2E tests (diffs).

## Binary Compilation
- **Decision**: Compile once in `TestMain`.
- **Rationale**: Compiling per test is too slow. Compiling once ensures all tests run against the same artifact.
- **Mechanism**: `exec.Command("go", "build", "-o", tempBin, "./cmd/aps")`.

## Environment Isolation
- **Decision**: Per-test `t.TempDir()` as `HOME`.
- **Rationale**: Ensures tests do not interfere with each other or the user's actual `~/.agents`.
- **Constraint**: `aps` uses `os.UserHomeDir()`. We need to ensure setting `HOME` (or `USERPROFILE` on Windows) works. Go's `os.UserHomeDir()` respects these variables.

## Webhook Testing
- **Decision**: Use a helper to find a free port or use port 0.
- **Mechanism**: `aps webhook serve --addr 127.0.0.1:0` prints the chosen port?
- **Correction**: `aps webhook serve` currently takes `--addr`. We can pick a random port or just use `:0` if the CLI prints the listener address. The current implementation logs `listening on ...`. We can parse stdout or just pick a random high port in test.
