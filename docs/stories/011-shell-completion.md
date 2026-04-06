# Shell Completion

**ID**: 011
**Feature**: Shell Integration
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want tab completion for profiles so that I can easily select the right agent.

## Acceptance Scenarios

1. **Given** valid profiles, **When** I trigger completion on `aps [TAB]`, **Then** profile IDs are suggested alongside subcommands.
2. **Given** `aps agent-a [TAB]`, **When** I trigger completion, **Then** it suggests commands or files (standard shell behavior).

## Tests

_No dedicated tests yet._
