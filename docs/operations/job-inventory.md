# Job Runner Inventory

Audit of raw goroutine and `time.Ticker` callsites in the `aps` codebase,
with a recommended `kit/runtime/job` primitive for each. The categories
follow the 2026-05-04 reliability audit (§3): retry/backoff loops,
subscriber goroutines, and scheduled sweeps.

Scope: production code only (test helpers and `_test.go` excluded; the
in-test `go func` sites in `tests/e2e/*`, `internal/testing/helpers.go`,
and the protocol/isolation `*_test.go` files are deliberately out of
scope — they exercise concurrency, they do not own daemons).

Primitive vocabulary (see `kit/go/runtime/job`):

- **Job** — a single unit of work with retry/backoff. Enqueue once,
  handler runs to terminal succeeded/failed.
- **Service** — the backend that stores and routes jobs. For aps the
  target is `durabletask` (SQLite at `xdg.DataDir("aps") + "/jobs.db"`).
- **Poller** — long-running claim loop that drains a queue and dispatches
  to handler funcs by job Type. Replaces ad-hoc `for { ... sleep }`
  daemons.
- **Bus subscriber** — `runtime/bus` (kit-managed) is the right fit when
  the work is in-process, fire-and-forget, and durability is not needed.
  Listed here when a site is misclassified as a job candidate.

## Summary

