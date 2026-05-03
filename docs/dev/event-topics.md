# Bus Topics Catalog — aps + ecosystem

Cross-tool index of bus topics emitted across the IC stack.
Authoritative when in conflict with downstream re-summaries.

- Status: Living catalog (v0.1)
- Author: $USER
- Related: tlc `bus-topics-spec-0.1.md`, uhp `spec/events.yaml`,
  story `051-listener-daemon`, scenario 2 (always-aware-agents)

## 1. Scope

Documents what each emitter publishes onto the shared kit/bus event
hub so cross-process subscribers (e.g. aps listener daemon) can
build against a stable contract. Authoritative source per emitter
is its `internal/events/` package or equivalent spec file — this
doc is the umbrella index.

Out of scope: transport (kit/bus picks in-proc vs dpkms hub),
auth, persistent replay (kit/bus is best-effort fan-out).

## 2. Naming convention

`<namespace>.<entity>.<action>` — lowercase, dot-separated, ASCII.

- `<namespace>` ∈ { `aps`, `tlc`, `ctxt`, `uhp` (events.yaml uses
  PascalCase event names; not bus topics yet) }
- `<entity>` — domain noun
- `<action>` — past-tense verb on entity, OR present-tense for
  message-flow events (`inbox_message_added`)

Wildcards (subscriber-side, kit/bus rules):

- `*` = single segment (`aps.profile.*`)
- `#` = zero+ trailing segments (`aps.#`, `tlc.task.#`)

## 3. Common envelope (kit/bus.Event)

| Field       | Type        | Notes                                       |
|-------------|-------------|---------------------------------------------|
| `topic`     | string      | Full topic name; subscriber match key       |
| `source`    | string      | Emitter id (`aps`, `tlc`, `ctxt`)           |
| `timestamp` | RFC3339 UTC | Set at publish via `bus.NewEvent`           |
| `payload`   | object      | Per-topic schema; JSON-encoded on the wire  |

Recommended additions (NOT yet emitted; consumers tolerant):

- `event_id` (uuid v7) — dedupe key
- `session_id` — link to originating CLI session
- `schema_version` — `"0.1"` until bumped
- `workspace_id` (string, ULID) — scopes event to a wsm workspace.
  Subscribers MAY filter by active workspace; null = global event.
  Emit-side: publishers MUST set from current aps profile's
  `WorkspaceLink.name`; if profile has no link, leave nil. Cross-ref:
  `~/.ops/docs/research/wsm-integration-audit-2026-04-30.md` (T-0179)
  and T-0192 (spec evolution).

### 3.1 Payload type erasure (cross-process)

`payload` is typed `any` in Go (`kit/bus.Event.Payload`). In-process
subscribers receive the exact Go value publishers passed to
`bus.NewEvent` (e.g. `aps.ProfileCreatedPayload`). Cross-process
subscribers (NetworkAdapter, SQLiteAdapter, dpkms hub) receive the
JSON-decoded form: objects → `map[string]any`, arrays → `[]any`,
numbers → `float64`. The publisher's Go struct type does NOT
survive the wire — by design (T-0178).

Go consumers wanting struct-typed access SHOULD re-marshal hop:

```go
raw, _ := json.Marshal(e.Payload)
var p aps.ProfileCreatedPayload
_ = json.Unmarshal(raw, &p)
```

Existing helpers like `payloadString(p, "ProfileID")` in
`tests/e2e/bus/helpers_test.go` show the alternative — type-asserted
field-by-field map access. Either pattern is fine; pick re-marshal
when struct shape is stable, map access when scanning unknown
payloads.

Non-Go consumers (Python listener, webhook bridges) decode the JSON
object directly — no special treatment needed.

Cross-ref: `kit/go/runtime/bus/event.go` (Payload doc comment), tlc
`bus-topics-spec-0.1.md §4.1`.

### 3.2 Subscriber filter pattern (workspace_id)

Multi-workspace agents (one aps profile, N active workspaces) need
listeners to drop events from non-active workspaces. Pattern:

```
// pseudocode — listener handler
profile := aps.LoadProfile(profileID)
active := profile.WorkspaceLink.Name  // "" if no link
bus.Subscribe("tlc.#", func(ctx, e) {
    ws := e.Envelope["workspace_id"]  // nil → global
    if ws != nil && active != "" && ws != active {
        return  // skip cross-workspace event
    }
    dispatch(ctx, e)
})
```

Rules:

- nil `workspace_id` → treat as global; never filtered out.
- profile with no `WorkspaceLink` → no filter; receive all.
- mismatch (`ws != active`) → drop before dispatch.
- listener route DSL (story `052-listener-routing-config`) supports
  `where: workspace == active` predicate as syntactic sugar over above.

## 4. aps topics (this repo)

Source of truth: `internal/events/events.go`. Source id: `aps`.

| Topic                   | Origin    | Payload type             | Trigger                                    |
|-------------------------|-----------|--------------------------|--------------------------------------------|
| `aps.profile.created`   | emitted   | ProfileCreatedPayload    | After profile create succeeds              |
| `aps.profile.updated`   | emitted   | ProfileUpdatedPayload    | After profile update succeeds              |
| `aps.profile.deleted`   | emitted   | ProfileDeletedPayload    | After profile delete succeeds              |
| `aps.adapter.linked`    | emitted   | AdapterLinkedPayload     | After adapter link to profile              |
| `aps.adapter.unlinked`  | emitted   | AdapterUnlinkedPayload   | After adapter unlink from profile          |
| `aps.session.started`   | emitted   | SessionStartedPayload    | After session register                     |
| `aps.session.stopped`   | emitted   | SessionStoppedPayload    | After unregister or status→inactive/errored |

