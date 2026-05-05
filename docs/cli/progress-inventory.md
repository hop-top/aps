# Progress inventory — >2s aps ops without progress feedback

Conservative audit (`T-0462`) of aps commands that block for >2s in
realistic use without emitting structured progress per
`cli-conventions-with-kit.md` §6.5.

Inputs: source @ `aps-progress-mandate` worktree
(`3235095`), `kit/console/progress` API, audit reference
`~/.ops/reviews/aps-kit-integration-audit-2026-05-04.md` §4.6.

Today aps does not import `hop.top/kit/go/console/progress`. kit/cli
already wires the active `progress.Reporter` into `cmd.Context()` —
adopters call `progress.FromContext(cmd.Context()).Emit(ctx, …)`.
No additional flag plumbing is required: `--quiet`,
`--progress-format`, and `--format json` selection already happens
inside kit/cli (see
`kit/go/console/cli/progress_reporter.go`).

## Categories

- **spinner** — indeterminate (network round-trip, peer wait)
- **progress-bar** — known total (bytes, files)
- **streamed-events** — already produces events; needs UI surface
- **stage** — multi-phase fixed steps (load → encode → transmit → ack)

## Inventory

Each row: command, file:line of the blocking call, what blocks, why
slow, candidate reporter shape, feasibility.

### Confirmed >2s candidates (real network or peer wait)

| # | Command | File:line | Blocking call | Why slow | Reporter |
|---|---------|-----------|---------------|----------|----------|
| 1 | `aps a2a tasks send` | `internal/a2a/client.go:104` (called from `internal/cli/a2a/send_task.go:62`) | `c.client.SendMessage(ctx, params)` | JSON-RPC POST to peer; peer may take seconds to enqueue/process the task before returning the first response | stage: `connect → send → ack` (spinner-style, indeterminate per phase) |
| 2 | `aps a2a tasks cancel` | `internal/a2a/client.go:165` (called from `internal/cli/a2a/cancel_task.go:41`) | `c.client.CancelTask(ctx, params)` | Peer signals running worker, may wait for soft-cancel ack | stage: `connect → cancel` |
| 3 | `aps a2a tasks subscribe` | `internal/a2a/client.go:187` (called from `internal/cli/a2a/subscribe_task.go:47`) | `c.client.SetTaskPushConfig(ctx, config)` | Peer registers webhook; full handshake | stage: `connect → register` |
| 4 | `aps a2a card fetch` | `internal/cli/a2a/fetch_card.go:43` | `http.DefaultClient.Do(req)` | Plain HTTP GET to peer's well-known card; cold-start can be slow | stage: `fetch → parse` |
| 5 | `aps device pair` (`aps adapter pair`) | `internal/cli/adapter/pair.go:286` (`waitForPairing`) | `<-ticker.C` polling for `server.ActiveConnections() > 0`; bounded by `--qr-expires` (default 15m) | Wait for human + device to scan QR and connect | spinner with elapsed; the existing 1s ticker is the perfect cadence to emit `progress.Event{Phase:"wait", Item:"device", Extra:{elapsed}}` |
| 6 | `aps run <profile> -- <cmd>` | `internal/core/execution.go:253` (`cmd.Run()` in `runCommandWithProcessIsolation`) | Subprocess execution of arbitrary user command | User-supplied command; can be a build, an LLM agent, a long-running shell. Blocks indefinitely with no aps-side feedback | streamed-events: a single `progress.Event{Phase:"exec", Item:command}` at start, `Phase:"exit", OK:&ok` at end is enough to give agents a structured envelope around the user command. Anything finer-grained belongs to the child process |

### Excluded (does NOT meet >2s threshold today)

Conservative exclusion — false positives are worse than misses.

