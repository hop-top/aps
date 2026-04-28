# Agentic Multi-Protocol Runtime (AMPR) — Design

**Date:** 2026-04-07
**Status:** Draft
**Track:** [aps-agent](../../.tlc/tracks/aps-agent/)
**Subcommand:** `aps agent`

---

## Problem

APS has chassis of agent platform. Missing screws to start running.

Shipped surface: Agent Skills (agentskills.io standard, federated from Claude Code, Cursor, VS Code, Gemini CLI, Zed, Windsurf), three-layer scope, capability bundles, squad topologies, A2A protocol, ACP for editor integration, AGNTCY trust + observability + identity. Profiles sandboxed, sessions tracked, secrets managed, webhooks routed.

Missing: anything that drives an LLM. A2A `Executor.Execute()` is placeholder. ACP `session/prompt` returns `ErrNotImplemented`. Skills discoverable, renderable into context — no caller consumes catalog or dispatches tool calls.

This doc specifies LLM-driving loop. One new package, seven wiring points: CLI frontend, Agent Protocol HTTP+SSE, A2A executor, ACP session handler, webhooks, messenger adapters, squad peer-call tools. Everything else APS has — and the bits consumed from hop.top toolkit — reused as-is.

---

## What APS already has

Loop is the only entirely-new runtime component besides MCP client. Everything below shipped or wired in `main`. Some pieces are placeholders the loop *replaces* rather than wraps.

### From APS itself

**Fully shipped — consumed as-is:**

| Primitive | Package | What it does |
|---|---|---|
| Agent Skills | `internal/skills/` | Discovery from profile/global/user/IDE paths, frontmatter parser, hierarchical override, XML/JSON/YAML adapters, secret replacement, telemetry |
| Three-layer scope | `internal/core/scope/` | Profile ∩ squad ∩ workspace intersection; resolves effective `Rule` per turn |
| Capability bundles | `internal/core/bundle/` | Named presets that union-merge into profile scope; binary policies; `deny_flags`; per-tool runtime overrides |
| Squad topologies | `internal/core/squad/` | Team-Topologies primitives; contracts with input/output schemas; router; evolution monitor; topology checklist; `ContextLoad` struct (loop populates) |
| Profile execution | `internal/core/execution.go:RunAction` + `internal/core/isolation/manager.go:ExecuteAction` (per-OS) | Process, platform, container isolation; env injection; secrets. Built-in tools and skill scripts both ride this path. |
| Agent Protocol HTTP+SSE | `internal/adapters/agentprotocol/` + `internal/core/protocol/core.go:APSAdapter.ExecuteRun` | HTTP runs/threads/skills handlers; SSE streaming; routes runs to `RunAction` |
| A2A protocol scaffolding | `internal/a2a/` (server, client, agent card, three transports, storage) | Full A2A v0.3.4 stack |
| ACP protocol scaffolding | `internal/acp/` (server, sessions, fs, terminal, permissions, capabilities) | Full ACP stack minus placeholders below |
| AGNTCY observability | `internal/agntcy/observability/` | OTel spans, metrics, propagation |
| AGNTCY identity | `internal/agntcy/identity/` | DID generation, message signing, Verifiable Credentials |
| AGNTCY trust verifier (logic) | `internal/agntcy/trust/verifier.go` | Full Ed25519 + DID verification |

**Placeholders or bugs the loop must fix:**

| Item | Problem | Fix |
|---|---|---|
| `internal/a2a/executor.go:Execute()` | Body is `Sprintf("Processed: %s", textPart.Text)` echo at lines 59-68 | Phase 4 replaces body with loop invocation, preserves event-emission scaffolding |
| `internal/acp/server.go:handleSessionPrompt`, `handleSessionCancel` | Both return `ErrNotImplemented` | Phase 4 fills in both handlers using same loop |
| AGNTCY trust hook callsite at `internal/a2a/server.go:184-191` | Calls `verifier.Verify(ctx, "", "", nil)` with empty inputs — verification cannot reject anything except missing-DID-when-required, signature verification always skipped | Phase 1 one-line fix to use `VerifyHTTP(ctx, r, body)` so loop reads verified sender DID |
| Agent Protocol HTTP loop integration | Routes to action executor only; no LLM-driven runs | Phase 4 extends `core.ExecuteRun` to recognise agent action ID that drives loop instead of running script (existing action paths unchanged) |

**Documented but does not exist:**

| Item | Doc claim | Reality |
|---|---|---|
| `internal/acp/mcp_bridge.go` | "Connects MCP servers as callable tools" (`docs/dev/acp-implementation.md`) | File does not exist. No MCP code anywhere in `internal/`. **Phase 3 builds `internal/core/mcp/` from scratch.** |

### From the hop.top toolkit

**Already imported by APS** (in `go.mod`):

| Component | Import path | Used for |
|---|---|---|
| `kit/cli` | `hop.top/kit/cli` | cobra+fang+viper root command factory; `--quiet`/`--no-color`; CharmTone theme. `aps agent` registers under this. |
| `kit/log` | `hop.top/kit/log` | Viper-aware logger with hop.top theme |
| `kit/output` | `hop.top/kit/output` | Renderer + hint helpers for CLI output |
| `cxr` | `hop.top/cxr` | Execution router with `Handler` interface, `ProcessHandler`, `Capabilities` probing. Currently used for adapter dispatch. Loop's tool dispatchers adopt `cxr` so dispatch is uniform across adapters and tools (D11). |
| `upgrade` (top-level) | `hop.top/upgrade` | Self-upgrade against GitHub releases. Migrating to `hop.top/kit/upgrade`. |

**Newly required by this track** (not yet in `go.mod`):

