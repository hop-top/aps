# A2A Protocol Research

**Date**: 2026-01-20
**Updated**: 2026-01-21
**Purpose**: Research existing Agent-to-Agent communication protocols and standards

---

## Overview

This document explores existing agent communication protocols, patterns, and best practices. **Critical discovery**: Official Agent2Agent (A2A) Protocol exists at https://a2a-protocol.org.

---

## Critical Discovery: Official A2A Protocol

### A2A Protocol Overview

**Source**: https://a2a-protocol.org
**Developed by**: Google (donated to Linux Foundation)
**Status**: Production-ready (v0.3.4 released Jan 14, 2026)
**License**: Apache 2.0

**Purpose**: Open standard for agent-to-agent communication, enabling interoperability between agents built on different frameworks (LangGraph, CrewAI, Semantic Kernel, etc.).

**Key Features**:
- **Task-based messaging**: Central to protocol, with full lifecycle management
- **Agent Cards**: Discovery mechanism for agent capabilities
- **Multi-protocol bindings**: JSON-RPC 2.0, gRPC, HTTP+JSON/REST
- **Streaming & Async**: Native support for long-running operations
- **Enterprise-ready**: Authentication, authorization, security, tracing
- **Extensibility**: Custom extensions and capability declarations

**Official SDKs**:
- Go: https://github.com/a2aproject/a2a-go ✅ **Relevant to APS**
- Python: https://github.com/a2aproject/a2a-python
- JavaScript: https://github.com/a2aproject/a2a-js
- Java: https://github.com/a2aproject/a2a-java
- C#/.NET: https://github.com/a2aproject/a2a-dotnet

**Decision**: **Adopt official A2A Protocol** instead of creating custom protocol.

### Why Official A2A?

**Interoperability**:
- APS agents can communicate with any A2A-compliant agent
- Ecosystem benefits from standard, not siloed solution
- Google + Linux Foundation backing ensures long-term viability

**Zero Protocol Engineering**:
- Battle-tested specification with real-world usage
- Comprehensive documentation and examples
- Active community support and maintenance

**Go SDK Available**:
- `github.com/a2aproject/a2a-go` is production-ready
- Aligns with APS's Go implementation
- Well-documented with pkg.go.dev reference

**Enterprise-Grade**:
- Built-in authentication (OAuth2, mTLS, OpenID Connect)
- Authorization and security model
- Observability and tracing support

**Complementary to Other Standards**:
- ACP (Agent Client Protocol): Editor ↔ Agent communication
- MCP (Model Context Protocol): Agent ↔ Tools/Data communication
- A2A (Agent2Agent Protocol): Agent ↔ Agent communication

**Stack Integration**:
```
User ↔ Editor (ACP)
         ↓
      Agent (MCP ← Tools/Data)
         ↓
      Other Agents (A2A)
```

---

## Existing Agent Communication Protocols

### 1. Agent Communication Language (ACL)

**Standard**: FIPA (Foundation for Intelligent Physical Agents)

**Description**:
- Standard for agent communication in multi-agent systems
- Defines message formats, interaction protocols, and semantics

**Message Format**:
```
performative: request
sender: agent-a
receiver: agent-b
content: "Execute task X"
language: SL0
ontology: task-ontology
```

**Performatives**: request, query, inform, subscribe, etc.

**Strengths**:
- Well-established standard
- Formal semantics
- Rich interaction protocols

**Weaknesses**:
- Complex to implement
- Not designed for modern cloud environments
- Overkill for simple agent communication

**Relevance**: Use for message semantics (request, query, inform), simplify implementation

---

### 2. JSON-RPC

**Standard**: JSON-RPC 2.0

**Description**:
- Lightweight remote procedure call protocol using JSON
- Request-response pattern with batch support

**Message Format**:
```json
{
  "jsonrpc": "2.0",
  "method": "runTask",
  "params": {"command": "deploy"},
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {"status": "success"},
  "id": 1
}
```

