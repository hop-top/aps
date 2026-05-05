---
status: paper
---

# Aps Chat — Native Profile-Backed LLM REPL

**ID**: 055
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a profile owner, I want `aps chat <profile-id>` to open a multi-turn
conversation with the profile acting as an AI assistant — its persona is the
system prompt, its secrets supply the LLM credentials, its session shows up
in `aps session list` — so the profile is the assistant identity, not the
shell context.

Today's compose-via-`aps run`:

```sh
aps run noor -- foo "hello"
```

works (verified e2e 2026-05-02) — env injection makes credentials available
to `foo`. But: profile persona is not the system prompt, sessions don't
appear in aps's session registry, voice/text are routed through different
binaries (`foo`/`aps voice`), and downstream tools have no programmatic
hook to drive a profile-scoped chat.

This story replaces the run-foo dance with a first-class `aps chat`.

## Out of Scope

- Replacing `foo` as a general LLM CLI. `foo` stays the standalone REPL
  for non-profile work.
- Image / multi-modal inputs (text only for v1).

## Acceptance Scenarios

1. **Given** a profile `noor` with persona `{tone: professional, style: terse}`
   and a configured LLM provider key in `secrets.env`, **When** I run
   `aps chat noor`, **Then** an interactive REPL opens, the persona renders
   as the system prompt, and the assistant replies in the configured tone.

2. **Given** an active `aps chat noor` session, **When** I send 3 user
   turns, **Then** all 3 are recorded in core/session.Registry with
   `Type=SessionTypeChat` and visible via
   `aps session list --type chat`.

3. **Given** the kit RouteLLM config wired into the profile or workspace
   YAML, **When** the chat picks a model, **Then** the routing decision
   is logged at debug level (model + tier chosen) and respects
   `--model <override>` when supplied.

4. **Given** a chat session with id `S-abc`, **When** I run
   `aps session attach S-abc`, **Then** the prior turns are replayed and
   I can continue the conversation. (`aps session detach` works
   symmetrically.)

5. **Given** a profile without any LLM provider key in secrets, **When** I
   run `aps chat <id>`, **Then** the command exits with code 5 (unauthorized)
   and the error names the missing keys (e.g. `OPENAI_API_KEY`,
   `ANTHROPIC_API_KEY`).

6. **Given** the `--once "<prompt>"` flag, **When** invoked, **Then** the
   chat fires a single turn, prints the response, persists the session, and
   exits 0 — script-friendly, no REPL.

7. **Given** profiles `noor` (ops) and `farid` (finance), **When** I run
   `aps chat noor --invite farid` and ask "what's our Q3 burn?",
   **Then** both profiles see the prompt, each replies in its own
   persona-styled turn, and the transcript identifies each speaker by
   profile URI.

8. **Given** an active multi-profile chat where the human steps out via
   `:auto`, **When** the agents take 5 turns autonomously, **Then**
   turn-taking respects the configured policy (round-robin by default),
   the loop terminates on `:done`-emitting consensus or after
   `--max-auto-turns N` (default 10), and every turn is persisted with
   the speaker's profile id.

## Tests

### E2E

- `tests/e2e/chat/chat_repl_test.go` — `TestChat_OpensRepl_PersonaApplied`
- `tests/e2e/chat/chat_session_persist_test.go` — `TestChat_SessionAppearsInRegistry`
- `tests/e2e/chat/chat_routellm_test.go` — `TestChat_RouteLLM_PicksConfiguredTier`
- `tests/e2e/chat/chat_attach_test.go` — `TestChat_AttachReplaysPriorTurns`
- `tests/e2e/chat/chat_unauth_test.go` — `TestChat_ExitsCode5OnMissingKeys`
- `tests/e2e/chat/chat_once_test.go` — `TestChat_OnceFlagOneShotPersistsSession`
- `tests/e2e/chat/chat_multi_invite_test.go` — `TestChat_MultiProfile_InvitedRepliesInPersona`
- `tests/e2e/chat/chat_multi_auto_test.go` — `TestChat_MultiProfile_AutoMode_RespectsTurnPolicy`

### Unit

- `internal/cli/chat/chat_test.go` — REPL loop, key-binding, history nav
- `internal/cli/chat/persona_test.go` — Profile.Persona → system prompt
  renderer (tone/style/risk → directive lines)
- `internal/cli/chat/routellm_test.go` — config resolution
  (profile.yaml > workspace.yaml > config.yaml > kit defaults)

## Implementation Notes

### Surface

```
aps chat <profile-id>                       # interactive REPL
aps chat <profile-id> --once "<msg>"        # one-shot, exit after first reply
aps chat <profile-id> --model <id>          # override RouteLLM choice
aps chat <profile-id> --no-stream           # block until full response
aps chat <profile-id> --attach <S-id>       # continue an existing chat session
aps chat <id1> --invite <id2>[,<id3>]       # multi-profile chat, human-driven
aps chat <id1>,<id2>                        # shorthand for --invite
aps chat <id1> --invite <id2> --max-auto-turns N  # cap autonomous loops
```

In-REPL meta-commands for multi-profile sessions:
- `:auto` — yield to autonomous agent-to-agent turns (until `:done`
  consensus or `--max-auto-turns` cap)
- `:human` — reclaim the speaker turn
- `:done` — agent emits consensus signal; loop ends

Belongs to the `interact` group (matches `run`, `serve`, `voice`,
`session` per cli-conventions §4.1).

### Composition (kit primitives — no hand-rolling)

