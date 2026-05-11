# Incoming Execution Surfaces Findings

Status: investigation notes, not a final plan.
Date: 2026-05-11
Tasks: T-0587, T-0588, T-0589, T-0590

## Purpose

Capture what is currently documented versus what the implementation appears to expose for incoming requests, listeners, execution, and replies. This note is intentionally descriptive. It should be used before planning documentation fixes, feature work, or claims about support.

## Documented Surfaces Reviewed

The Markdown docs in `docs/agent` and `docs/dev` describe three main intake patterns:

- A2A server: profile receives agent tasks over A2A.
- Messenger adapters: platform messages normalize, route to actions, and optionally reply.
- ACP: editor/client JSON-RPC sessions for filesystem, terminal, and session operations.

The docs also mention an event listener/daemon concept around bus topics, but frame richer routing/handler behavior as future work.

## Implementation Surfaces Confirmed

### `aps serve` Agent Protocol HTTP API

This is a real request/reply execution surface and is underrepresented in the earlier high-level summary.

Entry points:

- `internal/cli/serve.go`
- `internal/adapters/agentprotocol/adapter.go`
- `internal/adapters/agentprotocol/runs.go`
- `internal/adapters/agentprotocol/runs_advanced.go`
- `internal/adapters/agentprotocol/threads.go`
- `internal/core/protocol/core.go`
- `tests/e2e/agent_protocol_test.go`

Observed behavior:

- Starts an HTTP protocol server.
- Mounts `/health` and Agent Protocol routes.
- Uses `adapters.DefaultManager()` at startup; that default manager currently registers the Agent Protocol adapter only.
- Supports run creation, wait, stream, background runs, thread/session routes, agent discovery, skills, and store endpoints.
- Executes profile actions through `protocol.APSAdapter.ExecuteRun`.
- Captures stdout for synchronous run responses and supports SSE streaming for run output.
- Optional bearer-token auth exists at the `aps serve` layer.

Open questions:

- Whether docs clearly distinguish this from A2A and generic webhooks.
- Whether the global `ProtocolRegistry` is intended to feed `aps serve`; current traced startup path uses the adapter manager instead.

### `aps webhook server`

This is a real generic webhook-to-action execution surface.

Entry points:

- `internal/cli/webhook/server.go`
- `internal/core/webhook.go`
- `tests/e2e/webhook_test.go`
- `tests/e2e/webhook_gap_test.go`

Observed behavior:

- Listens on `/webhook`.
- Requires POST.
- Reads `X-APS-Event`.
- Optionally validates `X-APS-Signature` HMAC.
- Uses `--event-map event=profile:action`.
- Runs mapped profile action synchronously via `core.RunAction`.
- Returns JSON status such as `executed`, `dry_run`, or error.
- Does not capture action stdout/stderr into the HTTP response; action output streams to process stdio.
- `--profile` can auto-enable webhook capability.

Open questions:

- Whether documentation currently presents this clearly as a first-class incoming execution path.
- Whether output capture/reply semantics are intentionally status-only or incomplete.

### A2A Server

This is a real A2A listener, but execution behavior is currently placeholder-level.

Entry points:

- `internal/cli/a2a/server.go`
- `internal/a2a/server.go`
- `internal/a2a/executor.go`
- `tests/e2e/a2a_server_test.go`
- `tests/e2e/a2a_client_test.go`

Observed behavior:

- Starts a standalone HTTP JSON-RPC server.
- Exposes `/.well-known/agent-card`.
- Stores tasks in filesystem-backed A2A storage.
- Handles task get/cancel/push-config flows.
- Executor emits task status transitions and replies to the first text part with `Processed: <text>`.

Gap signal:

- Docs imply profile-backed task processing. The current executor does not appear to run profile actions, chat, or LLM logic yet.

### ACP Server

ACP is a real JSON-RPC intake path, primarily for editor/client sessions rather than general webhook/task execution.

Entry points:

- `internal/cli/acp/server.go`
- `internal/acp/server.go`
- `internal/acp/terminal.go`
- `internal/acp/filesystem.go`

Observed behavior:

