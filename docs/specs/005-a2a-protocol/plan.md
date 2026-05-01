# A2A Protocol Adoption Plan for APS

**Status**: Architecture Update Required
**Created**: 2026-01-20
**Updated**: 2026-01-21
**Scope**: Adopt official A2A Protocol for agent-to-agent communication

---

## Executive Summary

### Critical Discovery

APS's specs/005-a2a-protocol/ created a custom "A2A Protocol" that shares the same name as an existing, official standard: **Agent2Agent (A2A) Protocol** from a2a-protocol.org (developed by Google, donated to Linux Foundation).

### Required Action

**Replace custom A2A protocol with official A2A Protocol adoption**.

### Benefits of Official A2A Adoption

- **Zero Protocol Engineering**: Official spec is battle-tested, production-ready
- **Ecosystem Interoperability**: APS agents can communicate with any A2A-compliant agent
- **Go SDK Available**: `github.com/a2aproject/a2a-go` (v0.3.4, actively maintained)
- **Enterprise-Grade**: Built-in authentication, streaming, async operations
- **Community Support**: Open source, Apache 2.0 license, active development

---

## Protocol Comparison

### Custom APS "A2A" Protocol

**Characteristics**:
- Custom JSON message format
- IPC/HTTP/WebSocket transports
- Conversation concept (duo/group)
- Message patterns: request, response, publish, subscribe
- Per-conversation ordering
- Local filesystem storage

**Issues**:
- Name collision with official standard
- Reinventing wheel (no ecosystem benefit)
- Maintenance burden on APS team
- No interoperability with other agents

### Official A2A Protocol

**Characteristics**:
- Task-based messaging with Agent Cards
- JSON-RPC 2.0, gRPC, HTTP+JSON/REST bindings
- Message lifecycle management (TaskStatus)
- Streaming and async operations
- Push notifications for long-running tasks
- Multi-turn conversations
- Enterprise authentication (OAuth2, mTLS, OpenID Connect)

**Benefits**:
- Industry standard (Google + Linux Foundation)
- Official Go SDK available
- Interoperability with A2A ecosystem
- Extensible via protocol extensions
- Well-documented with examples

---

## Updated Architecture

### Before (Custom Protocol)

```
┌─────────────────────────────────────────────────────┐
│         APS Custom A2A Protocol                  │
├─────────────────────────────────────────────────────┤
│  • Custom JSON message format                   │
│  • IPC/HTTP/WebSocket transports               │
│  • Conversation-based messaging                 │
│  • Request/response + pub/sub                 │
└─────────────────────────────────────────────────────┘
                    │
                    ▼
          ┌─────────────────┐
          │ APS Profiles  │
          └─────────────────┘
```

### After (Official A2A Protocol)

```
┌─────────────────────────────────────────────────────┐
│         Official A2A Protocol                   │
│    (a2a-protocol.org + a2a-go SDK)           │
├─────────────────────────────────────────────────────┤
│  • Task-based messaging                        │
│  • JSON-RPC 2.0 / gRPC / HTTP+JSON bindings │
│  • Agent Cards for discovery                  │
│  • Streaming + async operations               │
│  • Enterprise authentication                   │
└─────────────────────────────────────────────────────┘
                    │
         ┌──────────┼──────────┐
         │          │          │
         ▼          ▼          ▼
    ┌────────┐ ┌────────┐ ┌────────┐
    │Profile │ │Profile │ │External│
    │  (APS) │ │  (APS) │ │  A2A   │
    │        │ │        │ │ Agents │
    └────────┘ └────────┘ └────────┘
```

---

## Migration Plan

### Phase 1: Protocol Adoption (Week 1)

**Objective**: Replace custom spec with official A2A protocol reference

**Tasks**:
1. ✅ Identify official A2A protocol
2. ⏳ Archive custom protocol documentation
3. ⏳ Create adoption guide referencing official spec
4. ⏳ Update all references from "our A2A" to "official A2A"

**Deliverables**:
- [ ] A2A Protocol Adoption Guide (this document)
- [ ] Archived custom protocol in `legacy/` directory
- [ ] Updated plan.md referencing official A2A
- [ ] Updated research.md with official protocol findings

---

### Phase 2: SDK Integration (Week 2-3)

**Objective**: Integrate `github.com/a2aproject/a2a-go` SDK

**Tasks**:
1. ⏳ Add `a2a-go` to `go.mod`
2. ⏳ Create APS-specific A2A Server implementation
3. ⏳ Map APS profiles → A2A agents
4. ⏳ Implement Agent Card generation for APS profiles
5. ⏳ Add A2A Client for profile-to-profile communication

