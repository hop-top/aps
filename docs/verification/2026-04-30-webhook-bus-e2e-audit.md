# Webhook + Bus e2e audit

Author: jadb
Date: 2026-04-30
Task: T-0093 (`~/.ops/.tlc`)
Branch: `docs/webhook-bus-verify`

## Summary

Webhook surface has one happy-path e2e (`TestWebhookServer`) covering
auth + action trigger + 401/400 paths — all in-process spawn of `aps`
binary. Toggle stories (037/038/039) covered by 16 short profile-yaml
tests in `protocol_toggle_test.go`. Bus surface has ZERO cross-process
coverage: `internal/events/events_test.go` round-trips through an
in-memory `bus.New()` (same Go test binary), and core/session/adapter
tests use a `fakePublisher` stub. No e2e spawns a separate `aps`
publisher process + separate subscriber process sharing
`APS_BUS_ADDR`/`APS_BUS_TOKEN` against a real dpkms hub. Stories 051
(listener-daemon) and 052 (listener-routing-config) are NOT yet
drafted in this repo (only 050 exists) — they will land on top of
this gap, so primitives need cross-process e2e first.

## Per-test inventory (17 tests)

| File::Function | Exercises | Cross-proc? |
|---|---|---|
| `tests/e2e/webhook_test.go::TestWebhookServer` | 401 missing-sig, 401 bad-sig, 400 no-event, 200 + action proof file | Y (spawns `aps webhook server`) |
| `tests/e2e/protocol_toggle_test.go::TestWebhookToggle_Enable` | toggle on → yaml has `- webhooks` | Y (spawns `aps`) |
| `…::TestWebhookToggle_Disable` | toggle on→off → yaml stripped | Y |
| `…::TestWebhookToggle_ForceDisable` | `--enabled=off` → yaml stripped | Y |
| `…::TestWebhookServer_AutoEnable` | `webhook server --profile` auto-enables, then killed | Y (spawn+kill, no req sent) |
| `…::TestA2AToggle_{Enable,Disable,CustomConfig,ForceEnable}` | yaml deltas | Y |
| `…::TestA2AServer_AutoEnable` | spawn+kill | Y |
| `…::TestACPToggle_{Enable,Disable,CustomConfig}` | yaml deltas | Y |
| `…::TestACPServer_AutoEnable` | spawn+kill | Y |
| `…::TestToggle_Invalid{EnabledValue,Protocol,Transport}` | error paths | Y |
| `internal/events/events_test.go::TestPublisher_DeliversToSubscriber` | pub→sub round-trip | N (in-mem `bus.New()`) |
| `…::TestPublisher_PatternMatchesSessionTopics` | wildcard `aps.session.*` | N |
| `internal/core/profile_events_test.go::Test{CreateProfile,DeleteProfile,Add/RemoveCapability}_Emits…` | core hits `pkgPublisher` | N (fakePublisher) |
| `internal/core/session/registry_events_test.go::Test{Register,Unregister,UpdateStatus_*,CleanupInactive}_Emits…` | session lifecycle topics | N (fakePublisher) |

## Coverage matrix (story acceptance → test → cross-proc)

| Story::Criterion | Test | X-proc |
|---|---|---|
| 004#1 valid POST triggers action | `TestWebhookServer` | Y |
| 004#2 missing sig → 401 | `TestWebhookServer` | Y |
| 009#1 valid signed → 200 + action | `TestWebhookServer` | Y |
| 009#2 invalid sig → 401 | `TestWebhookServer` | Y |
| 009#3 unmapped event → 400 | partial: `TestWebhookServer` covers no-event-header (400); UNMAPPED event-name not tested | Y (partial) |
| 039#1 toggle enables | `TestWebhookToggle_Enable` | Y |
| 039#2 toggle disables | `TestWebhookToggle_Disable` | Y |
| 039#3 `--enabled=on` reconfirms | NOT TESTED (only `=off` force tested) | — |
| 039#4 server `--profile` auto-enables | `TestWebhookServer_AutoEnable` (no live request sent) | Y (partial) |
| bus aps.profile.* end-to-end via dpkms | NONE | N |
| bus aps.adapter.* end-to-end via dpkms | NONE | N |
| bus subscriber receives webhook-triggered event | NONE (and emit wiring itself reserved per `events.go:29-38`) | N |

Totals: 11 criteria mapped, 9 fully covered, 2 partial, 3 missing.
Cross-process count: 9/11 (all webhook/toggle); bus 0/3.

## Gap list

Highest-value gaps (ordered by scenario-2 blocker risk):

1. **Bus pub-sub cross-process** — no test boots dpkms hub (or a stub
   ws bus server), spawns `aps profile new` with `APS_BUS_ADDR/TOKEN`,
   and a separate subscriber process that asserts
   `aps.profile.created` arrival. Blocks stories 051/052.
2. **Bus adapter.linked/.unlinked cross-process** — same shape as #1,
   for `aps adapter link/unlink`. Blocks listener routing.
3. **Bus token-missing graceful path** — no e2e asserts the stderr
   warning + non-zero behavior survival when `APS_BUS_TOKEN` is unset
   while `APS_BUS_ADDR` is set.
4. **Webhook 009#3 unmapped event** — current test only sends NO event
   header; need a request with `X-APS-Event: not.in.map` → 400.
5. **Webhook server-auto-enable + live request** — `TestWebhookServer_
   AutoEnable` kills before sending a request; should send a signed
   POST and assert action ran (closes 039#4 fully).
6. (lower) **Webhook 039#3 reconfirm** — toggle `--enabled=on` on
   already-enabled profile; assert idempotent.

## Recommendations (prioritized)

P1 (blocks tools-showcase scenario 2 — listener daemon demo):
- gap 1: cross-process bus pub-sub for `aps.profile.*`
- gap 2: cross-process bus pub-sub for `aps.adapter.*`

P2 (correctness / completeness):
- gap 4: unmapped-event 400
- gap 5: live request after auto-enable
- gap 3: token-missing fallback

Cap of 5 gap-tasks per task spec. Drop gap 6 (lowest value).

## Cross-process testing pattern (proposal)

No existing pattern in repo. Two viable approaches:

**A. Two-process Go test (preferred for bus)**
- Test boots a stub bus server (kit/bus exposes `bus.New()` with
  `WithNetwork`; can serve via `httptest.NewServer` wrapping a ws
  upgrader OR run `dpkms` as a subprocess if available in PATH).
- Test sets `APS_BUS_ADDR=ws://127.0.0.1:<port>/ws/bus` +
  `APS_BUS_TOKEN=test`.
- Spawn 1: `aps profile new noor` (publisher).
- Spawn 2: `aps bus subscribe aps.profile.*` (if the CLI supports
  it) OR a minimal Go subscriber goroutine using kit/bus client.
- Assert subscriber received topic + payload within 2s.

**B. xrr cassette (preferred when dpkms unavailable)**
- Record once against a live dpkms; replay via `XRR_MODE=replay`.
- Cassette dir under `tests/e2e/cassettes/bus/`. Pattern matches
  `tlc flow test` precedent (per CLAUDE.md xrr section).
- Cross-runtime YAML cassettes mean the test stays hermetic in CI.

Recommendation: ship A first (one Go test using kit/bus in-mem ws
server); promote to B only if dpkms presence becomes a CI flake.

For unmapped-event + live-request gaps (4, 5): extend
`TestWebhookServer` pattern — spawn `aps webhook server`, send
HTTP, assert proof file. No new pattern needed.
