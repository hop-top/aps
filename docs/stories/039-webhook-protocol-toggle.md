# Webhook Protocol Toggle

**ID**: 039
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to enable or disable the Webhook protocol for my profile so that I can control when my profile receives webhook events.

## Acceptance Scenarios

1. **Given** a profile without Webhooks enabled, **When** I run `aps webhook toggle --profile <id>`, **Then** Webhooks are enabled for the profile.
2. **Given** a profile with Webhooks enabled, **When** I run `aps webhook toggle --profile <id>`, **Then** Webhooks are disabled and removed from the profile.
3. **Given** a profile with Webhooks enabled, **When** I run `aps webhook toggle --profile <id> --enabled=on`, **Then** the configuration is preserved and reconfirmed.
4. **Given** a profile, **When** I run `aps webhook server --profile <id>`, **Then** if Webhooks are not enabled, they are auto-enabled and the server starts.

## Tests

### E2E
- `tests/e2e/webhook_test.go` — `TestWebhookToggle_Enable`, `TestWebhookToggle_Disable`, `TestWebhookToggle_ServerAutoEnable`
