# aps domain model fit — kit/go/runtime/domain

Last run: 2026-04-29
Task: T-0355
Author: jad

Survey of every aps entity against the kit `domain` package surface.
Identifies which abstractions (Entity, Repository, Service,
StateMachine, Validator, AuditRepository) provide value vs which add
noise. Drives subsequent adoption tasks T-0356, T-0357, T-0358.

## kit/go/runtime/domain surface

| Type / func | Purpose |
|---|---|
| `Entity` | `interface { GetID() string }` — base constraint |
| `Repository[T Entity]` | generic CRUD: Create, Get, List, Update, Delete |
| `Service[T Entity]` | wraps Repository with optional Validator + AuditRepository + EventPublisher |
| `Query` | shared list/search params: Limit, Offset, Sort, Search |
| `Validator[T]` | `Validate(ctx, T) error` |
| `AuditRepository` | append-only entity audit trail |
| `EventPublisher` | `Publish(ctx, topic, source, payload) error` (already adopted in phase 5) |
| `StateMachine` | rule-checked transitions + pre/post bus events |
| Sentinels | `ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrInvalidTransition` |

## Entity inventory

| Entity | File | ID field | CRUD shape today | domain.Entity? | Repository? | StateMachine? | Validator? | AuditRepo? | Adopt? |
|---|---|---|---|---|---|---|---|---|---|
| `Profile` | `internal/core/profile.go` | `ID string` | `CreateProfile / GetProfile / ListProfiles / SaveProfile / DeleteProfile` (file-backed YAML) | yes (add `GetID()`) | partial — package-level funcs, not a repo type | no — `Profile` has no lifecycle states | yes — many implicit invariants today (ID format, email, accounts) | no | **yes (T-0356)** — start with Entity only; defer repo until file-backed repo is generalised |
| `SessionInfo` | `internal/core/session/registry.go` | `ID string` | `Register / Unregister / Get / List / UpdateStatus / UpdateHeartbeat / CleanupInactive` on `SessionRegistry` | yes (add `GetID()`) | shape matches Repository but registry has bespoke methods (`UpdateStatus`, `UpdateHeartbeat`) — wrap, don't replace | **yes** — `active → inactive`, `active → errored`, `inactive → active` (T-0357) | maybe — heartbeat freshness etc; defer | no | **yes (T-0357)** — adopt StateMachine for status transitions; keep registry CRUD bespoke |
| `Workspace` | `internal/core/collaboration/workspace.go` | `ID string` (uuid) | `WorkspaceManager.Create / Get / List / Delete` over collaboration.Storage | yes (add `GetID()`) | strong fit — Manager already mirrors Repository | no — workspace lifecycle is open-ended, not state-machine | yes — name/owner invariants | could be useful (audit log already exists separately) | defer (out of phase 6 scope) |
| `Capability` | `internal/core/capability/types.go` | `Name string` (no ID) | install/list/remove via filesystem walk | yes (use `Name` as ID) | partial — could front a sqlstore-backed cache | no | yes — kind/type validation | no | **partial (T-0358)** — introduce `sqlstore` for the capability cache (key=name, value=struct) rather than re-walking the filesystem each invocation; full Repository wrapping deferred |
| `Bundle` | `internal/core/bundle/types.go` | `Name string` | yaml-loaded only, no mutation | yes (use `Name`) | low value — bundles are read-only configs | no | yes — version, extends loop detection | no | defer — read-only config doesn't benefit from Repository |
| `Action` | `internal/core/action.go` | `ID string` | `LoadActions / GetAction / SaveAction / DeleteAction` per profile | yes (already has `ID`) | medium fit — but per-profile scope adds a key-prefix the generic Repository doesn't model | no | yes (command, args) | no | defer — needs profile-scoping support in domain.Repository or a custom impl; revisit after T-0358 lessons |
| `Squad` | `internal/core/squad/types.go` | `ID string` | manager-style CRUD over storage | yes | strong fit | no | yes (members non-empty) | no | defer — orthogonal to phase 6 priorities |
| `Webhook` | `internal/core/webhook.go` | `ID string` (delivery_id) | event log (append-only) | yes | repository would over-model an append-only log; sqlstore-backed kv would be a better fit | no | minimal | no | defer |

## Decisions for this phase

### T-0356 — `Profile` adopts `domain.Entity`

Trivial — add `GetID() string { return p.ID }`. Buys:

- generic Repository wiring later (when we move profiles off raw YAML files).
- usable as `[]Entity` for cross-cutting tooling (audit, listing, etc).

Out of scope: replacing the package-level `CreateProfile`/`SaveProfile`
funcs with a `Service[Profile]`. The aps profile store is currently a
filesystem walk + per-profile YAML; lifting it onto Repository is its
own track (would touch every adapter import path).

### T-0357 — `Session` adopts `domain.StateMachine`

`SessionStatus` already has three terminal-ish states. Today,
`UpdateStatus` accepts any value. With the state machine:

```text
rules := map[domain.State][]domain.State{
    "active":   {"inactive", "errored"},
    "inactive": {"active", "errored"},
    "errored":  {}, // terminal — must Unregister to leave
}
```

Wins:

- forbids invalid transitions like `errored → active` (currently silent
  bug — operators have to manually unregister + re-register).
- bus pre-hook gives subscribers veto power over status change (e.g. a
  policy gate blocking errored→active without operator action).
- `domain.ErrInvalidTransition` available to callers via `errors.Is`.

Implementation: keep `SessionRegistry.UpdateStatus` as the public seam;
delegate transition validation to a package-private `*StateMachine`
field; on error, return the `*TransitionError` wrapped in our existing
error chain. The bus publisher used by the state machine is the same
one wired in T-0353 — pre/post events fire on `domain.state.*` topics
which are already routable through the audit subscriber (T-0354) once
its translator gains a `TransitionPayload` case.

Note: a state machine adoption only fires `domain.state.pre/post-transition`
events with a `TransitionPayload`. The aps-specific
`aps.session.stopped` events emitted by `UpdateStatus` continue to fire
**in addition** — they carry the richer aps payload that subscribers
already consume.

### T-0358 — `sqlstore` for capability cache

Capabilities are currently rediscovered by walking
`~/.aps/capabilities/<profile>/...` every command invocation. Each
capability has stable metadata (Name, Type, InstalledAt, Path, Links).
That's a textbook key-value cache.

Adopt `kit/go/storage/sqlstore`:

- one db at `<aps-data>/capabilities.db`
- key = `<profile>/<capability-name>`
- value = JSON-encoded `Capability`
- TTL: short (e.g. 5 minutes) so on-disk truth still wins eventually
- migration SQL: index on `key LIKE '<profile>/%'` for per-profile listing

The slow path (filesystem walk) remains the source of truth and
re-populates the cache on miss.

## Out of scope for phase 6

- Profile migration to `domain.Repository` (would touch every YAML caller).
- Workspace, Squad, Action repository wrapping (orthogonal).
- Validator extraction (existing implicit validation works; lift later
  once we hit a contention point).
- AuditRepository for entities (the bus-subscriber audit added in
  T-0354 covers the timeline use case for now).
