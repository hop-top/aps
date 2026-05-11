# Service-Oriented UX Draft

Status: draft proposal, not an implementation plan.
Date: 2026-05-11
Task: T-0591
Related: `docs/dev/incoming-execution-surfaces-findings.md`

## Purpose

Define a user-facing UX model for letting APS profiles receive requests, execute work, and reply through external channels. This draft uses `service` as the primary term instead of `incoming`.

The design goal is to make one question obvious:

> If something sends a request to this APS surface, what happens, and what comes back?

## Naming Direction

`service` works better than `incoming` if APS treats each exposed surface as an operational runtime attached to a profile.

Working definition:

> An APS service is a named runtime or endpoint that lets a profile receive external input, observe events, execute actions, or expose a protocol.

This term fits:

- HTTP APIs.
- Webhooks.
- Messenger connections.
- A2A servers.
- ACP client sessions.
- Event listeners.
- Mobile pairing channels.
- Voice channels, if and when they are mounted as reachable listeners.

Potential naming risk:

- `service` can be confused with backend process lifecycle, such as `aps voice service start`.
- The CLI should distinguish profile-facing services from internal backend services.

Preferred mitigation:

- Use `aps service` for profile-facing surfaces.
- Rename or describe backend-only processes as `backend` or `runtime` in future docs, for example `aps voice backend start`.

## Core UX Model

Users should not have to choose between internal concepts such as protocol adapters, bridges, handlers, standalone servers, or registry paths during first setup.

Instead, expose one top-level concept:

```bash
aps service
```

Each service has:

- A user-owned identifier.
- A service type.
- An optional adapter.
- A profile.
- Type- and adapter-specific options and environment bindings.
- A reachability status.
- An execution status.
- A reply behavior.

## Service Types

The service identifier is not the service type or adapter name. Users choose identifiers that match their domain, such as `support-bot`, `github-deploy`, `agent-api`, or `ops-events`.

`--type` names the user-facing category. `--adapter` names a concrete provider when the type has multiple providers.

| User intent | Canonical type | Adapter examples | Backing surface |
| --- | --- | --- |
| External app calls a profile API | `api` | `agent-protocol` | Agent Protocol through `aps serve` |
| Generic event triggers a profile action | `webhook` | generic, GitHub webhook, Stripe-style webhook | `aps webhook server` |
| Another agent sends a task | `a2a` | A2A JSON-RPC | A2A server |
| Editor or client controls a profile session | `client` | `acp` | ACP |
| Chat-like platform messages a profile | `message` | `telegram`, `slack`, `discord`, `sms`, `whatsapp` | Messenger adapters |
| Work-item or inbox item reaches a profile | `ticket` | `email`, `github`, `gitlab`, `jira`, `linear` | Ticket/message adapters |
| Internal tool event is observed | `events` | bus topics | `aps listen` |
| Mobile app pairs to a profile | `mobile` | APS mobile | Mobile adapter pairing server |
| Voice channel reaches a profile | `voice` | `web`, `tui`, `telegram`, `twilio` | Voice adapters |

## Type Aliases

`--type` should accept canonical types and adapter aliases. Adapter aliases expand through kit's aliasing layer, then persist as canonical service config.

Examples:

| User input | Canonical config |
| --- | --- |
| `--type message --adapter slack` | `type: message`, `adapter: slack` |
| `--type slack` | `type: message`, `adapter: slack` |
| `--type telegram` | `type: message`, `adapter: telegram` |
| `--type discord` | `type: message`, `adapter: discord` |
| `--type sms` | `type: message`, `adapter: sms` |
| `--type whatsapp` | `type: message`, `adapter: whatsapp` |
| `--type ticket --adapter github` | `type: ticket`, `adapter: github` |
| `--type email` | `type: ticket`, `adapter: email` |
| `--type github` | `type: ticket`, `adapter: github` |
| `--type gitlab` | `type: ticket`, `adapter: gitlab` |
| `--type jira` | `type: ticket`, `adapter: jira` |
| `--type linear` | `type: ticket`, `adapter: linear` |

Alias rule:

- Use kit's aliasing mechanism for these expansions, not ad hoc command-specific string switches.
- If the current kit alias store is command-only, add a typed alias namespace in kit and have `aps service` consume that.
- Store only canonical config on disk.
- Show both the user input and resolved config in `--dry-run` and `service show`.

## Maturity Labels

Every service should report its maturity. This prevents the current problem where docs describe component-level code as if it were a complete user-facing route.

| Label | Meaning |
| --- | --- |
| `ready` | Reachable from a user command, executes real profile-backed work, and has documented reply behavior. |
| `status-only` | Executes work, but replies with status metadata rather than action output. |
| `observe-only` | Receives or observes events but does not dispatch actions. |
| `placeholder` | Listener is reachable, but execution is stubbed or acknowledgement-only. |
| `component` | Code exists, but no normal user command was traced that mounts it. |
| `planned` | Design/documentation exists, but implementation is not present or not verified. |

## Service Status Shape

`aps service status` should make the behavior explicit:

```text
ID            TYPE      ADAPTER    PROFILE     RECEIVES        EXECUTES          REPLIES        MATURITY
agent-api     api       agent      worker      HTTP /v1/runs   profile action    JSON/SSE       ready
github-hook   webhook   generic    ops         POST /webhook   profile action    status JSON    status-only
worker-a2a    a2a       jsonrpc    worker      JSON-RPC /      placeholder       A2A task       placeholder
dev-acp       client    acp        dev         stdio JSON-RPC  session ops       JSON-RPC       ready
watcher       events    bus        noor        bus topics      none              JSONL          observe-only
support-bot   message   slack      assistant   platform msg    placeholder       platform JSON  component
repo-inbox    ticket    github     maintainer  repo events     placeholder       status JSON    component
mobile-link   mobile    aps        assistant   WS command      ack only          WS status      placeholder
```

The important columns are:

- `RECEIVES`: what input arrives and where.
- `EXECUTES`: what APS does with it.
- `REPLIES`: what the caller gets back.
- `MATURITY`: whether the service should be treated as production-ready.

## Proposed CLI

### Discovery

```bash
aps service list
aps service status
aps service show <service-id>
aps service routes <service-id>
aps service logs <service-id>
```

### Creation

Creation should have one grammar:

```bash
aps service add <service-id> --type <type-or-adapter-alias> --profile <profile-id> [options]
```

Examples:

```bash
aps service add agent-api --type api --profile worker --addr 127.0.0.1:8080
aps service add github-hook --type webhook --profile ops --route push=deploy
aps service add worker-a2a --type a2a --profile worker
aps service add dev-acp --type client --adapter acp --profile dev --transport stdio
aps service add support-bot --type slack --profile assistant --env SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN
aps service add sms-alerts --type sms --profile assistant --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
aps service add repo-inbox --type github --profile maintainer --env GITHUB_TOKEN=secret:GITHUB_TOKEN
aps service add watcher --type events --profile noor --topics "aps.#,tlc.#"
aps service add mobile-link --type mobile --profile assistant
aps service add voice-web --type voice --adapter web --profile assistant
```

The four stable parts are:

1. `aps service add`
2. `<service-id>`
3. `--type <type-or-adapter-alias>`
4. `--profile <profile-id>`

Everything else is type- or adapter-specific configuration: address, route, channel, token secret, transport, topics, reply policy, delivery mode, or environment bindings.

### Type-Scoped Options

`aps service add` should have a small common option set and a type/adapter-scoped option set.

Common options:

```text
<service-id>
--type <type-or-adapter-alias>
--adapter <adapter>
--profile <profile-id>
--env <KEY=VALUE>
--label <KEY=VALUE>
--description <text>
--dry-run
```

Type and adapter options are selected by `--type` plus the resolved adapter. They should not all appear in base help because that would make the command unreadable and would imply every option applies to every service.

Base help:

```bash
aps service add --help
```

Should show:

```text
Usage:
  aps service add <service-id> --type <type-or-adapter-alias> --profile <profile-id> [options]

Common options:
  --type           Service type or adapter alias: api, webhook, message, slack, ticket, github, ...
  --adapter        Concrete adapter when --type is canonical: slack, telegram, github, jira, ...
  --profile        Profile that owns the service
  --env            Environment binding, repeatable
  --label          Metadata label, repeatable
  --description    Human-readable description
  --dry-run        Validate without writing

Type help:
  aps service add --type slack --help
  aps service add --type message --adapter slack --help
  aps service add --type webhook --help
```

