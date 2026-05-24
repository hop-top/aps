# Redact integration audit

Audit of `kit/core/redact` adoption across the four canonical choke points
in aps. Reference: `~/.ops/docs/kit-conventions/reference/core-redact.md`.

The integration is anchored at `internal/logging/redact.go`, which wraps
`kit/core/redact.Default()` and adds aps-domain rules (`aps-bearer-header`,
`aps-x-api-key-header`, `aps-aps-signature`, `aps-generic-bearer`) plus a
local allowlist for RFC1918, loopback IPs, and the `@example.com` /
`@hop.top` fixture domains.

## Choke point coverage

### A. Logger sink — VERIFIED COMPLETE

`internal/logging/logger.go::SetViper` (called from
`internal/cli/root.go::init`) builds the kit logger and wraps its writer
with `logging.NewWriter(os.Stderr)`. `SetSlogDefault` does the same for
the process-wide `slog.Default` so any stdlib `slog.X` call routes
through the redacting sink.

No production code constructs a logger directly. The only `log.New(...)`
call in aps lives at `internal/logging/logger.go:21` (the boot-time
default created before `SetViper` replaces it) and is unreachable once
`init` has run. No `kitlog.New` or `slog.New` outside the logging
package.

### B. Stdout/stderr formatter helpers — VERIFIED COMPLETE (HIGH-severity sites)

`internal/logging/output.go` provides `logging.Print/Println/Printf/Fprint*`
helpers that route formatted output through `Apply` before writing.

HIGH-severity stdout sites named in the redact-inventory:

| Surface | File | Status |
|---|---|---|
| `aps env` (eval-friendly export) | `internal/cli/env.go:34` | `logging.Println` |
| `aps run` child stdio (non-TTY) | `internal/core/execution.go:30-35` (`stdioWriter`) | `logging.NewWriter` |
| `aps adapter exec` action output | `internal/cli/adapter/exec.go:80` | `logging.Print` |
| `aps session inspect` env values | `internal/cli/session/inspect.go:110, 136` | `logging.Apply` + `logging.Println` |
| `aps a2a show` text parts | `internal/cli/a2a/get_task.go:110` (textual) | `logging.Apply` |
| `aps a2a show --format json` | `internal/cli/a2a/get_task.go` (JSON) | `logging.ApplyBytes` (fixed in this PR) |

LOW-severity sites that print curated values (skill metadata, version
strings, identity DID/badge, listing tables of profile names) keep
using bare `fmt.*` per the design note in `internal/logging/output.go`.

### C. HTTP response body — VERIFIED CORE COMPLETE, gaps deferred

Canonical wrapping helpers route through `logging.ApplyBytes`:

| Wrapper | Location |
|---|---|
| `respondJSON` | `internal/core/webhook.go:185` |
| `sendJSON` | `internal/adapters/agentprotocol/adapter.go:590` |
| `writeJSON` | `internal/adapters/messenger/handler.go:948` |
| `writeError` | `internal/adapters/messenger/handler.go:970` |
| `sendError` | `internal/adapters/agentprotocol/adapter.go:569` (Apply on message) |

Unwrapped `json.NewEncoder(w).Encode(...)` callsites found:

| Site | Category | Disposition |
|---|---|---|
| `internal/core/webhook.go:95` | "no event mapping" 400 path | DEFERRED — file is owned by aps-job-runner-T-0471 worktree |
| `internal/core/webhook.go:192` | fallback after `json.Marshal` failure inside the wrapper | DEFERRED (same owner; also fallback-on-error path, low blast radius) |
| `internal/core/protocol/http_bridge.go:73, 175, 195` | bridge metadata + JSONRPC echo/error | DEFERRED — see follow-up task |
| `internal/adapters/agentprotocol/adapter.go:419` | static "item stored successfully" message | DEFERRED — adapters/ owned by aps-job-runner-T-0471 |
| `internal/adapters/agentprotocol/adapter.go:595` | fallback after `json.Marshal` failure inside `sendJSON` | DEFERRED (same owner; fallback path) |
| `internal/adapters/agentprotocol/runs_advanced.go:41, 88` | "run started" + "not implemented" static messages | DEFERRED (same owner) |
| `internal/adapters/messenger/handler.go:953, 980` | fallback after `json.Marshal` failure | DEFERRED (adapters/) |
| `internal/adapters/messenger/handler.go:989` (`writeText`) | challenge handshakes (WhatsApp / Slack verification echo) | DEFERRED (adapters/, short alphanumeric echo, low leak risk) |

