# Agent Lifecycle

> **Status**: The gaps originally documented in this audit have been
> resolved by the `address-gaps-20260407` track (commits 870c78c..bd00a23).
> This document is now a reference for the current session and profile
> lifecycle; see git history for the fix commits.

Internal reference for how APS profiles and sessions are born,
transitioned, and torn down. Covers the actual state model in
`internal/core/session/registry.go` and the callers that move state.

Audience: contributors modifying lifecycle code, debugging stuck
sessions, or implementing new isolation backends.

## Two lifecycles, one relationship

APS has **two distinct lifecycles** that often get conflated:

1. **Profile lifecycle** — config object, filesystem-backed,
   long-lived, no runtime state.
2. **Session lifecycle** — runtime instance of a profile, in-memory
   + registry-backed, short-to-medium-lived.

**Relationship**: one profile has 0..N sessions. A profile can
exist forever with zero sessions. A session cannot exist without a
profile.

```
Profile (config on disk)
   │
   ├── Session A  (active, tmux-backed, isolation:process)
   ├── Session B  (inactive, awaiting cleanup)
   └── Session C  (active, via A2A protocol)
```

---

## Profile lifecycle

### States

Profiles have **no explicit status field**. Existence is the state.
A profile either exists on disk (under `<data>/profiles/<id>/`) or
it does not.

### Transitions

| Transition          | Code path                                         | CLI command              |
| ------------------- | ------------------------------------------------- | ------------------------ |
| Create              | `core.CreateProfile` (profile.go:343)             | `aps profile new`        |
| Save / update       | `core.SaveProfile` (profile.go:315)               | various                  |
| Add capability      | `core.AddCapabilityToProfile`                     | `aps profile capability add` |
| Remove capability   | `core.RemoveCapabilityFromProfile` (profile.go:447) | `aps profile capability remove` |
| Link workspace      | `core.SetProfileWorkspace`                        | `aps profile workspace set` |
| Export              | profile_export.go                                 | `aps profile share`      |
| Import              | profile_bundle.go                                 | `aps profile import`     |
| Delete              | `core.DeleteProfile` (profile.go:476)             | `aps profile delete`     |

### Delete semantics

`core.DeleteProfile(id, force)` removes the profile directory
under `<data>/profiles/<id>` after checking the session registry
for non-terminal sessions tied to the profile. Without `--force`,
delete refuses if any active session exists. With `--force`, the
profile is removed and active sessions are orphaned — they keep
running but lose their profile context. Operators should normally
terminate sessions first via `aps session terminate <id>` and
then run `aps profile delete <id>`.

The CLI wrapper (`internal/cli/profile.go` `profileDeleteCmd`)
prompts for interactive confirmation unless `--yes` is passed.

---

## Session lifecycle

### States

Defined in `internal/core/session/registry.go:24-28`:

```go
const (
    SessionActive   SessionStatus = "active"
    SessionInactive SessionStatus = "inactive"
    SessionErrored  SessionStatus = "errored"
)
```

### State diagram (actual, not aspirational)

```
         ┌─────────────┐
         │             │
         │  (nothing)  │
         │             │
         └──────┬──────┘
                │
                │ Register()
                ▼
         ┌─────────────┐
         │             │
         │   active    │◄──┐
         │             │   │ attach/detach
         └──┬───────┬──┘   │ (no state change)
            │       │
            │       │ UpdateStatus(SessionErrored)
            │       │ (isolation teardown failure)
            │       ▼
            │  ┌─────────────┐
            │  │             │
            │  │   errored   │ (terminal; persists in registry
            │  │             │  until `aps session delete`)
            │  └──────┬──────┘
            │         │
            │         │ aps session delete
            │         ▼
            │  ┌─────────────┐
            │  │  (removed)  │
            │  └─────────────┘
            │
            │ UpdateStatus(SessionInactive)
            │ (from `aps session terminate`)
            ▼
         ┌─────────────┐
         │             │
         │  inactive   │
         │             │
         └──────┬──────┘
                │
                │ Unregister()
                ▼
         ┌─────────────┐
         │             │
         │  (removed)  │
         │             │
         └─────────────┘
```

### The `errored` state — wired from isolation teardown failures

Since the address-gaps-20260407 track (T3), isolation backend
cleanup paths transition the session to `SessionErrored` when
`tmux kill-session` returns a non-benign error during teardown.
See `internal/core/isolation/{process,darwin,linux}.go`
`cleanupTmux()`. Benign tmux errors ("can't find session",
"no server running", "no sessions", "error connecting to") still
go through the clean-shutdown path that calls `Unregister`.

Errored sessions are **left in the registry** so operators can
observe them via `aps session list`. They do not auto-transition
further; an operator must run `aps session delete <id>` after
investigating to remove the entry.

The benign-error check is centralized in
`internal/core/session/tmux.go` (`IsBenignTmuxError`) and shared
between the isolation backends and `cli/session/terminate.go`.

### Heartbeat is wired from the ACP dispatcher

