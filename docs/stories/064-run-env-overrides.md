---
status: shipped
---

# 064 - Per-Invocation Environment Overrides for `aps run`

**ID**: 064
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P2
**Status**: shipped
**Track**: aps-run-env-flag

## Story

As an aps user, I want to set or override environment variables for a
single `aps run` invocation without editing the profile so I can swap
in an alternate API key, test a feature flag, or load a project-local
`.env` file without polluting the profile's `secrets.env`.

```sh
aps run <profile> --env KEY=VAL -- <command>
aps run <profile> --env-file path/to/.env -- <command>
aps run <profile> --env-file .env --env DEBUG=1 -- <command>
```

Both flags are repeatable. Validation is fatal-by-error so a typo
fails before the child process is spawned.

## Precedence

Lowest to highest; later entries win on duplicate keys:

1. Parent process `os.Environ()`.
2. Profile-injected variables (`APS_*`, secrets, git/ssh, capabilities,
   bundles — `buildEnvVars` output).
3. `--env-file` entries (in flag order; later files win over earlier).
4. `--env` entries (in flag order; later `--env` for the same key wins).

The resolved slice is de-duplicated last-wins before exec so the child
sees a single value per key.

## Validation

- `--env KEY=VAL` requires a non-empty `KEY` matching
  `[A-Za-z_][A-Za-z0-9_]*` and the literal `=` separator.
- `--env-file PATH` fails if the file is missing or unreadable.
- Each line in a dotenv file must be blank, a `#` comment, or a
  `KEY=VALUE` (optionally prefixed with `export `). Malformed lines
  fail with the file path and line number.

Dotenv values may be wrapped in matching single or double quotes
(quotes are stripped). Unquoted values trim ` # …` trailing comments.
No escape sequences are expanded.

## Redaction

`--env OPENAI_API_KEY=<sk-…>` makes the secret end up in the child's
real environment legitimately — but the aps-side log capture of the
child's stdout/stderr still flows through the redacting
`logging.NewWriter` boundary (story 058). The token never reaches the
aps process's stdout, stderr, or logger sinks; the child can do
whatever it wants with the value internally.

`--no-redact` and `APS_DEBUG_NO_REDACT=1` continue to bypass redaction
for break-glass diagnosis.

## Acceptance Scenarios

1. **Given** any profile, **When** I run
   `aps run <profile> --env FOO=bar -- env`, **Then** the child
   environment contains `FOO=bar`.
2. **Given** a dotenv file at `/tmp/x.env`, **When** I run
   `aps run <profile> --env-file /tmp/x.env -- env`, **Then** the
   child sees every `KEY=VALUE` entry from the file.
3. **Given** `--env-file` and `--env` both set the same key, **Then**
   the `--env` value wins.
4. **Given** two `--env FOO=…` flags, **Then** the second value wins.
5. **Given** `--env FOO=…` overrides a profile-injected variable,
   **Then** the override wins.
6. **Given** `--env FOO` (missing `=`), **Then** the command fails
   before exec with an error guiding the user to `KEY=VALUE`.
7. **Given** `--env-file /missing.env`, **Then** the command fails
   with the missing path quoted in the error.
8. **Given** a dotenv file with a malformed line on line N, **Then**
   the error names the file and the line number N.
9. **Given** `--env OPENAI_API_KEY=<sk-…>`, **When** the child runs
   `env`, **Then** the secret bytes do not appear on aps's stdout or
   stderr; the env-line shape stays visible with a redaction tag.

## Tests

### Unit
- `internal/core/env_overrides_test.go` (parser + precedence)

### E2E
- `tests/e2e/run_env_override_test.go` (flag wiring, precedence,
  validation, redaction)

## Dependencies

- Story 002 — generic command execution.
- Story 058 — default child-process output redaction (no new redact
  wiring required; the existing `stdioWriter` boundary catches override
  values automatically).
