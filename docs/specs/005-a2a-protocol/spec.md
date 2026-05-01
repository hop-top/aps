# A2A Protocol Specification for APS

**Status**: Adopting Official Standard
**Version**: Official A2A v0.3.4
**Date**: 2026-01-21
**Scope**: APS integration with Agent2Agent (A2A) Protocol

---

## Overview

APS adopts the **official A2A Protocol** developed by Google and donated to the Linux Foundation. This document specifies how APS integrates with the standard A2A Protocol.

**Official Specification**: https://a2a-protocol.org/latest/specification/

**Go SDK**: https://github.com/a2aproject/a2a-go

---

## Key Decisions

### 1. Protocol Adoption ✅

**Decision**: Adopt official A2A Protocol instead of creating custom APS protocol.

**Rationale**:
- Name collision with existing standard
- Zero protocol engineering effort
- Ecosystem interoperability
- Official Go SDK available
- Enterprise-grade features

**Reference**: See `plan.md` for complete adoption plan

---

### 2. Architecture Mapping ✅

**APS Component → A2A Concept Mapping**:

| APS Component | A2A Concept | Description |
|---------------|--------------|-------------|
| Profile | Agent | Isolated execution environment |
| Profile Config | Agent Card | Discovery metadata and capabilities |
| Conversation | Task | Multi-turn communication context |
| Message | Message | Communication unit |
| Participant | Participant | Task participant profiles |
| Transport | Protocol Binding | IPC/HTTP/gRPC communication |

---

## A2A Protocol Summary

### Core Concepts

**Task**: Central abstraction representing work to be done. Tasks have a lifecycle (submitted, working, completed, failed, cancelled) and maintain message history.

**Agent Card**: Discovery mechanism. JSON-based metadata document describing agent's identity, capabilities, skills, interface, and authentication requirements.

**Message**: Communication unit with one or more Parts (TextPart, FilePart, DataPart). Messages have roles (user, agent) and are part of Task history.

**Protocol Bindings**: Multiple ways to implement A2A protocol:
- **JSON-RPC 2.0**: Simple, widely adopted
- **gRPC**: High performance, streaming
- **HTTP+JSON/REST**: Standard, firewall-friendly

**Streaming & Async**: Native support for long-running operations via:
- Server-Sent Events (SSE) for streaming
- Push notifications (webhooks) for async updates

---

## A2A Operations

### Core Operations

**SendMessage**:
- Create task and send message
- Returns direct response or Task object
- Supports synchronous request/response

**SendMessageStream**:
- Create task with streaming updates
- Real-time feedback via TaskStatusUpdateEvent and TaskArtifactUpdateEvent
- Returns initial Task + stream of updates

**GetTask**:
- Retrieve task state and message history
- Optional history length parameter
- Returns Task with messages and artifacts

**ListTasks**:
- Query tasks with filters (status, context, timestamp)
- Pagination support
- Optional artifact inclusion

**CancelTask**:
- Cancel long-running task
- Returns confirmation

**SubscribeToTask**:
- Subscribe to push notifications for task updates
- Configure webhook URL
- Receive TaskStatusUpdateEvent and TaskArtifactUpdateEvent

---

## APS Integration Architecture

### Component Diagram

```
┌─────────────────────────────────────────────┐
│         APS Profile                          │
├─────────────────────────────────────────────┤
│  • Profile Config                              │
│  • Isolation Tier                              │
│  • A2A Settings                               │
└─────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────┐
│      APS A2A Adapter Layer                 │
├─────────────────────────────────────────────┤
│  • Agent Card Generator                       │
│  • A2A Server (using a2asrv)            │
│  • A2A Client (using a2aclient)          │
│  • Transport Adapters (IPC, HTTP, gRPC)     │
└─────────────────────────────────────────────┘
           │
    ┌──────┼──────┐
    │      │      │
    ▼      ▼      ▼
┌────────┐┌────────┐┌────────┐
│  IPC   ││  HTTP  ││  gRPC  │
│ Queue  ││ Server ││ Server │
└────────┘└────────┘└────────┘
    │      │      │
    └──────┼──────┘
           ▼
┌─────────────────────────────────────────────┐
│      Official A2A Protocol                 │
│    (a2a-protocol.org + a2a-go SDK)         │
└─────────────────────────────────────────────┘
```