| # | Command | File:line | Why excluded |
|---|---------|-----------|--------------|
| E1 | `aps a2a tasks send-stream` | `internal/cli/a2a/send_stream.go:21` | Returns an error today (`not yet supported by a2a-go SDK v0.3.4`). Re-evaluate when streaming lands |
| E2 | `aps directory register` | `internal/agntcy/discovery/client.go:35` | Stub: `// TODO: When github.com/agntcy/dir/client is available, call Push() here.` Returns the local OASF record without a network call; <50ms |
| E3 | `aps directory discover` | `internal/agntcy/discovery/client.go:61` | Stub: returns `[]DiscoveryResult{}` with no network call |
| E4 | `aps directory show` | `internal/agntcy/discovery/client.go:71` | Stub: generates from local profile; <50ms |
| E5 | `aps capability install` | `internal/core/capability/manager.go:34` | "For MVP: If source exists locally, copy it." No download/clone today. Local recursive copy of a capability dir is sub-second for typical sizes. Re-evaluate when remote sources land (the inline TODO calls this out) |
| E6 | `aps profile share` / `aps profile import` | `internal/core/profile_bundle.go:32,90` | Single YAML marshal/unmarshal of profile + actions list. Local file ops; <100ms for typical profiles. No archive packing |
| E7 | `aps voice service start` | `internal/voice/backend.go:91` | `exec.Command(...).Start()` returns once spawn syscall completes; no wait. Backend is a separate long-lived daemon |
| E8 | `aps webhook serve` | `internal/core/webhook.go:38` | Per-event handler latency is bounded by the user action; the server itself is just `http.Serve`. Outbound delivery (per-event POST) is not implemented in aps today — only inbound receive. The original audit text "webhook delivery" maps to `aps a2a tasks subscribe` (row 3), which IS the outbound webhook registration |
| E9 | `aps listen` | `internal/cli/listen.go` | Long-running daemon; the listener doesn't have a "completion" — progress would be misleading. Startup is bus subscribe, sub-second |
| E10 | `aps a2a tasks list` | `internal/cli/a2a/list_tasks.go` | SDK v0.3.4 returns `ListTasks not supported`; stubbed (`internal/a2a/client.go:152`) |
| E11 | `aps a2a tasks show` (`get`) | `internal/cli/a2a/get_task.go:52` | Reads from local `storage.Get`, NOT `client.GetTask`. Sub-100ms file read; the SDK's `GetTask` is unused by aps today |
| E12 | `aps upgrade` | `internal/cli/upgrade.go:35` | Delegates to `kit/go/core/upgrade.RunCLI`. kit owns the progress story for the upgrade flow |

## Top 5 sites for T-0463

Ranked by adopter benefit (`agents reading the JSONL stream` and
`humans staring at a terminal`):

1. **`aps a2a tasks send`** (row 1). The canonical >2s op that
   started the audit. Highest visibility; touched by every A2A
   integration test and demo.
2. **`aps a2a card fetch`** (row 4). Smallest blast radius — pure
   HTTP GET, easy to instrument with `connect → fetch → parse`.
   Useful proof-of-life for the wiring before tackling SDK calls.
3. **`aps a2a tasks cancel`** (row 2). Companion to `send`; same SDK
   client, same shape.
4. **`aps a2a tasks subscribe`** (row 3). Mentioned by name in the
   audit; covers the "webhook delivery" case (outbound webhook
   registration).
5. **`aps run`** (row 6). Highest UX impact — agents currently see a
   wall of silence around arbitrary user commands. A structured
   envelope (`exec` start + `exit` end with OK flag) gives every
   downstream `aps run`-shaped invocation a uniform progress contract
   without changing child stdio.

Deferred to a follow-up: row 5 (`device pair` spinner over the 1s
ticker — has a non-trivial UI today with QR code rendering and is
best left for a dedicated voice-ps sibling).

## Wiring shape

All top-5 sites use the same minimal pattern:

```go
import "hop.top/kit/go/console/progress"

r := progress.FromContext(cmd.Context())
r.Emit(ctx, progress.Event{Phase: "connect", Item: targetProfile})
…
r.Emit(ctx, progress.Event{Phase: "send", Item: targetProfile})
result, err := c.client.SendMessage(ctx, params)
ok := err == nil
r.Emit(ctx, progress.Event{Phase: "ack", OK: &ok})
```

No flag changes, no new imports beyond `kit/console/progress`. kit/cli
selects Discard / Human / JSONL based on `--quiet` and
`--progress-format`.

## Audit-spec mismatches and gaps

- The original audit line names "`a2a tasks send-stream`" as a
  >2s candidate; in reality this command is unimplemented (E1) and
  cannot be wired today. Picked `a2a card fetch` instead as the
  small-scope proof.
- "Webhook delivery >2s" maps to `a2a tasks subscribe` (peer
  registers webhook) and not to `aps webhook serve` (which is the
  inbound receiver). Documented as E8 above.
- `kit/console/progress` has no `Stage` / `Steps` primitive — the
  Reporter is a single `Emit(Event)` interface. Multi-phase wiring
  is implemented by emitting one `Event` per phase, which matches
  the §6.5 JSONL example. No kit-side gap to file.
