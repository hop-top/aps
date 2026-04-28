# A2A Protocol - Quickstart for APS

**Purpose**: Get started with Agent-to-Agent communication in APS using official A2A Protocol

---

## What is A2A?

A2A (Agent2Agent) Protocol is an **open standard** developed by Google and donated to the Linux Foundation. It enables seamless communication and collaboration between AI agents, providing a common language for agents built using diverse frameworks.

**Official Documentation**: https://a2a-protocol.org/latest/

**Go SDK**: https://github.com/a2aproject/a2a-go

**Key Features**:
- **Task-based messaging**: Central abstraction with full lifecycle management
- **Agent Cards**: Discovery mechanism for agent capabilities
- **Multi-protocol bindings**: JSON-RPC 2.0, gRPC, HTTP+JSON/REST
- **Streaming & Async**: Native support for long-running operations
- **Enterprise-ready**: Authentication, authorization, security, tracing
- **Extensibility**: Custom extensions and capability declarations

---

## Quick Setup

### 1. Enable A2A on a Profile

```bash
# Create new profile with A2A enabled
aps profile new orchestrator \
  --display-name "Task Orchestrator" \
  --enable-a2a

# Verify A2A is enabled
aps profile show orchestrator
```

**Profile Configuration**:
```yaml
id: orchestrator
display_name: "Task Orchestrator"

a2a:
  enabled: true
  # Preferred protocol binding (jsonrpc, grpc, http)
  protocol_binding: "jsonrpc"
  # Security scheme for discovery
  security_scheme: "apikey"
```

---

### 2. Create Your First A2A Task

A2A uses a task-based model. Create a task to send a message to another profile.

```bash
# Send a task request (synchronous)
aps a2a send-task worker \
  --type query \
  --payload '{"query": "status"}'

# Send a task request with streaming (real-time updates)
aps a2a send-stream worker \
  --type task \
  --payload '{"command": "deploy", "target": "app1"}'
```

**What Happens**:
1. APS resolves the target profile's Agent Card
2. Creates an A2A Task via `a2aclient.SendMessage()`
3. Task is routed to target profile via appropriate transport
4. Target profile processes and returns response
5. Task state stored in `~/.agents/a2a/tasks/<task-id>/`

---

### 3. List and View Tasks

```bash
# List all tasks for a profile
aps a2a list-tasks orchestrator

# List tasks by status
aps a2a list-tasks orchestrator --status working

# List tasks by context (grouping)
aps a2a list-tasks orchestrator --context deploy-team

# View task details and message history
aps a2a get-task <task-id>

# Cancel a task
aps a2a cancel-task <task-id>
```

**Task States**:
- `submitted`: Task created, waiting to be processed
- `working`: Task is being processed
- `completed`: Task completed successfully
- `failed`: Task failed with error
- `cancelled`: Task was cancelled

---

### 4. Agent Card Discovery

Agent Cards are A2A's discovery mechanism. They describe an agent's capabilities, skills, and authentication requirements.

```bash
# Fetch an Agent Card from external agent
aps a2a fetch-agent-card http://10.0.0.1:8080/.well-known/agent-card

# Verify Agent Card signature
aps a2a verify-agent-card <card-file>

# Display Agent Card capabilities
aps a2a show-capabilities <card-file>

# Register profile for network discovery
aps a2a register \
  --endpoint http://10.0.0.1:8080 \
  --protocol-binding grpc

# Discover agents on network
aps a2a discover --network 192.168.1.0/24
```

**Agent Card Example**:
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
  "agentSkills": [
    {
      "name": "deploy",
      "description": "Deploy applications",
      "inputSchema": {...}
    }
  ],
  "securitySchemes": [
    {
      "type": "apiKey",
      "description": "API Key authentication"
    }
  ]
}
```

---

## Common Scenarios

### Scenario 1: Task Delegation

Profile A (orchestrator) delegates tasks to Profile B (worker):

```bash
# Profile A creates task for Profile B
aps a2a send-task worker \
  --type task \
  --payload '{"command": "deploy", "target": "app1"}'

# Profile B receives task via A2A protocol
# Profile B processes and returns result
# Task stored in history
```

**A2A Flow**:
1. A2A Client creates Task with Message
2. Task routed via transport (IPC, HTTP, gRPC)
3. Target A2A Server receives Task
4. Agent Executor processes Task
5. Response sent back as Task update
6. Task marked as `completed`

---

### Scenario 2: Real-Time Streaming

Profile A streams task updates from Profile B:

```bash
# Profile A creates streaming task
aps a2a send-stream worker \
  --type task \
  --payload '{"command": "deploy"}'

