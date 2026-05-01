# A2A Protocol Examples

This directory contains examples demonstrating the A2A (Agent-to-Agent) protocol integration in APS.

## Prerequisites

1. APS installed and configured
2. At least two profiles created with A2A enabled

## Examples

### 1. Basic Server-Client Communication

**Scenario**: Start an A2A server for one profile and send a task from another.

#### Step 1: Create Profiles

```bash
# Create worker profile with A2A enabled
aps profile create worker --display-name "Worker Agent"

# Enable A2A for worker profile
# Edit ~/.local/share/aps/profiles/worker/profile.yaml and add:
# a2a:
#   enabled: true
#   protocol_binding: "jsonrpc"
#   listen_addr: "127.0.0.1:8081"
#   public_endpoint: "http://localhost:8081"

# Create orchestrator profile
aps profile create orchestrator --display-name "Orchestrator Agent"
```

#### Step 2: Start A2A Server

```bash
# Terminal 1: Start A2A server for worker profile
aps a2a server --profile worker

# Output:
# A2A server started for profile: worker
# Listening on: 127.0.0.1:8081
# Agent Card: http://127.0.0.1:8081/.well-known/agent-card
```

#### Step 3: Fetch Agent Card

```bash
# Terminal 2: Fetch the worker's Agent Card
aps a2a card fetch --url http://127.0.0.1:8081/.well-known/agent-card

# Output:
# Agent Card fetched from: http://127.0.0.1:8081/.well-known/agent-card
# URL: http://127.0.0.1:8081
# Transport: jsonrpc
```

#### Step 4: Send Task

```bash
# Send a task to worker profile
aps a2a tasks send --target worker --message "Process deployment for app1"

# Output:
# Task created/updated: task-abc123
# Status: submitted
# Last message ID: msg-def456
```

#### Step 5: Check Task Status

```bash
# List all tasks for worker profile
aps a2a tasks list --profile worker

# Output:
# TASK ID      STATUS      CREATED              MESSAGES
# task-abc123  completed   2026-01-30 10:15:30  2

# Get detailed task information
aps a2a tasks show task-abc123 --profile worker

# Output:
# Task ID: task-abc123
# Status: completed
#
# Message History (2 messages):
# ---
#
# Message 1 (ID: msg-def456, Role: user):
#   Part 1 [text]: Process deployment for app1
#
# Message 2 (ID: msg-ghi789, Role: agent):
#   Part 1 [text]: Deployment completed successfully
```

### 2. Task Cancellation

```bash
# Send a long-running task
aps a2a tasks send --target worker --message "Long running operation"

# Cancel the task
aps a2a tasks cancel <task-id> --target worker

# Verify cancellation
aps a2a tasks show <task-id> --profile worker
# Status should show: cancelled
```

### 3. Push Notifications

```bash
# Start a webhook server (example using netcat)
# Terminal 1:
nc -l 9000

# Terminal 2: Subscribe to task updates
aps a2a tasks subscribe <task-id> --target worker --webhook http://localhost:9000/hook

# Terminal 3: Send a message to trigger updates
aps a2a tasks send --target worker --task-id <task-id> --message "Update task"

# Terminal 1 will receive webhook notifications as the task progresses
```

### 4. Agent Card Management

```bash
# Show Agent Card for a profile
aps a2a card show --profile worker

# Output:
# Agent Card for Profile: worker
# Display Name: Worker Agent
# URL: http://127.0.0.1:8081
# Transport: jsonrpc
# Description: APS Profile Agent
#
# Capabilities:
#   - Streaming: true
#   - Push Notifications: true
#   - State Transition History: false

# Fetch and save Agent Card as JSON
aps a2a card fetch --url http://127.0.0.1:8081/.well-known/agent-card --format json > worker-card.json
```

### 5. Multi-Profile Workflow

**Scenario**: Orchestrator delegates tasks to multiple worker profiles.

