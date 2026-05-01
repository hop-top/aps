# A2A Protocol - Decisions and Resolved Questions

**Date**: 2026-01-20
**Updated**: 2026-01-21
**Status**: CRITICAL DECISION - Adopt Official A2A Protocol

---

## Overview

This document summarizes all decisions for APS A2A protocol adoption. **Critical decision**: Replace custom APS A2A protocol with official A2A Protocol from https://a2a-protocol.org.

---

## Critical Decision: Protocol Choice

### Decision: **ADOPT OFFICIAL A2A PROTOCOL**

**Question**: Should APS create custom protocol or adopt official standard?

**Decision**: **Adopt official A2A Protocol**

**Rationale**:
- **Name Collision**: Custom protocol shares name with existing standard
- **Zero Engineering**: Official protocol is battle-tested, production-ready
- **Ecosystem**: Interoperability with A2A-compliant agents
- **Go SDK**: Official `github.com/a2aproject/a2a-go` SDK available
- **Enterprise-Grade**: Built-in authentication, streaming, async operations
- **Community**: Open source, Apache 2.0, active development

**Impact**:
- Replace all custom protocol specifications with official A2A references
- Integrate `a2a-go` SDK into APS codebase
- Update CLI commands to use A2A task model instead of custom conversations
- Map APS profiles → A2A agents via Agent Cards
- Adapt APS isolation tiers to A2A security schemes

**Timeline**: 6 weeks for complete migration (see `plan.md`)

**Reference**: Official A2A spec at https://a2a-protocol.org/latest/specification/

---

## Resolved Technical Questions

### 1. Protocol Adoption ✅

**Question**: Custom APS A2A vs. official A2A Protocol?

**Decision**: **Official A2A Protocol**

**Rationale**:
- Custom protocol: Maintenance burden, no ecosystem, name collision
- Official protocol: Zero engineering, ecosystem benefits, Go SDK available

**Implementation**:
- Archive custom protocol in `legacy/` directory
- Add `github.com/a2aproject/a2a-go` dependency
- Implement A2A Server using `a2asrv` package
- Implement A2A Client using `a2aclient` package
- Map APS profiles to A2A agents via Agent Cards

**Alternative Not Chosen**:
- Custom protocol: ❌ Reinventing wheel, no interoperability

**Reference**: `plan.md` - Complete adoption plan

---

### 2. Message Format ✅

**Question**: Custom JSON message format vs. A2A Message schema?

**Decision**: **A2A Message (Protocol Buffer schema)**

**Rationale**:
- A2A has well-defined message schema from Protocol Buffers
- Supports multiple content types (TextPart, FilePart, DataPart)
- Task lifecycle built into protocol
- JSON-RPC, gRPC, and HTTP+JSON/REST bindings available

**A2A Message Structure**:
```protobuf
message Message {
  string id = 1;                           // UUID v4
  repeated Part parts = 2;                  // Content parts
  repeated Role roles = 3;                  // Message roles
  // ... additional fields per A2A spec
}
```

**Implementation**:
- Use `a2a-go` SDK's message types
- No custom serialization needed
- Automatic versioning support

**Custom Format Not Chosen**:
- APS custom JSON: ❌ Non-standard, no ecosystem

**Reference**: https://a2a-protocol.org/latest/specification/#4-protocol-data-model

---

### 3. Conversation Model ✅

**Question**: Custom conversations (duo/group) vs. A2A Tasks?

**Decision**: **A2A Tasks with multi-turn conversations**

**Rationale**:
- A2A Tasks are central abstraction with full lifecycle
- Supports multi-turn conversations via message history
- TaskStatus tracks state (submitted, working, completed, failed, cancelled)
- Streaming and async operations built-in
- Agent Cards define capabilities for task execution

**Task Structure**:
```protobuf
message Task {
  string id = 1;                           // Task ID
  string context_id = 2;                    // Optional context grouping
  repeated string participant_ids = 3;         // Participant profiles
  repeated Message messages = 4;             // Message history
  TaskStatus status = 5;                     // Current status
  repeated Artifact artifacts = 6;             // Task outputs
  // ... additional fields per A2A spec
}
```

**Mapping**:
- APS Conversation → A2A Task
- APS Duo/Group → Multi-turn Task with participants
- APS Conversation History → Task Messages
- APS Conversation ID → Task ID

**Custom Model Not Chosen**:
- APS custom conversations: ❌ Non-standard, no ecosystem

