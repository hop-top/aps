# Voice Profile Configuration

**ID**: 040
**Feature**: Voice
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to enable voice on a profile and configure its backend, channels, and persona prompt so that the profile can accept real-time spoken conversations.

## Acceptance Scenarios

1. **Enable voice on a profile**
   - **Given** a profile YAML with no voice block
   - **When** I add `voice: {enabled: true}` and reload the profile
   - **Then** voice is enabled with auto-detected backend and no active channels

2. **Select a specific backend type**
   - **Given** a profile with `voice.backend.type: personaplex-mlx`
   - **When** the voice service starts
   - **Then** it uses the personaplex-mlx binary configured in `~/.config/aps/config.yaml`

3. **Delegate to an external backend**
   - **Given** a profile with `voice.backend.url: ws://192.168.1.10:8998`
   - **When** the voice service starts
   - **Then** it connects to the external instance and does not launch a local process

4. **Auto-detect backend by platform**
   - **Given** a profile with `voice.backend.type: auto` (or omitted)
   - **When** the voice service starts on macOS
   - **Then** it prefers `personaplex-mlx` then `moshi-mlx`, falling back to `compatible`
   - **And** on Linux it prefers `personaplex-cuda` then `moshi`

5. **Enable web and TUI channels**
   - **Given** a profile with `voice.channels.web: true` and `voice.channels.tui: true`
   - **When** the voice service starts
   - **Then** both the WebSocket endpoint at `/ws` and the Unix socket adapter are active

6. **Enable Telegram voice channel**
   - **Given** a profile with `voice.channels.telegram.enabled: true` and `bot_token_secret` set
   - **When** a user sends an audio message to the bot
   - **Then** the message is routed to the profile's voice session via the messenger webhook pipeline

7. **Enable Twilio voice channel**
   - **Given** a profile with `voice.channels.twilio.enabled: true`, `phone_number`, `account_sid_secret`, and `auth_token_secret` set
   - **When** a caller dials the configured number
   - **Then** Twilio streams audio to `/twilio/media-stream` and the call is handled by the profile

8. **Custom prompt template**
   - **Given** a profile with `voice.prompt_template: "You are a concise assistant. Answer in one sentence."`
   - **When** a voice session starts
   - **Then** the custom template is injected into the backend verbatim, overriding auto-generation

9. **Persona-derived prompt**
   - **Given** a profile with `persona.tone: professional`, `persona.style: concise`, `persona.risk: low` and no `voice.prompt_template`
   - **When** a voice session starts
   - **Then** the backend receives a generated prompt describing the persona in natural language

10. **Voice ID selection**
    - **Given** a profile with `voice.voice_id: NATF0`
    - **When** a voice session starts
    - **Then** the backend uses the specified voice ID for speech synthesis

## Tests

### E2E
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_EnableOnProfile`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_BackendTypeSelection`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_DelegateExternalBackend`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_AutoDetectBackendByPlatform`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_EnableWebAndTUIChannels`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_EnableTelegramChannel`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_EnableTwilioChannel`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_CustomPromptTemplate`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_PersonaDerivedPrompt`
- planned: `tests/e2e/voice_config_test.go::TestVoiceConfig_VoiceIDSelection`

### Unit
- `internal/voice/config_test.go` — Config type defaults and YAML unmarshalling
- `internal/voice/prompt_test.go` — Prompt generation from persona fields; custom template override
- `internal/voice/backend_test.go` — `ResolveType` platform-preference order; external URL passthrough