- Starts a standalone protocol server.
- Current transport defaults to stdio.
- `aps acp enable --transport stdio|http|ws` stores transport/listen/port settings in the profile.
- `aps acp server` currently creates the ACP server and calls `Start` with nil config; `internal/acp/server.go` currently returns stdio transport regardless of saved profile transport settings.
- Supports initialization, auth, sessions, prompts, filesystem operations, terminal operations, and permission/session modes.
- WebSocket/HTTP bridge concepts appear in docs and supporting files, but need separate verification before describing them as a wired CLI path.

Open questions:

- Which non-stdio transports are fully user-accessible through CLI today.
- How ACP should be positioned relative to `aps serve`.
- Whether HTTP/WebSocket ACP support is planned-only, partially implemented, or wired through a different entry point.

### `aps listen`

This is an event intake surface, not yet an execute/reply surface.

Entry points:

- `internal/cli/listen.go`
- `tests/e2e/listen/`

Observed behavior:

- Requires `--profile`.
- Requires bus to be enabled.
- Subscribes to bus topic patterns.
- Prints matching events as JSONL to stdout.
- Supports `--topics` and `--exit-after-events`.
- Comments explicitly state handler dispatch/routing lands later.

Gap signal:

- Useful for awareness and piping, but not yet a direct action dispatch or reply mechanism.

### Messenger Webhook Handler

The handler exists, but production mounting and real action dispatch are unclear/incomplete.

Entry points:

- `internal/adapters/messenger/handler.go`
- `internal/adapters/messenger/router.go`
- `internal/adapters/messenger/normalizer.go`
- `internal/core/messenger/manager.go`
- `internal/cli/adapter/messenger_alias.go`
- `internal/voice/voice_messenger.go`
- `internal/adapters/messenger/*_test.go`

Observed behavior:

- HTTP handler expects `/messengers/{platform}/webhook`.
- Normalizes Telegram, Slack, GitHub, and email payloads.
- Routes by messenger name and channel ID to a profile/action mapping.
- Denormalizes action results into platform-shaped responses.
- Audio attachments can bypass routing into a voice handler.
- `aps messenger start` / `aps adapter start` uses the core adapter lifecycle. Messenger adapters default to subprocess strategy, looking for executables such as `aps-adapter-<name>` or a local `run` file.
- Built-in adapter startup currently marks runtime state as running; it does not mount the Go messenger webhook handler.
- Script adapter execution exists through adapter action commands, but persistent script start is explicitly unsupported.
- `aps messenger test` exercises normalization/routing behavior and simulates dispatch/send status; it is not proof of live platform delivery.

Gap signals:

- `MessageRouter.ExecuteAction` is currently placeholder dispatch text, not real action execution.
- Search did not find the handler mounted by `aps serve` or another production HTTP server path.
- `aps messengers start` appears to be adapter/device subprocess lifecycle, not necessarily this Go HTTP handler.

### Mobile Adapter Pairing Server

This is a real network-facing intake path, but it is a device pairing/control channel rather than general profile request/reply automation.

Entry points:

- `internal/cli/adapter/pair.go`
- `internal/core/adapter/mobile/server.go`

Observed behavior:

- `aps adapter pair` starts a mobile adapter server and shows a QR/pairing code.
- The server exposes `/aps/adapter/{profileID}/health`, `/pair`, and `/ws`.
- Pairing creates an adapter identity and token.
- The WebSocket path authenticates device connections and accepts messages of type `command`.
- `handleCommand` currently parses the command payload, emits a `running` status, then emits `received`.
- The command execution path contains a TODO to execute through APS core later.
- `runPair` builds the displayed endpoint with `https://`, but the traced server startup path does not pass a TLS certificate option into `NewAdapterServer`; this needs confirmation before documenting the endpoint scheme as HTTPS.

Gap signal:

- This is reachable from a CLI command, but it does not currently execute profile actions or return action results.

### Voice Intake Components

Voice intake components exist, but CLI/server mounting needs more verification.

Entry points:

- `internal/voice/adapter_web.go`
- `internal/voice/adapter_twilio.go`
- `internal/voice/voice_messenger.go`
- `internal/cli/voice.go`

Observed behavior:

- `WebAdapter` serves WebSocket voice sessions at `/ws`.
- `TwilioAdapter` accepts media streams at `/twilio/media-stream`.
- `MessengerVoiceHandler` receives audio messages from the messenger pipeline and emits voice channel sessions.
- `aps voice start` currently registers a voice session; it does not appear to mount the web or Twilio HTTP handlers by itself.

Open questions:

- Whether another server path mounts these adapters.
- Whether docs overstate voice intake availability as an externally reachable service.

## Route Reachability Matrix

This matrix is a working snapshot from CLI route tracing plus an `xray explore` pass over route and server symbols. "Reachable" means a normal user-facing command appears to start the listener directly, not just that a handler type exists in tests or component code.

| Surface | User command or entry | Transport / route | Executor path | Reply semantics | Current classification |
| --- | --- | --- | --- | --- | --- |
| Agent Protocol HTTP | `aps serve` | HTTP on `--addr`; `/health`; `/v1/runs*`; `/v1/threads*`; `/v1/agents*`; `/v1/store*`; `/v1/skills*` | `protocol.NewAPSAdapter()` through Agent Protocol handlers; run endpoints call profile/action execution paths | JSON run/thread/store responses; synchronous run responses include captured stdout in `output`; SSE stream endpoints emit stdout chunks | Reachable, profile-backed execution |
| Generic webhook server | `aps webhook server` | HTTP POST `/webhook`; event selected by `X-APS-Event`; optional `X-APS-Signature` | `core.RunAction(profileID, actionID, body)` from `--event-map` | JSON status/error with delivery metadata; action stdout/stderr stream to process stdio | Reachable, profile-backed execution, status-only reply |
| A2A server | `aps a2a server --profile <id>` | HTTP JSON-RPC `/`; agent card at `/.well-known/agent-card` | `internal/a2a.Executor.Execute` | A2A task/status responses; current text response is `Processed: <text>` | Reachable protocol listener, placeholder execution |
| ACP stdio server | `aps acp server <profile>` | JSON-RPC over stdio | `internal/acp.Server.handleRequest`; session, filesystem, terminal, prompt handlers | JSON-RPC responses on stdout | Reachable client/session protocol, not general webhook automation |
| ACP HTTP/WebSocket | Profile config via `aps acp enable --transport=http|ws`; component helpers in ACP transport files | Intended HTTP/WS concepts; WebSocket helper exposes `/acp` when started directly | No traced CLI path passes ACP transport config into `Server.Start`; `createTransport` returns stdio | Not reachable through the traced server command | Component/config exists, CLI wiring incomplete or missing |
| Event listener | `aps listen --profile <id>` | Bus subscription patterns from `--topics`; no HTTP route | Bus subscriber callback encodes events | JSONL event records to stdout | Reachable event intake, no action dispatch/reply |
| Messenger adapter lifecycle | `aps messenger start <name>` / `aps adapter start <name>` | Starts adapter runtime; messenger defaults to subprocess strategy | External `aps-adapter-<name>` or local `run` executable for subprocess; built-in strategy only marks runtime running | CLI status/PID; platform replies depend on external subprocess, if any | Reachable lifecycle command, not the built-in Go webhook handler |
| Messenger test | `aps messenger test <name>` | No listener; local pipeline simulation | Resolves channel route; execute/denormalize/send steps are simulated | CLI step report; `--send` marks send as successful without verified platform delivery in this path | Reachable test harness, not live intake |
| Messenger webhook handler | Component `internal/adapters/messenger.Handler` | HTTP POST `/messengers/{platform}/webhook` if mounted | `MessageRouter.HandleMessage`; `ExecuteAction` returns placeholder dispatch text | Platform-shaped response via denormalizer or generic status | Component exists, no traced production mount, placeholder execution |
| Mobile pairing server | `aps adapter pair --profile <id>` | HTTP/WebSocket under `/aps/adapter/{profileID}`: `/health`, `/pair`, `/ws` | Pairing/token flow; WebSocket `command` handler currently acknowledges only | WebSocket status messages `running` then `received`; no profile action result | Reachable pairing/control channel, command execution TODO |
| Voice backend lifecycle | `aps voice service start` | Starts configured backend process; no APS HTTP route mounted by this command | External backend binary | CLI status only | Reachable process lifecycle, not an intake route by itself |
| Voice session registration | `aps voice start --profile <id>` | No listener mounted; registers a session for channel metadata | `voice.RegisterSession` | CLI prints session ID | Reachable session registration, not an external listener |
| Voice Web/Twilio adapters | Component constructors `NewWebAdapter`, `NewTwilioAdapter` | WebSocket `/ws`; WebSocket `/twilio/media-stream` if mounted | Channel session audio/text channels | WebSocket audio/text frames | Component exists, no traced CLI mount |
| Generic protocol HTTP bridge | Component `internal/core/protocol/http_bridge.go` | Generic HTTP/JSON-RPC bridge handlers if constructed | Placeholder bridge responses; `ProtocolServerAdapter.RegisterRoutes` is no-op | JSON placeholder status/method/path | Component exists, not a traced user-facing listener |