**Reference**: https://a2a-protocol.org/latest/specification/#411-task

---

### 4. Communication Patterns ✅

**Question**: Custom patterns (request, response, publish, subscribe) vs. A2A task lifecycle?

**Decision**: **A2A task operations (SendMessage, SendMessageStream)**

**Rationale**:
- A2A `SendMessage`: Create task, return direct response or task
- A2A `SendMessageStream`: Real-time updates via streaming
- A2A `GetTask`: Retrieve task state and history
- A2A `ListTasks`: Query tasks with filters
- A2A `CancelTask`: Stop long-running operations

**Operations**:
```
SendMessage:     Request/response, creates Task
SendMessageStream: Streaming updates for real-time feedback
GetTask:         Fetch task state and history
ListTasks:        Query tasks by status, context, etc.
CancelTask:       Cancel long-running operation
SubscribeToTask:  Receive push notifications
```

**Pattern Mapping**:
- APS `request/response` → A2A `SendMessage`
- APS `publish/subscribe` → A2A `SendMessage` with topic context
- APS `conversation` → A2A multi-turn Task via `SendMessageStream`

**Custom Patterns Not Chosen**:
- APS custom patterns: ❌ Non-standard, limited functionality

**Reference**: https://a2a-protocol.org/latest/specification/#31-core-operations

---

### 5. Transport Bindings ✅

**Question**: Custom IPC/HTTP/WebSocket vs. A2A protocol bindings?

**Decision**: **A2A official bindings + custom IPC via extensions**

**Rationale**:
- A2A provides JSON-RPC 2.0, gRPC, HTTP+JSON/REST bindings
- Multiple bindings available out-of-the-box
- Custom transport via A2A extensibility mechanism
- Profile config can specify preferred binding

**Available Bindings**:
- **JSON-RPC 2.0**: Simple, widely adopted
- **gRPC**: High performance, streaming, strong typing
- **HTTP+JSON/REST**: Standard, firewall-friendly

**Custom IPC Transport**:
- Implement via A2A extensions mechanism
- Maintain APS filesystem-based queues
- Work with all isolation tiers

**Transport Selection**:
- Profile config specifies preferred binding
- Fallback between bindings
- Per-message override possible

**Custom Transports Not Chosen**:
- APS custom IPC/HTTP/WebSocket: ❌ Non-standard, no ecosystem

**Reference**: https://a2a-protocol.org/latest/specification/#5-protocol-binding-requirements

---

### 6. Agent Discovery ✅

**Question**: Custom profile registry vs. A2A Agent Cards?

**Decision**: **A2A Agent Cards for discovery**

**Rationale**:
- Agent Cards are A2A's discovery mechanism
- JSON-based, signed, declarative
- Published at `/.well-known/agent-card`
- Contains capabilities, skills, interface, auth requirements

**Agent Card Structure**:
```json
{
  "agentProvider": {
    "name": "APS",
    "version": "1.0.0"
  },
  "agentCapabilities": {
    "supportedInterfaces": ["jsonrpc", "grpc"],
    "extensions": ["aps-isolation"]
  },
  "agentSkills": [...],
  "agentInterfaces": [...],
  "securitySchemes": [...]
}
```

**APS Profile → Agent Card**:
- Profile ID → Agent ID
- Profile capabilities → Agent Skills
- Isolation tier → Security Scheme
- A2A config → Supported Interfaces

**Custom Registry Not Chosen**:
- APS custom registry: ❌ Non-standard, limited discovery

**Reference**: https://a2a-protocol.org/latest/specification/#8-agent-discovery-the-agent-card

---

### 7. Authentication ✅

**Question**: Custom HMAC-SHA256 vs. A2A authentication schemes?

**Decision**: **A2A standard authentication schemes**

**Rationale**:
- A2A supports OpenID Connect (OAuth2), API Keys, HTTP Basic, mTLS
- Enterprise-grade authentication out-of-the-box
- Agent Cards declare supported schemes
- No custom authentication implementation needed

**Supported Schemes**:
- **OpenID Connect**: Web-based OAuth2 flow
- **API Key**: Simple token-based
- **HTTP Basic**: Username/password
- **Mutual TLS**: Certificate-based

**Isolation Tier Mapping**:
- **Process (Tier 1)**: Optional authentication, API Key
- **Platform (Tier 2)**: mTLS or API Key
- **Container (Tier 3)**: mTLS required

**Custom Auth Not Chosen**:
- APS custom HMAC-SHA256: ❌ Non-standard, limited use cases