```bash
# Create multiple worker profiles
aps profile create worker-1
aps profile create worker-2
aps profile create worker-3

# Start servers for all workers (in separate terminals)
aps a2a server --profile worker-1 &  # Port 8081
aps a2a server --profile worker-2 &  # Port 8082
aps a2a server --profile worker-3 &  # Port 8083

# Note: Configure each worker with different listen_addr in profile.yaml

# Send tasks from orchestrator to all workers
aps a2a tasks send --target worker-1 --message "Deploy service-a"
aps a2a tasks send --target worker-2 --message "Deploy service-b"
aps a2a tasks send --target worker-3 --message "Deploy service-c"

# Monitor all tasks
for worker in worker-1 worker-2 worker-3; do
  echo "=== Tasks for $worker ==="
  aps a2a tasks list --profile $worker
done
```

## Programmatic Usage (Go)

### Server Example

```go
package main

import (
    "context"
    "path/filepath"

    "oss-aps-cli/internal/core"
    "oss-aps-cli/internal/a2a"
)

func main() {
    ctx := context.Background()

    // Load profile
    profile, err := core.LoadProfile("worker")
    if err != nil {
        panic(err)
    }

    // Create storage config
    agentsDir, _ := core.GetAgentsDir()
    config := &a2a.StorageConfig{
        BasePath: filepath.Join(agentsDir, "a2a", profile.ID),
    }

    // Create and start server
    server, err := a2a.NewServer(profile, config)
    if err != nil {
        panic(err)
    }

    if err := server.Start(ctx); err != nil {
        panic(err)
    }

    // Server runs until context is cancelled
    select {}
}
```

### Client Example

```go
package main

import (
    "context"

    a2asdk "github.com/a2aproject/a2a-go/a2a"
    "oss-aps-cli/internal/core"
    "oss-aps-cli/internal/a2a"
)

func main() {
    ctx := context.Background()

    // Load target profile
    targetProfile, err := core.LoadProfile("worker")
    if err != nil {
        panic(err)
    }

    // Create client
    client, err := a2a.NewClient("worker", targetProfile)
    if err != nil {
        panic(err)
    }

    // Create message
    message := &a2asdk.Message{
        ID:   a2asdk.NewMessageID(),
        Role: a2asdk.MessageRoleUser,
        Parts: []a2asdk.Part{
            a2asdk.TextPart{Text: "Deploy application"},
        },
    }

    // Send message
    task, err := client.SendMessage(ctx, message)
    if err != nil {
        panic(err)
    }

    println("Task created:", string(task.ID))
}
```

## Troubleshooting

### Server Won't Start

**Issue**: `failed to start A2A server: address already in use`

**Solution**: Check if another server is running on the same port or change the `listen_addr` in profile configuration.

### Agent Card Not Found

**Issue**: `failed to fetch agent card: status 404`

**Solution**: Ensure the A2A server is running and the URL includes the correct path `/.well-known/agent-card`.

### Task Stuck in "Submitted" Status

**Issue**: Task remains in submitted state

**Solution**: Check server logs and ensure the executor is properly handling tasks. The server may be running but not processing tasks.

### Permission Denied Errors

**Issue**: `failed to create storage: permission denied`

**Solution**: Ensure the `<data>/a2a/` directory has proper permissions (0700) and your user owns it.

## Advanced Topics

### Custom Transport Configuration

Edit profile's `profile.yaml`:

```yaml
a2a:
  enabled: true
  protocol_binding: "grpc"  # or "jsonrpc", "http"
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "https://my-agent.example.com:8081"
```

### Security Configuration

For production deployments, configure mTLS or API key authentication in the profile's A2A settings.

### Task Storage

Task data is stored in:
```
~/.local/share/aps/a2a/<profile-id>/tasks/<task-id>/
```

You can inspect task state by reading the JSON files directly.

## Further Reading

- [A2A Protocol Specification](../../specs/005-a2a-protocol/spec.md)
- [Official A2A Documentation](https://a2a-protocol.org/latest/)
- [A2A Go SDK](https://github.com/a2aproject/a2a-go)
