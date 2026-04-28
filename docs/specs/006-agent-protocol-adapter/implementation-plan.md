# Implementation Plan: Agent Protocol Adapter

**Feature**: 006-agent-protocol-adapter
**Status**: Draft
**Created**: 2026-01-21
**Estimated Duration**: 6 weeks

---

## Architecture Decisions (Open Questions Resolved)

### 1. Store Scope
**Decision**: Profile-scoped namespaces
- Format: `{profile_id}/{key}`
- Prevents cross-profile data leakage
- Natural mapping to APS profile isolation model
- Example: `agent-a/user_prefs` vs `agent-b/user_prefs`

### 2. Action Schema Generation
**Decision**: Multi-tier approach
- **Tier 1**: Parse action manifests (actions.yaml) → infer basic structure
- **Tier 2**: Add optional `.schema.json` files alongside actions for detailed schemas
- **Tier 3**: Auto-detect JSON schema from shebang/doc comments (future)
- **Initial implementation**: Tier 1 only - generate minimal schema from manifest

### 3. Session Persistence
**Decision**: In-memory with optional disk persistence
- In-memory for fast operations
- Existing SessionRegistry already persists to disk (`~/.aps/sessions/registry.json`)
- Leverage existing mechanism for thread metadata
- Thread-specific state stored in `~/.aps/sessions/threads/{thread_id}.json`

### 4. Authentication
**Decision**: Multiple auth modes (optional)
- **Mode A**: No auth (default, for local development)
- **Mode B**: Static Bearer token (simple production setup)
- **Mode C**: API key lookup file (multi-tenant)
- **Mode D**: HMAC signature (webhook compatibility)
- **Implementation**: Start with Mode A + B, extensible via middleware

---

## Directory Structure

```
add-agent-protocol-support/
├── internal/
│   ├── core/
│   │   └── protocol/                    # NEW: Protocol adapter layer
│   │       ├── interface.go            # APSCore interface definition
│   │       ├── types.go                # Core types (RunInput, RunState, etc.)
│   │       └── core.go                 # APSCore implementation
│   │
│   └── adapters/                        # NEW: Protocol adapters
│       ├── registry.go                 # Adapter registration system
│       └── agentprotocol/              # Agent Protocol adapter
│           ├── adapter.go              # ProtocolAdapter interface impl
│           ├── handlers.go             # HTTP handler functions
│           ├── runs.go                 # /runs endpoint logic
│           ├── threads.go              # /threads endpoint logic
│           ├── agents.go               # /agents endpoint logic
│           ├── store.go                # /store endpoint logic
│           ├── sse.go                  # SSE streaming utilities
│           └── types.go                # Agent Protocol types
│
├── cmd/aps/
│   └── main.go                        # Updated: init() adapters
│
└── tests/
    ├── e2e/
    │   └── agent_protocol_test.go     # NEW: E2E protocol tests
    └── unit/
        ├── core/
        │   └── protocol_test.go       # NEW: Core tests
        └── adapters/
            └── agentprotocol_test.go  # NEW: Adapter tests
```

---

## Implementation Phases

### Phase 1: Core Interface Layer (Week 1)
**Goal**: Abstract APS execution into protocol-agnostic interface

#### Tasks:
1. **Create `internal/core/protocol/interface.go`**
   - Define `APSCore` interface with all methods from spec
   - Define `RunInput`, `RunState`, `SessionState`, `AgentInfo` types
   - Define `RunStatus`, `StreamMode` constants
   - Add comprehensive godoc comments

2. **Create `internal/core/protocol/types.go`**
   - Implement all data structures
   - Add JSON tags for HTTP serialization
   - Add validation methods (e.g., `Validate() error`)

3. **Create `internal/core/protocol/core.go`**
   - Implement `APSCore` as adapter over existing functions
   - `ExecuteRun()`: Call `RunAction()` or `RunCommand()` based on input
   - `GetAgent()`: Call `LoadProfile()` + map to `AgentInfo`
   - `GetAgentSchemas()`: Parse actions, generate minimal JSON Schema
   - `CreateSession()`: Create session in registry + optional thread storage
   - `Store*()`: Implement simple KV store in `~/.aps/store/`
   - Add error mapping: APS errors → HTTP-compatible errors