**Code Structure**:
```
internal/
  a2a/
    server.go          # A2A server implementation using a2asrv
    client.go          # A2A client implementation using a2aclient
    agentcard.go       # APS profile → Agent Card mapping
    transport/         # Transport adapters for APS isolation
      ipc.go           # IPC transport wrapper
      http.go          # HTTP transport wrapper
      websocket.go     # WebSocket transport wrapper
```

**Deliverables**:
- [ ] A2A server component using `a2asrv` package
- [ ] A2A client component using `a2aclient` package
- [ ] Profile → Agent Card mapping layer
- [ ] Transport adapters for APS isolation tiers

---

### Phase 3: Isolation Integration (Week 4)

**Objective**: Adapt A2A to work across APS isolation tiers

**Tasks**:
1. ⏳ Implement IPC transport using A2A over filesystem queues
2. ⏳ Expose A2A HTTP/gRPC endpoints for network communication
3. ⏳ Configure Agent Cards with isolation-aware capabilities
4. ⏳ Add authentication/authorization per isolation tier
5. ⏳ Test A2A communication across isolation boundaries

**Isolation Mapping**:

| APS Isolation Tier | A2A Transport | Authentication |
|-------------------|----------------|----------------|
| Process (Tier 1) | IPC (local) | Optional signatures |
| Platform (Tier 2) | HTTP/gRPC | mTLS or API keys |
| Container (Tier 3) | HTTP/gRPC | mTLS + container auth |

**Deliverables**:
- [ ] IPC transport for A2A (Tier 1)
- [ ] HTTP/gRPC transport for A2A (Tier 2/3)
- [ ] Isolation-aware authentication layer
- [ ] Cross-tier communication tests

---

### Phase 4: CLI Integration (Week 5)

**Objective**: Update CLI commands to use official A2A

**Tasks**:
1. ⏳ Replace custom `aps a2a` commands with A2A SDK calls
2. ⏳ Update TUI to display A2A tasks instead of conversations
3. ⏳ Add Agent Card discovery commands
4. ⏳ Integrate A2A streaming responses
5. ⏳ Update documentation to reference official A2A

**CLI Command Mapping**:

| Old (Custom) | New (Official A2A) |
|---------------|-------------------|
| `aps a2a start-duo` | Create new task via A2A Client |
| `aps a2a list-conversations` | `aps a2a tasks list` |
| `aps a2a show-conversation` | `aps a2a tasks show` |
| `aps a2a send` | Client.SendMessage() |
| `aps a2a subscribe` | Client.SendMessageStream() |

**Deliverables**:
- [ ] Updated CLI commands using A2A SDK
- [ ] TUI integration with A2A task display
- [ ] Agent Card discovery commands
- [ ] Streaming response support

---

### Phase 5: Testing & Validation (Week 6)

**Objective**: Ensure A2A adoption meets all requirements

**Tasks**:
1. ⏳ Unit tests for A2A server/client integration
2. ⏳ Integration tests for isolation tiers
3. ⏳ Interoperability tests with other A2A agents
4. ⏳ Performance benchmarks
5. ⏳ Security audit

**Test Scenarios**:
- [ ] Profile A creates task for Profile B (same machine, IPC)
- [ ] Profile A creates task for Profile B (network, HTTP)
- [ ] Profile A streams task updates to Profile B
- [ ] APS profile communicates with external A2A agent
- [ ] Authentication/authorization per isolation tier
- [ ] Long-running task with push notifications

**Deliverables**:
- [ ] Comprehensive test suite
- [ ] Interoperability validation
- [ ] Performance benchmarks
- [ ] Security audit report

---

## Updated Data Model Mapping

### Concepts Mapping

| Custom APS Concept | Official A2A Concept |
|--------------------|---------------------|
| Conversation | Task |
| Message | Message |
| Duo/Group | Multi-turn Task |
| Participant | Participant profiles |
| Conversation ID | Task ID |
| Message pattern | Task lifecycle |
| Pub/Sub topic | Subscription context |

### Storage Mapping

**Custom Storage**:
```
~/.agents/conversations/
  <conv-id>/
    meta.json
    messages/
      <timestamp>_<msg-id>.json
```

**A2A Storage** (via SDK):
```
~/.agents/a2a/
  tasks/
    <task-id>/           # Task metadata
      meta.json
      messages/            # Message history
        <timestamp>.json
      artifacts/            # Task outputs
```

---

## Updated System Architecture

### Component Overview

1. **A2A Server** (`internal/a2a/server.go`)
   - Uses `github.com/a2aproject/a2a-go/a2asrv`
   - Exposes APS profiles as A2A agents
   - Implements Agent Cards for discovery