**Strengths**:
- Simple, well-defined
- Widely adopted
- Batch request support
- Notification support (no response required)

**Weaknesses**:
- Only request-response
- No pub/sub
- No conversation concept

**Relevance**: Official A2A uses JSON-RPC 2.0 as one of its bindings

---

### 3. MQTT

**Standard**: ISO/IEC 20922

**Description**:
- Lightweight publish/subscribe messaging protocol
- Designed for IoT and network-constrained environments

**Message Format**:
```
TOPIC: deployments
PAYLOAD: {"app": "app1", "version": "v1.0"}
QoS: 1
RETAIN: false
```

**Strengths**:
- Excellent pub/sub support
- Lightweight (small packet size)
- Quality of Service levels
- Built-in session persistence

**Weaknesses**:
- Requires broker (centralized)
- Not designed for agent-to-agent direct communication
- Limited semantic richness

**Relevance**: Use pub/sub concepts, but A2A provides decentralized alternative

---

### 4. AMQP

**Standard**: ISO/IEC 19464

**Description**:
- Advanced Message Queuing Protocol
- Supports multiple messaging patterns

**Message Format**:
```
Exchange: "tasks"
Routing Key: "deploy"
Body: {"command": "deploy"}
Properties: {"delivery-mode": 2}
```

**Strengths**:
- Rich feature set (exchanges, queues, routing)
- Reliability guarantees
- Supports multiple patterns

**Weaknesses**:
- Complex protocol
- Requires broker
- Heavy for simple use cases

**Relevance**: A2A provides similar functionality without broker requirement

---

### 5. gRPC

**Standard**: gRPC 1.x

**Description**:
- High-performance RPC framework using Protocol Buffers
- Supports streaming and bidirectional communication

**Message Format**:
```protobuf
service AgentService {
  rpc SendMessage (MessageRequest) returns (MessageResponse);
  rpc Subscribe (SubscribeRequest) returns (stream Event);
}
```

**Strengths**:
- High performance (binary, HTTP/2)
- Strong typing (Protocol Buffers)
- Bi-directional streaming
- Code generation

**Weaknesses**:
- Complex to implement
- Requires Protocol Buffers schema
- Not JSON-friendly
- Not suitable for filesystem-based IPC

**Relevance**: Official A2A includes gRPC binding for high-performance scenarios

---

### 6. WebSocket + JSON

**Standard**: RFC 6455 (WebSocket)

**Description**:
- Real-time bidirectional communication over HTTP
- Payload format is application-defined (JSON common)

**Message Format**:
```json
{
  "type": "request",
  "id": "req-123",
  "payload": {"command": "deploy"}
}
```

**Strengths**:
- Real-time bidirectional
- Simple to implement
- Works over web
- Payload-agnostic

**Weaknesses**:
- No built-in reliability
- Connection management required
- No pub/sub semantics (must be built on top)

**Relevance**: Official A2A supports HTTP+JSON/REST binding

---

## Communication Patterns

### Pattern 1: Request/Response (Synchronous)

**Description**: One agent sends request, waits for response

**Use Cases**:
- Task delegation with result
- Queries for information
- Remote procedure calls

**Implementation**:
- Correlation ID matches request to response
- Timeout handling
- Retry logic
- Error handling

**Examples**: JSON-RPC, gRPC, HTTP REST

**APS Application**: Worker/orchestrator pattern, queries for status

**A2A Implementation**: `SendMessage` creates Task, returns result directly

---

### Pattern 2: Publish/Subscribe (Asynchronous)

**Description**: One agent publishes, many subscribe

**Use Cases**:
- Event broadcasting
- Status updates
- Notifications
- Real-time monitoring

**Implementation**:
- Topic-based routing
- Subscription registry
- Wildcard subscriptions (optional)
- Message filtering

**Examples**: MQTT, Redis Pub/Sub, Kafka

**APS Application**: Deployment events, alerts, status broadcasts

**A2A Implementation**: Subscribe to topic, SendMessage with topic context

---

### Pattern 3: Conversation (Long-Running)