| Concern | Use |
|---|---|
| LLM client | `kit/go/ai/llm` (multimodal, fallback, hooks) |
| Routing / model selection | `kit/go/ai/llm/routellm` configured per-profile |
| REPL TUI | `kit/go/console/tui` (input, scrollback, status line) |
| Markdown rendering of replies | `kit/go/console/markdown` (T-0377 deferred no-op for aps; this story re-opens it) |
| Persona → system prompt | aps-domain — small renderer in `internal/cli/chat/persona.go` |
| Session persistence | `internal/core/session.Registry` extended with `SessionTypeChat` |
| Secrets | `kit/go/storage/secret` (already wired via T-0378) |

### Integration Boundary

`hop.top/foo` remains the standalone ad-hoc LLM CLI. APS may compose
with it through the generic external-CLI bridge (`aps run <profile> --
foo ...`), but native `aps chat` MUST NOT shell out to `foo` as its
implementation boundary. Native chat should use `kit/go/ai/llm` and
`kit/go/ai/llm/routellm` directly so profile persona, session events,
model routing, cancellation, and multi-profile turn-taking are first-
class APS concepts rather than parsed terminal output.

Related bridge coverage lives in story 061 and covers `claude`,
`codex`, `gemini`, and `opencode` as external child processes.

### Persona Mapping

```
Profile.Persona { Tone, Style, Risk } →
  System prompt prefix:
    "You are <DisplayName>. Tone: <tone>. Style: <style>. Risk
     posture: <risk>. Honor scope/limits per the profile config
     attached below.\n\n<rest>"

Profile.Limits.MaxRuntimeMinutes → REPL idle timeout
Profile.Scope → reflected in system prompt as boundaries
```

### RouteLLM Wiring

Resolution order (lowest → highest precedence):

1. `kit/go/ai/llm/routellm.DefaultRouterConfig()`
2. System config (kit/config layered)
3. User config: `~/.config/aps/llm.yaml`
4. Profile config: `profile.yaml#llm` block (extends LLMConfig type)
5. CLI flag: `--model`

Profile YAML schema addition:

```yaml
llm:
  default_model: claude-sonnet-4-5
  routers: [match, similarity]
  router_config:
    threshold: 0.7
  fallback:
    - claude-haiku-4-5
    - gpt-4o-mini
```

### Session Registry Extension

`SessionType` already has `SessionTypeStandard` + `SessionTypeVoice`
(T-0364). Add `SessionTypeChat`. `aps session list --type chat`
filter falls out of the existing `--type` flag implementation.

### Exit Codes

Per cli-conventions §8.1 (T-0381 already wired):

- 3 — profile not found
- 4 — chat session id collision (rare, on `--attach <existing>`)
- 5 — unauthorized: no LLM provider key in profile secrets

## Resolved Design Decisions

### Streaming UX in TUI — reflow-all for v1, kit `StreamView` upstream as follow-up

V1 ships token-by-token streaming via the simple "reflow-all" approach:
each token arrives as a `tea.Msg`, appends to the buffer, `View()`
re-renders the streaming pane. ~50 LOC. Visible jitter on long replies
is acceptable for a first cut.

Follow-up: contribute a `tui.StreamView` primitive upstream to
`kit/go/console/tui` — pairs with the AppShell gap flagged in T-0380's
close note. Smooth append-render reusable across aps/ctxt/dpkms.
Tracked as a kit upstream issue, not a blocker for chat.

### History storage — session events

Each turn (user / assistant / tool-call) is persisted as a
`core/session.Registry` event under `Type=SessionTypeChat`. Matches
voice (T-0364) — single source of truth, audit-trail-consistent,
bus-emitable for downstream subscribers (T-0354). `aps session
attach <id>` replays via the existing event-walk path.

Follow-up: ship `aps session export <id> --format md` to derive a
human-readable markdown transcript from events on demand. Tracked
as a separate story.

### Multi-profile cross-talk — in scope

Chat handles **human ↔ profile** AND **profile ↔ profile** through the
same surface. Two participation modes:

- **Single-profile** (default): `aps chat <id>` — human drives, one
  profile responds.
- **Multi-profile**: `aps chat <id> --invite <other-id>[,<other-id>]`
  or `aps chat <id1>,<id2>` — multiple profiles in one session, each
  contributing its persona as a system prompt section. Human can
  speak (gated turn) or step out and let agents converse autonomously.

Cross-talk dispatch reuses A2A under the hood (`aps a2a tasks send`
post-T-0365 already handles profile-to-profile messaging) — the chat
surface is the human-facing layer that drives the A2A wire format
when participants are remote, or in-process when they're local.

Design constraints to resolve in implementation:
- Turn-taking policy (round-robin / user-mention / auto-yield)
- Context-budget split when N participants share one model context
  window
- Persona collision rule (when two profiles claim the same role,
  e.g. both "auditor")
- Identity rendering in the transcript (who said what — uses
  `Profile.URI()` from T-0341)
- Session ownership when A2A is involved (the initiator's session
  is canonical; remote participants emit echo events)

## Related

- T-0378 — kit/storage/secret adoption (shipped)
- T-0364 — voice session collapse / SessionType field (shipped)
- T-0377 — kit/console/markdown adoption (shipped no-op; this story
  re-opens it for chat reply rendering)
- T-0380 — internal/tui → kit/console/tui (shipped partial; full
  AppShell pattern would benefit chat too)
- foo (`hop.top/foo`) — standalone LLM CLI/REPL; aps chat replaces
  it for profile-scoped use cases, foo stays for ad-hoc

## Refs

Verified compose path (today): `aps run noor -- foo "<msg>"` works
end-to-end as of 2026-05-02 — env injection from `secrets.env` to
`foo` confirmed. This story replaces that with a native surface.