### 4.1 Payload field reference

Concrete struct shapes (Go) — see `internal/events/events.go`:

| Payload                  | Fields                                                                |
|--------------------------|-----------------------------------------------------------------------|
| ProfileCreatedPayload    | ProfileID, DisplayName, Email, Department, Capabilities []string      |
| ProfileUpdatedPayload    | ProfileID, Fields []string (changed names), Department                |
| ProfileDeletedPayload    | ProfileID                                                             |
| AdapterLinkedPayload     | ProfileID, AdapterType, AdapterID                                     |
| AdapterUnlinkedPayload   | ProfileID, AdapterType, AdapterID                                     |
| SessionStartedPayload    | SessionID, ProfileID, Command, PID, Tier                              |
| SessionStoppedPayload    | SessionID, ProfileID, Reason ("unregister"\|"inactive"\|"errored")    |

## 5. tlc topics

Authoritative spec: `tlc/docs/bus-topics-spec-0.1.md`. Source id: `tlc`.
Format: `tlc.<entity>.<verb>`. Entities: `task`, `track`, `flow`. Verbs
include `created`, `updated`, `claimed`, `unclaimed`, `assigned`,
`unassigned`, `completed`, `reopened`, `deleted`, `blocked`, `unblocked`,
`status_changed`, `activated`, `started`, `step_completed`, `failed`.

Do not duplicate the catalog here — read the spec. v0.2 will rename
hyphenated legacy verbs (`status-changed`, `step-completed`) to
underscore form.

Delivery: at-least-once. Same-entity FIFO; cross-entity unordered.
Publish swallows errors (`state_updater.go:233`); audit consumers
reconcile against `task_logs`.

## 6. ctxt topics

Source of truth: `ctxt/internal/events/topics.go`. Source id: `ctxt`.

| Topic                    | Origin  | Trigger                              |
|--------------------------|---------|--------------------------------------|
| `ctxt.object.ingested`   | emitted | After object ingest pipeline         |
| `ctxt.object.updated`    | emitted | After object mutation                |
| `ctxt.object.deleted`    | emitted | After object hard-delete             |
| `ctxt.job.completed`     | emitted | Background job success terminal      |
| `ctxt.job.failed`        | emitted | Background job error terminal        |

Plugin-bus (separate from kit/bus): `ctxt.refresh.trigger`,
`ctxt.notification.create` — emitted via `pluginapi.Event` from
`internal/plugin/capability.go`. Different transport; not in scope
for cross-tool subscribers.

## 7. uhp events (NOT bus topics yet)

Source: `uhp/spec/events.yaml`. UHP exposes a hook envelope/handler
protocol — NOT a kit/bus topic stream. 36 catalog entries (SessionStart,
PreToolUse, TaskCreated, WorktreeCreate, inbox_message_*, …) deliver
to handler scripts via stdin JSON. Each declares `origin: native |
extension | derived`; adapter capabilities.yaml drives synthesis.

Bridging out of scope for v0.1. When bridged, name would be
`uhp.<category>.<event>` (e.g. `uhp.tool.pretooluse`).

## 8. Delivery + ordering guarantees

| Property                | aps    | tlc    | ctxt   |
|-------------------------|--------|--------|--------|
| At-least-once           | yes    | yes    | yes    |
| Same-entity FIFO        | yes    | yes    | yes    |
| Cross-entity ordering   | none   | none   | none   |
| Publish error swallowed | yes    | yes    | yes    |
| Cross-process via dpkms | when wired | when wired | when wired |

Consumers MUST be idempotent: kit/bus may redeliver on subscriber
panic + restart. Synthesize a dedupe key from
`(topic, source, timestamp, payload.<id_field>)` until v0.2 adds
`event_id`.

## 9. Subscribers

- aps listener daemon — story `051-listener-daemon`. Plans to
  subscribe `aps.#`, `tlc.#`, `ctxt.#` and route per profile rules
  (story `052-listener-routing-config`).
- collaboration audit subscriber —
  `internal/core/collaboration/audit_subscriber.go` subscribes
  `aps.#` and writes audit log entries.
- scenario 2 always-aware-agents — design relies on aps listener
  consuming tlc + ctxt topics for cross-tool reactions.

## 10. Versioning

- `schema_version` field expected at envelope level v0.2+.
- Add-only payload evolution: new fields OK; renames/removals
  require version bump per emitter spec.
- Topic name renames: hyphen→underscore migration tracked in tlc
  bus-topics-spec §8. aps + ctxt have no legacy hyphen forms.

## 11. References

- aps: `internal/events/{events,publisher}.go`
- tlc: `tlc/docs/bus-topics-spec-0.1.md`
- ctxt: `ctxt/internal/events/topics.go`
- uhp: `uhp/spec/events.yaml`
- kit/bus: `hop.top/kit/go/runtime/bus`
- kit reorg map: `docs/plans/2026-04-29-kit-reorg-adoption/domain-mapping.md`