**Reference**: https://a2a-protocol.org/latest/specification/#7-authentication-and-authorization

---

### 8. Storage Backend ✅

**Question**: Custom filesystem storage vs. A2A storage interface?

**Decision**: **Custom APS storage using A2A task lifecycle**

**Rationale**:
- A2A doesn't prescribe storage backend
- APS maintains filesystem-based storage model
- Implement A2A Task lifecycle over APS storage
- Preserve existing APS architecture

**Storage Model**:
- APS filesystem storage: `~/.agents/a2a/tasks/`
- Task metadata: `meta.json`
- Message history: `messages/<timestamp>.json`
- Artifacts: `artifacts/<artifact-id>/`

**A2A Integration**:
- Implement A2A Task lifecycle
- Store A2A Messages in APS format
- Expose storage via A2A Server interface

**Custom Storage Not Chosen**:
- Change to A2A-prescribed storage: ❌ Breaks APS architecture

**Reference**: A2A storage is implementation choice

---

### 9. CLI Commands ✅

**Question**: Keep custom `aps a2a` commands vs. A2A SDK integration?

**Decision**: **Update CLI to use A2A SDK**

**Rationale**:
- Maintain CLI UX (user shouldn't see protocol change)
- Internally use A2A SDK for operations
- Map CLI commands to A2A operations

**Command Mapping**:
| Old (Custom) | New (A2A SDK) | A2A Operation |
|---------------|------------------|----------------|
| `aps a2a start-duo` | `aps a2a create-task` | `SendMessage` |
| `aps a2a list-conversations` | `aps a2a tasks list` | `ListTasks` |
| `aps a2a show-conversation` | `aps a2a tasks show` | `GetTask` |
| `aps a2a send` | `aps a2a tasks send` | `SendMessage` |
| `aps a2a subscribe` | `aps a2a tasks subscribe` | `SubscribeToTask` |
| `aps a2a register` | `aps a2a publish-card` | Agent Card |

**Custom CLI Not Chosen**:
- Keep custom CLI with custom protocol: ❌ No ecosystem benefit

---

### 10. Legacy Data Migration ✅

**Question**: Migrate existing custom conversations to A2A tasks?

**Decision**: **Keep legacy data read-only, no auto-migration**

**Rationale**:
- Different protocols (custom JSON vs A2A schema)
- Risk of data loss during migration
- Users should control migration
- New tasks use A2A from day 1

**Migration Path**:
- Legacy conversations in `legacy/conversations/` (read-only)
- New A2A tasks in `~/.agents/a2a/tasks/`
- Optional migration tool provided (user-controlled)
- Backward compatibility layer in CLI

**Auto-Migration Not Chosen**:
- Automatic migration: ❌ Risk of data loss, user control

---

## Architectural Decisions

### A2A Server Architecture

**Decision**: **Use `a2asrv` package with APS customizations**

**Components**:
- **A2A Server**: `github.com/a2aproject/a2a-go/a2asrv`
- **APS Agent Executor**: Custom executor mapping profiles → agents
- **APS Transport Layer**: Custom IPC transport via A2A extensions
- **APS Storage Backend**: Filesystem storage implementing A2A lifecycle

**Implementation**:
```go
// internal/a2a/server.go
func NewAPAServer(profile *core.Profile) (*a2asrv.Server) {
    executor := newAPSProfileExecutor(profile)
    handler := a2asrv.NewHandler(executor, options...)
    
    // Wrap in transports
    grpcHandler := a2agrpc.NewHandler(handler)
    jsonrpcHandler := a2asrv.NewJSONRPCHandler(handler)
    
    return grpcHandler  // or jsonrpcHandler
}
```

---

### A2A Client Architecture

**Decision**: **Use `a2aclient` package with APS customizations**

**Components**:
- **A2A Client**: `github.com/a2aproject/a2a-go/a2aclient`
- **APS Agent Card Resolver**: Custom resolver for APS profiles
- **APS Transport Selection**: Prefer IPC for local, fallback to HTTP/gRPC

**Implementation**:
```go
// internal/a2a/client.go
func NewAPSClient(targetProfileID string) (*a2aclient.Client, error) {
    // Resolve Agent Card
    card := resolveAPSProfileCard(targetProfileID)
    
    // Create client from card
    client, err := a2aclient.NewFromCard(ctx, card, options...)
    
    return client, err
}
```

---

### Isolation Integration Architecture

**Decision**: **Map APS isolation tiers to A2A security schemes**

**Mapping Table**:

| APS Isolation Tier | A2A Security Scheme | Transport | Auth |
|-------------------|----------------------|-----------|-------|
| Process (Tier 1) | API Key (optional) | IPC (custom) | HMAC-SHA256 |
| Platform (Tier 2) | mTLS or API Key | HTTP/gRPC | Shared secret |
| Container (Tier 3) | mTLS (required) | HTTP/gRPC | Container identity |

**Implementation**:
- Custom IPC transport for Tier 1 (filesystem queues)
- HTTP/gRPC for Tier 2/3 (network communication)
- Agent Cards declare security scheme per tier
- Enforce isolation boundaries at transport layer

---

## Design Principles

### 1. Adopt, Don't Invent

**Approach**: Use official standards when available

**Examples**:
- A2A Protocol (not custom A2A)
- ACP (Agent Client Protocol) for editor integration
- MCP (Model Context Protocol) for tools

**Benefit**: Zero engineering, ecosystem benefits, community support

---

### 2. Map, Don't Break

**Approach**: Preserve APS architecture while adopting A2A

**Examples**:
- Profiles remain profiles (not agents)
- Filesystem storage preserved (not replaced)
- Isolation tiers maintained (not changed)
- CLI UX preserved (internal A2A integration)

**Benefit**: Smooth migration, user-friendly, maintainable

---

### 3. Extend, Don't Fork

**Approach**: Use A2A extensibility mechanism for custom features

**Examples**:
- Custom IPC transport via A2A extensions
- APS-specific capabilities in Agent Cards
- Custom storage backend implementing A2A lifecycle

**Benefit**: Maintain A2A compliance while adding APS-specific features

---

### 4. Interoperate, Don't Isolate

**Approach**: Enable APS agents to communicate with any A2A agent

**Examples**:
- APS profiles discover external A2A agents
- External agents discover APS profiles via Agent Cards
- Task delegation across implementations
- Multi-turn conversations across agents

**Benefit**: Ecosystem participation, not siloed solution

---

## Not Chosen Alternatives

### Custom APS A2A Protocol ❌

**Reason**: Name collision, no ecosystem, maintenance burden

**Chosen Instead**: Adopt official A2A Protocol

---

### Keep Custom Transport Implementation ❌

**Reason**: Non-standard, limited to APS

**Chosen Instead**: A2A bindings + custom transport via extensions

---

### Auto-Migrate Legacy Data ❌

**Reason**: Risk of data loss, protocol incompatibility

**Chosen Instead**: Keep legacy read-only, user-controlled migration

---

### Dual Protocol Support (Custom + A2A) ❌

**Reason**: Increased complexity, unclear migration path

**Chosen Instead**: Direct replacement with deprecation period

---

## Future Considerations

### v1.0 Features (Immediate)

- A2A basic operations (SendMessage, GetTask, ListTasks)
- JSON-RPC 2.0 and gRPC bindings
- Agent Card discovery
- Basic authentication (API Key, HTTP Basic)

### v0.3.4+ Features (Current)

- Streaming updates (SendMessageStream)
- Push notifications (SubscribeToTask)
- Enterprise authentication (OpenID Connect, mTLS)
- Protocol extensions support

### Future Enhancements

- Custom IPC transport optimization
- APS-specific extensions (isolation awareness)
- Enhanced Agent Card capabilities
- Advanced security schemes

---

## References

- **Official A2A Spec**: https://a2a-protocol.org/latest/specification/
- **A2A Go SDK**: https://github.com/a2aproject/a2a-go
- **Adoption Plan**: `plan.md`
- **Research**: `research.md`

---

## Summary

**Critical Decision**: Adopt official A2A Protocol instead of creating custom APS A2A.

**All Technical Questions Resolved**:
1. ✅ Protocol choice: Official A2A
2. ✅ Message format: A2A Message schema
3. ✅ Conversation model: A2A Tasks
4. ✅ Communication patterns: A2A operations
5. ✅ Transport bindings: A2A bindings + extensions
6. ✅ Agent discovery: Agent Cards
7. ✅ Authentication: A2A security schemes
8. ✅ Storage backend: Custom APS storage with A2A lifecycle
9. ✅ CLI commands: A2A SDK integration
10. ✅ Legacy data: Read-only, no auto-migration

**Next Steps**: Implement per `plan.md` adoption plan (6 weeks).
