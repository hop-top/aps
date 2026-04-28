---
title: "APS Server Documentation Matrix"
tracks:
  - server-docs-matrix
tasks:
  - title: "index: draft docs/user/servers/README.md skeleton"
    description: |
      Hybrid index page. TL;DR cheat sheet row, then three tables:
      (1) servers × transports, (2) servers × access-surfaces,
      (3) APS↔external-CLI integration pairs. Checkmarks legend:
      ✅ stable / ⚠️ experimental / 🚧 planned / ❌ not supported.
      Link to per-server pages (empty stubs OK at this stage).
    effort: S
    priority: P0
    tags: [phase:1, domain:docs, scope:user]

  - title: "template: finalize per-server page template"
    description: |
      Sections 1–14 as agreed. Section 9 split into 9a "APS as
      server, CLI as client" (MCP + ACP server + HTTP ingress)
      and 9b "CLI as server, APS as client" (driving gemini
      --acp, opencode acp, codex app-server). Each CLI subsection
      MUST cite official docs URL, not --help.
    effort: S
    priority: P0
    tags: [phase:1, domain:docs, scope:user]
    blocked-by: [0]

  - title: "server page: docs/user/servers/a2a.md"
    description: |
      Migrate and expand a2a-quickstart.md + a2a-examples.md into
      the template. Cover transports (HTTP/JSON-RPC), access
      surfaces (local/LAN/TS/CF), CLI consumers (curl + aps a2a
      client). Link MCP/ACP cross-refs. Keep quickstart as a
      pointer doc or fold it in.
    effort: M
    priority: P0
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "server page: docs/user/servers/acp.md"
    description: |
      Migrate acp-quickstart.md into the template. Critical
      section: ACP as cross-vendor protocol — document both
      APS-as-server (existing) and APS-as-client driving
      gemini --acp / opencode acp (9b). Cite
      https://agentclientprotocol.com and
      https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/acp-mode.md
      and https://opencode.ai/docs/acp/.
    effort: M
    priority: P0
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "server page: docs/user/servers/agntcy.md"
    description: |
      New doc. Cover gRPC transport, access surfaces, and how to
      hit it from buf/grpcurl. No documented AI-CLI consumer
      path — call out explicitly in section 9 as "none (gap)".
      Reference internal/agntcy/ for current state.
    effort: M
    priority: P1
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "server page: docs/user/servers/voice.md"
    description: |
      New doc. Cover voice listener transport (WebRTC/SIP —
      verify against internal/voice/). Access surfaces
      constrained by NAT/STUN/TURN. Section 9 consumers:
      phone dial-in, browser WebRTC. Cite docs/dev/voice.md.
    effort: M
    priority: P1
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "server page: docs/user/servers/messengers.md"
    description: |
      Replace flat docs/user/messengers.md with template-shaped
      page per messenger (telegram, discord, slack, github,
      email). For each: inbound transport (webhook vs long-poll),
      required tunnel (cloudflared for webhooks, none for
      long-poll), HMAC/signature verification. Cross-link
      remote-access.md.
    effort: L
    priority: P0
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "new: docs/user/servers/mcp.md (APS as MCP server)"
    description: |
      Currently no docs. Covers how APS exposes profile
      capabilities as an MCP server to Claude/Gemini/Codex/
      OpenCode. Section 9 is the main body — one subsection
      per CLI with the ACTUAL registration path, verified
      against official docs. See spec.md §MCP for confirmed
      facts per CLI.
    effort: L
    priority: P0
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "new: docs/user/servers/cli-clients.md"
    description: |
      Cross-cutting doc explaining APS as CLIENT, not server.
      Covers: (a) driving gemini --acp as subprocess via stdio
      JSON-RPC, (b) driving opencode acp same way, (c) speaking
      codex app-server v2 JSON-RPC over stdio or experimental
      WebSocket+JWT. Flag 🚧 for any APS-side client that does
      not exist yet. Each section cites official protocol docs.
    effort: L
    priority: P0
    tags: [phase:2, domain:docs, scope:user]
    blocked-by: [1]

  - title: "dev: docs/dev/panels.md (TUI panel architecture)"
    description: |
      Per user scope split, panels are dev-only. Document
      internal/tui panel layout, lifecycle, how panels bind to
      profile sessions, and any local-only access (no network
      listener). Not in the external-access matrix.
    effort: M
    priority: P1
    tags: [phase:3, domain:docs, scope:dev]
    blocked-by: [1]

  - title: "gap: APS-as-MCP-server implementation status"
    description: |
      Audit internal/ for MCP server exposure of profile
      capabilities. If absent/partial, file follow-up track.
      This is the lowest-common-denominator integration for all
      four AI CLIs — highest leverage gap.
    effort: S
    priority: P0
    tags: [phase:1, domain:audit]

  - title: "gap: APS ACP-client mode (drive gemini/opencode)"
    description: |
      Audit internal/acp for client-side support that can spawn
      and drive gemini --acp or opencode acp as a subprocess.
      Likely missing (existing ACP code is server-side). File
      follow-up track if confirmed.
    effort: S
    priority: P1
    tags: [phase:1, domain:audit]

  - title: "gap: Codex app-server client"
    description: |
      Check for any internal/codex or similar package speaking
      Codex App Server Protocol v2 (JSON-RPC 2.0 stdio + optional
      WebSocket with JWT/HMAC). Almost certainly absent. Decide
      whether to ship (would enable deep Codex integration) or
      defer. Spec:
      https://developers.openai.com/codex/app-server
      https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md
    effort: S
    priority: P2
    tags: [phase:1, domain:audit]

  - title: "gap: WebSocket transport on A2A/ACP"
    description: |
      Current A2A/ACP are HTTP/JSON-RPC. WebSocket row in the
      matrix will be ❌ for APS-as-server. Decide if/when to
      add WebSocket support (useful for browser clients and
      codex-compat listener mode). Spec ask only — no
      implementation in this track.
    effort: XS
    priority: P3
    tags: [phase:1, domain:audit]

  - title: "gap: mDNS / local discovery"
    description: |
      OpenCode exposes --mdns for service discovery. Evaluate
      whether APS should advertise profiles via mDNS for
      LAN auto-discovery by peer agents. Doc-only decision.
    effort: XS
    priority: P3
    tags: [phase:1, domain:audit]

  - title: "index: fill matrix checkmarks after audits"
    description: |
      Once audit tasks complete, populate the three tables in
      docs/user/servers/README.md with actual ✅/⚠️/🚧/❌.
      Each 🚧 cell links to a follow-up track. Each ✅ cell
      links to the per-server page.
    effort: S
    priority: P0
    tags: [phase:3, domain:docs]
    blocked-by: [10, 11, 12, 13, 14]

  - title: "cross-link: update docs/user/README.md + remote-access.md"
    description: |
      Add servers/ section to the user README index. Update
      remote-access.md to link to the matrix page. Add links
      FROM a2a-quickstart.md and acp-quickstart.md to the new
      canonical server pages (or mark them as superseded).
    effort: XS
    priority: P1
    tags: [phase:3, domain:docs]
    blocked-by: [2, 3, 7]
