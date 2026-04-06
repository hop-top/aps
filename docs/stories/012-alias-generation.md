# Alias Generation

**ID**: 012
**Feature**: Shell Integration
**Persona**: [User](../personas/user.md)
**Priority**: P3

## Story

As a user, I want to generate shell aliases for my profiles so I can invoke them directly by name.

## Acceptance Scenarios

1. **Given** profile `agent-a`, **When** I run `aps alias`, **Then** output includes `alias agent-a='aps agent-a'`.
2. **Given** a collision (profile name same as system command), **When** I run `aps alias`, **Then** it warns about the conflict.

## Tests

_No dedicated tests yet._