---

### APS A2A Server

**Purpose**: Expose APS profiles as A2A agents

**Components**:
- **Agent Executor**: Maps APS profile to A2A agent
- **Agent Card Generator**: Creates Agent Card from profile config
- **Transport Adapters**: IPC, HTTP, gRPC implementations
- **Storage Backend**: APS filesystem storage with A2A task lifecycle

**Implementation**:
```go
// internal/a2a/server.go
package a2a

import (
    "github.com/a2aproject/a2a-go/a2asrv"
    "github.com/a2aproject/a2a-go/a2a"
)

func NewAPAServer(profile *core.Profile) (*a2asrv.Server, error) {
    // Create APS-specific agent executor
    executor := NewAPSProfileExecutor(profile)
    
    // Create A2A handler with options
    options := []a2asrv.RequestHandlerOption{
        a2asrv.WithCustomOptions(GetAPSOptions()),
    }
    
    handler := a2asrv.NewHandler(executor, options...)
    
    // Wrap in transports
    grpcHandler := a2agrpc.NewHandler(handler)
    jsonrpcHandler := a2asrv.NewJSONRPCHandler(handler)
    
    return grpcHandler, nil
}
```

---

### APS A2A Client

**Purpose**: Enable APS profiles to communicate with other A2A agents

**Components**:
- **Agent Card Resolver**: Resolves target profile's Agent Card
- **Transport Selection**: Chooses optimal transport (IPC, HTTP, gRPC)
- **Task Management**: Creates and manages A2A tasks

**Implementation**:
```go
// internal/a2a/client.go
package a2a

import (
    "github.com/a2aproject/a2a-go/a2aclient"
    a2a "github.com/a2aproject/a2a-go/a2a"
)

func NewAPSClient(targetProfileID string) (*a2aclient.Client, error) {
    // Resolve Agent Card for target profile
    card, err := ResolveAPSProfileCard(targetProfileID)
    if err != nil {
        return nil, err
    }
    
    // Create A2A client from Agent Card
    options := a2aclient.FactoryOptions{
        // Add APS-specific options
    }
    
    client, err := a2aclient.NewFromCard(ctx, card, options...)
    if err != nil {
        return nil, err
    }
    
    return client, nil
}
```

---

### Agent Card Generation

**Purpose**: Generate A2A Agent Card from APS profile configuration

**Agent Card Structure**:
```json
{
  "agentProvider": {
    "name": "APS",
    "version": "1.0.0",
    "description": "Agent Profile System"
  },
  "agentCapabilities": {
    "supportedInterfaces": ["jsonrpc", "grpc"],
    "extensions": ["aps-isolation", "aps-profile"]
  },
  "agentSkills": [
    {
      "name": "execute",
      "description": "Execute commands in isolated environment",
      "inputSchema": {
        "type": "object",
        "properties": {
          "command": {"type": "string"},
          "args": {"type": "array"}
        }
      }
    }
  ],
  "agentInterfaces": [
    {
      "interfaceId": "jsonrpc",
      "version": "2.0",
      "url": "http://127.0.0.1:8080/jsonrpc"
    },
    {
      "interfaceId": "grpc",
      "version": "1.0",
      "url": "127.0.0.1:8080"
    }
  ],
  "securitySchemes": [
    {
      "type": "apiKey",
      "description": "API Key authentication",
      "in": "header",
      "name": "X-API-Key"
    },
    {
      "type": "mtls",
      "description": "Mutual TLS authentication",
      "mtlsEndpoint": "https://example.com/mtls"
    }
  ]
}
```

**Generation Logic**:
```go
// internal/a2a/agentcard.go
func GenerateAgentCard(profile *core.Profile) (*a2a.AgentCard, error) {
    // Map profile config to Agent Card
    provider := &a2a.AgentProvider{
        Name:    "APS",
        Version: "1.0.0",
    }
    
    capabilities := &a2a.AgentCapabilities{
        SupportedInterfaces: []string{"jsonrpc", "grpc"},
        Extensions:          []string{"aps-isolation"},
    }
    
    // Map profile's a2a config to Agent Card
    // ...
    
    return &a2a.AgentCard{
        AgentProvider:    provider,
        AgentCapabilities: capabilities,
        // ... additional fields
    }, nil
}
```

