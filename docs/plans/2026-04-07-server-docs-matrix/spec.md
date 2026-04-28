# APS Server Documentation Matrix — Spec

Companion to [plan.md](./plan.md). Grounded citations for every
integration claim the plan depends on. No `--help` inferences.

## Scope split

- **docs/user/** — A2A, ACP, AGNTCY, MCP, voice, messengers,
  APS-as-client. User-facing "turn it on and reach it".
- **docs/dev/** — TUI panels. Internals only. Not part of the
  external-access matrix.

## Confirmed facts per external CLI

### Claude Code (Anthropic)

**MCP** — HIGH confidence
- Registration: `claude mcp add [--transport stdio|http|sse] <name> …`
- Config: `~/.claude.json` (user/local), `.mcp.json` (project),
  `managed-mcp.json` (managed).
- Remote: HTTP and SSE. WebSocket hinted in subagent docs
  (`stdio|http|sse|ws`) but not at top level.
- Auth: bearer headers via `--header`, OAuth 2.0 via `/mcp`,
  `${VAR}` env expansion in `.mcp.json`.
- Source: https://code.claude.com/docs/en/mcp

**Agents** — HIGH confidence
- `--agent <name>` REQUIRES pre-registration in
  `.claude/agents/*.md`, `~/.claude/agents/*.md`, or plugin
  `agents/`, or via `--agents` JSON in the same command.
- `--agents '<json>'` is the only fully-inline path. Supported
  keys: description, prompt, tools, disallowedTools, model,
  permissionMode, mcpServers, hooks, maxTurns, skills,
  initialPrompt, memory, effort, background, isolation, color.
- `--add-dir` grants file access ONLY; does not load
  `.claude/agents/` from those dirs.
- Source: https://code.claude.com/docs/en/sub-agents

**ACP** — NOT SUPPORTED. No documentation mention.

**Remote/server mode** — HIGH
- `claude remote-control` exists but is bound to claude.ai /
  claude app. Not a third-party protocol. APS cannot host a
  server that Claude Code attaches to.
- Source: https://code.claude.com/docs/en/cli-reference

**Discrepancy flag**: docs state "`claude --help` does not list
every flag". Always trust docs over --help output.

### Gemini CLI (Google)

**MCP** — HIGH confidence
- Config: `~/.gemini/settings.json` or project `.gemini/settings.json`
  under `mcpServers`.
- Transports: stdio (`command`+`args`), SSE (`url`), HTTP
  streaming (`httpUrl`).
- Fields: `timeout`, `trust`, `includeTools`, `excludeTools`,
  `headers`.
- Auth: OAuth 2.0 for remote MCP. `authProviderType` options:
  `dynamic_discovery`, `google_credentials`,
  `service_account_impersonation`.
- Source: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md

**Agents** — HIGH (MEDIUM on absence-of-inline-JSON)
- Subagents: `.gemini/agents/*.md` with YAML frontmatter
  (name, description, tools, model, temperature, max_turns,
  timeout_mins).
- Agent Skills: `.gemini/skills/<name>/SKILL.md`. System prompt
  gets name+description; `activate_skill` tool loads full content.
- Extensions: `gemini-extension.json` manifest. PRIMARY
  third-party injection mechanism. Can ship MCP servers,
  commands, GEMINI.md, skills, hooks, themes.
- No documented inline `--agents` JSON flag equivalent to
  Claude Code.
- Sources:
  https://geminicli.com/docs/core/subagents/
  https://geminicli.com/docs/cli/skills/
  https://geminicli.com/docs/extensions/writing-extensions/

**ACP** — HIGH confidence
- `gemini --acp` (deprecated alias: `--experimental-acp`) starts
  Gemini as a **server** in client-server ACP relationship.
- Transport: stdio. Protocol: JSON-RPC 2.0.
- Methods: `initialize`, `authenticate`, `newSession`,
  `loadSession`, `prompt`, `cancel`.
- Proxied filesystem service so agent file access flows through
  client.
- Client may provide MCP server details during `initialize`.
- Standard maintained at agentclientprotocol.com (also adopted
  by OpenCode).
- APS integration: act as ACP **client**, drive Gemini as
  subprocess.
- Source: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/acp-mode.md

**Remote mode** — Only ACP. No HTTP/WebSocket server documented.

**Workspace**: `--include-directories` creates virtual workspace.
Whether `.gemini/` configs load from those dirs is NOT documented.
Open issue #13669 reports implementation gap.

### Codex CLI (OpenAI)

**MCP** — HIGH confidence
- Config: `~/.codex/config.toml` under `[mcp_servers.<id>]`.
- Transports: stdio (`command`, `args`, `cwd`, `env`), HTTP
  streaming (`url`). WebSocket NOT documented.
- Auth: `bearer_token_env_var`, `http_headers`,
  `env_http_headers`. `codex mcp login` / `logout`.
- Tool filtering: `enabled_tools`, `disabled_tools`. Per-tool
  approval: `[mcp_servers.<id>.tools.<tool>] approval_mode`.
- Source: https://developers.openai.com/codex/config-reference

**Agents** — MEDIUM
- NO formal custom-agent definition. AGENTS.md is instruction
  injection only.
- Profiles in `[profiles.<name>]` are config variants
  (model/reasoning overrides), not agent definitions.
  Selectable with `-p <name>`.
- Source: https://github.com/openai/codex/blob/main/docs/agents_md.md

**ACP** — NOT SUPPORTED. No mention in Codex docs.

**Remote/server mode — app-server** — HIGH confidence
- `codex app-server` speaks **Codex App Server Protocol v2**.
- Protocol: JSON-RPC 2.0. Transports:
  - stdio (newline-delimited JSON, default)
  - WebSocket (experimental): `codex app-server --listen ws://127.0.0.1:4500`
- Primitives: thread, turn, item.
- Required handshake: `initialize` → `initialized` notification
  → `thread/start` or `thread/resume` → stream → `turn/completed`.
- **WebSocket auth only**: capability tokens OR HMAC-signed JWT
  bearer tokens (optional issuer/audience validation) via
  `Authorization: Bearer <token>`. stdio has no auth (process
  boundary).
- Backpressure: JSON-RPC error `-32001`.
- Schema: open source, clients can generate TS/JSON Schema.
- Sources:
  https://developers.openai.com/codex/app-server
  https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md

**Discrepancy flag**: `codex --remote ws://...` top-level flag
in --help output is **NOT documented** in the official CLI
reference. Treat as unofficial / possibly undocumented WS
listener pass-through. Do not rely on it in docs.

### OpenCode

**MCP** — HIGH confidence
- Config: `opencode.json` / `opencode.jsonc` under `mcp` key.
- Types: `"type": "local"` (stdio command) or `"type": "remote"`
  (HTTP url).
- Auth: Dynamic Client Registration on 401. Static headers
  supported: `"headers": { "Authorization": "Bearer {env:KEY}" }`.
- WebSocket NOT documented.
- Source: https://opencode.ai/docs/mcp-servers/

**Agents** — HIGH
- JSON in `opencode.json` under `agent` key, or Markdown files
  in `~/.config/opencode/agents/` (global) / `.opencode/agents/`
  (project). Filename = agent name.
- `opencode agent create` scaffolder; `opencode agent list`.
- Source: https://opencode.ai/docs/agents/

**ACP** — HIGH
- `opencode acp` starts OpenCode as ACP-compatible subprocess,
  JSON-RPC over stdio.
- OpenCode is the server; editor/tool is the client.
- Same ACP standard as Gemini CLI.
- Limitations: `/undo`, `/redo` currently unsupported via ACP.
- Auth: not addressed in docs.
- Source: https://opencode.ai/docs/acp/

**Remote/server mode — HTTP** — HIGH
- `opencode serve` starts headless HTTP server with OpenAPI 3.1
  spec at `http://<host>:<port>/doc`.
- `opencode attach [url]` connects TUI to running backend.
- Auth: `OPENCODE_SERVER_PASSWORD` env (HTTP basic;
  username override via `OPENCODE_SERVER_USERNAME`).
- Flags: `--port`, `--hostname`, `--mdns`, `--cors`.
- Third-party server-side reimplementation: feasible (spec is
  published) but NOT explicitly blessed. Treat as unblessed.
- Source: https://opencode.ai/docs/server/

## Cross-CLI capability summary

| Capability                    | Claude | Gemini | Codex | OpenCode |
| ----------------------------- | ------ | ------ | ----- | -------- |
| MCP server (stdio)            | ✅     | ✅     | ✅    | ✅       |
| MCP server (HTTP/SSE)         | ✅     | ✅     | ✅    | ✅       |
| MCP OAuth                     | ✅     | ✅     | bearer| ✅ DCR    |
| Inline agent via CLI flag     | ✅     | ❌     | n/a   | ❌       |
| File-based agent registration | ✅     | ✅     | n/a   | ✅       |
| CLI exposes ACP server        | ❌     | ✅     | ❌    | ✅       |
| CLI exposes custom protocol   | remote-control (claude.ai only) | — | app-server v2 | OpenAPI 3.1 |
| APS-implementable server side | no     | no     | no    | maybe    |

## Gaps implied by confirmed facts

These become audit tasks (11–15 in plan.md frontmatter):

1. **APS-as-MCP-server** — highest leverage. All four CLIs
   consume MCP. If APS does not expose profile capabilities
   as an MCP server, that is the biggest gap.

2. **APS ACP-client mode** — existing APS ACP code is
   server-side only. Driving `gemini --acp` or `opencode acp`
   as a subprocess requires a client implementation.

3. **APS Codex app-server client** — most spec-rich deep-
   integration surface available. Requires JSON-RPC 2.0 v2
   client speaking thread/turn/item primitives.

4. **WebSocket transport on A2A/ACP** — useful for browsers
   and would enable compat with Codex-style WS listeners.
   Currently ❌.

5. **mDNS advertise** — OpenCode has it. APS does not. Doc-
   only evaluation.

## Non-goals

- Implementing any of the gap items in this track — plan is
  documentation-first. Gap audits surface candidates; separate
  tracks build them.
- IDE integrations (VS Code, Cursor, JetBrains) — deferred.
- MCP-client side of APS (consuming other MCP servers) — out
  of scope; this track is about exposing APS, not consuming.
- `codex --remote` flag — undocumented, not relied on.

## Source index

- https://code.claude.com/docs/en/mcp
- https://code.claude.com/docs/en/sub-agents
- https://code.claude.com/docs/en/cli-reference
- https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md
- https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/acp-mode.md
- https://geminicli.com/docs/core/subagents/
- https://geminicli.com/docs/cli/skills/
- https://geminicli.com/docs/extensions/writing-extensions/
- https://developers.openai.com/codex/app-server
- https://developers.openai.com/codex/config-reference
- https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md
- https://github.com/openai/codex/blob/main/docs/agents_md.md
- https://opencode.ai/docs/mcp-servers/
- https://opencode.ai/docs/agents/
- https://opencode.ai/docs/acp/
- https://opencode.ai/docs/server/
- https://agentclientprotocol.com