---

# APS Server Documentation Matrix

Plan for documenting every server APS can launch, every transport
it speaks, every access surface it can be reached on, and every
external AI CLI (or peer) that can connect to it.

## Problem

APS can launch multiple listeners (A2A, ACP, AGNTCY, voice, per-
messenger adapters) across multiple transports (HTTP/JSON-RPC,
gRPC, WebSocket, SSE, webhook, long-poll) with multiple access
surfaces (local, LAN, Tailscale, Cloudflare Tunnel) and multiple
possible consumers (Claude Code, Gemini CLI, Codex CLI, OpenCode,
browsers, curl, phone dial-in).

Current docs cover fragments: `a2a-quickstart.md`, `a2a-examples.md`,
`acp-quickstart.md`, flat `messengers.md`, and newly-added
`remote-access.md`. No single page answers "what can APS do, with
which client, over which network path?" and no gap report flags
missing capabilities.

## Goal

1. A single **index page** at `docs/user/servers/README.md` that
   answers the above in three tables and a TL;DR cheat sheet.
2. **Per-server pages** under `docs/user/servers/<name>.md`
   following a common template.
3. A **gap report** derived from the index that lists every ❌/🚧
   cell as a tracked follow-up.
4. TUI/panel internals documented separately in `docs/dev/panels.md`
   (out of the external-access matrix per user scope split).

## Scope