**Description**: Multi-turn communication with history tracking

**Use Cases**:
- Collaborative problem-solving
- Multi-step workflows
- Negotiations
- Threaded discussions

**Implementation**:
- Conversation metadata
- Message history
- Participant management
- Status tracking (active, archived, closed)

**Examples**: Chat applications, email threads

**APS Application**: Multi-agent project work, collaborative workflows

**A2A Implementation**: Multi-turn Task with message history

---

### Pattern 4: Pipeline/Chain

**Description**: Sequential message passing through agents

**Use Cases**:
- Processing pipelines
- Multi-stage workflows
- Data transformation

**Implementation**:
- Routing rules
- Forwarding logic
- Error propagation

**Examples**: Unix pipes, ETL pipelines

**APS Application**: Build pipelines, data processing workflows

**A2A Implementation**: Agent delegates sub-tasks via A2A messages

---

## Security Considerations

### 1. Authentication

**Methods**:
- **Pre-shared keys**: Simple but limited scalability
- **Public key cryptography**: More scalable (Ed25519, RSA)
- **Tokens (JWT)**: Stateless, expiration support
- **HMAC signatures**: Message integrity and authenticity

**A2A Implementation**:
- OpenID Connect (OAuth2) for web
- API Keys for service-to-service
- mTLS for high-security scenarios
- HTTP Basic Auth (simple)

**Recommendation**: Use A2A's built-in authentication schemes

---

### 2. Authorization

**Approaches**:
- **Allowlists**: Explicitly permitted senders
- **Blocklists**: Explicitly blocked senders
- **Capability-based**: Permissions attached to messages
- **Topic-based ACLs**: Access control per topic

**A2A Implementation**:
- Agent Cards declare capabilities
- Extensions for custom permissions
- Context-based access control

**Recommendation**: Use A2A's Agent Card capability model

---

### 3. Encryption

**Options**:
- **Transport encryption**: TLS for HTTP/WebSocket
- **End-to-end encryption**: Encrypt payload, only recipient can decrypt
- **Hybrid**: TLS for transport + optional E2E

**A2A Implementation**:
- HTTPS/TLS required for network
- mTLS for mutual authentication
- Custom extensions for E2E encryption

**Recommendation**: A2A's built-in TLS/mTLS support is sufficient

---

### 4. Message Integrity

**Techniques**:
- **HMAC signatures**: Detect tampering
- **Digital signatures**: Non-repudiation
- **Checksums**: Simple integrity checks

**A2A Implementation**:
- Custom extensions for signatures
- Agent Card signing for discovery
- Message verification optional

**Recommendation**: Use A2A's extensibility mechanism for custom signatures

---

## Transport Layer Research

### IPC (Inter-Process Communication)

**Mechanisms**:
- **Unix domain sockets**: Fast, filesystem-based
- **Named pipes**: Cross-platform, limited features
- **Shared memory**: Fastest, complex synchronization
- **Filesystem queues**: Simple, polling overhead

**APS Choice**: Filesystem queues (simplest, works with isolation tiers)

**A2A Consideration**: Custom transport via A2A extensibility mechanism

---

### HTTP

**Advantages**:
- Universal
- Firewall-friendly
- Easy debugging
- Well-understood

**Disadvantages**:
- Connection overhead
- Polling for events (long-polling workaround)

**A2A Implementation**: HTTP+JSON/REST binding available

---

### WebSocket

**Advantages**:
- Real-time bidirectional
- Low latency
- Efficient
- Works over HTTP

**Disadvantages**:
- Connection management complexity
- Stateful
- Connection loss handling

**A2A Implementation**: HTTP binding supports WebSocket for streaming

---

### gRPC

**Advantages**:
- High performance (binary)
- Streaming support
- Strong typing

**Disadvantages**:
- Requires Protocol Buffers
- Not filesystem-friendly
- More complex

**A2A Implementation**: Native gRPC binding available

---

## Storage Research

### Filesystem-Based Storage

