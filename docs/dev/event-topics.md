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
| `aps.webhook.received`  | RESERVED  | WebhookReceivedPayload   | NOT YET WIRED — see §4.1                   |
| `aps.action.ran`        | RESERVED  | ActionRanPayload         | NOT YET WIRED — see §4.1                   |

### 4.1 Reserved-but-not-emitted

Two constants exist in `events.go` so subscribers can register
handlers ahead of emit wiring landing. Per T-0093 audit (kit reorg
adoption, domain-mapping.md):

- `aps.webhook.received` — webhook handlers live in
  `internal/cli/webhook/`; no unified post-receive hook yet.
- `aps.action.ran` — actions dispatched through
  `internal/core/action.go`; no unified post-run hook yet.

Subscribers SHOULD register but MUST NOT assume delivery until
emit sites land. Track via T-0098 follow-ups.

### 4.2 Payload field reference

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
| WebhookReceivedPayload   | ProfileID, Event, Source — RESERVED                                   |
| ActionRanPayload         | ProfileID, ActionID, ExitCode — RESERVED                              |

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