**In scope (docs/user/):** A2A, ACP, AGNTCY, MCP (as server), voice,
all messenger adapters, APS-as-client modes (driving gemini --acp /
opencode acp / codex app-server).

**In scope (docs/dev/):** TUI panel architecture (panels.md).

**Out of scope:** IDE integrations (VS Code, Cursor, JetBrains),
MCP clients consuming APS workflows, any new transport
implementation work (plan only identifies gaps — does not build
them).

## Structure — index page

`docs/user/servers/README.md` — hybrid index with:

1. **TL;DR cheat sheet** — "If you want X → go to Y" for the 80%
   case.
2. **Table 1: Servers × Transports** — what each APS server can
   speak on the wire.
3. **Table 2: Servers × Access surfaces** — which reach modes are
   supported per server (local, LAN, Tailscale, Cloudflared,
   browser, curl).
4. **Table 3: APS ↔ External CLI integration pairs** — the
   actionable backlog. Rows are `(APS role) ↔ (external CLI role)`
   pairs. Every cell is ✅ stable / ⚠️ experimental / 🚧 planned /
   ❌ not supported.
5. **Legend** + link to gap follow-up tracks.

## Structure — per-server page template

Finalized in task 1. Sections:

```
1. Overview
2. Capability matrix (this server's row from the index)
3. Configuration (profile.yaml snippet)
4. Launching
5. Connecting — local
6. Connecting — LAN
7. Connecting — Tailscale
8. Connecting — Cloudflared
9. Connecting — from AI CLIs
   9a. APS as server, CLI as client
       - Claude Code  (MCP registration: claude mcp add …)
       - Gemini CLI   (.gemini/settings.json mcpServers)
       - Codex CLI    ([mcp_servers.*] in ~/.codex/config.toml)
       - OpenCode     (opencode.json mcp)
   9b. CLI as server, APS as client
       - Gemini CLI   (APS spawns `gemini --acp`, JSON-RPC stdio)
       - OpenCode     (APS spawns `opencode acp`, JSON-RPC stdio)
       - Codex CLI    (APS spawns `codex app-server`, v2 JSON-RPC)
10. Connecting — from browser
11. Connecting — from CLI client (curl / aps / grpcurl / websocat)
12. Authentication
13. Troubleshooting
14. See also
```

All section-9 claims MUST cite official docs URLs, not `--help`.

## Research baseline — grounded facts

The following are confirmed from official docs (see spec.md for
full citations). These drive Table 3 of the index.

### All four CLIs support APS-as-MCP-server
- Claude Code: `claude mcp add` with stdio/HTTP/SSE transports,
  `.mcp.json` project config, OAuth via `/mcp`. `--add-dir` does
  **not** load agent configs.
- Gemini CLI: `.gemini/settings.json` `mcpServers` with stdio/SSE/
  HTTP-streaming, OAuth 2.0 for remote.
- Codex CLI: `~/.codex/config.toml` `[mcp_servers.*]` with stdio/
  HTTP, `bearer_token_env_var`, `codex mcp login`.
- OpenCode: `opencode.json` `mcp` with `type: local|remote`,
  Dynamic Client Registration OAuth.

### ACP is cross-vendor (Gemini + OpenCode)
- Gemini: `gemini --acp` starts a JSON-RPC 2.0 stdio **server**.
  Methods: `initialize`, `authenticate`, `newSession`,
  `loadSession`, `prompt`, `cancel`. Proxied filesystem service.
- OpenCode: `opencode acp` same pattern.
- An APS **ACP-client** implementation could drive both CLIs as
  subprocesses. Existing APS ACP code is server-side only —
  gap confirmed in audit task.

### Codex app-server is the deep integration surface
- `codex app-server` speaks **Codex App Server Protocol v2**
  (JSON-RPC 2.0 over stdio or experimental WebSocket with
  JWT/HMAC bearer auth).
- No `codex --remote` top-level flag exists in official docs.
- APS would need to implement a Codex-app-server **client**
  (not server) to drive Codex deeply. Likely gap.

### Claude has no third-party server mode
- `claude remote-control` exists but is **bound to claude.ai**.
- APS cannot host a server that Claude Code attaches to.
- Integration is MCP-only (claude → APS), no reverse path.

### Claude `--agent` vs `--agents`
- `--agent <name>` requires pre-registration (`.claude/agents/`,
  plugin, or `--agents` in same command).