`SessionRegistry.UpdateHeartbeat` (`registry.go:233`) refreshes
`LastSeenAt` for a session. Since T4, the ACP server dispatcher
calls `APSAdapter.HeartbeatSession`
(`internal/core/protocol/core.go:348`) on every session-scoped
JSON-RPC request via `internal/acp/server.go:222-228`, gated by
the `sessionScopedMethods` set (`session/prompt`, `session/cancel`,
`session/set_mode`, `session/load`, etc.). Heartbeat failures are
logged but do not block dispatch — the downstream handler returns
its own not-found error if the session is genuinely gone.

`LastSeenAt` is now updated by:

- `Register()` — on creation
- `UpdateStatus()` — on `aps session terminate`
- `UpdateHeartbeat()` — on every session-scoped ACP request

The reaper (see "CleanupInactive is scheduled by a background
reaper" below) reads `LastSeenAt` to decide which sessions are
stale enough to remove.

### CleanupInactive is scheduled by a background reaper

`CleanupInactive(timeout time.Duration)` removes sessions whose
`LastSeenAt` is older than `timeout` and persists the result.
`DefaultTimeout` is 30 minutes.

A reaper goroutine is spawned from `GetRegistry()` inside the same
`sync.Once` that loads the registry from disk. It ticks every
`ReaperTickInterval` (5 minutes) and calls
`CleanupInactive(DefaultTimeout)` on each tick. The two intervals
are deliberately separate: the tick controls how often we wake up,
the timeout controls how stale a session must be before it is
eligible for reaping.

The production reaper runs with `context.Background()` and is
torn down by process exit — APS is a long-running CLI with no
natural cancellation point. Tests that need to exercise the
reaper call `startReaper(ctx, r, tick)` directly with their own
cancellable context and a short tick on a `NewForTesting()`
registry, so they can stop the goroutine cleanly without leaking
into other tests.

### Transitions that actually happen

| Transition                      | Trigger                                             | Code location                          |
| ------------------------------- | --------------------------------------------------- | -------------------------------------- |
| (none) → `active`               | isolation adapter starts session                    | `isolation/process.go:199`, `darwin.go:222`, `linux.go:234` |
| (none) → `active`               | A2A protocol creates session                        | `core/protocol/core.go:313`            |
| `active` → `active` (re-attach) | CLI                                                 | `cli/session/attach.go`                |
| `active` → `active` (detach)    | CLI                                                 | `cli/session/detach.go`                |
| `active` → `inactive`           | `aps session terminate`                             | `cli/session/terminate.go:56`          |
| `*` → (none)                    | `aps session delete`                                | `cli/session/delete.go:58`             |
| `*` → (none)                    | isolation adapter cleanup                           | `isolation/process.go:211`, `darwin.go:289`, `linux.go:301` |

There are **two Unregister call sites** — CLI delete and
isolation cleanup — but `Unregister` is idempotent (T0,
`registry.go:157-171`): it `delete`s from the map without
checking existence and persists the result. Whichever caller
runs second gets a clean no-op, not an error. Callers that need
guaranteed teardown should still call `terminate` first, then
`delete` after observing `inactive` status, so the isolation
backend has a chance to kill the process tree before the
registry entry disappears.

---

## The `SessionInfo` struct

Defined at `registry.go:36-51`:

```go
type SessionInfo struct {
    ID          string            `json:"id"`
    ProfileID   string            `json:"profile_id"`
    ProfileDir  string            `json:"profile_dir,omitempty"`
    Command     string            `json:"command"`
    PID         int               `json:"pid"`
    Status      SessionStatus     `json:"status"`
    Tier        SessionTier       `json:"tier,omitempty"`
    TmuxSocket  string            `json:"tmux_socket,omitempty"`
    TmuxSession string            `json:"tmux_session,omitempty"`
    ContainerID string            `json:"container_id,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    LastSeenAt  time.Time         `json:"last_seen_at"`
    Environment map[string]string `json:"environment,omitempty"`
    WorkspaceID string            `json:"workspace_id,omitempty"`
}
```

Fields to note:

- **`PID`** — the tmux session's PID on Unix backends. Not the
  PID of the interactive shell inside tmux. Killing this PID
  tears down the whole tmux session.
- **`TmuxSocket`** — populated by process/darwin/linux isolation
  backends. Empty for A2A-protocol-created sessions.
- **`TmuxSession`** — populated by all three isolation backends
  (`internal/core/isolation/{process,darwin,linux}.go`
  `registerSession`). The legacy `Environment["tmux_session"]`
  entry is preserved for backwards compatibility. The
  `tmuxSessionName` helper at `cli/session/terminate.go` resolves
  via fallback chain `TmuxSession` → `Environment["tmux_session"]`
  → `ID`.
- **`ContainerID`** — reserved for container isolation; currently
  unused in production paths.
- **`Tier`** — `basic`/`standard`/`premium`. Set to `TierStandard`
  by all current isolation backends. No code branches on tier yet.
- **`LastSeenAt`** — updated by `UpdateHeartbeat` on every ACP
  session-scoped request (T4). Reflects real activity. The
  reaper (T5) uses this for staleness checks.

---

## Persistence

The session registry is a singleton with disk-backed JSON:

- Location: `<data>/sessions/registry.json`
- Load: `LoadFromDisk()` called once in `GetRegistry()` via
  `sync.Once`
- Save: every mutator (`Register`, `Unregister`, `UpdateStatus`,
  `UpdateHeartbeat`, `UpdateSessionMetadata`, `CleanupInactive`)
  calls `saveToDiskLocked()` while holding the write lock (T1).
  On persistence failure, the in-memory mutation is rolled back
  before the error is returned to the caller, so the in-memory
  view stays consistent with disk.

`SaveToDisk()` (the public, lock-acquiring variant) remains
available for callers that need to flush after a manual edit,
but normal mutators do not require it.

---

## Registering from a new isolation backend

Checklist when adding a new isolation adapter (container, VM,
remote, etc.):

1. After successfully starting the sandbox, construct a
   `SessionInfo` with:
   - `ID` — deterministic from your backend (tmux session name
     is the current convention)
   - `ProfileID` + `ProfileDir` — from the `context`
   - `Status: session.SessionActive`
   - `Tier` — pick one; `TierStandard` is the current default
   - Backend-specific fields (`ContainerID`, `TmuxSocket`, etc.)
   - `CreatedAt` + `LastSeenAt` = `time.Now()`
2. Call `registry.Register(sess)`. On error, tear down the
   sandbox — a failed register means you leaked the sandbox.
3. In your cleanup path, on a clean teardown call
   `registry.Unregister(id)` (idempotent — safe if the CLI
   already deleted it). On a real teardown failure, call
   `registry.UpdateStatus(id, session.SessionErrored)` BEFORE
   any unregister so the errored state is observable via
   `aps session list`. Operators remove errored entries with
   `aps session delete <id>` after investigation.
4. Use `session.IsBenignTmuxError` to distinguish benign
   already-dead tmux errors from real teardown failures so you
   don't mark sessions errored on a benign race.

---

## Debugging stuck sessions

### Symptom: `aps session list` shows `active` but nothing runs

The session's isolation backend crashed without calling
`Unregister`, or the ACP client stopped sending requests so
`UpdateHeartbeat` never fired. `LastSeenAt` is stale. The reaper
(see "CleanupInactive is scheduled by a background reaper") will
eventually remove it once `LastSeenAt` exceeds `DefaultTimeout`
(30 minutes). To remove it immediately:

```bash
aps session delete <id>
```

### Symptom: `aps session terminate` returns a joined error but registry shows `inactive`

Since commit a4d4720 (T2), `terminate` always reaches
`UpdateStatus(SessionInactive)` even on partial teardown failure
(`cli/session/terminate.go:87-96`). Step errors are collected via
`errors.Join` and returned alongside the status update.

**Cause**: tmux was already dead when we tried to kill it (e.g.
crashed), or the recorded PID was stale. The graceful poll
(`waitForProcessExit`, `terminate.go:166-196`) escalates to
SIGKILL on timeout, so a live PID will be reaped; a stale PID
returns ESRCH and is treated as already-exited.

**Action**: read the joined error text. If the process and tmux
session are genuinely gone, run `aps session delete <id>` to
drop the registry entry. If something is still running, re-run
with `--force` to SIGKILL the recorded PID immediately
(`terminate.go:52-64`).

### Symptom: session re-appears after `delete`

Should not happen post-T1 — every mutator persists under the
write lock and rolls back on disk-write failure. If you observe
this, suspect a stale process holding an old in-memory registry
(e.g. a long-lived background process started before the on-disk
state changed) or a second `<data>` directory shadowing the
expected one. Check `APS_DATA_PATH` in the offending process.

---

## History

This document was originally written as an audit of gaps in the
session and profile lifecycle. All seven documented gaps —
missing `DeleteProfile`, dead `SessionErrored`, dead
`UpdateHeartbeat`, unscheduled `CleanupInactive`, non-persisting
mutators, broken `terminate` teardown, and racy `Unregister`
call sites — were resolved by the `address-gaps-20260407` track
(commits `870c78c..bd00a23`). See `git log` on that range for
the per-task fix commits.

---

## References

- `internal/core/session/registry.go` — state model + registry
- `internal/core/session/tmux.go` — `IsBenignTmuxError` helper
- `internal/core/isolation/process.go:170-212` — darwin/linux
  process isolation Register/Unregister
- `internal/core/isolation/darwin.go:222,289` — macOS-specific
- `internal/core/isolation/linux.go:234,301` — Linux-specific
- `internal/core/protocol/core.go:313,361` — A2A protocol path
- `internal/cli/session/attach.go`, `detach.go`, `terminate.go`,
  `delete.go` — CLI commands
- `internal/core/profile.go:315,343,447` — profile lifecycle
  (create/save/modify; no delete)
- `internal/core/assets/docs/SESSIONS.md` — older internal doc
  with example usage; cross-reference against current registry
  signatures before copying snippets
