---
status: paper
---

# Thread-Scoped Runs

**ID**: 033
**Feature**: Agent Protocol Adapter
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P3

## Story

As an external client, I want to scope runs to threads so that I can maintain conversational context across multiple interactions with an agent.

## Acceptance Scenarios

1. **Given** a thread, **When** I create a run within it, **Then** the run has access to the thread's history.
2. **Given** multiple threads, **When** I list runs for a specific thread, **Then** only that thread's runs are returned.

## Tests

### E2E
- planned: `tests/e2e/thread_runs_test.go::TestThreadRun_AccessesThreadHistory`
- planned: `tests/e2e/thread_runs_test.go::TestThreadRun_ListScopedToThread`
