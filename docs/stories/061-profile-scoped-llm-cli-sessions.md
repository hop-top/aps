---
status: shipped
---

# 061 - Profile-Scoped External LLM CLI Sessions

**ID**: 061
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P1
**Status**: shipped
**Track**: aps-profile-llm-sessions
**Tasks**: T-0568, T-0569, T-0571

## Story

As an aps user who already uses agent CLIs like Claude Code, Codex,
Gemini CLI, or OpenCode, I want to launch those tools under a selected
APS profile so they inherit that profile's identity, git config, and
secrets without needing a native `aps chat` implementation first.

This story makes the current composition contract explicit:

```sh
aps run <profile> -- claude  ...
aps run <profile> -- codex   ...
aps run <profile> -- gemini  ...
aps run <profile> -- opencode ...

aps <profile> claude ...
```

Native `aps chat <profile>` remains a separate first-class REPL story.
This story is the external-CLI bridge.

## Acceptance Scenarios

1. **Given** a profile with API keys in `secrets.env`, **When** I run
   `aps run <profile> -- claude "review this branch"`, **Then** the
   `claude` process receives the profile's injected `APS_PROFILE_*`
   variables and profile secrets.

2. **Given** the same profile, **When** I run
   `aps run <profile> -- codex "write tests"`, **Then** the `codex`
   process receives the same profile-scoped environment.

3. **Given** the same profile, **When** I run
   `aps run <profile> -- gemini "summarize docs"`, **Then** the
   `gemini` process receives the same profile-scoped environment.

4. **Given** the same profile, **When** I run
   `aps run <profile> -- opencode "inspect failures"`, **Then** the
   `opencode` process receives the same profile-scoped environment.

5. **Given** any supported external LLM CLI, **When** I invoke it with
   shorthand syntax (`aps <profile> <cli> ...`), **Then** APS forwards
   argv unchanged after the CLI name and injects the same profile
   environment as `aps run`.

6. **Given** the child CLI writes stdout or stderr, **When** APS
   forwards the output, **Then** the streams remain visible to the
   caller while default redaction still protects secret values.

7. **Given** the child CLI exits non-zero, **When** APS exits, **Then**
   APS returns the child exit code so scripts and agents can react
   deterministically.

8. **Given** CI does not have real vendor CLIs or network credentials,
   **When** E2E tests run, **Then** tests use deterministic local stub
   executables named `claude`, `codex`, `gemini`, and `opencode`.

## Implementation Notes

- The implementation intentionally uses the generic execution surface,
  not vendor-specific adapters.
- Tests use `--no-redact` only where they must assert raw secret
  injection. Default redaction remains covered by story 058.
- The `aps run` path now preserves wrapped `exec.ExitError` exit codes
  through the root exit-code mapper, matching shorthand behavior.

## Tests

### E2E

- `tests/e2e/run_test.go`
  - `TestProfileScopedExternalLLMCLIs`
  - `TestProfileScopedExternalLLMCLIExitCode`

### Unit

- `internal/cli/exit/exit_test.go`
  - `TestCode` covers direct and wrapped child `exec.ExitError`
    preservation.

## Dependencies

- Story 002 — generic command execution.
- Story 010 — shorthand execution.
- Story 058 — default child-process output redaction.
- Story 055 — native `aps chat`, intentionally separate.