### D. Persisted log files — VERIFIED COMPLETE for adapter subprocess, telemetry fixed in this PR

`logging.NewWriter` wraps:

| Persisted writer | Location |
|---|---|
| Adapter subprocess `stdout.log` | `internal/core/adapter/manager.go:222` |
| Adapter subprocess `stderr.log` | `internal/core/adapter/manager.go:223` |
| Child process stdout/stderr in `aps run` | `internal/core/execution.go:30-35` |
| Skills telemetry JSONL | `internal/skills/telemetry.go:148-149` (fixed in this PR) |

Other on-disk writers reviewed:

| Site | Category | Disposition |
|---|---|---|
| `internal/core/metrics.go:49` | usage events; `RecordUsage` currently has no production callers | DEFERRED — no live leak path |
| `internal/core/messenger/audit.go:137` | capability-changes audit JSONL | DEFERRED (messenger; adapters/ owner) |
| `internal/core/messenger/logging.go:198` | per-workspace message logs | DEFERRED (messenger; adapters/ owner) |
| `internal/core/multidevice/offline_queue.go:75` | offline queue persistence | DEFERRED — sync events carry IDs not secrets |
| `internal/core/multidevice/event_store.go:124` | multidevice event store | DEFERRED — same as above |
| `internal/core/multidevice/manager.go:137` | multidevice manager state | DEFERRED — same |

## Bypass paths

### `Default()` lazy load — VERIFIED

`hop.top/kit/go/core/redact.Default()` (kit v0.4.0-alpha.4) uses a package
`sync.Once`. aps's `logging.Redactor()` wraps it in a second `sync.Once`
that adds the aps-domain rules.

`logging.Redactor()` is only called from `Apply`, `ApplyBytes`, and
`redactWriter.Write`, all of which short-circuit on `!Enabled()` before
touching the redactor. Boot-time CLI init (`internal/cli/root.go::init`)
does not call any of them, so the gitleaks/Presidio corpora load on
first redaction, not on `aps --help` or fast-fail paths.

### `--no-redact` flag — VERIFIED

Declared on the root command at `internal/cli/root.go:174`. Parsed into
`noRedactFlag`. The kit `PrePersistentRunE` hook
(`applyNoRedactToggle`) calls `logging.SetRedactEnabled(!noRedactFlag)`
which writes `redact.enabled` into the bound viper. `logging.Enabled()`
reads it.

E2E coverage: `tests/e2e/redact_test.go::TestRedact_NoRedactFlagShowsRawValue`.

### `APS_DEBUG_NO_REDACT` env — VERIFIED

`logging.Enabled()` checks `os.Getenv("APS_DEBUG_NO_REDACT")` first
(line 253) and short-circuits on any truthy value. This wins over both
the viper key and `--no-redact`.

E2E coverage: `tests/e2e/redact_test.go::TestRedact_EnvBypassShowsRawValue`.

Unit coverage: `internal/logging/redact_test.go::TestEnabled_EnvBypass`,
`TestApply_BypassReturnsRaw`, `TestNewWriter_PassesThroughWhenDisabled`.

## Fixes landed in this PR

1. `internal/cli/a2a/get_task.go` — `--format json` path now routes
   serialized `Task` bytes through `logging.ApplyBytes` so it matches
   the textual path's guarantee.
2. `internal/skills/telemetry.go::writeEvent` — appended event JSONL
   now writes through `logging.NewWriter` so `ErrorMsg` (populated
   from action failure stderr via `TrackFailure`) is redacted on disk.

## Deferred follow-up

All deferred gaps consolidated into one follow-up task on the
`aps-redact-logs` track. The fixes are mechanical (route through an
existing wrapper or wrap the `os.OpenFile` target with
`logging.NewWriter`); deferred only to avoid worktree collisions
with `aps-job-runner-T-0471` (which owns `internal/core/webhook.go`,
listener daemons, and `adapters/*`) and `aps-run-env-flag-T-0576`
(which owns `cmd/`, `internal/core/execution.go`, and
`internal/core/redact/`).

See the follow-up tlc task on the `aps-redact-logs` track for the
full callsite list and disposition.