4. **Create `tests/unit/core/protocol_test.go`**
   - Test `APSCore` methods in isolation
   - Mock existing core functions
   - Verify type conversions and error handling

**Deliverables**: Protocol-agnostic core layer, full unit test coverage

---

### Phase 2: Adapter Registry (Week 1-2)
**Goal**: Declarative adapter registration system

#### Tasks:
1. **Create `internal/adapters/registry.go`**
   - Define `ProtocolAdapter` interface with `Name()` and `RegisterRoutes()`
   - Define `AdapterRegistry` struct
   - Implement `Register(name, adapter)` method
   - Implement `GetAdapter(name)` method
   - Implement `ListAdapters()` method
   - Implement `RegisterAll(mux, core)` to mount all registered adapters

2. **Update `cmd/aps/main.go`**
   - Import adapters package
   - Call `adapters.RegisterDefaults()` in init()
   - Add `serve` command registration (see Phase 5)

**Deliverables**: Dynamic adapter registration, extensible for future protocols

---

### Phase 3: Agent Protocol Adapter - Core (Week 2)
**Goal**: Basic Agent Protocol endpoint implementation

#### Tasks:
1. **Create `internal/adapters/agentprotocol/types.go`**
   - Define Agent Protocol request/response types
   - Define `CreateRunRequest`, `RunResponse`, `ThreadRequest`, etc.
   - Map Agent Protocol types to core types
   - Add JSON schema tags

2. **Create `internal/adapters/agentprotocol/adapter.go`**
   - Implement `ProtocolAdapter` interface
   - Define `AgentProtocolAdapter` struct
   - Implement `RegisterRoutes()` to mount all handlers
   - Add helper methods for error response formatting

3. **Create `internal/adapters/agentprotocol/handlers.go`**
   - `handleRunWait`: Execute sync run, return result
   - `handleRunStatus`: Get run state by ID
   - `handleRunCancel`: Cancel running run
   - `handleCreateThread`: Create session/thread
   - `handleGetThread`: Get session state
   - `handleAgentSearch`: List agents (profiles)
   - `handleGetAgent`: Get agent metadata
   - `handleGetAgentSchemas`: Return action schemas
   - Add proper error handling (404, 400, 409, 500)

**Deliverables**: Functional Agent Protocol endpoints (non-streaming)

---

### Phase 4: Streaming Support (Week 3)
**Goal**: SSE streaming for `/runs/stream` endpoint

#### Tasks:
1. **Create `internal/adapters/agentprotocol/sse.go`**
   - Implement SSE utilities
   - Define `SSEWriter` struct for writing events
   - Methods: `WriteEvent(event, data)`, `Flush()`, `Close()`
   - Handle client disconnection detection
   - Add event types: `output`, `done`, `error`

2. **Update `internal/adapters/agentprotocol/runs.go`**
   - Implement `handleRunStream` handler
   - Execute action in background goroutine
   - Capture stdout/stderr via `io.Pipe()`
   - Stream chunks as SSE events
   - Handle process termination and status updates
   - Cleanup on context cancellation

3. **Update `internal/core/protocol/core.go`**
   - Modify `ExecuteRun()` to support streaming output
   - Return channel or callback for streaming chunks
   - Ensure cancellation propagates to child process

4. **Add streaming tests**
   - Test SSE event ordering
   - Test client disconnection cleanup
   - Test cancellation mid-stream

**Deliverables**: Full SSE streaming support with proper resource cleanup

---

### Phase 5: CLI Integration (Week 3-4)
**Goal**: New `aps serve` command to start protocol server

#### Tasks:
1. **Create `internal/cli/serve.go`**
   - Define `serveCmd` Cobra command
   - Flags: `--port`, `--addr`, `--adapter`, `--auth-token`, `--log-level`
   - Implement `ServeProtocols()` function
   - Setup HTTP server with `http.ServeMux`
   - Register adapters via `adapters.RegisterAll()`
   - Add graceful shutdown (SIGINT/SIGTERM)
   - Add structured logging (start, stop, errors)
   - Add health check endpoint (`GET /health`)

