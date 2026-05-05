---
status: implemented
---

# Voice Backend Service

**ID**: 041
**Feature**: Voice
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to start, stop, and check the status of the voice backend service from the CLI so that I can control when speech processing is available.

## Acceptance Scenarios

1. **Start the voice backend**
   - **Given** a `voice.backends` entry in `~/.config/aps/config.yaml`
   - **When** I run `aps voice service start`
   - **Then** the configured backend binary is launched as a detached child process
   - **And** the child PID is persisted to `$XDG_RUNTIME_DIR/aps/voice.pid`
   - **And** I see "Voice backend started."

2. **Start with no backend configured**
   - **Given** no backend binary is configured and no `voice.backend.url` is set
   - **When** I run `aps voice service start`
   - **Then** the command fails with an error indicating no backend binary is configured

3. **Stop the voice backend**
   - **Given** the backend service is running (a live PID file is on disk)
   - **When** I run `aps voice service stop`
   - **Then** the process recorded in the PID file is signalled (SIGTERM, then SIGKILL after a grace period)
   - **And** the PID file is removed
   - **And** I see "Voice backend stopped."

4. **Stop when not running**
   - **Given** no PID file is present (or it is stale)
   - **When** I run `aps voice service stop`
   - **Then** the command succeeds silently and any stale PID file is swept

5. **Check service status — running**
   - **Given** a live PID file is on disk
   - **When** I run `aps voice service status`
   - **Then** I see "running"

6. **Check service status — stopped**
   - **Given** no PID file is present (or it is stale)
   - **When** I run `aps voice service status`
   - **Then** I see "stopped"

7. **Refuse concurrent start**
   - **Given** the backend service is already running
   - **When** I run `aps voice service start` again
   - **Then** the command fails with an error noting the existing PID

## Implementation notes

Lifecycle uses `hop.top/kit/go/console/ps` for the read path
(`EntryFromPIDFile` → liveness via signal-0 probe) and
`hop.top/kit/go/core/xdg` for the canonical PID-file location
(`$XDG_RUNTIME_DIR/aps/voice.pid`, with a `$TMPDIR` fallback when
runtime dir is unavailable). Process spawn uses `Setpgid` so the
backend outlives the CLI invocation.

## Tests

### E2E
- `tests/e2e/voice_service_test.go::TestVoiceService_StatusStopped_FreshHome`
- `tests/e2e/voice_service_test.go::TestVoiceService_StatusRunning_FromPIDFile`
- `tests/e2e/voice_service_test.go::TestVoiceService_Stop_TerminatesAndCleansPIDFile`
- `tests/e2e/voice_service_test.go::TestVoiceService_Stop_NoOpWhenNotRunning`
- `tests/e2e/voice_service_test.go::TestVoiceService_Stop_ZombiePIDFile`
- `tests/e2e/voice_service_test.go::TestVoiceService_Start_ErrorWhenNoBackendConfigured`

### Unit
- `internal/voice/backend_test.go` — `Start`/`Stop`/`IsRunning`/`Status`,
  external URL no-op, missing/unknown binary errors, PID-file write,
  zombie-PID sweep, concurrent-start refusal.
