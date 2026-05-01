---
status: shipped-no-e2e
---

# Listener Daemon

**ID**: 051
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a profile owner, I want a long-running listener that subscribes to bus topics, webhook
ingress, and A2A streams so my agent reacts to events without me polling.

## Acceptance Scenarios

1. **Given** a profile config with `listen.topics`, **When** I run `aps listen --profile <id>`,
   **Then** it subscribes to bus + webhook + A2A streams matching that config.
2. **Given** a published `tlc.task.assigned` event whose assignee matches the profile, **When**
   the listener receives it, **Then** the configured handler is invoked (action / adapter /
   skill / webhook).
3. **Given** a transient bus disconnect, **When** reconnect succeeds, **Then** subscriptions
   resume from the last position with exponential backoff.
4. **Given** a handler invocation fails, **When** the listener processes the failure, **Then**
   the event is logged and the listener does NOT crash (fail-soft).
5. **Given** SIGTERM or SIGINT, **When** received, **Then** the listener drains in-flight
   handlers and exits cleanly.

## Tests

### E2E
- `tests/e2e/listen/listen_subscribe_test.go` — `TestListen_SubscribesToConfiguredTopics`
- `tests/e2e/listen/listen_dispatch_test.go` — `TestListen_DispatchesToAction`
- `tests/e2e/listen/listen_reconnect_test.go` — `TestListen_ReconnectsWithBackoff`
- `tests/e2e/listen/listen_failsoft_test.go` — `TestListen_FailSoftOnHandlerError`
- `tests/e2e/listen/listen_shutdown_test.go` — `TestListen_GracefulShutdown`
