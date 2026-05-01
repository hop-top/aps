---
status: shipped-no-e2e
---

# Listener Routing Config

**ID**: 052
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Related Stories**: [051](051-listener-daemon.md)
**Priority**: P1

## Story

As a profile owner, I want declarative routing rules in profile YAML so I configure listener
behavior without code.

## Acceptance Scenarios

1. **Given** a profile YAML with `listen.topics[].pattern: "tlc.task.assigned"` +
   `.where: "assignee == <profile_id>"` + `.run.action: <name>`, **When** a matching event
   arrives, **Then** the named action runs with the event payload as input.
2. **Given** multiple rules with overlapping patterns, **When** an event arrives, **Then**
   first-match wins (rules ordered top-to-bottom).
3. **Given** a `where` expression that fails to evaluate, **When** the listener encounters it,
   **Then** the rule is skipped with a warning (does not crash the daemon).
4. **Given** `run.adapter: <name>` instead of `run.action`, **When** an event matches, **Then**
   `aps adapter exec <name>` is invoked with payload-derived inputs.
5. **Given** `run.webhook: <url>`, **When** an event matches, **Then** an HTTP POST is sent to
   the URL with the envelope as the body.

## Profile YAML schema (illustrative)

```yaml
listen:
  topics:
    - pattern: "tlc.task.assigned"
      where: "assignee == 'noor'"
      run:
        action: triage-task
    - pattern: "aps.adapter.email.received"
      where: "to contains 'noor@'"
      run:
        adapter: email
        with:
          op: reply-draft
    - pattern: "ctxt.capture.*"
      run:
        webhook: http://localhost:9000/hook
```

## Tests

### E2E
- `tests/e2e/listen/route_pattern_match_test.go` — `TestRoute_PatternMatch`
- `tests/e2e/listen/route_where_filter_test.go` — `TestRoute_WhereExpression`
- `tests/e2e/listen/route_first_match_test.go` — `TestRoute_FirstMatchWins`
- `tests/e2e/listen/route_action_dispatch_test.go` — `TestRoute_ActionDispatch`
- `tests/e2e/listen/route_adapter_dispatch_test.go` — `TestRoute_AdapterDispatch`
- `tests/e2e/listen/route_webhook_dispatch_test.go` — `TestRoute_WebhookDispatch`
- `tests/e2e/listen/route_invalid_where_test.go` — `TestRoute_InvalidWhereSkipsRule`