## Current Working Classification

Likely complete enough to document as active request/reply surfaces:

- `aps serve` Agent Protocol HTTP API.
- `aps webhook server`.

Active protocol listener with incomplete execution semantics:

- A2A server.

Active client/session protocol, not general inbound automation:

- ACP server.

Active pairing/control listener with placeholder command execution:

- Mobile adapter pairing WebSocket server.

Active event subscription only:

- `aps listen`.

Component-level or partially wired intake:

- Messenger webhook handler.
- Voice WebSocket/Twilio/messenger voice adapters.

## Documentation Gap Classification

This table compares current documentation claims against the reachability matrix. It does not prescribe fixes yet; it identifies where the docs and implementation need reconciliation.

| Area | Documentation location | Claim shape | Observed implementation | Classification | Starting point |
| --- | --- | --- | --- | --- | --- |
| Agent Protocol HTTP | `docs/dev/architecture/protocol-server-architecture.md`; `docs/agent/skills.md`; skills docs | Agent Protocol HTTP endpoints exist | `aps serve` does expose Agent Protocol HTTP routes and profile-backed run execution | Documentation mostly valid, but underrepresented in incoming-surface docs | Add `aps serve` as a first-class incoming request/reply surface in user/dev docs |
| Protocol registry startup | `docs/dev/architecture/protocol-server-architecture.md`; `docs/dev/protocol-interface-unification.md` | `runServe` registers HTTP adapters via global protocol registry and may start standalone servers | Traced `runServe` uses `adapters.DefaultManager()`; global `ProtocolRegistry` appears separate from the active `aps serve` startup path | Documentation misleading or stale | Start from `internal/cli/serve.go`, `internal/adapters/manager.go`, `internal/adapters/protocol_registry.go` |
| Generic webhooks | `docs/dev/bundles.md`; `docs/dev/architecture/12-factor-implementation.md`; sparse current dev docs | Webhooks are a capability/service and external trigger path | `aps webhook server` is real and executes mapped actions, but response is status-only and some architecture docs point to older paths/commands | Documentation missing plus stale references | Add current `aps webhook server` behavior and response limits; verify or retire older `webhook register` examples |
| A2A task processing | `docs/agent/a2a-integration.md`; `docs/dev/a2a-implementation.md` | Profiles receive and process tasks; examples describe useful task responses, streaming, push, and workflows | A2A server/listener is real; executor currently emits status and `Processed: <text>` placeholder, not profile action/chat execution | Feature incomplete and docs overstate behavior | Start from `internal/a2a/executor.go` and clarify which A2A flows are implemented versus protocol scaffolding |
| A2A transport/auth claims | `docs/dev/a2a-implementation.md` | IPC, HTTP, gRPC, auth schemes, mTLS/OAuth, push notifications are described as supported implementation details | Current reachability pass only confirmed the standalone HTTP JSON-RPC server and agent card path; transport/auth breadth needs separate verification | Further investigation needed; possible documentation overreach | Start from `internal/a2a/transport/`, agent card generation, and e2e tests before making support claims |
| ACP command and transport | `docs/dev/acp-implementation.md`; `docs/dev/architecture/protocol-server-architecture.md` | `aps acp start --profile ... --transport ws --port ...`; stdio/WebSocket/HTTP bridge transports managed by ACP | Actual command is `aps acp server <profile>`; toggle stores transport settings, but server start passes nil config and `createTransport` returns stdio | Documentation misleading; feature partially implemented but not wired | Start from `internal/cli/acp/server.go`, `internal/cli/acp/toggle.go`, `internal/acp/server.go`, `internal/acp/transport_ws.go` |
| ACP status | `docs/dev/acp-implementation.md` | "Phase 6 Complete" and "Implementation Status: Complete" | Core stdio JSON-RPC server exists, but documented HTTP/WS transport and some MCP bridge behavior remain placeholder or unwired | Documentation overstates completeness | Split status into stdio/core complete versus remote transport/bridge incomplete |
| Messenger live intake | `docs/agent/messenger-patterns.md`; `docs/dev/messenger-architecture.md` | Incoming platform messages normalize, route, execute actions, and optionally reply | Component handler normalizes/routes, but no traced production mount; `ExecuteAction` is placeholder; `messenger test` simulates execute/send | Feature exists but not wired; feature incomplete; docs overstate live execution | Start from `internal/adapters/messenger/handler.go`, `router.go`, and `internal/cli/adapter/*` lifecycle |
| Messenger CLI examples | `docs/agent/messenger-patterns.md`; `docs/dev/messenger-architecture.md` | Uses `aps messengers create --template=subprocess --language=python`, `aps profile link-messenger`, `aps adapter test` | Current CLI uses `aps messenger` alias over adapter commands; create flags are `--type`, `--strategy`; links appear under adapter/messenger link commands | Documentation stale | Refresh command examples from current cobra commands before changing feature behavior |
| Voice channels | `docs/dev/voice.md` | APS routes sessions across web, TUI, Telegram, Twilio; web serves UI/proxy; messenger and Twilio adapters bridge audio into pipeline | Backend lifecycle and session registration exist; Web/Twilio adapter components exist; no traced CLI command mounts those HTTP/WebSocket handlers; voice action pipeline claims need verification | Documentation misleading or aspirational | Start from `internal/cli/voice.go`, `internal/voice/adapter_web.go`, `adapter_twilio.go`, `voice_messenger.go` |
| Event listener daemon | `docs/dev/event-topics.md` | Listener daemon subscribes and future route DSL will dispatch profile rules | `aps listen` subscribes and prints JSONL only; docs generally frame richer routing as planned | Documentation mostly aligned | Keep as future-oriented; cross-link `aps listen` current behavior |
| Mobile pairing | `docs/dev/adapters.md`; `docs/dev/bundles.md`; current findings | Mobile pairing via QR and WebSockets | `aps adapter pair` starts pairing server and WebSocket, but command messages are acknowledged only; endpoint scheme may be documented/advertised as HTTPS without confirmed TLS in traced path | Feature incomplete; further verification needed | Start from `internal/cli/adapter/pair.go` and `internal/core/adapter/mobile/server.go` |
| Generic protocol HTTP bridge | `docs/dev/architecture/protocol-server-architecture.md`; `docs/dev/protocol-interface-unification.md` | HTTP bridge can expose ACP/stdio protocols remotely | Bridge components return placeholder responses; `ProtocolServerAdapter.RegisterRoutes` is no-op; no traced user-facing mount | Component exists but not wired; docs overstate practical availability | Start from `internal/core/protocol/http_bridge.go` and registry usage |

## Next Investigation Starting Points

1. Add test-coverage evidence to the matrix where it changes confidence, especially for `aps serve`, `aps webhook server`, A2A, ACP, messenger, mobile pairing, and voice.
2. Decide whether the next deliverable is documentation correction, feature completion, or both for each high-risk area.
3. For Telegram/chat work, start from `internal/adapters/messenger`, the adapter lifecycle in `internal/core/adapter`, and the `aps-chat` track, because current messenger action dispatch and A2A executor are not yet profile-backed chat execution paths.
4. Confirm whether mobile pairing is intended to advertise `https://` by default or whether TLS setup is missing from the traced `aps adapter pair` path.
