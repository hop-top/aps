# uma and aps Conversation History: Mapping Options

Status: decided, Option A (dual-write). The decision, the answers to the
[open questions](#open-questions) and what stays open are recorded in
[messenger-architecture.md](messenger-architecture.md#conversation-history). The text below is the
analysis as proposed; it is not rewritten. Since it was written: a host now mounts the bridge
(`internal/adapters/messenger/uma_mount.go`), the bridge tests assert history, an answered
event is not run again after a restart, and a quoted WhatsApp Cloud reply keeps its thread.

A `--type uma` message service keeps its messages in two places: the uma engine the bridge hosts,
and aps session history (`message_turns`). This page compares the ways to reconcile them and
recommends one. Background: [uma bridge](messenger-architecture.md#uma-bridge),
[conversation policy](message-conversation-policy.md),
[user guide](../user/conversations.md).

## Summary

- Keep aps history as the read model and keep writing it from the aps runtime path (dual-write).
  The bridge already composes that way; the work is hardening and tests, not a new mechanism.
- Do not build read-through over uma `Messages.List` now. With today's in-memory uma store it
  loses every conversation on restart, and `aps service conversation` runs in a different process
  from the engine and cannot reach it.
- Revisit an event-fed projection once uma has a durable store in aps and the bridge resumes its
  subscription instead of bootstrapping it.

Sections: [the problem](#the-problem), [options](#options), [comparison](#comparison),
[recommendation](#recommendation), [open questions](#open-questions),
[evidence and probes](#evidence-and-probes).

Code references are `file:line` in this tree. A bare file name lives in
`internal/core/messenger` (`history*.go`, `runtime.go`, `types.go`, `sms.go`, `whatsapp.go`),
`internal/adapters/umabridge` (`bridge.go`, `provider.go`, `catalog.go`, `authorizer.go`) or
`internal/adapters/messenger` (`router.go`, `handler.go`). References prefixed `uma:` are the uma
repository at commit `650e527`, the commit `go.mod` pins. Anything not read or run is marked
**Unverified**.

## The problem

### What the two stores hold

aps keeps one row per turn in a SQLite table, in `<data-dir>/messages/conversations.db`.
A turn carries the policy keys `msgconv:v1:...` and `msgsess:v1:...`
(`internal/core/messenger/types.go:116-163`), a direction, the provider message id, sender, text,
attachments, the routed profile and action, and a timestamp
(`internal/core/messenger/history.go:41-57`). Rows get an autoincrement `seq`, and only the newest
500 per conversation survive (`history_sqlite.go:37-57`, `:123-128`; `history.go:26`).

uma keeps conversations, contacts, messages, operations and an event log. A message has a uuidv7
`Ref.ID`, provider ids in `External`, a revision, edit, delete and reaction state, and no
direction field: a message with an `Origin` was authored through uma, which is how outbound is
recognized (`uma:model/model.go:180-237`, `uma:store/store.go:249`). uma has no per-conversation
cap. Its retention policy has no message horizon (`uma:uma.go:115-148`).

The bridge builds each engine on `memory.New` and does not require durability
(`internal/adapters/umabridge/bridge.go:125-135`). The memory store reports `Durable=false`, never
frees messages or conversations, and takes about 3-6 KB per message
(`uma:store/memory/memory.go:37-50`, `:95`; `uma:README.md:795-815`). The only other maintained
store is PostgreSQL (`uma:docs/store-backends.md`).

### Who reads history

| Reader | What it needs | Where |
| --- | --- | --- |
| `prior_turns` on action stdin | Newest N turns of one session, oldest first; N is the service option `history_turns`, default 20 | `internal/adapters/messenger/router.go:272`, `:330-350`; `handler.go:986-999`; `history.go:22` |
| `aps service conversation list` / `show` | All conversations and turns, from a separate CLI process that opens the SQLite file directly and treats a missing file as empty | `internal/cli/service/conversation.go:184-200` |
| Retention | Newest 500 turns per conversation, pruned on append | `history_sqlite.go:123-128` |
| Chat execution mode | Inbound turn recorded for the query surface; the chat runtime keeps its own transcript | `internal/adapters/messenger/handler.go:807-812` |

### Who writes it today, for a bridged service

The bridge hands `Config.History` to `messenger.Runtime` (`bridge.go:33-34`, `:145-147`), and the
runtime appends an outbound turn after delivery (`internal/core/messenger/runtime.go:424-430`,
`:436-448`). The inbound turn is appended by the aps router before the action runs
(`router.go:271-273`, `:354-365`), not by the bridge. A host that passes the same store to the
router and to `Config.History` gets both directions with no further code.

So dual-write is the existing shape. The user guide's "aps history is not yet mapped from it"
(`docs/user/messengers.md:346-347`) means uma ids are not mapped, not that nothing is recorded.
Two gaps remain. No host constructs the bridge in this tree yet (no importer of
`internal/adapters/umabridge` outside the package), and no bridge test asserts history: the
harness uses a stub executor and no store (`umabridge/bridge_test.go`).

### What breaks at cutover

These are the facts every option has to survive.

- **Restart empties uma.** `provision` and the subscription bootstrap both run on every start
  because the store is empty (`bridge.go:169-171`, `:222-225`). Conversation and message ids are
  regenerated, and provider-event de-duplication resets
  ([messenger-architecture.md](messenger-architecture.md#uma-bridge), "Not done here").
- **Keys continue only by construction.** For SMS and Twilio WhatsApp the bridge builds the same
  sender and channel strings as the native providers
  (`provider.go:68-72`, `:92`; `internal/core/messenger/sms.go:300-301`, `catalog.go:93-100`) and
  sets `messenger_name` to the service id (`provider.go:97`). The `msgconv` key therefore matches
  the native one only if the operator keeps the same service id. The key embeds the service id, so
  a new id starts a new history.
- **WhatsApp Cloud threads are dropped.** The native normalizer sets `Thread` from `context.id`
  (`internal/core/messenger/whatsapp.go:361`), so a reply is its own session. The bridge never
  sets `Thread` (`provider.go:86-103`), so every message lands in the sender's conversation-level
  session. uma keeps the reply context only in the raw payload extension
  (`uma:adapter/messaging/meta/whatsapp/webhook.go:313`).
- **Sender display name differs.** The bridge uses the address as name (`provider.go:90`); the
  native Cloud path uses the profile name (`whatsapp.go:343-348`).
- **Outbound is recorded when queued, not when sent.** The bridge's receipt is `accepted`
  (`provider.go:133-141`), and the runtime records the turn on any non-error receipt
  (`runtime.go:429-430`). The conversation policy says outbound turns are recorded "after
  successful provider delivery" ([policy](message-conversation-policy.md#conversation-store)).
- **Edits, deletes and reactions do not reach aps.** The bot subscribes to `message.received`
  only (`bridge.go:222`). For the three bridged adapters this is mostly moot: none declares
  `edit` or `delete`, and only `meta.whatsapp` reports reactions
  (`uma:docs/adapters/twilio.sms.md:41-43`, `uma:docs/adapters/twilio.whatsapp.md`,
  `uma:docs/adapters/meta.whatsapp.md`). The policy already excludes reactions without text from
  turns.

## Options

### Option A: dual-write (aps history written from the aps path)

**Data flow.** Unchanged from today. uma records the inbound message and emits
`message.received`; the bot turn normalizes it; the aps router appends the inbound turn, reads
`prior_turns` from SQLite and runs the action; the runtime delivers through `Messages.Reply` and
appends the outbound turn. uma stays source of truth for provider state, aps for the transcript
actions see.

**Ids and keys.** aps derives `msgconv`/`msgsess` from the `NormalizedMessage`, never from uma.
Mapping is by convention: uma conversation = (service, receiving number, sender), because the
bridge's inbox is per-contact with `LockToSingle` (`bridge.go:204-205`) and the bridge's
authorizer has no action that closes or resolves a conversation (`authorizer.go:22-33`), so a
sender never gets a successor conversation. A probe confirmed one conversation per sender and
`ReplyRoute.ID` equal to the sender address. The turn's `message_id` is the provider id for
inbound (`provider.go:73-79`) and the answered inbound id for outbound (`history.go:134-136`).
`PlatformMetadata` carries `uma_message` and `uma_conversation` (`provider.go:96-102`), but the
table has no column for them, and under the memory store they would dangle after a restart.

**Failure modes.**

| Situation | Behaviour |
| --- | --- |
| aps append fails | Logged, routing continues (`router.go:359-363`, `runtime.go:441-446`); uma has the message, aps does not, nothing reconciles |
| Restart, then provider redelivers an old event | uma de-dup is gone, so the action runs again and a second reply is sent; history records both. Probe: one provider event, two process lifetimes, two outbound turns |
| Same-process redelivery | uma de-duplicates; no second turn (`bridge_test.go:375`; probe) |
| Provider rejects the send | uma operation ends `rejected`, but aps already recorded an outbound turn. Probe: 1 turn, operation `rejected` |
| Crash between uma commit and aps append | Message in uma, missing in aps; moot while uma is in memory |
| Edit, delete, reaction | Never reach aps; moot for current adapters |
| Ordering | `seq` is append order. Inbound is appended when the bot turn runs, not at uma commit. **Unverified:** whether two quick messages from one sender can append out of uma recording order |

The table has no unique key (`history_sqlite.go:37-57`), so nothing makes appends idempotent. A
unique index on `(service_id, direction, message_id)` would hide the duplicate rows after a
restart but not the duplicate action run and reply, which is the real defect. The durable fix is
de-duplication that survives restart: a durable uma ingress store, or a bridge pre-check against
aps history before running the action.

**Cost.** Small. No new mechanism: a bridge-level test with a real store, a decision on queued
versus sent outbound turns, and a doc update. A pre-check or unique index is a small additive
change; the table has no migration framework (`CREATE TABLE IF NOT EXISTS` only), so any new
column needs an `ALTER`.

**Tests.** Extend the umabridge harness with a SQLite store and the real router: both directions
recorded; a second message sees the first as `prior_turns`; a new bridge on the same file sees
old turns; a rejected send follows the chosen policy; a post-restart redelivery follows the
chosen policy. Add one CLI test reading the same file.

**`aps service conversation`.** Unchanged and restart-safe: it reads the same SQLite file as for
native providers.

### Option B: read-through (aps history reads built from uma `Messages.List`)

**Data flow.** Implement `ConversationStore` over a uma client. `AppendTurn` becomes a no-op for
bridged services, since uma already recorded the message. `RecentTurns` resolves the aps key to a
uma conversation, lists its messages and maps them to turns. `ListConversations` lists
conversations and fetches each one's last message. The bot actor already holds `message.read`,
`conversation.read` and `inbox.read` (`authorizer.go:27-33`).

**Ids and keys.** The aps key must map to a uma conversation without a stored link. Probed:
`Conversations.List` returns it with `ReplyRoute.ID` equal to the sender address (the Twilio
WhatsApp `whatsapp:` prefix is stripped by uma and re-added by `catalog.go:93-100`), so the
resolver is a scan or an in-process map filled at ingress. `seq` has no uma counterpart and would
be synthesized from list position, which is unstable if rows are ever pruned. `profile_id` and
`action_name` are aps-only and would be empty unless the bridge attaches them to uma message
metadata. Session scope collapses to conversation scope, as in A. For a future bridged adapter
with per-room or per-thread selection (`uma:model/records.go:30-31`), `msgconv` (the room) would
span several uma conversations (the threads), so `RecentTurns` for a conversation would fan out.
Today's catalog is direct-message only (`catalog.go:46-118`).

**Mapping detail.** Direction is `Origin != nil` (probe). Outbound `message_id` must be rebuilt
from `InReplyTo`'s provider id to match aps's convention. Probe: a rejected reply is still listed
as a message with no provider id, so read-through must also consult operation state to avoid
showing unsent replies as assistant turns.

**Failure modes.**

| Situation | Behaviour |
| --- | --- |
| Restart (memory store) | All history gone, including for conversations that existed before. `prior_turns` empty, `service conversation` empty. A regression against native providers |
| Restart (PostgreSQL) | History survives, but the bridge must resume its subscription rather than bootstrap it again; see the note below the table |
| uma read error | Same degrade as today: empty `prior_turns`, logged (`router.go:339-345`) |
| Newest-N cost | `Messages.List` is forward-only, so the tail means paging the conversation; cost grows with total history on every inbound message. See the note below the table |
| Redelivery | Handled by uma while its dedup lives; same as A in memory |
| Edits, deletes, redactions | Reflected for free: tombstones have cleared content, redaction scrubs text |

Note on restart with PostgreSQL: the bridge opens its subscription with
`Bootstrap(StartBeginning)` (`bridge.go:222-225`) because the memory store is always empty. Against
a populated durable store that would start from the beginning of the event log and run an action
for every past message; the host package offers `Resume` with a persisted marker instead
(`uma:companions/bot/host/prepare.go:52-66`). **Unverified:** not run against PostgreSQL.

Note on newest-N cost: `Messages.List` is forward-only in recording order
(`uma:uma.go:645-668`). Newest-first exists only on the internal store port
(`uma:store/store.go:249`). The caller can bound the scan with `Since` (`OccurredAt`), but each
call still scans at most 500 stored items, and the memory store builds and orders the whole table
before cutting a page (`uma:store/memory/memory.go:25-30`).

**Cutover.** Pre-cutover native turns exist only in SQLite, so reads for a continued
conversation id must union legacy rows and uma messages and merge their order by time.

**Cost.** Large. A store adapter with per-service routing (one shared store serves every service,
and engines are per service, `messenger-architecture.md:133`), key resolution, newest-N paging,
legacy union, an operation-state filter, durable uma (PostgreSQL), a persisted subscription
marker, and a way for the CLI to reach a running engine. **Unverified:** whether any server API
could serve the CLI; none exists for conversations today (`ConversationStore` is used only by the
CLI and the messenger adapter).

**Tests.** The store adapter against the conformance-style cases in
`history_test.go`, a durable-uma restart test, a legacy-union test, a rejected-send test, and a
cost test at a few thousand messages.

**`aps service conversation`.** Breaks unless the CLI can reach the engine. The CLI is a separate
process (`conversation.go:184-200`); the memory store is unreachable from it, and a PostgreSQL
read from the CLI needs the engine's credentials and tenant setup. The practical fallback keeps
the CLI on SQLite, which brings back a second copy and so is Option A.

### Option C: event-fed projection (hybrid)

**Data flow.** aps history stays the read model, but turns are written from uma's event stream
instead of the aps runtime path. The bot subscription widens from `message.received` to
`message.created` (emitted at commit, before provider delivery, for replies and notes) and
`delivery.updated` (`uma:model/payloads.go:48-52`, `uma:model/names.go:186-190`). A projector
upserts one turn per uma message, keyed by uma message ref, and marks or drops a reply whose
operation ends `rejected`. The router stops appending inbound turns for bridged services, because
`ExecuteAction` appends unconditionally (`router.go:273`) and would double-write.

**Ids and keys.** Same aps keys as A, plus a nullable `source_ref` column with a unique index.
The ref is only stable across restarts with a durable uma store.

**Failure modes.** Idempotent upsert absorbs event redelivery; a persisted marker plus
`host.Resume` absorbs restarts; sent-versus-queued becomes explicit. It adds a window between
`message.created` and `delivery.updated`, a projector that can lag, and edit and delete handling
through `message.updated`. With the memory store it adds nothing over A, because everything
disappears at restart anyway.

**Cost.** Medium-large: schema change with `ALTER`, projector, marker persistence, router switch,
and new event handling in the bridge, which today ignores everything but `message.received`.

**Tests.** Event fixtures for each kind, redelivery and restart with a persisted marker,
rejected-send, and a no-double-write check on the inbound turn.

**`aps service conversation`.** Unchanged: it still reads SQLite.

### Option D: one-time import through uma's history capability

**Idea.** Backfill `prior_turns` after a restart or at cutover by calling `Runtime.Import`.

**Why it does not hold up.**

- Only `twilio.sms` declares `history`. `twilio.whatsapp` has it "not built" and `meta.whatsapp`
  has no provider call (`uma:docs/adapters/twilio.sms.md:116-190`,
  `uma:docs/adapters/twilio.whatsapp.md:46`, `uma:docs/adapters/meta.whatsapp.md:43`).
- It is inbound only: the number's own sends are excluded
  (`uma:docs/adapters/twilio.sms.md:122-127`), so aps would have half of every exchange. Reach is
  10,000 messages from the newest.
- The bridge's authorizer does not grant `channel.import` (`authorizer.go:22-26`).
- An imported message still emits `message.received`
  (`uma:docs/api-coverage.md:221`), and neither `handle` nor `NormalizeIngress` checks
  `ImportedBy` (`bridge.go:259-286`, `provider.go:62-103`). Importing would re-run the action and
  send a reply for every historical message. **Unverified:** not run; the documentation and the
  bridge code read the same way.

It is rejected as a mapping strategy. If a backfill is ever wanted, the bridge would first have
to skip imported messages.

## Comparison

| Concern | A: dual-write | B: read-through | C: event-fed projection | D: import |
| --- | --- | --- | --- | --- |
| `prior_turns` after restart (memory uma) | Kept (SQLite) | Lost | Kept (SQLite) | Inbound only, twilio.sms only |
| `aps service conversation` | Unchanged | Needs engine access from CLI; breaks with memory uma | Unchanged | Unchanged |
| Key continuity from native providers | Same service id keeps it; Cloud threads dropped | Same, plus legacy union needed | Same as A | n/a |
| Newest-N read cost | Indexed SQLite query | Forward scan of history per message | Indexed SQLite query | n/a |
| Retention | 500 per conversation | None in uma; unbounded | 500 per conversation | n/a |
| Redelivery after restart | Duplicate action, reply, turns | Handled by uma de-dup only if uma is durable; otherwise as A | Idempotent turns; action still repeats without durable uma | Re-runs actions for the whole window |
| Unsent or rejected replies | Recorded as turns (probe) | Listed unless operation state is checked | Explicit via `delivery.updated` | n/a |
| Edits, deletes, redactions | Not reflected | Reflected | Reflected via `message.updated` | n/a |
| Needs durable uma | No | Yes | Yes, to pay off | No |
| New code | Tests, docs, small hardening | Store adapter, resolver, union, durable setup | Projector, schema, marker, router switch | Authorizer, filter, import trigger |
| Reversible | Yes | Hard once legacy turns diverge | Yes | Yes |

## Recommendation

Adopt Option A, and treat C as the target once uma is durable.

**Reasoning.**

1. aps history is the only store that survives a restart today and is the only one the CLI
   process can read. uma is in memory by design in this bridge; every conversation and every
   id is gone on restart (probe). A read-through built on it would make bridged services strictly
   worse than native ones on the first restart.
2. The policy identity (`msgconv`/`msgsess`, thread scope, 500-turn cap, `history_turns`) is aps's
   contract with actions. Deriving it from `NormalizedMessage` keeps it independent of uma's
   conversation selection, which differs by mode and will differ more for future adapters.
3. The write path exists. The remaining decisions are policy decisions, not new mechanisms.
4. Read-through is expensive in the exact place it matters: newest-N on every inbound message
   against a forward-only list, plus a legacy union for converted services.

**Hardening to do inside A** (each is small, none is implemented here):

- Decide whether outbound turns record at queue time or on delivery settlement, and align
  [the policy](message-conversation-policy.md#conversation-store) with it
  ([open question 2](#open-questions)).
- Add the bridge-level history tests above.
- Make redelivery safe across restarts, by one of: a durable uma store, or a bridge pre-check of
  aps history before running the action.
- Document the WhatsApp Cloud session-scope change, or map `context.id` to `Thread` in the
  bridge.

**What would change the recommendation.**

- A durable uma store in aps (PostgreSQL), with the bridge resuming its subscription from a
  persisted marker. Then C becomes worth building: idempotent turns by uma ref, honest outbound
  state, edit and delete propagation. B stays unattractive until uma offers a newest-first read
  on the public API.
- A requirement that privacy erasure in uma (`Privacy.Redact`) must also scrub `prior_turns`.
  Under A this needs an aps-side purge keyed by provider message id.
- A bridged adapter with real edits or deletes in the context window.
- aps retiring SQLite history for other platforms, which removes the reason to keep a second copy.

## Open questions

Each question is for the maintainer and names what it blocks.

1. **Will bridged services run on durable uma state (PostgreSQL), and when?** Blocks the choice
   between A and C and the redelivery fix. If never, C is off the table and the pre-check route
   is the redelivery answer.
2. **Should an outbound turn mean "queued" or "delivered"?** Today it means queued
   (`provider.go:133-141`). Delivered needs a `delivery.updated` hook and a window where the turn
   is absent. Blocks the policy wording and the rejected-send test.
3. **Is losing WhatsApp Cloud reply threads at cutover acceptable?** The native path scopes a
   session to the replied-to message (`whatsapp.go:361`). Blocks the bridge normalizer change and
   the migration note. For Twilio SMS and WhatsApp it does not apply.
4. **Must a converted service keep its history?** Keeping continuity means the operator reuses the
   service id, and `service add --type uma` should warn about it. Blocks the cutover runbook.
5. **Is a double reply after a restart acceptable for now?** Probe: one provider event handled
   after a restart ran the action and sent a second reply. Blocks whether the pre-check ships
   before the bridge is mounted in `aps serve`.
6. **Should uma ids be persisted on turns?** Useless with memory uma; useful for C and for
   support. Blocks the `ALTER` and its unique index.
7. **Where does the decision get recorded?** The task asked for it in `messenger-architecture.md`,
   which is under concurrent change; this page links to it instead. After a decision,
   "mapping uma conversations onto aps session history" in the "Not done here" paragraph and
   `docs/user/messengers.md:346-347` need rewording. Blocks the doc follow-up only.

## Evidence and probes

Read and cited above: the aps history store, router, runtime, `service conversation` command and
umabridge package; uma's `Store`, `Messages.List`, `InboundMessages`, memory store and its
documented limits, selection rules, the adapter documents and the API coverage table.

Probes run, all in a scratch copy of this tree outside the repository, as in-package tests of
`umabridge` against uma at `650e527`, with the repository's fake Twilio server and a temporary
SQLite store. Nothing was committed.

| Probe | Result |
| --- | --- |
| Three inbound SMS, each answered, listed through `Messages.List` | Oldest first; inbound has no `Origin`, author kind `person`, provider SID in `External`; outbound has `Origin`, `InReplyTo`, its own SID once sent; page size 2 walked to `Complete` |
| Same provider event posted twice in one process | One action run, one reply, still six messages |
| Same provider event, two engine lifetimes sharing one SQLite store | Action ran in each lifetime; conversation and message ids differ; two outbound aps turns for one provider event |
| Provider answers 400 on send | One aps outbound turn recorded; the uma operation ends `rejected`; the reply message is still listed, with no provider id |
| Second sender, conversation listing | A second conversation; `ReplyRoute.ID` is the sender address; mode `per_contact`, status `open` |
| `Messages.List` with `Since` in the future | Empty page, no error |

**Unverified:** append ordering for rapid successive messages; whether `meta.whatsapp` sender and
channel ids equal the native Cloud ones; any PostgreSQL behaviour; provider redelivery windows;
whether a server API could serve the CLI.