Type help:

```bash
aps service add --type slack --help
```

Should resolve `slack` through kit aliasing to `type=message, adapter=slack`, then show common options plus Slack-specific options.

```text
Resolved:
  type: message
  adapter: slack

Slack options:
  --env SLACK_BOT_TOKEN=secret:<name>
  --receive polling|webhook
  --allowed-channel <id>       Repeatable
  --default-action <action>
  --reply text|none|auto
```

Validation contract:

- The service command validates common options.
- The selected type validates type-level options.
- The selected adapter validates adapter-level options.
- Type- and adapter-specific options should only be shown after alias expansion resolves the canonical type and adapter.
- Stored config may preserve unknown future fields for forward compatibility.
- Interactive CLI input should warn or fail on unknown options. Silent ignore is risky because misspelled flags create services that look configured but cannot work.

If permissive behavior is required for compatibility, use an explicit flag:

```bash
aps service add support-bot --type slack --profile assistant --ignore-unknown-options
```

Default behavior should be strict for direct CLI input.

### Type And Adapter Option Stress Test

The grammar holds if every type and adapter can express its needs as options after the same four-part command.

| Input | Canonical config | Example | Accepted option families | Help behavior |
| --- | --- | --- | --- |
| `--type api` | `type: api`, `adapter: agent-protocol` | `aps service add agent-api --type api --profile worker --addr 127.0.0.1:8080 --auth bearer:AGENT_API_TOKEN` | `--addr`, `--auth`, `--cors`, `--log-level` | `--type api --help` shows HTTP API options |
| `--type webhook` | `type: webhook`, `adapter: generic` | `aps service add github-hook --type webhook --profile ops --addr 127.0.0.1:9000 --secret GITHUB_WEBHOOK_SECRET --route push=deploy` | `--addr`, `--secret`, `--route`, `--allow-event`, `--dry-run-events` | `--type webhook --help` shows event mapping and signature options |
| `--type a2a` | `type: a2a`, `adapter: jsonrpc` | `aps service add worker-a2a --type a2a --profile worker --addr 127.0.0.1:8081 --public-endpoint http://localhost:8081` | `--addr`, `--public-endpoint`, `--transport`, `--auth` | `--type a2a --help` shows A2A server/card options and maturity warnings |
| `--type client --adapter acp` | `type: client`, `adapter: acp` | `aps service add dev-acp --type client --adapter acp --profile dev --transport stdio` | `--transport`, `--mode`, `--allow-terminal`, `--allow-write` | Must clearly mark HTTP/WS as unavailable until wired |
| `--type slack` | `type: message`, `adapter: slack` | `aps service add support-bot --type slack --profile assistant --env SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN` | `--env`, `--receive`, `--allowed-channel`, `--default-action`, `--reply` | `--type slack --help` resolves alias and shows Slack options |
| `--type telegram` | `type: message`, `adapter: telegram` | `aps service add support-bot --type telegram --profile assistant --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN --receive polling` | `--env`, `--receive`, `--allowed-chat`, `--default-action`, `--reply` | `--type telegram --help` resolves alias and shows Telegram options |
| `--type sms` | `type: message`, `adapter: sms` | `aps service add sms-alerts --type sms --profile assistant --env SMS_PROVIDER_TOKEN=secret:SMS_PROVIDER_TOKEN` | `--env`, `--provider`, `--from`, `--allowed-number`, `--reply` | `--type sms --help` resolves alias and shows SMS options |
| `--type whatsapp` | `type: message`, `adapter: whatsapp` | `aps service add wa-support --type whatsapp --profile assistant --env WHATSAPP_TOKEN=secret:WHATSAPP_TOKEN` | `--env`, `--provider`, `--phone-number-id`, `--allowed-number`, `--reply` | `--type whatsapp --help` resolves alias and shows WhatsApp options |
| `--type github` | `type: ticket`, `adapter: github` | `aps service add repo-inbox --type github --profile maintainer --env GITHUB_TOKEN=secret:GITHUB_TOKEN` | `--env`, `--repo`, `--events`, `--default-action`, `--reply` | `--type github --help` resolves alias and shows GitHub ticket options |
| `--type jira` | `type: ticket`, `adapter: jira` | `aps service add jira-intake --type jira --profile triage --env JIRA_TOKEN=secret:JIRA_TOKEN` | `--env`, `--site`, `--project`, `--jql`, `--default-action` | `--type jira --help` resolves alias and shows Jira options |
| `--type events` | `type: events`, `adapter: bus` | `aps service add watcher --type events --profile noor --topics "aps.#,tlc.#" --format jsonl` | `--topics`, `--format`, `--exit-after-events` | `--type events --help` shows observe-only semantics |
| `--type mobile` | `type: mobile`, `adapter: aps` | `aps service add mobile-link --type mobile --profile assistant --port 8765 --capability run:stateless` | `--port`, `--bind-addr`, `--capability`, `--approval-required` | `--type mobile --help` shows pairing and command-execution maturity |
| `--type voice --adapter web` | `type: voice`, `adapter: web` | `aps service add voice-web --type voice --adapter web --profile assistant --backend auto --voice-id NATF0` | `--backend`, `--voice-id`, `--channel`, `--listen` | Separates channel service from backend process lifecycle |