2. **A2A Client** (`internal/a2a/client.go`)
   - Uses `github.com/a2aproject/a2a-go/a2aclient`
   - Enables APS profiles to initiate A2A tasks
   - Supports streaming and async operations

3. **Agent Card Manager** (`internal/a2a/agentcard.go`)
   - Generates Agent Cards from APS profile configs
   - Declares A2A capabilities
   - Specifies authentication requirements

4. **Transport Layer** (`internal/a2a/transport/`)
   - IPC adapter for local communication
   - HTTP/gRPC adapter for network communication
   - WebSocket adapter for real-time streaming

5. **Isolation Bridge** (`internal/a2a/isolation.go`)
   - Maps APS isolation tiers to A2A security schemes
   - Configures authentication per tier
   - Enforces isolation boundaries

---

## Security Model

### Official A2A Security Features

1. **Authentication**
   - OpenID Connect (OAuth2)
   - API Keys
   - HTTP Basic Auth
   - Mutual TLS (mTLS)

2. **Authorization**
   - Agent Card-based capability declaration
   - Extension-based permissions
   - Context-based access control

3. **Transport Security**
   - HTTPS/TLS required for network
   - mTLS for high-security scenarios
   - Optional for local IPC

4. **Message Security**
   - Signatures via extension mechanism
   - End-to-end encryption (custom)

### APS Isolation Mapping

| APS Isolation Tier | A2A Security Scheme | Implementation |
|-------------------|----------------------|------------------|
| Process (Tier 1) | Optional signatures | HMAC-SHA256 per profile |
| Platform (Tier 2) | API Keys or mTLS | Shared secret or cert |
| Container (Tier 3) | mTLS | Container identity + cert |

---

## Updated CLI Commands

### Profile Commands

```bash
# Create profile with A2A enabled (unchanged)
aps profile create agent-a --display-name "Agent A" --enable-a2a

# List A2A-enabled profiles
aps a2a list-profiles

# Show profile's Agent Card
aps a2a show-agent-card agent-a

# Register profile for discovery (network)
aps a2a register --endpoint http://10.0.0.1:8080

# Discover A2A agents on network
aps a2a discover --network 192.168.1.0/24
```

### Task Commands

```bash
# Create task (request/response)
aps a2a tasks send agent-b \
  --type query \
  --payload '{"query": "status"}'

# Create streaming task
aps a2a tasks stream agent-b \
  --type task \
  --payload '{"command": "deploy"}'

# List tasks
aps a2a tasks list --profile agent-a --status working

# Get task details
aps a2a tasks show <task-id>

# Cancel task
aps a2a tasks cancel <task-id>

# Subscribe to task updates (push notifications)
aps a2a tasks subscribe <task-id> --webhook http://localhost:9000/hook
```

### Agent Card Commands

```bash
# Fetch Agent Card
aps a2a fetch-agent-card http://10.0.0.1:8080/.well-known/agent-card

# Verify Agent Card signature
aps a2a verify-agent-card <card-file>

# Display Agent Card capabilities
aps a2a show-capabilities <card-file>
```

---

## Migration Impact Analysis

### Breaking Changes

1. **Message Format**: Custom JSON → A2A Message (Protocol Buffer schema)
2. **Storage Structure**: Conversations → Tasks
3. **CLI Commands**: Some command names change
4. **Configuration**: Profile A2A settings need mapping to Agent Cards

### Non-Breaking Changes

1. **Core Concepts**: Profiles remain profiles, isolation unchanged
2. **Isolation Tiers**: Tier model preserved
3. **Security Model**: Per-profile settings retained
4. **TUI UX**: Similar conversation/task viewing experience

### Migration Path

**For Existing Users**:

1. Old conversations remain accessible in legacy storage
2. New tasks use official A2A protocol
3. CLI provides backward compatibility layer
4. Migration tool to convert old conversations to A2A tasks (optional)

**For New Users**:

1. Direct adoption of official A2A
2. No legacy compatibility concerns
3. Full interoperability from day one

---

## Success Criteria

### Functional Requirements

- ✅ APS profiles communicate via official A2A protocol
- ✅ APS profiles can interoperate with external A2A agents
- ✅ Supports all isolation tiers (process, platform, container)
- ✅ Streaming and async operations functional
- ✅ Agent Card discovery working
- ✅ Enterprise authentication supported

### Non-Functional Requirements

- ✅ Message latency < 100ms for local IPC
- ✅ Message latency < 500ms for network HTTP
- ✅ Throughput > 1000 messages/second
- ✅ 99.9% availability
- ✅ Comprehensive test coverage (>90%)

### Interoperability Requirements

- ✅ APS agents can discover external A2A agents
- ✅ External A2A agents can discover APS profiles
- ✅ Task delegation between APS and external agents
- ✅ Multi-turn conversations work across implementations
- ✅ Streaming responses functional

