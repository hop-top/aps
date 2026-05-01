---
status: shipped
---

# Profile Shorthand Execution

**ID**: 010
**Feature**: Shell Integration
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want to execute commands in a profile without typing `run` or `--` so that I can work faster.

## Acceptance Scenarios

1. **Given** profile `agent-a`, **When** I run `aps agent-a`, **Then** it launches an interactive session (equivalent to `aps run agent-a -- $SHELL`).
2. **Given** profile `agent-a`, **When** I run `aps agent-a git status`, **Then** it executes `git status` in the profile context (equivalent to `aps run agent-a -- git status`).
3. **Given** a subcommand `profile`, **When** I run `aps profile`, **Then** it executes the `profile` subcommand, NOT a profile named "profile".

## Tests

### E2E
- `tests/e2e/run_test.go` — `TestShorthandExecution`