| Component | Import path | Used for |
|---|---|---|
| `kit/llm` | `hop.top/kit/llm` | Provider abstraction (Anthropic, OpenAI, Google, Ollama, +9 OpenAI-compatible); `Streamer`; `ToolCaller`; fallback chains; event hooks. Loop's only new external dep doing heavy lifting. |
| `kit/upgrade` | `hop.top/kit/upgrade` | Replaces `hop.top/upgrade`. Adds `kit/upgrade/skill/preamble.go` — generates upgrade preamble fragments for AI agent skill files, with `SnoozeLevel` controls. Loop calls `skill.Preamble(...)` when rendering skill into context. |
| `kit/tui` | `hop.top/kit/tui` | Pre-themed bubbletea/v2 components: `Badge`, `Confirm`, `Progress`, `List`, `Dialog`, `Spinner`, `Status`, `Pills`. REPL frontend (Phase 6) and `aps agent serve` status views use these. |
| `kit/config` | `hop.top/kit/config` | Layered YAML loader: system → user → project → env. Loop's `kit/llm` config resolution (D6) uses this. |
| `kit/xdg` | `hop.top/kit/xdg` | XDG Base Directory paths with macOS/Windows fallbacks. Used wherever loop reads/writes per-user state. |
| `kit/sqlstore` | `hop.top/kit/sqlstore` | SQLite-backed key-value store. Available if loop needs storage-like functionality (e.g. session state for `aps agent serve`). Not used to replace `tip`'s own persistence. |
| `tip` | `hop.top/tip` | JIT command correction: pre-flight `tip suggest` + post-failure rescue; learned-fix retrieval before LLM fallback. Wraps tool dispatch as thin layer. |

### Out-of-tree but federated automatically

Skill registry already discovers `~/.claude/skills/`, `~/.cursor/skills/`, `~/.vscode/skills/`, `~/.gemini/skills/`, `~/.config/zed/skills/`, `~/.windsurf/skills/`. Skill written for any of those tools visible to `aps agent` with no modification. Skill written for `aps agent` (canonical Claude Code form) visible to those tools without modification.

### Net new packages

Two:

1. **`internal/core/agent/`** — loop, tool registry, meta-tools, squad peer-call synthesizer, `kit/llm` integration, factory used by all seven call sites.
2. **`internal/core/mcp/`** — JSON-RPC client, stdio + WebSocket transports, server registry, tool discovery, dispatcher. Built fresh because existing ACP doc claimed MCP bridge that does not exist on disk.

Everything else: consumption, replacement, or wiring.

---

## The loop

`aps agent` is single Go package, `internal/core/agent/`, that drives an LLM through a turn-based conversation. One package, three CLI frontends, seven wiring points.

### What the loop is

Turn state machine. Each turn:

1. Builds `kit/llm.Request` from message history, resolved system prompt, resolved tool list.
2. Calls `Streamer.Stream(ctx, req)` (or `Complete` if streaming unavailable). Accumulates streamed tokens into pending assistant message.
3. On tool call from stream, dispatches via injected `ToolRegistry`. Appends result. Continues until stream emits `end_turn` stop reason.
4. Emits structured events at every step: `token`, `tool_start`, `tool_end`, `tool_iron`, `tool_rescue`, `turn_end`. Subscribers attach for streaming UIs, observability, telemetry.

Loop knows nothing about CLIs, A2A, ACP, squads directly. Accepts `Context`, `Config` (from `kit/config`), `ToolRegistry` (interface), starting message. Returns when turn ends or context cancels.

### Cancellation

Loop honours `ctx.Done()` at every safe point: between turns, between LLM stream chunks, before and during tool dispatch, inside long-running tool calls (`ToolRegistry.Dispatch` passes `ctx` through). Cancellation cooperative — tool that ignores `ctx` runs to completion before loop reads signal — but built-in tools and `cxr`-routed dispatchers honour it.

Cancelled loop returns `context.Canceled`. Emits final `turn_end` with `StopReason: "cancelled"`. Partial assistant messages NOT persisted to A2A storage; caller decides retry vs discard. ACP `session/cancel` and CLI `SIGINT` handler both produce cancel via same context.

### Error recovery

Three categories, three strategies:

1. **LLM provider errors** (network, rate limit, 5xx) — handled by `kit/llm`'s built-in fallback chain. Loop sees successful stream from next provider, or `ErrFallbackExhausted` if all fail. No loop-level retry.
2. **Tool dispatch errors** — wrapped by `tip` post-failure rescue. `tip` retrieves learned fix from vector store; if found, loop transparently substitutes corrected call and retries once. If still no fix, error surfaces to LLM as tool result with `isError: true`. LLM decides next action.
3. **Loop-internal errors** (registry corruption, schema validation failure on contract-typed peer call) — non-recoverable. Returned to caller via `Run() error`. No partial state persisted.

### Steering and follow-up

Borrowed from `pi-agent-core` pattern. Two queues:

- **Steering queue** — messages injected *during* running turn. Loop drains between tool calls and before each LLM call, prepends drained messages to next request. Use case: user typing "stop, do this instead" while agent is running tools.
- **Follow-up queue** — messages injected *after* loop would otherwise stop. When turn reaches `end_turn` and follow-up queue non-empty, loop drains queue and starts another turn instead of returning. Use case: queuing "also summarise the result" while agent is mid-task.

Both queues support `one-at-a-time` (default — drain one message per check) or `all` (drain everything queued). CLI REPL frontend wires `^C` to steering queue. A2A and ACP push to queues via existing message endpoints. Phase 6 wires both queues; v1 ships with `one-at-a-time` mode.

### Why one package, not several

Multiple call sites need the loop. Putting it in any one protocol package forces others to import across protocol boundaries — forbidden by existing layering rules (`internal/core` cannot import `internal/cli`; same spirit between protocol packages).

Loop lives in `internal/core/agent/` for same reason `scope`, `bundle`, `squad` live in `internal/core/`: cross-cutting primitive any caller can construct and run. Dependency direction stays inward: every caller depends on `internal/core/agent/`, never the reverse.

### The seven wiring points

