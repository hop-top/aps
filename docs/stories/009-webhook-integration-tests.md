---
status: shipped
---

# Webhook Integration Tests

**ID**: 009
**Feature**: E2E Tests
**Persona**: [Maintainer](../personas/maintainer.md)
**Related Personas**: [User](../personas/user.md) (validates [004](004-webhook-server.md))
**Priority**: P3

## Story

As a maintainer, I want verification that the webhook server correctly routes events and validates signatures so that the complex integration point has regression testing.

## Acceptance Scenarios

1. **Given** a running webhook server, **When** a valid signed request is sent, **Then** it returns 200 and triggers the action.
2. **Given** a running webhook server, **When** an invalid signature is sent, **Then** it returns 401.
3. **Given** a running webhook server, **When** an unmapped event is sent, **Then** it returns 400.

## Tests

### E2E
- `tests/e2e/webhook_test.go` — `TestWebhookServer`