This model keeps command shape stable while allowing each type and adapter to own its own complexity.

### Lifecycle

```bash
aps service start <service-id>
aps service stop <service-id>
aps service restart <service-id>
aps service test <service-id>
```

### Routing

```bash
aps service route add <service-id> --event github.push --action deploy
aps service route add <service-id> --channel 12345 --action reply
aps service route list <service-id>
aps service route delete <service-id> <route-id>
```

`--action reply` resolves within the service profile by default. Cross-profile routing can be explicit with `--target <profile-id>:<action-id>` if that use case is needed later.

## Golden Paths

The first supported service flows should be the ones that are already closest to complete.

### API Service

Use when an external tool wants an HTTP API to run profile actions.

```bash
aps service add agent-api --type api --profile worker --addr 127.0.0.1:8080
aps service start agent-api
aps service test agent-api --action summarize --input '{"text":"hello"}'
```

Expected status:

```text
Receives: HTTP Agent Protocol requests
Executes: profile actions
Replies: JSON run metadata with captured stdout, or SSE output stream
Maturity: ready
```

Reply semantics:

- `POST /v1/runs` and `POST /v1/runs/wait` execute synchronously and return run metadata plus captured action stdout in `output`.
- `POST /v1/runs/stream` returns stdout chunks as SSE `output` events and a final `done` event with run metadata.
- Background and existing-run endpoints are metadata-first; callers should use wait or stream routes when they need action output.

### Webhook Service

Use when a third-party system sends event payloads.

```bash
aps service add github-hook --type webhook --profile ops --route push=deploy
aps service start github-hook
aps service test github-hook --event push --payload ./fixtures/push.json
```

Expected status:

```text
Receives: POST /webhook with X-APS-Event
Executes: mapped profile action
Replies: status JSON, not action stdout
Maturity: status-only
```

Current decision for T-0612:

- Keep the generic webhook service `status-only` until there is a separate output-capable reply contract.
- `aps webhook server` runs the mapped profile action synchronously through `core.RunAction`, so a non-zero action result can still become an HTTP failure.
- The successful HTTP reply is delivery/execution metadata: `status`, `delivery_id`, `event`, `profile`, and `action`.
- Action stdout and stderr belong to the webhook server process streams today. They are not captured into the HTTP response and should not be shown as caller-visible webhook output in service UX.
- If output-capable webhooks are added later, require an explicit reply mode such as `--reply output|status`, a bounded response-size policy, and tests for stdout, stderr, exit status, timeouts, and secret redaction before changing maturity from `status-only`.

## Service Types That Need Honest UX

### A2A Service

A2A should be exposed as a service, but the UX must not imply profile-backed task execution until the executor is real.

Current honest status:

```text
Receives: A2A JSON-RPC task messages
Executes: placeholder text processing
Replies: A2A task response
Maturity: placeholder
```

