# Voice

APS currently exposes voice backend lifecycle commands and voice session registration. Web, Twilio, and messenger voice adapters exist as Go components, but the current CLI does not mount them as reachable profile-facing services.

The design intent is speech-to-speech service routing across web, terminal, messaging, and telephony channels. Treat that routing as component-level or planned unless a command in this document explicitly says it starts a listener.

## Quick Start

```bash
# Start the voice backend service
aps voice service start

# Register a voice session record (does not mount a web listener)
aps voice start --profile my-profile

# List active sessions
aps voice session list
```

## CLI Reference

```
aps voice start --profile <id> [--channel web|tui|telegram|twilio]
aps voice service start
aps voice service stop
aps voice service status
```

`aps voice service start|stop|status` controls the backend process only. `aps voice start` registers a voice session for metadata/routing work; it does not start the WebSocket or Twilio HTTP handlers. Use `aps session list --type voice` for voice session records.

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
# ~/.local/share/aps/profiles/my-profile/profile.yaml
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

## Current Maturity

| Surface | User command | Listener mounted | Execution/reply behavior | Maturity |
| --- | --- | --- | --- | --- |
| Backend lifecycle | `aps voice service start` | No APS intake route; starts the configured backend process | CLI status only | process lifecycle |
| Session registration | `aps voice start --profile <id>` | No | Registers a voice session record | component support |
| Web adapter | component `voice.NewWebAdapter` | Only if another caller mounts it | WebSocket audio/text frames | component |
| Twilio adapter | component `voice.NewTwilioAdapter` | Only if another caller mounts it | Twilio Media Streams WebSocket frames | component |
| Messenger voice handler | component `voice.NewMessengerVoiceHandler` | Only if attached to a mounted messenger handler | Emits voice channel sessions from audio attachments | component |

## Persona Prompt Auto-Generation

When `prompt_template` is empty, APS can generate the backend text prompt from `persona` fields. `tone`, `style`, and `risk` map to natural-language instructions intended for backend session setup. The mounted channel pipeline that injects those prompts is not exposed by the current CLI service path.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        APS CLI                              │
│  aps voice start --profile <id>                             │
│  aps voice service start|stop|status                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                  Voice Orchestrator                         │
│  (internal/voice/)                                          │
│  - Profile → backend config mapping                         │
│  - Service lifecycle management                             │
│  - Session registration                                     │
│  - Component channel adapters                               │
│  - Future mounted channel routing                           │
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

`NewWebAdapter(profileID)` implements an HTTP handler for WebSocket voice sessions at `/ws`. The current `aps voice` CLI does not mount this handler, serve a browser UI, or proxy it to a backend.

### Hex TUI (terminal)

A terminal channel remains design-level in this tree. The current CLI can register a session with `--channel tui`, but no traced command mounts a TUI voice transport.

### Messenger (Telegram, WhatsApp)

`NewMessengerVoiceHandler()` can be attached to the messenger webhook handler component. It detects audio attachments and emits `ChannelSession` objects. The current messenger live intake path is not mounted by the traced `aps messenger` lifecycle, so this is component-only until a mounted messenger route wires it.

### Telephony (Twilio)

`NewTwilioAdapter(phoneNumber, profileID)` implements an HTTP handler for Twilio Media Streams WebSocket connections at `/twilio/media-stream`. The current `aps voice` CLI does not mount this handler or expose a public telephony route.

## Session Routing Modes

**Explicit** — `aps voice start --profile <id>` records a voice session for a selected profile and channel. It does not accept audio by itself.

**Intent-based** — design-level. No current mounted voice service route was traced that classifies transcript chunks and migrates profiles.

**Mid-session switch** — component-level. `SessionManager.SwitchProfile` exists, but no user-facing mounted voice channel path was traced.

## Session Lifecycle

```
component channel connects, if mounted by a caller
      │
      ▼
SessionManager.Create(profileID, channelType)
      │
      ├── resolve profile voice config
      ├── backend/session component setup
      └── open component pipeline
            │
            ▼
      [active session]
            │
            ├── audio/text frames arrive through component adapters
            │     └── mounted execution/reply pipeline not yet exposed by CLI
            │
            └── channel disconnects / timeout → Session.Close()
```

## Backend Agnosticism

APS speaks the PersonaPlex/Moshi WebSocket protocol and accepts any conforming backend. Set `backend.type: compatible` in the profile to use any compliant implementation that is not one of the named types.

## Related

- [Design document](../plans/2026-03-16-voice-integration-design.md)
- [Messenger Architecture](readme.md#-messenger-architecture) — platform comparison, normalized message format, routing
- [Configuration](configuration.md) — XDG paths and secret resolution