---

## Isolation Integration

### Transport Mapping

| APS Isolation Tier | A2A Transport | Security Scheme | Implementation |
|-------------------|----------------|----------------|------------------|
| Process (Tier 1) | Custom IPC | API Key (optional) | Filesystem queues |
| Platform (Tier 2) | HTTP/gRPC | mTLS or API Key | Network communication |
| Container (Tier 3) | HTTP/gRPC | mTLS (required) | Volume mounts + network |

### IPC Transport (Custom via A2A Extensions)

**Purpose**: Local communication across isolation tiers

**Implementation**:
- Filesystem queues in `~/.agents/ipc/queues/<profile-id>/incoming/`
- A2A extensibility mechanism for custom transport
- Polling-based delivery (100ms interval)
- Strict file permissions (0700)

**Directory Structure**:
```
~/.agents/ipc/
  queues/
    <profile-id>/
      incoming/
        <timestamp>_<uuid>.json
```

**Message Format** (A2A Message):
```json
{
  "id": "msg-001",
  "parts": [
    {
      "type": "text",
      "text": "Deploy app1"
    }
  ],
  "roles": ["user"],
  "timestamp": "1705773600000000000"
}
```

---

### Network Transport (HTTP/gRPC)

**Purpose**: Cross-machine communication

**Implementation**:
- A2A HTTP+JSON/REST binding for simple scenarios
- A2A gRPC binding for high-performance streaming
- mTLS for container isolation (Tier 3)
- Agent Card discovery at `/.well-known/agent-card`

**Endpoint Examples**:
- HTTP: `http://127.0.0.1:8080/jsonrpc`
- gRPC: `127.0.0.1:8080`
- Agent Card: `http://127.0.0.1:8080/.well-known/agent-card`

---

## Storage Integration

### Storage Backend

**Choice**: Custom APS storage implementing A2A task lifecycle

**Rationale**:
- Preserve APS filesystem-based storage
- No external database dependency
- Human-readable JSON format
- A2A storage is implementation choice

**Directory Structure**:
```
~/.agents/a2a/
  tasks/
    <task-id>/
      meta.json              # Task metadata
      messages/            # Message history
        <timestamp>.json
      artifacts/           # Task outputs
        <artifact-id>/
          meta.json
          content
  agent-cards/          # Cached Agent Cards
    <agent-id>.json
  registry.json          # Local profile registry
```

### Task Metadata (meta.json)

```json
{
  "id": "task-123",
  "contextId": "deploy-team",
  "participantIds": ["worker", "orchestrator"],
  "status": {
    "state": "completed",
    "timestamp": "1705773900000000000"
  },
  "created": "1705773600000000000",
  "updated": "1705773900000000000",
  "expiration": null
}
```

### Message Format (messages/<timestamp>.json)

```json
{
  "id": "msg-456",
  "parts": [
    {
      "type": "text",
      "text": "Deployment complete"
    }
  ],
  "roles": ["agent"],
  "timestamp": "17057739000000000000"
}
```

---

## CLI Integration

### Command Mapping

| Old Command (Custom) | New Command (A2A) | A2A Operation |
|---------------------|-------------------|----------------|
| `aps a2a start-duo` | `aps a2a create-task` | SendMessage |
| `aps a2a list-conversations` | `aps a2a tasks list` | ListTasks |
| `aps a2a show-conversation` | `aps a2a tasks show` | GetTask |
| `aps a2a send` | `aps a2a tasks send` | SendMessage |
| `aps a2a subscribe` | `aps a2a tasks subscribe` | SubscribeToTask |
| `aps a2a register` | `aps a2a publish-card` | Agent Card |

### Example Commands