| # | Call site | What it builds | What it does with the loop |
|---|---|---|---|
| 1 | `cmd/aps/agent.go` (CLI) | Loop with built-in toolkit; optional `--skill` activation | Streams tokens to stdout (one-shot), `kit/tui` REPL (interactive), or local socket (`serve`); exits when turn ends |
| 2 | `internal/adapters/agentprotocol/runs.go` (HTTP+SSE) | Loop bound to Run/Thread; skill set from request payload | Streams loop events as SSE Run events via existing `sse.go`; persists run state via existing thread store |
| 3 | `internal/a2a/executor.go` | Loop bound to inbound A2A task; sender DID propagated from AGNTCY trust hook | Streams events back via `eventqueue.Queue`; persists task state via existing A2A storage; runs inside request's OTel span |
| 4 | `internal/acp/session.go` | Loop with ACP fs/terminal methods exposed as tools; session mode enforced via `beforeToolCall` | Streams `session/update` notifications; honours `default`/`auto_approve`/`read_only` mode; permission system intercepts sensitive calls |
| 5 | `internal/webhook/loop_dispatcher.go` (new) | Loop fed inbound webhook payload as prompt; profile resolved from webhook routing | Runs single turn; optionally posts final assistant message back to webhook source (e.g. GitHub PR comment); persists nothing unless post-back tool called |
| 6 | `internal/adapters/messenger/loop_dispatcher.go` (new) | Loop fed inbound messenger message + sender metadata as prompt | Posts final assistant message back through adapter's outbound API to originating channel (Telegram, Discord, Slack, etc.) |
| 7 | `internal/core/agent/peers.go` | Synthetic peer-call tools constructed from squad contracts (not top-level frontend; contributes tools to registry the loop sees from any of #1–#6) | Outbound A2A `SendMessage` to peer profiles, gated by contract input/output schemas, caller identity propagated for downstream `AllowedIssuers` checks |

Loop constructed same way at all six top-level sites (#1–#6) via single factory in `internal/core/agent/factory.go`. Squad peer tools (#7) are tool source, not frontend — any frontend's loop can pull them in. Differences between sites: *what tools the registry contains* and *what events the caller subscribes to*. Not the loop itself.

### Future call sites: voice

APS voice subsystem (`internal/voice/`) is speech-to-speech via PersonaPlex and Moshi. LLM lives inside voice backend, not inside `aps agent`. Bridging voice to text loop has three possible shapes:

1. Voice does STT only and ships text to loop. Breaks voice subsystem's sub-200ms latency budget.
2. Voice keeps own LLM but borrows `aps agent`'s tool registry via MCP. Voice doesn't call loop; calls *tools* loop would have called. Sound but separate integration point.
3. Second loop variant optimised for low-latency turn-taking, sharing tool registry. Substantial work.

None small enough for v1. Voice deliberately deferred. Follow-up track decides which shape. Text-loop architecture below built so option 2 (shared tool registry) is straightforward when its time comes.

---

## The tool layer

Tool layer is where most of the design work landed and where most of `aps agent`'s leverage comes from. Three representations connected by two transformations:

```
   author writes                                                  LLM sees
       ↓                                                              ↑
[any supported input] ──[normalize]──> [canonical on-disk] ──[collapse]──> [LLM view]
                              ↑                  ↓
                              │            (rewritten to disk)
                       write / install
```

Principle: **authors write what feels natural, system stores single canonical form, LLM sees small collapsed catalog**. Each layer optimises for different constraint.

### The input layer

Skill authors write `allowed-tools` in any of four supported forms:

| Form | Example | Audience |
|---|---|---|
| Claude Code scoped | `Bash(git:*) Bash(jq:*) Read Write` | Claude Code skill authors, federation use cases |
| Flat dotted | `bash tlc.task.create tlc.task.list read_file` | non-technical authors |
| Flat with name globs | `bash tlc.* read_file` | authors who want broad allowance |
| Mixed | `Bash(git:*) tlc.task.* read_file` | anyone |

Grammar covering all four: space-delimited tokens, where each token is name optionally followed by `(scope)`. Names may contain any character except whitespace, `(`, `)`. Names may contain `*` for globbing. Multiple tokens with same name and different scopes union (`Bash(git:*) Bash(jq:*)` allows both).

**Notation does not impose naming convention.** Tool names are freeform strings; `Bash` and `bash` equally valid. APS conventions (Claude-compatible PascalCase for built-ins, snake_case for APS-native, dotted for hop.top CLIs and MCP) live in registry, not parser.

### The canonical storage layer

Canonical form is **Claude Code scoped**. Two reasons:

1. **Lossless.** Scoped parens express anything flat forms can — subcommand restrictions become comma-separated scope (`tlc.task.create tlc.task.list` → `Tlc(task.create,task.list)`) — while flat forms cannot losslessly express scope DSLs richer than dotted tails. `Bash(git:*)` has no flat equivalent.
2. **Federation format.** Skills written in or for `~/.claude/skills/`, `~/.cursor/skills/` already in canonical form, round-trip with zero changes. APS-authored skills, after normalization on save, copy to other IDE skill directories without modification.

**Normalization happens at `aps skill install` and `aps skill validate`.** Rewrites file in place. Author sees convenience input transformed into canonical form, learns canon by osmosis, file storage form stable across authoring tools. Discovery passes (`registry.Discover`) never rewrite — only explicit author touchpoints do, so version-controlled skill files don't get silently churned.

Normalizer also performs safe simplifications: deduplication, glob subsumption (`Bash(git:*) Bash(git:push)` collapses to broader pattern), canonical-name resolution for aliases (`bash` → `Bash`, `tlc.task.create` → `Tlc(task.create)`). Unknown tool names produce warnings, not errors. Preserved verbatim — skill may target different environment with more tools.

### The runtime view

Agent loop reads canonical form, constructs *collapsed* tool catalog for LLM. Related fine-grained subcommands group under single parent tool with `oneOf`-discriminated parameter schema:

```
Canonical: Tlc(task.create,task.list,task.claim) Bash(git:*) Read

LLM catalog:
  - Bash         (with scope hint in description)
  - Read
  - Tlc          (oneOf {task.create | task.list | task.claim})
```

LLM sees three entries instead of five. With dozens of hop.top CLIs and their subcommands, this is the difference between 200-tool catalog and 20-tool catalog.

Collapsed schema for `Tlc` is JSON Schema `oneOf` with discriminated union on `subcommand` enum, where each branch carries right `args` shape for its subcommand. Verbosity lives behind `describe_tool`, which is the progressive-disclosure moment — point where LLM is specifically asking for detail. Always-loaded catalog only carries names and one-line descriptions; full collapsed schemas pulled on demand.

This is second level of progressive disclosure. First level: skill descriptions (rendered into prompt) versus skill bodies (fetched via `read_skill`). Second level: tool names with one-line summaries (in catalog) versus collapsed schemas (fetched via `describe_tool`).

### Tool registry composition

Runtime view is the *output*. Input to runtime collapse step is resolved tool set for current turn. Registry assembles from multiple sources, then filters through several layers:

```
Sources                   Filters                      Output
─────────                 ───────                      ──────
built-in toolkit  ──┐
bundle binaries   ──┤
MCP bridge        ──┤──> registry ──> scope ─┐
peer-call tools   ──┤                         │
skill-bundled     ──┘                          │
                                       active skills ─┐
                                                       │
                                              bundle policies ─┐
                                                                │
                                                            collapse ──> LLM view
```

The five sources:

1. **Built-in toolkit** — `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`, `ReadSkill`, `DescribeTool`. Shipped with `aps agent`. Dispatched via `cxr` so they run under active profile's isolation tier with same policies as any other APS-managed execution.
2. **Bundle binaries** — every binary listed in active bundles' `requires` becomes flat tool (e.g. `developer` bundle's `git`, `gh`, `docker`). Subject to bundle's `deny_flags` and `runtime_overrides`. Dispatched via `cxr` for same isolation guarantees.
3. **MCP servers** — every tool from every MCP server configured in profile, fetched via new `internal/core/mcp/` package built fresh in Phase 3. (Existing ACP doc described MCP bridge that does not actually exist on disk; audit confirmed no MCP code anywhere in `internal/`.) Profiles declare MCP servers in `mcp:` config block; each server entry specifies `name`, `transport: stdio|ws`, `command` (for stdio) or `url` (for ws), env, args.
4. **Squad peer-call tools** — synthesised from squad contracts where active profile is on consumer side. One tool per `x-as-a-service` contract; parameter schema = contract `input_schema`; dispatcher = outbound A2A `SendMessage` to member of provider squad. Validated against `output_schema` on return. Disappears from registry when contract timeboxes expire (D2 + D3 defaults).
5. **Skill-bundled scripts** — when skill is active, scripts from its `scripts/` directory become tools (named `<skill>.<script>` by default). Dispatched through existing `internal/core/execution.go:RunAction` + `internal/core/isolation/manager.go:ExecuteAction` per-OS path — same path profile actions ride. Skill registry already exposes `Skill.HasScript()`, `GetScriptPath()`, `ListScripts()`. Loop iterates active skills and registers each script as flat tool. This is the work the existing `aps skill run` CLI command was meant to do (currently TODO returning `not yet implemented`); loop's skill-script source closes that gap.

The four filters apply in order, all intersection-style (tool must survive every filter to be included):

1. **Profile/squad/workspace scope** — `scope.Rule.Tools` from `internal/core/scope` resolves most-restrictive set across all three layers. Done by existing scope logic.
2. **Active skill `allowed-tools`** — canonicalized patterns from each active skill. Tool must match at least one active skill's pattern (or be meta-tool like `read_skill` and `describe_tool`, which are always available regardless of skill activation).
3. **Bundle binary policies** — bundles can `block` specific binaries even if they would otherwise be in scope. Bundle `deny_flags` not enforced at registry time but at dispatch time, by intercepting tool calls before `cxr` runs them.
4. **AGNTCY trust constraints** (peer-call tools only) — for tools originating from squad contracts requiring specific issuers, calling profile's verified DID must match contract's `AllowedIssuers`. Other tool sources not affected.

**Five sources, four execution routes.** Built-in tools and bundle binaries dispatch via `cxr` (D11). Skill-bundled scripts dispatch via `RunAction`/`ExecuteAction` because they are profile-scoped scripts and existing action execution path already handles isolation, env injection, secrets for that case. MCP tools dispatch via `internal/core/mcp/`. Peer-call tools dispatch via `internal/a2a/client.go:SendMessage`. Loop's tool dispatcher is small switch on tool source, not single uniform path — but each source's dispatch *is* uniform within its category.

### Why this is more than just an allow-list

Three things this layer does that flat allow-list can't:

1. **Two-level progressive disclosure** — skills filter which tools are even considered, then `describe_tool` filters how much detail any one tool reveals. Agent's context budget stays small even with hundreds of registered tools across active profile.
2. **Composition without coupling** — built-ins, bundles, MCP, peer calls, skill scripts all flow through same registry and same filters. Adding new source (e.g. webhooks-as-tools, voice-as-tools) means writing one source adapter, not modifying loop.
3. **Squad topology becomes runtime tool topology** — when squad contract is active, its tool exists; when contract expires, gone. Inverse-Conway-maneuver from squad spec made executable: topology you designed is literally the agent's tool layer at runtime, with `ContextLoad` measuring the cost.

Three-layer model — input formats, canonical storage, runtime view — and five-source/four-filter registry composition together are the bulk of `aps agent`'s tool layer. Rest of design (progressive disclosure, squad-aware multi-agent, `tip` integration) builds on top of these primitives.

---

## Progressive disclosure

Architectural answer to: *how does an agent with access to dozens of skills and hundreds of tools fit any of that into a context window?*

Three levels in `aps agent`. Each with one-line entry visible by default and fetch-on-demand expansion behind meta-tool.

### Level 1 — Skills

Agent's system prompt always includes **skill catalog**: every skill the registry resolved for active profile, rendered via existing `Registry.ToPromptXMLFiltered` as one `<skill>` element each, with `<name>`, `<description>`, `<location>`. Roughly 50–100 tokens per skill. Skill *bodies* — Markdown content with instructions, examples, references — not loaded.

When LLM decides skill is relevant to current task, calls meta-tool `read_skill(name)`. Dispatcher consults registry, checks active filters, returns skill's body content. If skill filtered out (by platform, protocol, isolation level, compatibility), `read_skill` returns structured "filtered out" error distinct from "not found" — LLM learns whether skill exists at all versus exists but is unavailable.

`read_skill` always available, regardless of which skills active. Meta-tool, not regular tool. Never appears in any skill's `allowed-tools` list. **`read_skill` is read-only — does NOT activate the skill.** LLM uses it to drill into a skill's body for context. Skill *activation* (which determines `allowed-tools` filtering) is caller-set: `--skill` CLI flag, A2A request payload field, ACP session params, webhook config, messenger adapter config. LLM cannot grant itself tool access by reading a skill (D12).

### Level 2 — Tool catalog

Agent's system prompt also includes **collapsed tool catalog** — runtime view from Section 4. Each entry is parent tool name plus one-line description; per-subcommand schemas not loaded. Profile with `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`, `Tlc` (8 subcommands), `Wsm` (5 subcommands), `Mcp_filesystem` (4 tools): catalog has 9 entries, not 25. Profile that adds another bundle bringing in 30 hop.top CLI subcommands grows catalog by 1 entry (new parent), not 30.

When LLM wants to call tool whose detail it doesn't have, calls `describe_tool(name)`. Dispatcher returns full collapsed schema for that tool — `oneOf`-discriminated union for tools with subcommands, flat schema for tools without. Verbosity of `oneOf` schemas only ever materialises for tools LLM is specifically considering.

`describe_tool` honours scope and skill filters. Tool that exists in registry but is filtered out by active skill's `allowed-tools` returns "not found" — *not* "filtered out". Deliberate information-hiding choice: telling LLM that tool exists but is restricted invites it to ask why, which adds noise. From LLM's view, restricted tools simply do not exist in this turn (D13).

Both meta-tools live in `internal/core/agent/tools/meta.go`. Registered as flat tools with no subcommand collapse. Only tools that bypass skill `allowed-tools` filtering — LLM always has access to them.

### Level 3 — Context budget feedback

Third level isn't user-facing in same way. After every turn, loop measures actual context usage and writes result back to active squad's `ContextLoad`:

```go
contextLoad.ToolSchemas      = len(registry.Resolved())
contextLoad.DomainKnowledgeKB = len(renderedSystemPrompt) / 1024
contextLoad.InteractionProtos = countOfPeerCallTools(registry)
contextLoad.SessionMemoryKB   = len(messageHistory) / 1024
```

`squad.ContextLoad` struct already exists in `internal/core/squad/context_load.go`. Loop is its first producer. With these values populated, `coordination_ratio = (interaction_protos × 1KB + session_memory_KB) / total` becomes real runtime metric. Topology checklist's `topology-first` check (`coordination_ratio < 0.5`) reflects what loop is actually doing rather than static struct full of zeros.

Feedback loop closes here: squad whose context is dominated by peer-call tools and message history rather than domain knowledge is by definition badly scoped. `aps squad check` will surface it. Agent loop is source of truth for measurement; squad layer is where consequences are evaluated.

### Why three levels

Three properties fall out of three-level structure that wouldn't exist with flat catalog:

1. **Catalog stays bounded as surface grows.** Every new bundle, every new skill, every new MCP server contributes O(1) catalog entries, not O(n) per subcommand or per tool. Profile with 50 skills and 200 tools across 15 sources still presents fewer than 30 always-loaded entries to LLM.
2. **Agent learns its way down.** Default state: "you have access to all of this; ask if you need detail". Skills are entry points for *task families*, parent tools are entry points for *capability families*, LLM decides which to drill into based on prompt. Drilling in is single tool call, not context refill.
3. **Cost of being wrong is bounded.** If LLM `read_skill`s skill it doesn't actually need, body is in context for rest of turn but doesn't pollute future turns. If it `describe_tool`s wrong tool, same thing. No permanent context creep from speculative exploration.

Three levels work together: skills tell LLM *what kinds of tasks* are supported, tool catalog tells it *what kinds of actions* are available, `ContextLoad` tells squad layer *what the cost looks like in practice*. None sufficient alone; together they let agent operate with dozens of skills and hundreds of tools and still fit in 200K context window with room to actually think.

---

## Squad-aware multi-agent

Squad in APS is Team-Topologies team unit: set of profile members, domain, type (`stream-aligned` / `enabling` / `complicated-subsystem` / `platform`), set of contracts that govern how it interacts with other squads. Squad layer ships in `internal/core/squad/` with router, evolution monitor, exit-condition tracker, timebox tracker, topology checklist. None currently drives runtime behaviour — planning and accounting surface.

Agent loop is first consumer that turns squads into runtime. When profile is acting as squad member, loop's tool registry pulls in **squad peer-call tools** synthesised from squad's contracts. Squad topology becomes agent's runtime tool topology.

### Peer-call tools

For each `x-as-a-service` contract where active profile's squad is on consumer side, loop synthesises one tool:

```
Name:   peer_<provider_squad_id>__<capability>
Params: contract.input_schema     (used directly as LLM-facing param schema)
Dispatcher: outbound A2A SendMessage to member of provider squad
            (one is picked via squad router; others tried on failure)
```

**Schema is the contract.** Contract's `input_schema` field is JSON Schema; `kit/llm.ToolDef.Parameters` accepts JSON Schema; two passed through unchanged. No translation layer. No hand-written wrapper. Squad contracts are source of truth. Peer-call tool that violates a contract is impossible by construction.

When dispatcher receives tool call:

1. Validates call against `input_schema` (already done by `kit/llm` before dispatch — no-op in normal cases).
2. Constructs A2A `Message` with call args as message body.
3. Resolves target profile via squad router (one provider-squad member selected).
4. Calls `internal/a2a/client.go:SendMessage` with resolved agent card.
5. Blocks until task completes (D2 default: synchronous wrapper). Timeouts come from contract's `sla.max_latency`.
6. Validates response against `output_schema`.
7. Returns validated result to loop, which feeds it back to LLM as tool result.

If validation fails on either side, or if A2A call errors, loop receives structured error and surfaces it as `isError: true`. LLM decides whether to retry, pick different tool, or report failure.

### Contract lifecycle becomes tool lifecycle

Squad contracts have lifetimes. `collaboration` contract has `timebox` (duration measured against `started_at`). `facilitating` contract has `exit_condition` defining when enabling squad's engagement is complete. Both tracked by `internal/core/squad/timebox.go` and `internal/core/squad/exit_condition.go`.

Loop's tool registry honours these. Peer-call tool synthesised from a contract:

- **In registry** while contract is active (timebox not expired, exit condition not met)
- **Disappears** as soon as contract goes inactive
- **Reappears** if squad layer extends or recreates contract

Profile that just finished 2-week collaboration with another squad will, on its next turn, find peer-call tool gone — without explicit re-registration. Squad layer is source of truth for what tools exist. Loop reflects current state at registry-resolve time, which happens at start of every turn.

### Identity propagation

When peer-call tool is invoked, calling profile's verified identity (from AGNTCY trust hook fix in Phase 1) flows through to outbound A2A message as `X-Agent-DID` and `X-Agent-Signature` headers. Provider profile's trust verifier sees calling profile's DID and applies its own `AllowedIssuers` policy. Peer call from profile whose DID is not in provider's allowed issuers list is rejected at trust hook, before it ever reaches provider's executor or loop.

Chain that makes squad contracts cryptographically meaningful, not just policy-meaningful: contract says "consumer squad C may call provider squad P with these schemas", trust layer enforces it at message boundary using same DIDs the squad CRDs reference. Without AGNTCY fix in Phase 1, chain breaks at very first hop because inbound trust hook never reads headers.

### `ContextLoad` as topology validator

Per Section 5, loop writes back to `squad.ContextLoad` after every turn. For squad members, most interesting field is `interaction_protos` — count of peer-call tools in resolved registry. Squad whose members are running loops with many peer-call tools and few domain tools is, by squad spec's definition, badly topologically scoped: `coordination_ratio = (interaction_protos × 1KB + session_memory_KB) / total` will trend toward 1.

`aps squad check`'s `topology-first` check evaluates `coordination_ratio < 0.5` against this populated value. With loop running, `topology-first` becomes live metric: squad whose members spent the last day calling four peer squads will fail check, even if its CRDs and contracts are well-formed. Check moves from "are the contracts shaped right" to "is the actual runtime well-scoped".

Inverse-Conway-maneuver from squad spec made executable. Squad topology you designed is what agent actually has at runtime. If topology is wrong, runtime measurement will surface it via existing topology checklist — not as separate report, but as same `aps squad check` command that already exists.

### Why peer calls are not "just A2A"

A2A is the *transport*. Squad contract is what makes the call a *typed, gated, lifecycle-aware* tool instead of arbitrary message. Three things distinguish peer-call tool from raw `a2a.SendMessage`:

1. **Schema is the contract**, not freeform `parts`. LLM sees typed parameter schema and typed return schema. Validation enforced by both `kit/llm` (input) and loop (output).
2. **Lifecycle is the contract's lifecycle.** Tools appear and disappear with timeboxes and exit conditions. Raw A2A has no such notion.
3. **Authorization is the squad's `AllowedIssuers` plus the contract's existence.** Profile cannot call peer just because it knows agent card; squad layer must have contract that says it can, trust layer must verify DID.

Result: `aps agent` running as member of one squad can call its squad's contracted providers as if they were local tools. Squad spec's design constraints become runtime invariants enforced at LLM tool-call boundary.

---

## `tip` integration

`tip` is separate hop.top tool that learns command corrections. Model is `(command, error message) → fix`, with vector store for retrieval and LLM fallback when no learned fix matches. Runs as CLI wrapper around any command, as HTTP server, as Go SDK.

`aps agent` integrates `tip` as thin wrapper around tool dispatch, in two places: pre-flight (before tool runs) and post-failure (after tool fails). D4's default — both hooks active.

### Pre-flight: `tip suggest`

Before tool dispatcher executes, loop calls `tip.Suggest(toolName, args)` against local `tip` Go SDK. `tip` checks its learned-fixes store for arguments that match known typo or convention error. If fix found:

- Fix applied silently (corrected args replace LLM's args)
- Loop emits `tool_iron` event with `{originalArgs, correctedArgs, confidence}` so callers can show correction in UI
- Tool runs with corrected args

If no fix found, tool runs with LLM's original args.

### Post-failure: `tip` rescue

When tool dispatcher returns error, loop calls `tip.Rescue(toolName, args, error)` before returning error to LLM. `tip` first checks learned-fixes store; if fix matches, loop transparently retries tool call with corrected args. If retry succeeds, LLM never sees original error — sees successful result of retry, loop emits `tool_rescue` with same shape as `tool_iron`.

If `tip`'s fallback hits LLM (it has its own provider config), corrected args go through same retry path. If `tip` finds nothing or retry also fails, error surfaces to LLM as tool result with `isError: true`. LLM decides what to do next — same fallback as normal tool failure.

### Learning loop

Both hooks feed `tip`'s store. When LLM corrects its own mistakes (original call fails, LLM tries again with different args, second call succeeds), loop submits `(toolName, originalArgs, error, correctedArgs)` tuple to `tip` as learned fix candidate. Over time, `tip` accumulates per-profile, per-tool corrections without anyone explicitly training it. Next time LLM makes same mistake, `tip suggest` catches it pre-flight and LLM never sees the failure.

### Configuration and disable

`tip` integration on by default for `aps agent`. Can be disabled per-profile (`agent.tip.enabled: false`), per-tool (`agent.tip.exclude: [Bash]`), globally via `--no-tip` on CLI. Disabled means dispatch goes straight through with no interception.

`tip`'s own state lives wherever `tip` puts it (own XDG paths, separate from APS). Loop does not manage `tip`'s persistence — see Section 2's note on `kit/sqlstore`.

---

## Coexistence with prior decisions

This track adds the agent loop and the MCP client. Does not modify protocol layers, identity layer, squad layer's design, skill format, or existing isolation system. The list below: what `aps-agent` deliberately leaves alone, with pointers to docs that own each area.

### A2A protocol-level decisions

`docs/specs/005-a2a-protocol/adrs/` contains nine ADRs (001–009) covering serialization, message IDs, transport fallback, compression, large payload references, message expiration, ordering, batching. Plus `docs/specs/005-a2a-protocol/decisions.md` (16KB of resolved questions on protocol choice, message format, conversation model, communication patterns, transport bindings, agent discovery, authentication, storage backend, CLI commands, legacy data migration).

This track touches none of those decisions. Agent loop fills in body of `internal/a2a/executor.go:Execute()` — function that lives *inside* the protocol layer those ADRs define. Replacing body of an executor is not protocol change. If Phase 4 task appears to require ADR change, that's a bug in the task.

### AGNTCY layers

`docs/plans/2026-03-01-agntcy-integration-gap-analysis.md` documents five layers — Discovery (L1), Messaging (L2 / SLIM), Identity (L3), Observability (L4), Security (L5). Layers 1, 3, 4, 5 shipped on `main`. Layer 2 (SLIM) on `feat/agntcy-slim-transport` waiting for upstream Go bindings.

Loop **consumes** AGNTCY surface (verified DIDs from L5 trust, OTel spans from L4, identity from L3) but does not modify any of it — except for one-line trust hook callsite fix in Phase 1 (`server.go:184-191`), which is bug fix to wiring that already exists, not design change to trust verifier itself. SLIM remains out of scope; when it lands, loop's outbound A2A peer calls inherit it for free because they go through existing transport selector.

### Skill format and discovery

`internal/skills/` and user-facing skill docs at `docs/user/skills/` define SKILL.md format, four-layer discovery (profile/global/user/IDE), secret replacement system, telemetry. Track adds **two new behaviours** that bolt onto existing surface: (a) `allowed-tools` parser/normalizer/rewriter pipeline that fills in Phase 6 of existing skill IMPLEMENTATION_PLAN, (b) `read_skill` meta-tool that exposes skill bodies through registry. Neither modifies file format, discovery logic, filter logic, or federation paths. Skill written for Claude Code today still works in `aps agent` tomorrow without modification.

### Squad spec and topology

`docs/dev/squad-topologies-spec.md` (theory) and `docs/dev/squads.md` (implementation reference) define squad types, contracts, interaction modes, topology checklist, `ContextLoad` model. `internal/core/squad/` is the implementation.

Track adds **two consumers** of squad layer: (a) loop populates `ContextLoad` after every turn, turning it from static struct into live runtime metric, (b) loop synthesises peer-call tools from existing contracts, treating contract's `input_schema`/`output_schema` as LLM-facing tool schema. Neither changes squad CRDs, contract format, router, timebox/exit-condition logic, evolution monitor, or topology checklist itself. `aps squad check` still runs same eight checks; only `topology-first` reflects measured rather than default values.

### Isolation system and execution paths

`internal/core/execution.go:RunAction` and `internal/core/isolation/manager.go:ExecuteAction` (with per-OS implementations in `process.go`, `darwin.go`, `linux.go`, `windows.go`, `platform_sandbox.go`) define how APS runs profile-attached scripts under active isolation tier. Track uses this path **unchanged** for skill-bundled scripts (Section 4's source 5). Built-in tools and bundle binaries dispatch through `cxr` (D11), separate execution route that already exists in `internal/core/adapter/manager.go`.

Track does not introduce new isolation primitive, modify any per-OS isolation implementation, or bypass existing sandbox layers.

### Existing CLI commands

`aps profile`, `aps action`, `aps adapter`, `aps capability`, `aps bundle`, `aps squad`, `aps skill`, `aps a2a`, `aps acp`, `aps directory`, `aps identity`, `aps observability`, `aps policy trust`, `aps webhook`, `aps voice`, `aps session`, `aps docs` — all unchanged. Track adds `aps agent` as new top-level subcommand under existing `kit/cli` root. None of existing commands modified, deprecated, or behaviour altered.

One place where existing command's *internal* path changes: `aps skill run`. Today returns "not yet implemented". After Phase 4 it works because underlying skill-script execution path (which loop also uses for source 5) is now wired in.

### Other tracks

`docs/specs/006-agent-protocol-adapter/` defines Agent Protocol HTTP+SSE adapter that exists in `internal/adapters/agentprotocol/`. Loop integrates with adapter via option-1 pattern from Section 3 — extending `core.ExecuteRun` to recognise agent action ID — without modifying adapter's protocol implementation, SSE writer, or run/thread storage model.

`docs/specs/007-capability-management/` and `docs/specs/009-capability-bundles/` define capability and bundle systems consumed by Section 4's source 2 (bundle binaries). Unchanged.

`docs/specs/008-acp-protocol/` defines ACP server. Loop fills in `handleSessionPrompt` and `handleSessionCancel` (currently `ErrNotImplemented` stubs) and adds ACP fs/terminal methods to loop's tool registry as session-scoped tool source. Does not modify ACP transport, JSON-RPC framing, session manager, or permission system — only two stub handlers and tool exposure layer.

### What this list is for

Aim of this section: make boundary explicit so future contributor reading design doc knows two things: what they can rely on as already-decided (items above) and where to look for specs that own each area. Agent loop is consumer of a lot of existing surface. Temptation when implementing it: "fix things along the way". Don't. Bugs found in layers above belong in their own tracks, except where they directly block loop (AGNTCY trust hook callsite is only such case in v1, and it's explicit Phase 1 task).

---

## Open decisions

Twelve decisions surfaced during brainstorm. Two resolved by prior docs. Ten remain open with defaults marked **[D]**. Plan's tasks assume defaults; flipping any default requires revising corresponding tasks before that phase begins.

| # | Decision | Default | Status |
|---|---|---|---|
| **D1** | `allowed-tools` format | Three-layer model: any of {Claude scoped, flat dotted, flat with globs, mixed} as input; Claude Code scoped as canonical on-disk; collapsed `oneOf` as runtime view. Normalization rewrites file at install/validate. | **Resolved** by prior docs (`docs/user/skills/CREATING_SKILLS.md`) plus this track's brainstorm |
| **D2** | Peer-call tool synchrony | Synchronous wrapper — loop blocks turn while it streams peer task to completion | **Open** |
| **D3** | Squad contracts as peer-call schemas | Contract `input_schema` becomes LLM-facing parameter schema; `output_schema` validates response | **Open** |
| **D4** | `tip` integration depth | Both pre-flight (`tip suggest`) and post-failure (`tip rescue`) | **Open** |
| **D5** | MCP client | Build `internal/core/mcp/` from scratch in Phase 3 (existing ACP doc claimed MCP bridge that does not exist) | **Open** |
| **D6** | `kit/llm` config sourcing | Three-layer merge: profile.yaml `agent:` block → `~/.config/hop/llm.yaml` → `LLM_PROVIDER`/`LLM_API_KEY` env | **Open** |
| **D7** | Tool registry hierarchy mechanism | Explicit registration: tools declare parent name and prefix; "deepest registered prefix wins" disambiguates nested namespaces | **Open** |
| **D8** | Collapsed tool schema shape | `oneOf`-discriminated union with `subcommand` enum and per-branch `args` schemas | **Open** |
| **D9** | Claude-compatible built-in name aliases | Canonical = Claude name (`Bash`, `Read`, `Write`); flat snake_case (`bash`, `read_file`) accepted as input alias | **Open** |
| **D10** | Default subcommand alias generation | Auto-generated from parent prefix + subcommand name; explicit alias declarations override | **Open** |
| **D11** | `cxr` for built-in tool dispatch | Built-in tools and bundle binaries dispatch via `cxr.ProcessHandler`; skill scripts dispatch via `RunAction`/`ExecuteAction`; MCP via `internal/core/mcp/`; peer calls via `internal/a2a/client.go` | **Open** (informal; surfaced during brainstorm) |
| **D12** | Skill activation mechanism | Caller-set only — `--skill` CLI flag, A2A request payload, ACP session params, webhook config, messenger adapter config. `read_skill` is read-only; LLM cannot grant itself tool access. | **Resolved** during brainstorm |
| **D13** | `describe_tool` response for filtered-out tools | Return "not found" rather than "filtered out" — information hiding for restricted tools | **Open** |

### Bias toward locking before phase work begins

Defaults are not provisional. They reflect current best read after brainstorm, audit, and analysis in this design doc. Each has real reason in corresponding section above:

- **D2/D3** — Section 6, squad-aware multi-agent design. Sync because LLM expects sync; contract-as-schema because it makes peer calls type-safe by construction.
- **D4** — Section 7, `tip` integration. Both hooks because learning loop only closes when both feed `tip`'s store.
- **D5** — Section 2's audit + Section 4's tool sources. Build because there's nothing to promote.
- **D6** — Section 2's hop.top dependencies + spec. Three-layer merge mirrors `kit/llm`'s own merge.
- **D7/D8/D9/D10** — Section 4, tool layer. Explicit registration + `oneOf` + Claude compatibility + auto-aliases together make three-layer `allowed-tools` model coherent.
- **D11** — Section 4's footnote on execution routes. `cxr` for things already `cxr`-shaped; `RunAction` for things that already ride that path; new routes only where existing ones don't fit.
- **D12** — Resolved as caller-set only. Preserves principle that LLM doesn't unilaterally grant itself capabilities. Skill activation maps cleanly to existing `--skill` CLI flag.
- **D13** — Section 5's information-hiding choice. Defensive default; can be relaxed later without breaking compatibility.

### Decisions explicitly out of scope for this track

Few questions came up during brainstorm that could be decided now but are genuinely better as follow-up work:

- **Voice integration shape** (Section 3's "future call sites") — three options exist but none small enough for v1
- **`mcpgen` for hop.top CLIs** — auto-MCP every hop.top binary so they expose tool schemas via MCP rather than via bundle binaries; would unify Section 4's sources 2 and 3 but expands cross-repo footprint substantially
- **Long-running `aps agent serve` session persistence** beyond what A2A storage already provides
- **Multi-step squad contract negotiation** (collaboration/facilitating beyond static contract → tool mapping)
- **`tip`'s own Phase 1** as hop.top toolkit member (vector store schema, embedding model, fallback config) — `aps agent` consumes `tip` as-is; if `tip` itself needs work, separate track

These listed in spec's "Out of scope" section and will not be revisited in this design doc.

---

## Out of scope and what comes next

`aps-agent` v1 ships loop, tool registry, three-layer `allowed-tools` pipeline, seven wiring points, MCP client, AGNTCY trust hook fix, `tip` integration, squad peer-call synthesizer. Substantial scope. Several adjacent ideas surfaced during brainstorm and are deliberately left for follow-up tracks.

### Explicitly deferred

**Voice integration.** APS's voice subsystem is speech-to-speech via PersonaPlex and Moshi; LLM lives inside voice backend, not in `aps agent`. Bridging the two is separate architectural project. Three options exist (Section 3); recommended path is for voice to share agent's tool registry via MCP without sharing the loop. Follow-up track decides which option to take.

**`kit/mcpgen`.** Shared helper in `hop.top/kit` that turns any cobra-based hop.top CLI into MCP server "for free". Would let every hop.top binary (`tlc`, `wsm`, `rux`, `ibr`, `rsx`, `eva`, `ben`, `git-hop`, etc.) expose its subcommands as MCP tools, unifying Section 4's bundle-binaries source and MCP source into one mechanism. Substantial cross-repo work; v1 ships with bundle-binaries dispatch via `cxr` and new MCP client handles MCP servers separately.

**Action → skill migration.** Listed as Sprint 2 of existing skill `IMPLEMENTATION_PLAN.md`. Independent of loop. Could ride docs-refresh track.

**Long-running session persistence beyond A2A.** A2A's task storage handles persistence loop needs for v1. Richer "conversation" persistence (resume `aps agent serve` session across restarts, restore REPL transcript, etc.) is follow-up — `kit/sqlstore` is natural backing store.

**Multi-step contract negotiation.** Squad contracts in v1 are static — if contract exists between two squads, peer-call tool is in registry; if not, isn't. Dynamic contract creation (consumer squad asks for temporary collaboration with provider it doesn't currently have contract with) is follow-up track.

**`tip` self-improvements.** `aps agent` consumes `tip` as it stands today. If `tip` itself needs work (vector store schema, embedding model upgrades, fallback config), that's `tip` repo's track, not this one.

### Anticipated follow-up tracks

Work above naturally clusters into a few likely follow-ups:

| Likely follow-up | What it adds | Depends on `aps-agent` |
|---|---|---|
| `aps-voice-bridge` | Voice subsystem shares agent tool registry via MCP | Yes — needs MCP client + tool registry |
| `aps-kit-mcpgen` | Shared MCP server generator for cobra-based hop.top CLIs | No (independent) — but `aps-agent`'s MCP client consumes whatever it produces |
| `aps-skill-migrate` | Action → skill conversion CLI | No |
| `aps-agent-persistence` | Resume sessions across restarts via `kit/sqlstore` | Yes |
| `aps-squad-dynamic-contracts` | Runtime negotiation of new contracts between squads | Yes |
| `aps-tip-evolution` | Improvements to `tip` itself | No (lives in `tip` repo) |

None committed; table is forward-looking. Track for any of them gets created when its time comes, with `aps-agent` listed as dependency where it actually applies.

### What success looks like

Seven success criteria from spec, repeated for visibility:

- `aps agent run "..."` against any profile with valid LLM config completes a turn, calls a tool, returns a result.
- A profile receiving an A2A message via its executor produces an LLM-driven response (not the placeholder echo).
- A profile receiving an ACP `session/prompt` produces an LLM-driven response, including filesystem and terminal tool calls under active session mode.
- A squad with two member profiles and a single `x-as-a-service` contract: consumer profile's loop sees a `peer_<provider>__<capability>` tool whose schema matches contract's `input_schema`, and synchronous call returns validated `output_schema`-shaped result.
- A skill with `allowed-tools: "Bash(git:*) tlc.*"` restricts loop's registry to those tools while skill is active (verified via `describe_tool` returning not-found for excluded tool).
- A loop run populates active squad's `ContextLoad` with measured byte counts and tool-schema counts; `aps squad check` reports `topology-first` based on populated values rather than static defaults.
- `tip` records a fix when a tool call fails with a typo'd argument and the next call against the same args succeeds without hitting LLM provider for a correction.

When all seven hold against the live binary, v1 is done.