2. **Integrate auth middleware**
   - Implement `authMiddleware()` for token validation
   - Add optional auth modes (None, Bearer Token)
   - Apply middleware to protocol routes only

3. **Update `cmd/aps/main.go`**
   - Import serve command
   - Register in init()

4. **Add E2E tests**
   - Start server in subprocess
   - Test `POST /runs/wait` with curl/http client
   - Test `POST /runs/stream` with SSE client
   - Test error scenarios

**Deliverables**: Fully functional CLI command to serve Agent Protocol API

---

### Phase 6: Store Implementation (Week 4)
**Goal**: Complete `/store` endpoints

#### Tasks:
1. **Create `internal/core/store/`** (if not exists)
   - Define store interface
   - Implement disk-based KV store in `~/.aps/store/`
   - Namespace-aware operations
   - Thread-safe access (mutex)

2. **Create `internal/adapters/agentprotocol/store.go`**
   - `handleStorePut`: Create/update item
   - `handleStoreGet`: Get item
   - `handleStoreDelete`: Delete item
   - `handleStoreSearch`: Search items by query (prefix match)
   - Add validation for keys/namespaces

3. **Add store tests**
   - Test CRUD operations
   - Test namespace isolation
   - Test concurrent access

**Deliverables**: Working store API with profile-scoped isolation

---

### Phase 7: Advanced Features (Week 5)
**Goal**: Complete remaining endpoints and polish

#### Tasks:
1. **Implement remaining Agent Protocol endpoints**
   - `POST /runs` (background runs)
   - `GET /runs/{id}/wait` (wait for existing run)
   - `GET /runs/{id}/stream` (reconnect to stream)
   - `DELETE /runs/{id}` (cleanup)
   - `POST /threads/search` (search sessions)
   - `GET /threads/{id}/history` (session history)
   - `POST /threads/{id}/runs` (run in thread context)
   - `DELETE /threads/{id}` (delete thread)
   - `PATCH /threads/{id}` (update metadata)
   - `POST /store/namespaces` (list namespaces)

2. **Implement run history tracking**
   - Create `~/.aps/sessions/runs.json` for run records
   - Record start time, end time, status, output size
   - Implement `ListRuns(threadID)` for history

3. **Edge case handling**
   - EC-001: Run cancelled before process starts
   - EC-002: Client disconnects mid-stream
   - EC-003: Concurrent runs on same thread (409 Conflict)
   - EC-004: Invalid agent_id or action (404)
   - EC-005: Store namespace doesn't exist (auto-create)

**Deliverables**: Complete Agent Protocol compliance, all edge cases handled

---

### Phase 8: Testing & Documentation (Week 6)
**Goal**: Comprehensive testing and documentation

#### Tasks:
1. **Complete unit tests**
   - `tests/unit/adapters/agentprotocol_test.go`
   - Test all handlers with mocked core
   - Test error paths
   - Test concurrency

2. **Complete integration tests**
   - `tests/e2e/agent_protocol_test.go`
   - Test all user stories from spec
   - Test real profile/action execution
   - Test streaming with actual long-running actions

3. **Performance testing**
   - Verify SC-001: `POST /runs/wait` overhead < 100ms
   - Verify SC-002: First SSE event < 50ms
   - Verify SC-003: Cancel response < 1s

4. **Documentation**
   - Update README with API usage examples
   - Add `docs/agent-protocol.md` with endpoint reference
   - Add example curl commands
   - Document auth modes
   - Document adapter registration for future protocols

5. **Success criteria validation**
   - SC-001 through SC-005 validated
   - All acceptance scenarios from user stories tested

**Deliverables**: Production-ready implementation with full test coverage and documentation

---

## Testing Strategy

### Unit Tests
- **Mock-based**: Use `testify/mock` to isolate core functions
- **Coverage target**: > 80% for adapters, > 90% for core protocol
- **Test framework**: `testing` + `testify/assert`

### Integration Tests
- **E2E scenarios**: Real HTTP server, real profiles/actions
- **Test profiles**: Create temporary profiles in test setup
- **Cleanup**: Ensure temp profiles/sessions are deleted
- **Concurrent**: Run tests in parallel with `t.Parallel()`

