---
status: shipped
---

# Webhook Server

**ID**: 004
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md) (tested by [009](009-webhook-integration-tests.md))
**Priority**: P3

## Story

As a user, I want to trigger actions via webhooks so that I can integrate with external systems like GitHub.

## Acceptance Scenarios

1. **Given** a running webhook server mapping `event.x` to `profile:action`, **When** I POST to `/webhook` with matching headers, **Then** the action is triggered.
2. **Given** a secured webhook server, **When** I request without a signature, **Then** I receive 401 Unauthorized.

## Tests

### E2E
- `tests/e2e/webhook_test.go` — `TestWebhookServer`