The CLI can still allow:

```bash
aps service add worker-a2a --type a2a --profile worker
```

But `aps service status` and `aps service test` should state that execution is placeholder-level.

### ACP Client Service

ACP should be described as a client/session service, not a generic automation endpoint.

Current honest status:

```text
Receives: JSON-RPC over stdio
Executes: session, filesystem, terminal operations
Replies: JSON-RPC responses
Maturity: ready for stdio, planned or unwired for HTTP/WebSocket
```

Avoid documenting HTTP/WebSocket ACP startup as supported until the CLI actually starts those transports.

### Message Service

Message services are strong candidates for `aps service`, but the UX needs to separate adapter lifecycle from live action execution.

Current honest status:

```text
Receives: platform messages only if an adapter subprocess or mounted handler is actually running
Executes: placeholder in built-in Go handler; external subprocess behavior depends on adapter
Replies: platform-shaped response only in component handler path
Maturity: component or adapter-dependent
```

Suggested CLI behavior:

```bash
aps service add support-bot --type telegram --profile assistant --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN
aps service route add support-bot --channel 12345 --action reply
aps service test support-bot --channel 12345 --message "hello"
```

If real execution is not wired, the test output should say:

```text
Route resolved: support-bot/12345 -> assistant:reply
Execution: placeholder, profile action was not invoked
Delivery: not verified
Maturity: component
```

### Events Service

Events should be positioned as observation first.

Current honest status:

```text
Receives: bus topics
Executes: none
Replies: JSONL to stdout
Maturity: observe-only
```

Only promote it to an execution service after route DSL and handler dispatch exist.

Current decision for T-0613:

- Keep the events service `observe-only`.
- The current reachable command is `aps listen --profile <id>`, which subscribes to bus topic patterns and writes one JSONL record per observed event.
- There is no route table, target action resolution, handler dispatch, or caller reply path in this surface.
- `aps service add ... --type events` and `aps service status` should therefore present `EXECUTES: none`, `REPLIES: JSONL to stdout`, and `MATURITY: observe-only`.
- Do not accept `--route`, `--action`, or other executable-service options for events until route dispatch is implemented and covered by tests.

### Mobile Service

Mobile pairing fits `service` if described as a pairing/control service.

Current honest status:

```text
Receives: pairing requests and WebSocket command messages over HTTP/WS by default
Executes: pairing/token flow; no profile action execution
Replies: pairing responses and placeholder command acknowledgements
Maturity: placeholder for command execution
```

The UX should not imply HTTPS or remote profile execution from mobile until the CLI pairing path wires TLS certificates and command handling calls the APS core action path. `AdapterServer` has component-level TLS support when constructed with a certificate, but `aps adapter pair` does not expose that configuration today.

### Voice Service

Voice needs careful naming because `voice service` already means backend process lifecycle.

Suggested distinction:

- `voice backend`: PersonaPlex/Moshi process lifecycle.
- `voice service`: profile-facing channel that receives audio.

Current honest status:

```text
Receives: audio only through component adapters if another caller mounts them
Executes: backend process lifecycle and session registration only from current CLI commands
Replies: component-level audio/text frames; no traced profile-facing service mount
Maturity: component
```

## Configuration Model

A service should be persisted as a named object, not just flags passed to long-running commands.

Example shape:

```yaml
services:
  agent-api:
    type: api
    adapter: agent-protocol
    profile: worker
    listen: 127.0.0.1:8080
    auth:
      type: bearer
      secret: AGENT_API_TOKEN

  github-hook:
    type: webhook
    adapter: generic
    profile: ops
    listen: 127.0.0.1:9000
    secret: GITHUB_WEBHOOK_SECRET
    routes:
      - event: push
        action: deploy

  support-bot:
    type: message
    adapter: telegram
    profile: assistant
    env:
      TELEGRAM_BOT_TOKEN: secret:TELEGRAM_BOT_TOKEN
    routes:
      - channel: "12345"
        action: assistant:reply

  watcher:
    type: events
    adapter: bus
    profile: noor
    topics:
      - aps.#
      - tlc.#
```

