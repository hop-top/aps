# Voice

APS manages speech-to-speech backend services (PersonaPlex, Moshi) and routes voice sessions across web, terminal, messaging, and telephony channels. Each APS profile maps to a voice persona; APS auto-generates the backend's text prompt from the profile's `persona` fields.

## Quick Start

```bash
# Start the voice backend service
aps voice service start

# Start a voice session (defaults to web channel)
aps voice start --profile my-profile

# List active sessions
aps voice session list
```

## CLI Reference

```
aps voice start [--profile <id>] [--channel web|tui|telegram|twilio]
aps voice service start
aps voice service stop
aps voice service status
aps voice session list
aps voice session attach <id>
```

## Configuration

Voice is configured in two places: the global config for backend binaries and the profile config for per-profile settings.

### Global Config (`~/.config/aps/config.yaml`)

```yaml
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

`default_backend: auto` selects the best available backend for the platform:
- Apple Silicon: PersonaPlex MLX → Moshi MLX → PersonaPlex CUDA → Moshi
- NVIDIA GPU: PersonaPlex CUDA → Moshi → PersonaPlex MLX → Moshi MLX

If no binary is configured, APS emits an error suggesting you set `voice.backends` or use `backend.url` in the profile.

### Profile Config

```yaml
# ~/.agents/profiles/my-profile/profile.yaml
persona:
  tone: friendly
  style: concise
  risk: low

voice:
  enabled: true
  backend:
    url: ""        # empty = APS-managed; set to use an external instance
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

All secret values are key names resolved from `secrets.env` at runtime — the same convention used elsewhere in APS.

## Persona Prompt Auto-Generation

When `prompt_template` is empty, APS generates the backend text prompt from `persona` fields. `tone`, `style`, and `risk` map to natural-language instructions that are injected into the backend before each session. Setting `prompt_template` overrides auto-generation.

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
│  (browser)  │  (terminal)  │  (Telegram, │   (Twilio,      │
│             │              │   WhatsApp) │    phone calls) │
└─────────────┴──────────────┴─────────────┴─────────────────┘
```

### Components

`internal/voice/` has three components:

**BackendManager** — resolves backend type, starts/stops the managed binary, and provides the WebSocket URL. With `backend.url` set in the profile, it defers to the external instance and skips process management.

**SessionManager** — creates, lists, closes, and switches sessions. Each `Session` carries an ID, profile ID, channel type, and state (`active` | `closed`). `SwitchProfile` supports mid-session profile migration.

**ChannelAdapter / ChannelSession** — the interface all channel adapters implement. The orchestrator is channel-agnostic.

```go
type ChannelAdapter interface {
    Accept() (<-chan ChannelSession, error)
    Close() error
}

type ChannelSession interface {
    AudioIn()  <-chan []byte   // PCM frames from caller
    AudioOut() chan<- []byte   // PCM frames to caller
    TextOut()  chan<- string   // text fallback for messenger channels
    Meta()     SessionMeta    // profile hint, channel type, caller ID
    Close()    error
}
```

## Channel Adapters

### Web UI

Serves PersonaPlex's React client statically and proxies the WebSocket to the backend. APS injects the resolved profile config (voice ID, prompt) at page load. Minimal implementation — mostly a reverse proxy.

### Hex TUI (terminal)

A separate Haskell binary connects to APS over a local Unix socket. APS exposes a voice session API; the TUI consumes it. The TUI captures microphone input, renders the transcript, and pipes audio frames to the orchestrator.

### Messenger (Telegram, WhatsApp)

Hooks into APS's existing messenger layer. Incoming voice messages decode to PCM and enter the pipeline. Responses return as audio files or text depending on channel capability. The existing link store maps user accounts to profiles.

### Telephony (Twilio)

Twilio streams call audio over its Media Streams WebSocket. The adapter bridges Twilio's mulaw/8 kHz format to the backend's PCM/24 kHz. Phone numbers map to profiles via `voice.channels.twilio.phone_number` in the profile config.

## Session Routing Modes

**Explicit** — `aps voice start --profile <id>` selects a profile before connecting. The session runs under that profile for its duration.

**Intent-based** — a lightweight classifier runs on each transcript chunk. When it detects with high confidence that a different profile should handle the request, the session migrates — same audio connection, new persona prompt injected mid-session.

**Mid-session switch** — an explicit escape hatch. The user or an action can request a profile switch during an active session via `SessionManager.SwitchProfile`.

## Session Lifecycle

```
channel connects
      │
      ▼
SessionManager.Create(profileID, channelType)
      │
      ├── resolve profile voice config
      ├── BackendManager connect → WebSocket
      ├── inject persona prompt + voice_id
      └── open pipeline
            │
            ▼
      [active session]
            │
            ├── transcript arrives
            │     ├── intent routing → migrate to different profile
            │     ├── action keyword → ActionRouter.dispatch()
            │     └── conversational → pass back to backend
            │
            └── channel disconnects / timeout → Session.Close()
```

## Backend Agnosticism

APS speaks the PersonaPlex/Moshi WebSocket protocol and accepts any conforming backend. Set `backend.type: compatible` in the profile to use any compliant implementation that is not one of the named types.

## Related

- [Design document](../plans/2026-03-16-voice-integration-design.md)
- [Messengers Overview](../MESSENGERS_OVERVIEW.md) — voice message support for Telegram/WhatsApp
- [Configuration](configuration.md) — XDG paths and secret resolution