### Performance Tests
- **Benchmarking**: Use `testing.B` for critical paths
- **Metrics**: Measure execution overhead, streaming latency
- **Tools**: `pprof` for memory profiling if needed

### Compliance Tests
- **Agent Protocol spec compliance**: Run official test suite if available
- **Custom compliance tests**: Verify all required behaviors from spec

---

## Risk Assessment

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **SSE client disconnection cleanup** | High | Medium | Use context cancellation, detect `context.Canceled`, cleanup goroutines with `defer` |
| **Concurrent run state updates** | High | Medium | Use `sync.RWMutex` for run state map, atomic operations for status |
| **Process termination reliability** | High | Low | Test on macOS/Linux/Windows, fallback to `os.Interrupt` then `SIGKILL` |
| **Action schema inference accuracy** | Medium | High | Start with minimal schemas, allow explicit overrides in future |
| **Store persistence race conditions** | Medium | Medium | File-level locking (`flock`) or in-memory with periodic sync |

### Integration Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Breaking changes to existing core** | High | Low | Core is stable, only wrapping existing functions |
| **Session registry conflicts** | Medium | Medium | Namespace protocol sessions (prefix: `protocol-`) |
| **Memory leaks in long-running streams** | High | Medium | Use `defer cleanup` patterns, monitor with pprof in tests |
| **Port conflicts with webhook server** | Low | Low | Default ports differ (webhook: 3000, protocol: 8080) |

### Dependency Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Agent Protocol spec changes** | Medium | Low | Pin to specific version, follow upstream for updates |
| **LangChain compatibility** | Low | Low | No direct dependency, implementing spec directly |

---

## Success Criteria Checklist

- [ ] **SC-001**: `POST /runs/wait` returns action output within 100ms overhead
- [ ] **SC-002**: `POST /runs/stream` delivers first SSE event within 50ms
- [ ] **SC-003**: `POST /runs/{id}/cancel` terminates within 1 second
- [ ] **SC-004**: All Agent Protocol endpoints pass compliance tests
- [ ] **SC-005**: New adapter requires only files in `internal/adapters/`, no `internal/core/` changes

---

## Timeline Summary

| Phase | Duration | Key Deliverable |
|-------|----------|-----------------|
| 1 | Week 1 | Core interface layer |
| 2 | Week 1-2 | Adapter registry |
| 3 | Week 2 | Basic Agent Protocol endpoints |
| 4 | Week 3 | SSE streaming support |
| 5 | Week 3-4 | CLI `serve` command |
| 6 | Week 4 | Store implementation |
| 7 | Week 5 | Advanced features & edge cases |
| 8 | Week 6 | Testing, documentation, validation |

**Total Duration**: ~6 weeks

---

## Implementation Notes

1. **SSE Streaming**: Use `net/http` with `http.Flusher` interface, no external SSE libraries needed
2. **Process Management**: Leverage existing isolation adapters, wrap for async execution
3. **Error Mapping**: Map APS errors to HTTP codes:
   - Profile not found → 404
   - Invalid input → 400
   - Concurrent run → 409
   - System errors → 500
4. **Logging**: Use existing logger or add `log/slog` for structured logging
5. **Metrics**: Optional - consider adding Prometheus metrics in future
6. **Testing**: Use `net/http/httptest` for HTTP handler tests, `httptest.NewServer` for integration

---

## Dependencies

### External Dependencies
- **None**: Use Go standard library only (`net/http`, `context`, `encoding/json`)

### Internal Dependencies
- `internal/core/*` - Execution, profiles, actions
- `internal/core/session/*` - Session management
- `internal/core/isolation/*` - Isolation adapters
- `github.com/spf13/cobra` - CLI (already used)

---

## Open Questions for Review

1. **Authentication**: Should we implement all auth modes (A-D) or start simpler?
2. **Store implementation**: Should store be persistent by default or in-memory only?
3. **Run history**: Is run history required for v1, or can it be v2?
4. **Backwards compatibility**: Should existing webhook server continue to work independently?
5. **Default port**: Is 8080 appropriate, or should it be configurable (default to something else)?

---

## Next Steps

1. Review this plan with stakeholders
2. Clarify open questions
3. Begin Phase 1 implementation
4. Set up CI for running new tests
