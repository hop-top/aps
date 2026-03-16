# Voice Integration Design

**Date:** 2026-03-16
**Status:** Draft

## Overview

This document describes how APS integrates with speech-to-speech backends (PersonaPlex, Moshi) to give agent profiles a voice. APS manages the backend service lifecycle and orchestrates voice sessions across four channels: web UI, terminal (Hex TUI), messaging platforms (Telegram, WhatsApp), and telephony (Twilio).

Each APS profile maps to a voice persona. APS generates the backend's text prompt from the profile's `persona` fields, so profiles work out of the box without manual prompt writing.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        APS CLI                              │
│  aps voice start [--profile <id>]                           │
│  aps voice service start|stop|status                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                  Voice Orchestrator                         │
│  (internal/voice/)                                          │
│  - Profile → backend config mapping                         │
│  - Service lifecycle management                             │
│  - Session routing (explicit / intent-based)                │
│  - Transcript → Action pipeline                             │
│  - Response → channel pipeline                              │
└──────┬──────────────────────────────────┬───────────────────┘
       │                                  │
┌──────▼──────┐                 ┌─────────▼────────────┐
│   Backend   │                 │     APS Core          │
│   Service   │                 │  - Profile execution  │
│  (managed   │                 │  - Action dispatch    │
│   or remote)│                 │  - A2A/ACP routing    │
└──────┬──────┘                 └──────────────────────┘
       │
┌──────▼──────────────────────────────────────────────────────┐
│                    Channel Adapters                         │
├─────────────┬──────────────┬─────────────┬─────────────────┤
│  Web UI     │  Hex TUI     │  Messenger  │   Telephony     │
│  (browser,  │  (terminal,  │  (Telegram, │   (Twilio,      │
│   v1)       │   v2)        │   WhatsApp) │    phone calls) │
└─────────────┴──────────────┴─────────────┴─────────────────┘
```

APS is the orchestrator. The backend service is a voice I/O engine: it converts speech to text and text to speech. APS decides which actions to trigger, executes them under the correct isolated profile, and returns responses through the backend.

---

## Backend Agnosticism

APS speaks the PersonaPlex/Moshi WebSocket protocol and accepts any conforming backend. The `type` field selects the backend; `compatible` accepts any conforming implementation.

```yaml
# ~/.config/aps/config.yaml
voice:
  default_backend: auto
  backends:
    personaplex-cuda:
      bin: /opt/personaplex/server
      args: ["--port", "8998"]
    personaplex-mlx:
      bin: /opt/personaplex-mlx/server
      args: ["--port", "8998"]
    moshi:
      bin: /opt/moshi/server
      args: ["--port", "8998"]
    moshi-mlx:
      bin: /opt/moshi-mlx/server
      args: ["--port", "8998"]
```

`auto` detection on Apple Silicon prefers PersonaPlex MLX, then Moshi MLX. On NVIDIA GPU: PersonaPlex CUDA, then Moshi. APS emits a clear error if no backend is found.

---

## Profile Voice Configuration

Each profile opts into voice with a `voice` block. All fields are optional; APS provides sensible defaults.

```yaml
# ~/.agents/profiles/customer-support/profile.yaml
persona:
  tone: friendly
  style: concise
  risk: low

voice:
  enabled: true
  backend:
    url: ""        # empty = APS-managed; set to use external instance
    type: auto     # auto | personaplex-cuda | personaplex-mlx | moshi | moshi-mlx | compatible
  voice_id: "NATF0"
  prompt_template: ""  # empty = auto-generated from persona fields
  channels:
    web: true
    tui: true
    telegram:
      enabled: true
      bot_token_secret: TELEGRAM_BOT_TOKEN
    twilio:
      enabled: true
      phone_number: "+15551234567"
      account_sid_secret: TWILIO_ACCOUNT_SID
      auth_token_secret: TWILIO_AUTH_TOKEN
```

Secrets follow APS's convention: each value is a key name resolved from `secrets.env` at runtime.

---

## Persona Prompt Auto-Generation

When `prompt_template` is empty, APS generates the backend's text prompt from the profile's `persona` fields, closing the loop between profile metadata and voice behavior.

```go
// internal/voice/prompt.go

func (g *PromptGenerator) Generate(p *core.Profile) string {
    return fmt.Sprintf(
        "You are %s. Your communication style is %s and %s. %s",
        p.DisplayName,
        toneDescription(p.Persona.Tone),
        styleDescription(p.Persona.Style),
        riskGuidance(p.Persona.Risk),
    )
}