# Real-time updates streamed to console:
# [1] Task submitted
# [2] TaskStatus: working
# [3] Artifact: deployment-plan.txt
# [4] TaskStatus: completed
```

**Streaming Events**:
- `TaskStatusUpdateEvent`: Status changes (submitted → working → completed)
- `TaskArtifactUpdateEvent`: Output artifacts (logs, files, results)

---

### Scenario 3: Multi-Turn Conversation

Multiple profiles collaborate on a task:

```bash
# Profile A creates task for group
aps a2a send-task deploy-team \
  --to worker,monitor \
  --type task \
  --payload '{"command": "deploy", "target": "app1"}'

# Task message history tracks all messages
aps a2a get-task <task-id> --history
```

**Message History**:
```json
{
  "messages": [
    {
      "id": "msg-001",
      "role": "user",
      "parts": [{"type": "text", "text": "Deploy app1"}]
    },
    {
      "id": "msg-002",
      "role": "agent",
      "parts": [{"type": "text", "text": "Starting deployment..."}]
    },
    {
      "id": "msg-003",
      "role": "agent",
      "parts": [{"type": "text", "text": "Deployment complete"}]
    }
  ]
}
```

---

### Scenario 4: Cross-Machine Communication

Profiles on different machines collaborate:

```bash
# Machine 1 - Register profile
aps a2a register \
  --endpoint http://10.0.0.1:8080 \
  --protocol-binding grpc

# Machine 2 - Discover and communicate
aps a2a discover --network 192.168.1.0/24

# Create task for remote profile
aps a2a send-task agent-a@10.0.0.1:8080 \
  --type query \
  --payload '{"query": "status"}'
```

**Network Discovery**:
- Profiles register endpoints in registry
- DNS-based discovery for network addresses
- Agent Cards published at `/.well-known/agent-card`
- Automatic transport selection (fallback: gRPC → HTTP)

---

### Scenario 5: Long-Running Tasks with Push Notifications

Subscribe to task updates via webhook:

```bash
# Subscribe to task push notifications
aps a2a subscribe-task <task-id> \
  --webhook http://localhost:9000/hook

# Task continues processing in background
# Push notifications sent to webhook when status changes
# or artifacts generated
```

**Push Notification Payload**:
```json
{
  "taskUpdate": {
    "taskId": "<task-id>",
    "status": "completed",
    "timestamp": "2026-01-20T10:00:00Z"
  }
}
```

---

## Profile Configuration

### Minimal A2A Configuration

```yaml
id: worker
display_name: "Worker Profile"

a2a:
  enabled: true
  protocol_binding: "jsonrpc"
  security_scheme: "apikey"
```

### Network-Enabled Profile

```yaml
id: orchestrator
display_name: "Orchestrator"

a2a:
  enabled: true
  protocol_binding: "grpc"
  listen_addr: "127.0.0.1:8080"
  public_endpoint: "http://10.0.0.1:8080"
  security_scheme: "mtls"
```

### Platform Isolation (Tier 2)

```yaml
id: platform-worker
display_name: "Platform Worker"

a2a:
  enabled: true
  protocol_binding: "http"
  security_scheme: "mtls"
  isolation_tier: "platform"
  shared_ipc: "/run/aps/ipc"
```

### Container Isolation (Tier 3)

```yaml
id: container-worker
display_name: "Container Worker"

a2a:
  enabled: true
  protocol_binding: "grpc"
  security_scheme: "mtls"
  isolation_tier: "container"
  volume_mounts:
    - "/var/run/aps/ipc:/agents/ipc"
    - "/var/run/aps/a2a:/agents/a2a"
```

---

## Task Management

### Listing Tasks

```bash
# List all tasks
aps a2a list-tasks

# Filter by status
aps a2a list-tasks --status working

# Filter by context (grouping)
aps a2a list-tasks --context deploy-team

# Pagination
aps a2a list-tasks --page-size 50 --page-token <token>

# Include artifacts in results
aps a2a list-tasks --include-artifacts
```

### Task Details

```bash
# Show task metadata and status
aps a2a get-task <task-id>

# Show message history
aps a2a get-task <task-id> --history

# Show artifacts (outputs)
aps a2a get-task <task-id> --artifacts

# Show last N messages
aps a2a get-task <task-id> --history-limit 10
```

### Task Lifecycle

```bash
# Cancel long-running task
aps a2a cancel-task <task-id>

# Archive task (read-only)
aps a2a archive-task <task-id>

# Delete task (and history)
aps a2a delete-task <task-id>
```

---

## Transport Selection

### JSON-RPC 2.0

**Best for**: Simple, request/response scenarios

**Advantages**:
- Simple, widely adopted
- Human-readable (JSON)
- Easy debugging

**Configuration**:
```yaml
a2a:
  protocol_binding: "jsonrpc"
```

---

### gRPC

**Best for**: High-performance, streaming scenarios

**Advantages**:
- High performance (binary)
- Streaming support
- Strong typing

**Configuration**:
```yaml
a2a:
  protocol_binding: "grpc"
  listen_addr: "127.0.0.1:8080"
