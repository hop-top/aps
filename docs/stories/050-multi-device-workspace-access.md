---
status: shipped-no-e2e
---

# Multi-Device Workspace Access

**ID**: 050
**Feature**: Multi-Device Workspace
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to access the same workspace from multiple devices (laptop, phone, tablet) and have changes automatically synchronized across all devices, so that I can seamlessly continue my work regardless of which device I'm using.

## Acceptance Scenarios

1. **Device Linking**
   - **Given** I have a workspace and multiple devices
   - **When** I run `aps device link <device-id>` on the new device
   - **Then** the device is linked to the workspace and can receive updates

2. **Real-Time Synchronization**
   - **Given** I am on Device A and create a new profile
   - **When** the profile is created
   - **Then** Device B sees the new profile in real-time (within 100ms)

3. **Concurrent Modifications**
   - **Given** Device A and Device B are both modifying the same profile
   - **When** both devices update the configuration simultaneously
   - **Then** the conflict is resolved automatically (Last-Write-Wins) and both devices converge to the same state

4. **Device Presence**
   - **Given** I have multiple linked devices
   - **When** I run `aps device presence <workspace-id>`
   - **Then** I see the status of each device (online, away, offline) with last activity timestamp

5. **Access Control**
   - **Given** Device A has write permissions and Device B has read-only permissions
   - **When** Device B tries to create a new action
   - **Then** the operation is denied and an error message is shown

6. **Offline Synchronization**
   - **Given** Device A goes offline and creates profiles
   - **When** Device A reconnects to the network
   - **Then** all offline changes are synced to the workspace and pushed to other devices

7. **Device Presence Timeout**
   - **Given** Device A is online
   - **When** Device A loses connectivity for 30 seconds
   - **Then** the device status changes to "away"
   - **And** after 120 seconds, the status changes to "offline"

8. **Rate Limiting**
   - **Given** Device A has rate limiting of 100 requests per minute
   - **When** Device A exceeds 100 requests in 60 seconds
   - **Then** additional requests are rejected with a 429 status and a Retry-After header

## Tests

### Unit
- `tests/unit/core/multidevice/linker_test.go` — Device linking, permission updates
- `tests/unit/core/multidevice/presence_test.go` — Presence state machine, transitions
- `tests/unit/core/multidevice/access_control_test.go` — Permission evaluation, policy models
- `tests/unit/core/multidevice/conflict_test.go` — Conflict detection and resolution
- `tests/unit/core/multidevice/offline_test.go` — Offline queue, sync recovery

### E2E
- `tests/e2e/multidevice/event_broadcasting_test.go` — Event publishing, subscription, ordering
- `tests/e2e/multidevice/presence_tracking_test.go` — Heartbeat, timeout detection, state transitions
- `tests/e2e/multidevice/access_control_test.go` — Permission enforcement, rate limiting
- `tests/e2e/multidevice/conflict_resolution_test.go` — Automatic and manual conflict resolution
- `tests/e2e/multidevice/sync_test.go` — Offline sync, event recovery
- `tests/e2e/multidevice/integration_test.go` — Full multi-device workflow scenarios