Persisted service config makes status, restart, testing, and documentation simpler.

## Testing UX

Every service type should have a test command that reports the same four properties.

```bash
aps service test <name>
```

Output shape:

```text
Service: github-hook
Type: webhook
Adapter: generic
Profile: ops

Receive: ok
Route: push -> ops:deploy
Execute: ok
Reply: status JSON

Maturity: status-only
```

For incomplete services, tests should succeed only for verified layers:

```text
Service: support-bot
Type: message
Adapter: telegram
Profile: assistant

Receive: not verified
Route: ok
Execute: placeholder
Reply: simulated

Maturity: component
```

## Documentation Structure

Recommended docs:

1. `docs/user/services.md`
   - User-facing setup and status model.
   - Only supported behavior.

2. `docs/dev/services.md`
   - Service architecture and command mapping.
   - Relationship to adapters, protocols, kit aliases, and core execution.

3. `docs/dev/service-maturity.md`
   - Maturity labels and support policy.
   - Matrix of current service types.

4. Existing protocol docs remain, but should link back to services:
   - A2A is a service type.
   - ACP is a client service type.
   - Agent Protocol is the API service implementation.
   - Webhooks are the event-trigger service implementation.

## Implementation Principles

- Do not expose component-level handlers as ready services.
- Do not mark a service `ready` unless it has a user command, a reachable listener, real execution semantics, reply semantics, and tests.
- Keep protocol names visible in advanced details, not first-run setup.
- Implement service type aliases through kit aliasing, then persist canonical `type` and `adapter`.
- Keep `receives`, `executes`, and `replies` visible in status and test output.
- Prefer one service command group over scattered setup commands in docs.

## Recommended First Cut

Phase 1 should only bless the two strongest current paths:

- `api`: Agent Protocol via `aps serve`.
- `webhook`: generic webhook via `aps webhook server`.

Phase 2 should reconcile partially built paths:

- A2A executor: wire profile-backed execution or label as placeholder.
- ACP HTTP/WebSocket: wire transport config or document stdio only.
- Messenger: mount handler and invoke real profile actions, or document subprocess-only adapter behavior.
- Mobile: decide whether WebSocket commands execute profile actions.
- Voice: separate backend lifecycle from channel listener services.

## Follow-Up Tasks

Track: `aps-service-ux`

| Task | Purpose |
| --- | --- |
| T-0601 | Implement service type aliases through kit aliasing. |
| T-0603 | Define Agent Protocol service output and reply semantics. |
| T-0604 | Resolve `aps serve` ProtocolRegistry mismatch for service architecture. |
| T-0605 | Decide and wire generic protocol HTTP bridge service behavior. |
| T-0606 | Wire A2A service execution to profile-backed actions or document placeholder maturity. |
| T-0607 | Verify A2A transport auth and push support claims for service docs. |
| T-0608 | Fix ACP service command and transport mismatch. |
| T-0609 | Mount message service handlers through a user-facing service path. |
| T-0610 | Wire message service routing to real profile action execution. |
| T-0611 | Refresh message service CLI docs and examples. |
| T-0612 | Document or implement webhook service reply semantics. |
| T-0613 | Keep events service observe-only until route dispatch exists. |
| T-0614 | Resolve mobile service HTTPS endpoint and TLS behavior. |
| T-0615 | Wire mobile service WebSocket commands to APS core execution or mark placeholder. |
| T-0616 | Mount voice channel services or mark voice adapters component-only. |
| T-0617 | Add service reachability and maturity test coverage. |
| T-0618 | Add service gap coverage matrix to UX draft. |
| T-0619 | Correct ACP and protocol architecture docs for service maturity. |
| T-0596 | Add Discord support to the messenger normalizer. |
| T-0597 | Add SMS as a `message` adapter. |
| T-0595 | Add WhatsApp as a `message` adapter. |
| T-0598 | Add Jira as a `ticket` adapter. |
| T-0599 | Add Linear as a `ticket` adapter. |
| T-0600 | Add GitLab as a `ticket` adapter. |

The UX should be strict: a service is either ready, status-only, observe-only, placeholder, component-level, or planned. That keeps users from mistaking a scaffold for a working integration.
