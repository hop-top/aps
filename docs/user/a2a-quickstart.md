# A2A Protocol Quickstart

Get started with A2A (Agent-to-Agent) protocol in 5 minutes.

## Prerequisites

- APS installed
- Go 1.23+ (if building from source)

## Quick Setup

### 1. Create Two Profiles

```bash
# Create worker profile
aps profile create worker --display-name "Worker Agent"

# Create orchestrator profile
aps profile create orchestrator --display-name "Orchestrator Agent"
```

### 2. Enable A2A for Worker Profile

Edit `<data>/profiles/worker/profile.yaml`:

```yaml
id: worker
display_name: Worker Agent
a2a:
  enabled: true
  protocol_binding: jsonrpc
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://localhost:8081"
```

Save and close the file.

### 3. Start the A2A Server

```bash
aps a2a server --profile worker
```

You should see:
```
A2A server started for profile: worker
Listening on: 127.0.0.1:8081
Agent Card: http://127.0.0.1:8081/.well-known/agent-card

Press Ctrl+C to stop the server
```

Leave this running and open a new terminal.

### 4. Send Your First Task

In a new terminal:

```bash
aps a2a tasks send --target worker --message "Hello from A2A!"
```

Output:
```
Task created/updated: task-01HN123ABC456
Status: submitted
Last message ID: msg-01HN123DEF789
```

### 5. Check Task Status

```bash
# List all tasks
aps a2a tasks list --profile worker

# Get specific task details
aps a2a tasks show task-01HN123ABC456 --profile worker
```

## What Just Happened?

1. You created an A2A server for the `worker` profile
2. The server exposed an Agent Card at `/.well-known/agent-card`
3. You sent a task message from the CLI
4. The server processed the task and stored it
5. You queried the task status

## Next Steps

### Try More Commands

```bash
# View the Agent Card
aps a2a card show --profile worker

# Fetch Agent Card from URL
aps a2a card fetch --url http://127.0.0.1:8081/.well-known/agent-card

# Cancel a task
aps a2a tasks cancel <task-id> --target worker

# Subscribe to task updates
aps a2a tasks subscribe <task-id> --target worker --webhook http://localhost:9000/hook
```

### Create a Multi-Agent Workflow

1. Create more worker profiles (worker-2, worker-3)
2. Configure each with different ports
3. Start multiple servers
4. Send tasks to different workers
5. Monitor all tasks

See [a2a-examples.md](./a2a-examples.md) for detailed examples.

### Build Custom Agents

Use the Go SDK to create custom agent behaviors:

```go
import (
    "oss-aps-cli/internal/a2a"
    "oss-aps-cli/internal/core"
)

// Your custom executor implementation
type MyExecutor struct {
    *a2a.Executor
}

// Override Execute to implement custom logic
func (e *MyExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
    // Custom task execution logic
    return nil
}
```

## Architecture Overview

```
┌─────────────┐                    ┌─────────────┐
│ Orchestrator│                    │   Worker    │
│   Profile   │                    │   Profile   │
└─────┬───────┘                    └──────┬──────┘
      │                                   │
      │ 1. SendMessage (HTTP/JSON-RPC)    │
      │─────────────────────────────────>│
      │                                   │
      │                            2. Create Task
      │                            3. Execute Task
      │                            4. Store Result
      │                                   │
      │ 5. GetTask Response               │
      │<─────────────────────────────────│
      │                                   │
      │ 6. GetTask (Poll for status)      │
      │─────────────────────────────────>│
      │                                   │
      │ 7. Task + History                 │
      │<─────────────────────────────────│
      │                                   │
```

## Storage Structure

Tasks are stored in:
```
~/.local/share/aps/a2a/worker/tasks/task-01HN123ABC456/
├── meta.json                 # Task metadata and status
├── messages/                 # Message history
│   ├── msg-01HN123DEF789.json
│   └── msg-01HN123GHI012.json
└── event_*.json             # Event history
```

## Configuration Reference

### Profile A2A Settings

```yaml
a2a:
  enabled: true                      # Enable A2A for this profile
  protocol_binding: "jsonrpc"        # Transport: jsonrpc, grpc, http
  listen_addr: "127.0.0.1:8081"     # Server listen address
  public_endpoint: "http://localhost:8081"  # Public endpoint URL
```

### Supported Transports

- **jsonrpc** (default): JSON-RPC 2.0 over HTTP
- **grpc**: gRPC with Protocol Buffers
- **http**: RESTful HTTP+JSON

## Common Issues

**Port already in use**:
```bash
# Change listen_addr in profile.yaml
listen_addr: "127.0.0.1:8082"  # Use different port
```

**A2A not enabled**:
```bash
# Error: A2A is not enabled for profile worker
# Solution: Add a2a config to profile.yaml
```

**Permission denied**:
```bash
# Ensure data dir has correct permissions
chmod 700 ~/.local/share/aps
chmod 700 ~/.local/share/aps/a2a
```

## Resources

- [Full Examples](./a2a-examples.md)
- [A2A Specification](../../specs/005-a2a-protocol/spec.md)
- [Official A2A Docs](https://a2a-protocol.org/latest/)
- [APS Documentation](../../README.md)

## Help

```bash
# Get help for any command
aps a2a --help
aps a2a server --help
aps a2a tasks send --help

# View profile configuration
aps profile show worker
```

---

**Congratulations!** You've successfully set up A2A protocol communication between APS profiles.