**Advantages**:
- Simple
- No external dependencies
- Easy backup/restore
- Human-readable (JSON)

**Disadvantages**:
- Limited scalability
- Slower queries
- No built-in indexing

**APS Choice**: For v1.0, consider database for v2.0

**A2A Implementation**: Storage backend is implementation choice

---

### SQLite

**Advantages**:
- SQL queries
- Indexing
- ACID transactions
- Single-file database

**Disadvantages**:
- External dependency
- Not human-readable
- Concurrency limitations

**Consideration**: For large-scale deployments

---

### Redis

**Advantages**:
- In-memory (fast)
- Pub/sub support
- Data structures
- Persistence options

**Disadvantages**:
- External dependency
- Requires server process
- Not filesystem-based

**Consideration**: For high-throughput, network scenarios

---

## Best Practices

### 1. Message Design

**Do**:
- Use JSON for human-readability
- Include timestamps
- Use UUIDs for IDs
- Version protocol
- Keep messages small (< 1MB)

**Don't**:
- Send large payloads (use references instead)
- Embed secrets in messages
- Assume ordering guarantees
- Send unvalidated data

**A2A Alignment**: A2A follows all these best practices

---

### 2. Error Handling

**Do**:
- Define error response format
- Use standard error codes
- Include error details
- Log all errors
- Provide meaningful error messages

**Don't**:
- Hide error details from sender
- Ignore errors
- Assume success without confirmation

**A2A Alignment**: A2A has comprehensive error handling

---

### 3. Performance

**Do**:
- Batch small messages
- Use compression for large payloads
- Cache frequently accessed data
- Use connection pooling
- Implement backpressure

**Don't**:
- Send unnecessary messages
- Poll too frequently
- Create new connections for each message

**A2A Alignment**: A2A supports streaming, async, and push notifications

---

### 4. Security

**Do**:
- Sign all messages (configurable)
- Validate all inputs
- Use TLS for network transport
- Implement rate limiting
- Log security events

**Don't**:
- Trust senders without authentication
- Send secrets in plaintext
- Disable security for convenience

**A2A Alignment**: A2A has enterprise-grade security features

---

### 5. Reliability

**Do**:
- Implement retry logic with backoff
- Use message timeouts
- Track message delivery
- Store conversation history
- Handle connection failures gracefully

**Don't**:
- Assume network is reliable
- Send messages without timeout
- Ignore failed deliveries

**A2A Alignment**: A2A supports long-running operations and push notifications

---

## Lessons from Existing Systems

### 1. Use Existing Standards When Available

**Observation**: Custom protocols rarely achieve ecosystem adoption

**Lesson**: Adopt official standards when they exist

**APS Decision**: Adopt official A2A Protocol

---

### 2. Separate Transport from Semantics

**Observation**: Protocols that mix transport and semantics (AMQP) are harder to evolve

**Lesson**: Abstract transport layer

**A2A Approach**: Multiple protocol bindings (JSON-RPC, gRPC, HTTP)

---

### 3. Support Multiple Patterns

**Observation**: Single-pattern protocols (JSON-RPC) are limiting

**Lesson**: Support multiple communication patterns

**A2A Approach**: Request/response + streaming + multi-turn conversations

---

### 4. Provide Good Tooling

**Observation**: Protocols with good tooling (MQTT, gRPC) are more successful

**Lesson**: Build SDKs and documentation from day 1

