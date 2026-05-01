---
status: paper
---

# Voice Sessions

**ID**: 042
**Feature**: Voice
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to start voice sessions on different channels, list active sessions, close them, and switch the active profile mid-session so that I can manage live spoken conversations across web, terminal, phone, and messenger interfaces.

## Acceptance Scenarios

1. **Start a voice session on the web channel**
   - **Given** a profile with `voice.channels.web: true` and the voice backend running
   - **When** I run `aps voice start --profile <id> --channel web`
   - **Then** a WebSocket endpoint is available at `/ws`
   - **And** a browser or WebSocket client can connect and exchange audio frames

2. **Start a voice session on the TUI channel**
   - **Given** a profile with `voice.channels.tui: true`
   - **When** I run `aps voice start --profile <id> --channel tui`
   - **Then** a Unix domain socket is created
   - **And** the Hex TUI can connect and exchange audio frames

3. **Receive a voice session from a Telegram audio message**
   - **Given** a profile with Telegram voice channel enabled and a running webhook server
   - **When** a user sends an audio attachment to the Telegram bot
   - **Then** a voice session is created with `channel_type=telegram` and the caller's Telegram user ID
   - **And** the audio URL is available for downstream processing

4. **Receive a voice session from a Twilio phone call**
   - **Given** a profile with Twilio voice channel enabled and a running protocol server
   - **When** a caller dials the configured phone number
   - **Then** Twilio connects to `/twilio/media-stream`
   - **And** a voice session is created with `channel_type=twilio` and the caller's phone number as `caller_id`

5. **List active voice sessions**
   - **Given** one or more voice sessions are active
   - **When** I run `aps voice session list`
   - **Then** each session is shown with its ID, profile, channel type, and state

6. **List when no sessions are active**
   - **Given** no voice sessions exist
   - **When** I run `aps voice session list`
   - **Then** I see "No active voice sessions."

7. **Close a voice session**
   - **Given** an active voice session with a known ID
   - **When** the session is closed
   - **Then** its state transitions to `closed`

8. **Switch profile mid-session**
   - **Given** an active voice session bound to profile A
   - **When** the profile is switched to profile B on the same session
   - **Then** the session's `profile_id` is updated to B without terminating the connection

9. **Get a session by ID**
   - **Given** an active voice session
   - **When** it is looked up by ID
   - **Then** the session is returned with its current state

10. **Look up a non-existent session**
    - **Given** no session exists for a given ID
    - **When** it is looked up
    - **Then** an error is returned indicating the session was not found

## Tests

### E2E
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_StartOnWebChannel`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_StartOnTUIChannel`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_FromTelegramAudio`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_FromTwilioCall`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_ListActive`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_ListEmpty`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_CloseTransitionsState`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_SwitchProfileMidSession`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_GetByID`
- planned: `tests/e2e/voice_sessions_test.go::TestVoiceSession_LookupNotFound`

### Unit
- `internal/voice/session_test.go` — Create, Get, List, Close, SwitchProfile; not-found errors
- `internal/voice/channel_test.go` — ChannelSession and ChannelAdapter interface contracts
- `internal/voice/adapter_web_test.go` — WebAdapter session creation and ServeHTTP routing
- `internal/voice/adapter_tui_test.go` — TUIAdapter Unix socket accept loop
- `internal/voice/adapter_twilio_test.go` — TwilioAdapter Media Streams upgrade and session metadata
- `internal/voice/voice_messenger_test.go` — MessengerVoiceHandler audio attachment detection and session emission