| Category | Sites | Migration target |
|---|---:|---|
| Scheduled sweeps (ticker-driven) | 1 production daemon (+ 2 CLI poll loops) | Poller (sweeps) / leave as-is (CLI) |
| Subscriber / listener daemon | 1 (`aps listen`) | Already uses `bus.SubscribeAsync` — no migration |
| Retry/backoff loops | 0 in current code | n/a |
| Fire-and-forget action runs | 1 (`runs.create` background) | Job (`action.run` type, durabletask) |
| HTTP server lifecycle goroutines | 7 (serve, a2a, acp, mobile, service) | Out of scope — `http.Server.Serve` + `Shutdown` is the idiomatic Go server lifecycle; not a job runner concern |
| Process-wait goroutines | 8 (isolation/*, adapter/manager, voice, terminal, protocol/core, execution) | Out of scope — `cmd.Wait` in a goroutine is the standard exec pattern; not a job-runner concern |
| Stdin-pipe writers | 4 (protocol/core, execution, isolation/*) | Out of scope — local one-shot pipe writes |
| Streaming/keepalive | 2 (sse, chat/engine) | Out of scope — per-request streams, no daemon lifecycle |

**Net migration surface in Phase 1 (T-0472):** two callsites.

The audit framing — "webhook + listener daemons under
`internal/core/webhook.go` and adapter handlers" — needs correction.
`internal/core/webhook.go` is a synchronous `http.Handler`; it has no
retry loop and no goroutine. `internal/cli/listen.go` is already on
`bus.SubscribeAsync` (kit-managed). The actual reliability win is in
the **session reaper** (a real ticker-driven sweep) and the
**Agent Protocol background runs** (a fire-and-forget `go func()` with
no durability or observability). Detailed below.

## Migration targets

### 1. Session reaper — ticker-driven scheduled sweep

**File:** `internal/core/session/registry.go:147–171` (`startReaper`)

**Current shape:**

```go
go func() {
    ticker := time.NewTicker(tick)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            expired, err := r.CleanupInactive(DefaultTimeout)
            ...
        }
    }
}()
```

Spawned exactly once from `GetRegistry` (`sync.Once`) with
`context.Background()`. Runs for process lifetime, removes session
records whose `LastSeenAt` is older than `DefaultTimeout`.

**Primitive:** `Poller` driving an `aps.session.sweep` job type.

**Migration sketch:**

- Replace the `startReaper` goroutine with a kit `Poller` that claims
  jobs from queue `session-sweep`.
- A small "ticker" produces one `session.sweep` enqueue per
  `ReaperTickInterval` (or use `ScheduledAt` to chain the next sweep
  from the handler — kit pattern; cron-style work doesn't need a
  separate ticker).
- Handler invokes `r.CleanupInactive(DefaultTimeout)` and emits the
  same log lines.

**Wins:** observable state per sweep (kit publishes `job.succeeded` /
`job.failed`), graceful shutdown via the Poller's ctx, dead-letter
event if cleanup repeatedly fails, sweep history queryable via
`Service.List`.

**Risk:** the reaper currently runs unconditionally on `GetRegistry`
via `sync.Once`. Moving to a Poller means the durabletask Service must
be wired before the first registry access. Recommend lazy-init the
Service alongside `GetRegistry`, or move the Service to the root cobra
command's `PersistentPreRun` like `eventBus`.

### 2. Agent Protocol background run — fire-and-forget action exec

**File:** `internal/adapters/agentprotocol/runs_advanced.go:11–44`
(`handleRunsCreateBackground`)

**Current shape:**

```go
go func() {
    _, _ = a.core.ExecuteRun(r.Context(), input, nil)
}()
w.WriteHeader(http.StatusAccepted)
```

The HTTP handler returns `202 Accepted` immediately while a goroutine
spawned in the request scope runs the action. Three concrete problems:

1. `r.Context()` is cancelled when the handler returns — the spawned
   goroutine inherits a context that is already (or imminently)
   cancelled. The current `ExecuteRun` mostly ignores ctx-cancel at
   the right moments, but this is latent breakage.
2. Result is discarded — `_, _ =`. There is no way for the caller to
   poll status; the run state lives only in `APSAdapter.runRegistry`
   (in-process map) until process exit.
3. No retry, no observability, no durability.

**Primitive:** `Job` with type `action.run` on a durable backend.

**Migration sketch:**

- `handleRunsCreateBackground` enqueues a job with payload
  `{profile_id, action_id, thread_id, input}` and returns the job ID
  (or the runID, with the job stamped as Metadata).
- A `Poller` claims and dispatches to an `action.run` handler that
  calls `core.RunAction` (the same code path as `aps run`).
- `handleRunsWaitExisting` / `handleRunsStreamExisting` can read job
  status from the durabletask Service, surfacing real lifecycle.

**Wins:** survives process restart (durabletask persists), retryable
(`MaxAttempts` per enqueue), structured failure mode.

**Risk:** `ExecuteRun` currently writes stdout to a streaming
`StreamWriter` for the foreground path. The background path passes
`nil` for the stream — that contract holds under Job execution too.

## Bus subscriber — already idiomatic

**File:** `internal/cli/listen.go:117–181` (`listen`)

Uses `eventBus.SubscribeAsync(p, handler)`. The audit called out
listener daemons as a candidate, but kit's bus subscriber pool already
provides the goroutine management. **No migration.** If a future
listener needs durability (replay across restarts), that's a bus
backend concern (`runtime/bus` with a persistent provider), not a job
concern.

## Webhook server — not a retry loop

**File:** `internal/core/webhook.go`

The audit text reads "webhook delivery — retry/backoff loop", but the
current implementation is **synchronous and inbound only**: aps
receives webhooks, looks up `event → profile:action`, and calls
`RunAction` in the request goroutine. There is no outbound delivery,
no retry queue, no goroutine spawned per request.

**If/when aps gains outbound webhook delivery** (e.g. firing events to
configured subscribers), that path should be a Job from day one —
queue `webhooks`, type `webhook.deliver`, backoff strategy
`{Initial: 1s, Max: 5m, Factor: 2.0, Jitter: 0.5}` per the kit
runtime-job reference. Not in scope for T-0472.

## Out-of-scope sites (intentional)

For audit completeness — these `go func` / ticker callsites are
**not** migration candidates. They're documented here so reviewers can
quickly confirm nothing was missed.

### HTTP server lifecycle (7 sites)

`internal/cli/serve.go:144`, `internal/cli/service/status.go:536`,
`internal/cli/acp/server.go:86`, `internal/a2a/server.go:118,123,347`,
`internal/acp/server.go:271,279`, `internal/core/adapter/mobile/server.go:142,162`

All follow the canonical Go pattern:

```go
go func() {
    if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
        log.Error(...)
    }
}()
go func() {
    <-ctx.Done()
    server.Shutdown(...)
}()
```

This is the documented `net/http` server lifecycle. Wrapping it in a
Job adds indirection without value — `http.Server` already exposes
graceful shutdown, and the listener is bound to a fixed port. Leave
as-is.

### Process-wait goroutines (8 sites)

`internal/core/adapter/manager.go:240`,
`internal/core/isolation/docker.go:202`,
`internal/core/isolation/process.go:256`,
`internal/core/isolation/linux.go:268`,
`internal/core/isolation/darwin.go:256`,
`internal/core/isolation/windows.go:223`,
`internal/voice/backend.go:190`,
`internal/acp/terminal.go:129`,
`internal/core/protocol/core.go:138`

Pattern: `go func() { cmd.Wait(); ...cleanup }()`. The standard
`exec.Cmd` idiom for non-blocking subprocess management. Job-runner
abstractions don't help here — the work is a single `Wait()` call,
not a unit that can fail-and-retry.

### Stdin-pipe writers (4 sites)

`internal/core/protocol/core.go:210`,
`internal/core/execution.go:327`,
`internal/core/isolation/process.go` and the platform variants.

Pattern: `go func() { defer pipe.Close(); pipe.Write(payload) }()`.
Required because writing to a stdin pipe can block; the parent
needs to keep reading from the child. Local, one-shot, no retry needed.

### Per-request streams (2 sites)

`internal/adapters/agentprotocol/sse.go:111` (`SSEWriter.KeepAlive`):
ticker emits SSE comments for the duration of one HTTP response.
Lifecycle is bound to `s.done` (per-request channel). Not a daemon.

`internal/cli/chat/engine.go:56,86`: streaming reply chunk producer
for one chat turn. Bound to the request context, closed when the
caller stops consuming.

### CLI poll loops (2 sites)

`internal/cli/workspace/activity.go:184` (`activity --follow`) and
`internal/cli/adapter/pair.go:288` (`adapter pair` waiting room).
Foreground TTY commands that poll until SIGINT or completion. Not
daemons — they own no state that survives process exit.

### Lifecycle goroutines on context (3 sites)

`internal/a2a/server.go:118`, `internal/cli/acp/server.go:86`,
`internal/core/adapter/mobile/server.go:162`: simple `<-ctx.Done() →
stop()` shims. Standard wiring, not a job concern.

## Phase 2 alignment (T-0661)

T-0661 frames the broader migration: "kit/runtime/job with Service +
Poller pattern, durabletask SQLite backend". The two Phase 1 sites
above already adopt that exact pattern. When T-0472 lands, T-0661 is
satisfied insofar as the "first production adopter" milestone is
reached. The remaining T-0661 work is:

1. Documentation update: the `Adoption pull` section of
   `~/.ops/docs/kit-conventions/reference/runtime-job.md` should
   reference the aps adoption.
2. Operator-impact entry: durabletask introduces a new on-disk file
   (`$XDG_DATA_HOME/aps/jobs.db`) — operators need this in release
   notes.

If the migration in T-0472 leaves the session reaper or the agent-
protocol background runner outside the Service + Poller + durabletask
pattern, T-0661 picks up the gap. Otherwise T-0661 closes with a
note pointing at the T-0472 commits.

## Recommended sequence

1. **T-0472a** — Wire a singleton `job.Service` (durabletask, SQLite
   at `xdg.DataDir("aps") + "/jobs.db"`) into the root cobra command,
   alongside `eventBus`. Add to `drainBus` (rename to `drainRuntime`?)
   so `Close()` runs on shutdown.
2. **T-0472b** — Migrate `runs_advanced.go` background run to a
   `action.run` Job. Status endpoints read from the Service. This is
   the user-visible win — `runs.wait` / `runs.stream` start working
   for background runs.
3. **T-0472c** — Migrate `startReaper` to a Poller-driven sweep. Drop
   the goroutine in `startReaper`; the singleton wiring from (a) is
   reused.
4. **T-0661** — Doc update + release notes; close as satisfied by
   T-0472 if the implementation followed Service + Poller +
   durabletask.

End of inventory.
