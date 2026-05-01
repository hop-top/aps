---
status: paper
---

# Documentation Generation

**ID**: 005
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P3

## Story

As a user, I want to generate local documentation so that I have offline access to the system manual.

## Acceptance Scenarios

1. **Given** the system is installed, **When** I run `aps docs`, **Then** the `<data>/docs` directory is populated with markdown files.

## Tests

### E2E
- planned: `tests/e2e/docs_test.go::TestDocsGenerate_PopulatesDataDir`