```bash
# Create task (send message)
aps a2a create-task worker \
  --type task \
  --payload '{"command": "deploy"}'

# Stream task updates
aps a2a create-task worker \
  --type task \
  --payload '{"command": "deploy"}' \
  --stream

# List tasks
aps a2a tasks list --status working

# Get task details
aps a2a tasks show <task-id> --history

# Subscribe to task
aps a2a tasks subscribe <task-id> \
  --webhook http://localhost:9000/hook

# Cancel task
aps a2a tasks cancel <task-id>
```

---

## Security

### Authentication Schemes

**API Key**:
- Simple token-based authentication
- Stored in profile's `secrets.env`
- Environment variable: `A2A_API_KEY`

**mTLS**:
- Mutual TLS for high-security scenarios
- Required for container isolation (Tier 3)
- Certificates stored in profile's `secrets.env`

**OpenID Connect (OAuth2)**:
- Web-based authentication
- Agent Cards declare OAuth2 URLs
- Token caching in profile config

### Authorization

**Agent Cards** declare capabilities:
- AgentSkills define available operations
- Extensions declare custom features
- SecuritySchemes define auth requirements

**Profile ACLs**:
```yaml
a2a:
  allow_tasks:
    - orchestrator
    - supervisor
  block_tasks:
    - untrusted-agent
```

---

## Compliance

### A2A Protocol Compliance

APS must comply with official A2A Protocol:

**Required**:
- ✅ Implement A2A core operations (SendMessage, GetTask, ListTasks)
- ✅ Generate valid Agent Cards
- ✅ Support at least one protocol binding (JSON-RPC, gRPC, or HTTP)
- ✅ Implement Task lifecycle correctly
- ✅ Handle A2A error codes properly

**Optional**:
- ⏳ Streaming support (SendMessageStream)
- ⏳ Push notifications (SubscribeToTask)
- ⏳ Custom extensions via A2A extensibility mechanism
- ⏳ Multiple protocol bindings

**Reference**: https://a2a-protocol.org/latest/specification/ for complete requirements

---

## Migration from Custom Protocol

### Legacy Data

**Location**: `specs/005-a2a-protocol/legacy/`

**Access**: Legacy conversations remain read-only
```bash
# List legacy conversations
aps a2a list-conversations --legacy

# View legacy conversation
aps a2a show-conversation <conv-id> --legacy
```

**Migration Path** (optional):
```bash
# Migrate legacy conversation to A2A task
aps a2a migrate <conv-id> --to-task
```

**Note**: New tasks use official A2A protocol. Legacy data is not auto-migrated.

---

## Versioning

### APS A2A Integration Version

**Current**: v1.0 (adopting official A2A v0.3.4)

**Semantic Versioning**:
- **MAJOR**: Breaking changes to APS integration
- **MINOR**: New APS-specific features, backward compatible
- **PATCH**: Bug fixes, backward compatible

**A2A Protocol Versioning**:
- APS follows official A2A versioning
- A2A v0.3.4 → APS A2A Integration v1.0
- Future A2A updates → APS updates to follow

---

## References

### Official A2A Documentation
- **Specification**: https://a2a-protocol.org/latest/specification/
- **Documentation**: https://a2a-protocol.org/latest/
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **GitHub**: https://github.com/a2aproject/A2A

### APS Documentation
- **Plan**: `plan.md` - A2A adoption plan
- **Research**: `research.md` - Protocol research
- **Decisions**: `decisions.md` - Design decisions
- **Quickstart**: `quickstart.md` - Getting started guide

### Legacy (Custom Protocol)
- **Custom Spec**: `legacy/custom-spec.md` - Original custom protocol
- **ADR Directory**: `adrs/` - Architecture Decision Records

---

## Summary

APS adopts the **official A2A Protocol** instead of creating a custom protocol. This provides:

**Benefits**:
- Zero protocol engineering
- Ecosystem interoperability
- Official Go SDK
- Enterprise-grade features
- Community support

**Implementation**:
- APS A2A Server (using `a2asrv`)
- APS A2A Client (using `a2aclient`)
- Agent Card generation from profiles
- Transport adapters for isolation tiers
- Custom storage implementing A2A lifecycle

**Timeline**: 6 weeks for complete migration (see `plan.md`)

**Status**: Ready for implementation.