---

## Open Questions

### Technical Questions

1. **Legacy Data Migration**
   - **Question**: Should old conversations be migrated to A2A tasks?
   - **Recommendation**: Keep read-only access, don't auto-migrate
   - **Rationale**: Different protocols, risk of data loss, user choice

2. **Custom Transport Extensions**
   - **Question**: Should APS expose custom IPC transport via A2A extensions?
   - **Recommendation**: Use A2A's extensibility mechanism for custom transports
   - **Rationale**: Maintain A2A compliance while adding APS-specific features

3. **Agent Card Signing**
   - **Question**: Should APS Agent Cards be signed?
   - **Recommendation**: Optional signing, enforce for network discovery
   - **Rationale**: Balance security vs. usability

### Architectural Questions

1. **Dual Protocol Support**
   - **Question**: Support both custom and official A2A during transition?
   - **Recommendation**: No, direct replacement to avoid confusion
   - **Rationale**: Simpler codebase, clear migration path

2. **External Agent Integration**
   - **Question**: Should APS ship with pre-registered external A2A agents?
   - **Recommendation**: No, user-controlled discovery only
   - **Rationale**: Security, user autonomy, avoid vendor lock-in

---

## Risks and Mitigations

### Technical Risks

1. **SDK Learning Curve**
   - **Risk**: Team unfamiliar with `a2a-go` SDK
   - **Mitigation**: Training, documentation, sample implementations
   - **Timeline**: Week 2-3

2. **Isolation Compatibility**
   - **Risk**: A2A doesn't map perfectly to APS isolation tiers
   - **Mitigation**: Extensibility mechanism, custom transports
   - **Timeline**: Week 4

3. **Performance Overhead**
   - **Risk**: Official A2A has higher overhead than custom protocol
   - **Mitigation**: Benchmark, optimize, profile-specific tuning
   - **Timeline**: Week 6

### Operational Risks

1. **User Confusion**
   - **Risk**: Users confused by protocol change
   - **Mitigation**: Clear communication, migration guide, CLI compatibility
   - **Timeline**: Week 5-6

2. **Breaking Changes**
   - **Risk**: Existing integrations break
   - **Mitigation**: Deprecation period, backward compatibility layer
   - **Timeline**: Week 5-6

3. **External Dependencies**
   - **Risk**: Reliance on external A2A SDK
   - **Mitigation**: Vendor support, Apache 2.0 license, fork capability
   - **Timeline**: Ongoing

---

## Timeline (Revised)

### Phase 1: Protocol Adoption (Week 1)
- Day 1-2: Archive custom protocol, create adoption guide
- Day 3-4: Update documentation (plan.md, research.md, decisions.md)
- Day 5: Review and finalize architecture

### Phase 2: SDK Integration (Week 2-3)
- Week 2: Add SDK, implement server/client
- Week 3: Implement Agent Cards, mapping layer

### Phase 3: Isolation Integration (Week 4)
- Day 1-2: IPC transport for Tier 1
- Day 3-4: HTTP/gRPC for Tier 2/3
- Day 5: Cross-tier testing

### Phase 4: CLI Integration (Week 5)
- Day 1-2: Update CLI commands
- Day 3-4: TUI integration
- Day 5: Streaming support

### Phase 5: Testing & Validation (Week 6)
- Day 1-2: Unit and integration tests
- Day 3: Interoperability tests
- Day 4: Performance benchmarks
- Day 5: Security audit

**Total Timeline**: 6 weeks (1.5 months)

---

## References

### Official A2A Protocol
- **Specification**: https://a2a-protocol.org/latest/specification/
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **Documentation**: https://a2a-protocol.org/latest/
- **GitHub**: https://github.com/a2aproject/A2A

### APS Components
- Profile System (`internal/core/profile.go`)
- Isolation Architecture (`specs/001-build-cli-core/isolation-architecture.md`)
- Configuration Management (`internal/core/config.go`)

### Related Protocols
- **ACP**: https://agentclientprotocol.com (Editor ↔ Agent)
- **MCP**: https://modelcontextprotocol.io (Agent ↔ Tools)

---

## Summary

**Critical Change Required**: Replace custom APS A2A protocol with official A2A Protocol adoption.

**Benefits**:
- Zero protocol engineering
- Ecosystem interoperability
- Official Go SDK (battle-tested)
- Enterprise-grade features
- Community support

**Timeline**: 6 weeks for complete migration

**Risk**: Low - Official protocol is well-documented, actively maintained, and designed for exactly this use case.

**Next Steps**: Proceed with Phase 1 (Protocol Adoption) immediately.
