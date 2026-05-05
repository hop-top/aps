---
status: shipped
---

# 058 - Redact secrets and PII from aps logs and output

**ID**: 058
**Feature**: §4 item 3 (kit-integration audit 2026-05-04)
**Persona**: [User](../personas/user.md), [Operator](../personas/user.md)
**Priority**: P1
**Status**: shipped
**Author**: jadb
**Task**: T-0459, T-0460, T-0461

## Story

As an aps user collaborating with a coding agent (Claude / Cursor /
Codex / etc.), I want any secret or PII that flows through aps's logs,
adapter output, webhook responses, persisted log files, or child-
process stdio to be replaced with a tag (`<openai-api-key>`,
`<aps-bearer-header>`) by default — so the bytes never end up in the
agent's context, the CI log retention pipeline, the stdout-shipper to
Loki/Splunk, or a `tee`'d file someone shares in Slack.

This was driven by a concrete incident on 2026-05-02: a user ran
`aps run noor -- env` to debug a profile issue. The child process's
`env` listed `OPENAI_API_KEY=sk-…` to the parent terminal. The
tailing agent absorbed it into its prompt cache, transcript, and
context store. Recovery required clearing the cache and rotating the
key.

The audit at `~/.ops/reviews/aps-kit-integration-audit-2026-05-04.md`
§4 item 3 cataloged this as a generalized risk: aps logs profile
contents, adapter exec output, and webhook payloads with no redact
adoption. This story is the closing of that gap.

## Acceptance Scenarios

1. **Given** a profile with `OPENAI_API_KEY=sk-proj-<secret>` in
   `secrets.env`, **When** I run `aps run <profile> -- env`,
   **Then** stdout shows the env line with the value replaced by a
   redact tag and the raw `sk-proj-<secret>` does not appear.

2. **Given** the same profile, **When** I run
   `aps --no-redact run <profile> -- env`, **Then** stdout shows the
   raw `sk-proj-<secret>` (operator-confirmed bypass for diagnosis).

3. **Given** the same profile, **When** I run
   `APS_DEBUG_NO_REDACT=1 aps run <profile> -- env`, **Then** stdout
   shows the raw value (env-var bypass for CI/non-interactive
   contexts).

4. **Given** a log line containing `Authorization: Bearer <token>`,
   **When** the kit logger writes it to stderr, **Then** stderr
   contains `Authorization: Bearer <aps-bearer-header>` and the
   token bytes are not present (structured-log key-aware redaction).

5. **Given** a webhook handler that returns a 5xx with action
   stderr in the body, **When** the response is written, **Then**
   the JSON body has any matched secret replaced with a redact tag.

6. **Given** an adapter subprocess that echoes a secret to its
   stdout, **When** aps forwards the bytes to the persisted
   `device.Path/stdout.log`, **Then** the file contains the redact
   tag (defends against on-disk retention with the default
   `0644` mode).

7. **Given** a session whose `Environment` map carries an API key,
   **When** I run `aps session inspect <id>`, **Then** the value
   column shows the redact tag.

## Implementation Notes

Four canonical choke points (per the inventory in
[redact-inventory.md](../cli/redact-inventory.md)):

1. **Logger sink** — `internal/logging/logger.go::SetViper` wraps
   the kit logger writer with `logging.NewWriter(os.Stderr)`. Every
   `logging.GetLogger().X(...)` call is filtered.
2. **Stdout/stderr formatter** — `internal/logging/output.go` exposes
   `Print` / `Println` / `Printf` / `Fprint*` helpers. HIGH-severity
   stdout sites (env, adapter exec, a2a get-task, session inspect)
   migrated.
3. **HTTP response body** — `respondJSON` (webhook) and
   `sendError`/`sendJSON` (agentprotocol) marshal-then-redact-then-
   write.
4. **Persisted log files** — adapter subprocess `stdout.log` /
   `stderr.log` writers wrapped.
5. **Child process stdio** — `aps run` and adapter exec wrap
   `cmd.Stdout` / `cmd.Stderr` with redacting writers.

The package singleton `logging.Redactor()` lazily loads
`redact.Default()` (gitleaks corpus + Presidio PII pack), adds four
aps-domain header rules with a custom replacement that preserves
key names, and uses the Tag strategy.

The `--no-redact` flag is a kitcli Global on the root command and
inverts into the `redact.enabled` viper key in `PersistentPreRun`.
The `APS_DEBUG_NO_REDACT` env is checked on every `Apply` call so
operators can scope bypass per-command without flag plumbing.

## Tests

### E2E
- `tests/e2e/redact/redact_test.go`
  - `TestRedact_RunCommandRedactsChildEnv` (drives `aps run` over a
    secret-bearing profile)
  - `TestRedact_NoRedactFlagShowsRawValue`
  - `TestRedact_EnvBypassShowsRawValue`
  - `TestRedact_AuthorizationHeaderKeyAware`
  - `TestRedact_LoggerSinkRedactsBearerToken`

### Unit
- `internal/logging/redact_test.go` — 9 tests covering OpenAI key,
  Bearer header (key-aware), env bypass, viper key override, writer
  pass-through when disabled, allowlist fixtures.

## Dependencies

- `hop.top/kit/go/core/redact` — gitleaks + Presidio default corpus.
- `github.com/BurntSushi/toml` — TOML parser used by the rule loader
  (added to aps go.sum).
- CLI conventions §8.6 (delegation-safety persistent flag patterns).
