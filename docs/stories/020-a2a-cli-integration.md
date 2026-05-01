---
status: paper
---

# A2A CLI Integration

**ID**: 020
**Feature**: A2A Protocol
**Persona**: [User](../personas/user.md)
**Priority**: P5

## Story

As a user, I want CLI commands to manage A2A tasks so that I can create, list, and inspect tasks from the terminal.

## Acceptance Scenarios

1. **Given** a running A2A server, **When** I run `aps a2a tasks send`, **Then** a task is created and its ID is returned.
2. **Given** existing A2A tasks, **When** I run `aps a2a tasks list`, **Then** I see all tasks with their statuses.

## Tests

### E2E
- planned: `tests/e2e/a2a_cli_test.go::TestA2A_TasksSendReturnsID`
- planned: `tests/e2e/a2a_cli_test.go::TestA2A_TasksListShowsAllStatuses`