- `--agents '<json>'` is the only fully-inline path. APS can emit
  this per invocation.

### OpenCode publishes OpenAPI 3.1
- `opencode serve` exposes a documented OpenAPI 3.1 spec at
  `/doc`. `opencode attach <url>` could theoretically connect
  to an APS-implemented backend speaking the same spec.
- Status: feasible-but-unblessed. Not pursued in this track.

## Matrix — Table 3 (initial checkmarks, to be verified by audit)

| APS role ↔ CLI role                       | Status | Notes                                   |
| ----------------------------------------- | ------ | --------------------------------------- |
| APS MCP server ← Claude Code (client)     | 🚧     | audit: does APS expose MCP?             |
| APS MCP server ← Gemini CLI (client)      | 🚧     | same                                    |
| APS MCP server ← Codex CLI (client)       | 🚧     | same                                    |
| APS MCP server ← OpenCode (client)        | 🚧     | same                                    |
| APS A2A server ← curl/browser             | ✅     | exists; docs exist                      |
| APS A2A server ← any AI CLI               | ❌     | no CLI speaks A2A natively              |
| APS ACP server ← curl                     | ✅     | exists; docs exist                      |
| APS ACP client → Gemini CLI               | 🚧     | likely absent; audit                    |
| APS ACP client → OpenCode                 | 🚧     | likely absent; audit                    |
| APS Codex app-server client → Codex CLI   | ❌     | not built; decide scope                 |
| APS AGNTCY server ← grpcurl               | ✅     | exists; docs partial                    |
| APS AGNTCY server ← any AI CLI            | ❌     | none speaks AGNTCY                      |
| APS voice listener ← phone                | ⚠️     | experimental; verify                    |
| APS voice listener ← browser WebRTC       | ⚠️     | verify                                  |
| APS messenger inbound ← telegram long-poll| ✅     | no tunnel needed                        |
| APS messenger inbound ← discord webhook   | ✅     | needs cloudflared                       |
| APS messenger inbound ← slack webhook     | ✅     | needs cloudflared                       |
| APS messenger inbound ← github webhook    | ✅     | needs cloudflared                       |
| APS messenger inbound ← email             | ⚠️     | verify mechanism                        |
| APS WebSocket on A2A/ACP                  | ❌     | transport not supported                 |
| APS mDNS advertise                        | ❌     | not implemented                         |

Audit tasks (11–15) confirm every 🚧 row before the index publishes.

## Deliverables

- `docs/user/servers/README.md` — index + three tables
- `docs/user/servers/_template.md` — per-server template (or
  inline in CONTRIBUTING guidance)
- `docs/user/servers/a2a.md`
- `docs/user/servers/acp.md`
- `docs/user/servers/agntcy.md`
- `docs/user/servers/mcp.md`               (new — highest leverage)
- `docs/user/servers/voice.md`
- `docs/user/servers/messengers/telegram.md`
- `docs/user/servers/messengers/discord.md`
- `docs/user/servers/messengers/slack.md`
- `docs/user/servers/messengers/github.md`
- `docs/user/servers/messengers/email.md`
- `docs/user/servers/cli-clients.md`        (new — APS-as-client)
- `docs/dev/panels.md`                       (TUI internals)
- Updated `docs/user/README.md` index
- Updated `docs/user/remote-access.md` cross-links
- Audit findings folded back into the index as confirmed
  ✅/⚠️/🚧/❌ checkmarks

## Diagram

See [integration-pairs-v1.mmd](./integration-pairs-v1.mmd) for the
APS↔CLI role graph.

## Task list

Tasks are declared in frontmatter above. Phasing:

- **Phase 1 — scaffold + audit** (tasks 0, 1, 10, 11, 12, 13, 14)
  — stand up the index skeleton and finalize the template;
  audit current APS code for MCP server, ACP client, Codex
  app-server client, WebSocket, mDNS.
- **Phase 2 — write pages** (tasks 2–9) — fill each per-server
  page against the template.
- **Phase 3 — close the loop** (tasks 15, 16) — populate the
  matrix checkmarks from audit results and wire up cross-links.

## References

- plan.md#L1-L199 (this file)
- spec.md (companion — grounded citations)
- integration-pairs-v1.mmd (diagram)