**A2A Approach**: Official SDKs for 5 languages (Go, Python, JS, Java, C#)

---

### 5. Design for Evolution

**Observation**: Static protocols become obsolete

**Lesson**: Versioning and backward compatibility

**A2A Approach**: Semantic versioning, extensibility mechanism

---

## Recommendations for APS

### Protocol Choice

**Decision**: **Adopt official A2A Protocol**

**Rationale**:
- Zero protocol engineering effort
- Ecosystem interoperability
- Official Go SDK available
- Enterprise-grade features
- Community support and maintenance

---

### Implementation Approach

**Phase 1**: SDK Integration
- Add `github.com/a2aproject/a2a-go` dependency
- Implement A2A Server for APS profiles
- Implement A2A Client for profile-to-profile communication

**Phase 2**: Agent Cards
- Generate Agent Cards from APS profile configs
- Declare A2A capabilities
- Specify authentication requirements

**Phase 3**: Transport Adapters
- Map APS isolation tiers to A2A security schemes
- Implement custom IPC transport via A2A extensions (if needed)

**Phase 4**: CLI Integration
- Update commands to use A2A SDK
- Display A2A tasks instead of custom conversations
- Support Agent Card discovery

---

### Integration with APS Architecture

**Profiles → A2A Agents**:
- APS profile exposed as A2A agent
- Profile config → Agent Card
- Isolation tier → A2A security scheme

**Conversations → Tasks**:
- Multi-turn conversation → A2A Task
- Message history → Task message history
- Participants → Task participants

**Transports → A2A Bindings**:
- IPC → Custom transport via A2A extensions
- HTTP/WebSocket → A2A HTTP/gRPC bindings

---

### Interoperability Strategy

**APS ↔ External A2A Agents**:
- Agent Card discovery
- Task delegation
- Multi-turn collaboration
- Streaming responses

**APS Profiles ↔ APS Profiles**:
- Same A2A protocol
- Local IPC for efficiency
- Network communication when needed

---

## Open Questions

### Customization

1. **APS-Specific Features**
   - **Question**: How to add APS-specific features not in official A2A?
   - **Recommendation**: Use A2A's extensibility mechanism (extensions)
   - **Example**: APS isolation awareness, custom transports

2. **Storage Backend**
   - **Question**: Should APS use A2A's storage or custom implementation?
   - **Recommendation**: Custom storage using A2A task lifecycle
   - **Rationale**: Maintain APS's storage model while using A2A protocol

### Migration

3. **Legacy Data**
   - **Question**: What happens to existing custom conversations?
   - **Recommendation**: Keep read-only, don't auto-migrate
   - **Rationale**: Different protocols, risk of data loss

4. **Dual Protocol Support**
   - **Question**: Should APS support both custom and official A2A during transition?
   - **Recommendation**: No, direct replacement
   - **Rationale**: Simpler codebase, clear migration path

---

## References

### Official A2A Protocol
- **Specification**: https://a2a-protocol.org/latest/specification/
- **Documentation**: https://a2a-protocol.org/latest/
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **GitHub**: https://github.com/a2aproject/A2A

### Related Standards
- **ACP (Agent Client Protocol)**: https://agentclientprotocol.com
- **MCP (Model Context Protocol)**: https://modelcontextprotocol.io

### Other Standards (Historical Research)
- **FIPA ACL**: http://www.fipa.org/specs/fipa00037/
- **JSON-RPC**: https://www.jsonrpc.org/specification
- **MQTT**: http://mqtt.org/
- **AMQP**: https://www.amqp.org/
- **gRPC**: https://grpc.io/
- **WebSocket**: https://tools.ietf.org/html/rfc6455
- **CloudEvents**: https://cloudevents.io/

### Academic Papers
- "Agent Communication Languages: A Survey" (2002)
- "The Actor Model of Computation" (2003)
- "A Survey on Publish/Subscribe Systems" (2010)

### Books
- "Multi-Agent Systems: An Introduction to Distributed Artificial Intelligence" (1999)
- "Designing Data-Intensive Applications" (2017) - Chapter on messaging

---

## Summary

**Critical Discovery**: Official A2A Protocol exists at https://a2a-protocol.org, developed by Google, donated to Linux Foundation, with production-ready Go SDK.

**Recommendation**: **Adopt official A2A Protocol** instead of creating custom APS A2A protocol.

**Benefits**:
- Zero protocol engineering
- Ecosystem interoperability
- Official Go SDK (battle-tested)
- Enterprise-grade features
- Community support

**Next Steps**: See `plan.md` for detailed adoption plan.