```

---

### HTTP+JSON/REST

**Best for**: Network communication, firewall-friendly

**Advantages**:
- Standard, universal
- Firewall-friendly
- No special client required

**Configuration**:
```yaml
a2a:
  protocol_binding: "http"
  listen_addr: "127.0.0.1:8080"
```

### IPC (Custom via A2A Extensions)

**Best for**: Local communication across isolation tiers

**Advantages**:
- Always available
- Works with APS isolation model
- No network overhead

**Configuration**:
```yaml
a2a:
  protocol_binding: "custom:ipc"
  ipc_path: "/run/aps/ipc"
```

---

## Security

### Authentication

**API Key** (Simple):
```yaml
a2a:
  security_scheme: "apikey"
  api_key_env: "A2A_API_KEY"  # From secrets.env
```

**mTLS** (High Security):
```yaml
a2a:
  security_scheme: "mtls"
  mtls_cert: "/path/to/cert.pem"
  mtls_key: "/path/to/key.pem"
  mtls_ca: "/path/to/ca.pem"
```

**OpenID Connect** (OAuth2):
```yaml
a2a:
  security_scheme: "openid"
  openid_auth_url: "https://auth.example.com/auth"
  openid_token_url: "https://auth.example.com/token"
```

### Authorization

**Agent Cards** declare capabilities:
```json
{
  "agentCapabilities": {
    "extensions": ["deploy", "query"]
  }
}
```

**ACLs** in profile config:
```yaml
a2a:
  allow_tasks:
    - orchestrator
    - supervisor
  block_tasks:
    - untrusted-agent
```

---

## Debugging and Monitoring

### Check A2A Status

```bash
# Show A2A server status
aps a2a status server

# Show active tasks
aps a2a status tasks

# Show transport status
aps a2a status transport

# Show Agent Card
aps a2a status agent-card
```

### Stream Task Updates

```bash
# Stream task updates in real-time
aps a2a stream <task-id>

# Stream all tasks for a profile
aps a2a stream --profile worker

# Stream tasks by context
aps a2a stream --context deploy-team
```

### Task Statistics

```bash
# Show task statistics
aps a2a stats <task-id>

# Output:
# Task ID: task-123
# Status: completed
# Created: 2026-01-20 10:00:00
# Completed: 2026-01-20 10:05:00
# Duration: 5m 0s
# Message Count: 15
# Artifact Count: 3
```

---

## Filesystem Structure

```
~/.agents/
  a2a/
    tasks/
      <task-id>/
        meta.json              # Task metadata
        messages/            # Message history
          <timestamp>.json
        artifacts/           # Task outputs
          <artifact-id>/
            meta.json
            content
      index.json             # Search index
    agent-cards/          # Cached Agent Cards
      <agent-id>.json
    registry.json          # Local profile registry
  ipc/
    queues/               # IPC transport queues
      <profile-id>/
        incoming/
          <uuid>.json
  logs/
    a2a/
      server.log
      transport.log
```

---

## Next Steps

1. **Read Official Spec**: See https://a2a-protocol.org/latest/specification/ for complete protocol details
2. **Explore Go SDK**: See https://github.com/a2aproject/a2a-go for SDK documentation
3. **Check Examples**: See `a2a-samples` repo for detailed examples
4. **Setup Profiles**: Create profiles with A2A enabled
5. **Start Communicating**: Try the CLI commands above

---

## Troubleshooting

### Task Not Found

```bash
# Check task exists
aps a2a list-tasks --all

# Check task ID format (should be UUID)
aps a2a get-task <task-id>
```

### Agent Card Not Found

```bash
# Check endpoint is accessible
curl http://10.0.0.1:8080/.well-known/agent-card

# Check network connectivity
aps a2a discover --network 192.168.1.0/24
```

### Transport Connection Failed

```bash
# Check transport status
aps a2a status transport

# Check firewall settings (network communication)
# Check IPC permissions (local communication)

# Try fallback transport
aps a2a send-task <target> --protocol-binding http
```

### Isolation Issues

```bash
# Check isolation configuration
aps profile show <profile-id>

# Verify IPC directory permissions
ls -la /run/aps/ipc/

# Check volume mounts (container isolation)
docker inspect <container-id>
```

---

## Getting Help

- **Official Documentation**: https://a2a-protocol.org/latest/
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **Go Doc Reference**: https://pkg.go.dev/github.com/a2aproject/a2a-go
- **APS Documentation**: See `spec.md` for APS-specific A2A integration
- **Logs**: `~/.agents/logs/a2a/`
- **Debug Mode**: `aps a2a --debug <command>`
- **Status**: `aps a2a status` for system health

---

## Migration from Custom Protocol

If you have data from the previous custom A2A protocol:

```bash
# Legacy conversations are read-only
aps a2a list-conversations --legacy

# View legacy conversation
aps a2a show-conversation <conv-id> --legacy

# (Optional) Convert to A2A task
aps a2a migrate <conv-id> --to-task
```

**Note**: New tasks use official A2A protocol. Legacy conversations remain accessible in read-only mode.
