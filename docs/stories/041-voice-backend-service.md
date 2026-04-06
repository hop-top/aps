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
   - **Then** the configured backend binary is launched as a child process
   - **And** I see "Voice backend started."

2. **Start with no backend configured**
   - **Given** no backend binary is configured and no `voice.backend.url` is set
   - **When** I run `aps voice service start`
   - **Then** the command fails with an error indicating no backend binary is configured

3. **Stop the voice backend**
   - **Given** the backend service is running
   - **When** I run `aps voice service stop`
   - **Then** the process is terminated and I see "Voice backend stopped."

4. **Stop when not running**
   - **Given** no backend process is running
   - **When** I run `aps voice service stop`
   - **Then** the command succeeds silently (no-op)

5. **Check service status — running**
   - **Given** the backend service is running
   - **When** I run `aps voice service status`
   - **Then** I see "running"

6. **Check service status — stopped**
   - **Given** no backend service is running
   - **When** I run `aps voice service status`
   - **Then** I see "stopped"

## Tests

### Unit
- `internal/voice/backend_test.go` — `Start`, `Stop`, `IsRunning`; external URL no-op; missing binary error