var toneMap = map[string]string{
    "friendly":     "warm and approachable",
    "professional": "formal and precise",
    "casual":       "relaxed and conversational",
}

var riskMap = map[string]string{
    "low":    "Never speculate. If unsure, say so.",
    "medium": "Use best judgement, flag uncertainty.",
    "high":   "Act decisively with available information.",
}
```

When `prompt_template` is set, it takes precedence over auto-generation.

---

## Voice Orchestrator

`internal/voice/` contains three components: `SessionManager`, `BackendManager`, and `Pipeline`.

```
┌─────────────────────────────────────────────┐
│            Voice Orchestrator               │
│                                             │
│  SessionManager                             │
│  ├── create(profile, channel) → Session     │
│  ├── route(intent) → Profile  (intent mode) │
│  └── switch(session, profile) → Session     │
│                                             │
│  BackendManager                             │
│  ├── resolve(profile) → Backend             │
│  ├── start/stop/health()                    │
│  └── connect() → WebSocket                  │
│                                             │
│  Pipeline                                   │
│  ├── audio_in → backend → transcript        │
│  ├── transcript → ActionRouter              │
│  ├── ActionRouter → APS profile execution   │
│  └── result → backend → audio_out           │
└─────────────────────────────────────────────┘
```

### Session Lifecycle

```
channel connects
      │
      ▼
SessionManager.create(profile, channel)
      │
      ├── resolve profile voice config
      ├── BackendManager.connect() → WebSocket
      ├── inject persona prompt + voice_id
      └── open Pipeline
            │
            ▼
      [active session]
            │
            ├── transcript arrives
            │     ├── intent routing? → migrate to different profile
            │     ├── action keyword? → ActionRouter.dispatch()
            │     └── conversational? → pass back to backend
            │
            └── channel disconnects / timeout → Session.close()
```

### Session Routing Modes

Three modes compose naturally:

- **Explicit** (`aps voice start --profile <id>`): the user selects a profile before connecting. The session runs under that profile for its duration.
- **Intent-based** (ambient/always-on): a lightweight classifier runs on each transcript chunk. When it detects with high confidence that a different profile should handle the request, the session migrates — same audio connection, new persona prompt injected mid-session. This is the Siri-like mode.
- **Mid-session switch**: an explicit escape hatch. The user or an action can request a profile switch during an active session.

---

## Channel Adapters

All channel adapters implement a single interface; the orchestrator is channel-agnostic.

```go
type ChannelAdapter interface {
    Accept() (<-chan ChannelSession, error)
    Close() error
}

type ChannelSession interface {
    AudioIn()  <-chan []byte
    AudioOut() chan<- []byte
    TextOut()  chan<- string   // text-only fallback for messenger channels
    Meta()     SessionMeta    // profile hint, channel type, caller id
    Close()    error
}
```

### Web UI (v1)

Serves PersonaPlex's React client statically and proxies WebSocket to the backend. At page load, APS injects the resolved profile config (voice ID, prompt) into the client. The implementation is minimal — mostly a reverse proxy.

### Hex TUI (v2)

A separate Haskell binary connects to APS over a local socket. APS exposes a voice session API; the TUI consumes it. This separation keeps the language boundary clean and lets the TUI evolve independently. The TUI captures microphone input, renders the conversation transcript, and pipes audio frames to the orchestrator.

### Messenger (Telegram, WhatsApp)

Extends APS's existing messenger layer. Incoming voice messages decode to PCM and enter the pipeline. Responses return as audio files or text, depending on channel capability. The existing `link store` maps user accounts to profiles.

### Telephony (Twilio)

Twilio streams call audio over its Media Streams WebSocket. The adapter bridges Twilio's mulaw/8 kHz format to the backend's PCM/24 kHz. Phone numbers map to profiles via the profile's `twilio.phone_number` field. The adapter handles SIP signaling and call lifecycle.

---

## New CLI Commands

```
aps voice start [--profile <id>] [--channel web|tui|telegram|twilio]
aps voice service start
aps voice service stop
aps voice service status
aps voice session list
aps voice session attach <id>
```

---

## What This Does Not Cover

- Backend installation and model weight management (out of scope for APS)
- Audio format transcoding details (Twilio adapter internals)
- Hex TUI internal design (separate design document)
- Intent classifier implementation
