# A2A Protocol Integration for AI Agents

**For AI Agents**: How to use and understand the A2A protocol in APS.

---

## What is A2A?

A2A (Agent-to-Agent) is a standard protocol for communication between AI agents. In APS, each profile can act as an A2A agent, allowing profiles to send tasks and messages to each other.

Current APS maturity: the A2A listener, agent card, task storage, and task
status lifecycle are implemented. The built-in executor is placeholder-level
and replies with `Processed: <message>`; it does not run profile actions,
chat, or LLM-backed work yet.

---

## Key Concepts

### Agent Card

An **Agent Card** is like a business card for an AI agent. It tells other agents:
- Where to find you (URL)
- How to communicate with you (transport protocol)
- What you can do (capabilities)
- How to authenticate (security)

**Example Agent Card**:
```json
{
  "url": "http://127.0.0.1:8081",
  "preferredTransport": "jsonrpc",
  "description": "Worker agent for task processing",
  "capabilities": {
    "streaming": true,
    "pushNotifications": true
  }
}
```

### Task

A **Task** represents work to be done. It has:
- **ID**: Unique identifier
- **Status**: submitted → working → completed/failed/cancelled
- **History**: All messages exchanged
- **Artifacts**: Results or outputs

### Message

A **Message** is a communication unit within a task. It contains:
- **Role**: "user" (requester) or "agent" (responder)
- **Parts**: Text, files, or data
- **Task ID**: Which task it belongs to

---

## How to Use A2A

### As a Server (Receiving Tasks)

When you want to **receive** tasks from other agents:

1. **Enable A2A** in your profile configuration:
```yaml
a2a:
  protocol_binding: jsonrpc
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://localhost:8081"
```

2. **Start the A2A server**:
```bash
aps a2a server --profile <your-profile-id>
```

3. **Your agent is now discoverable** at:
   - Agent Card: `http://127.0.0.1:8081/.well-known/agent-card`
   - Task endpoint: `http://127.0.0.1:8081/`

### As a Client (Sending Tasks)

When you want to **send** tasks to other agents:

1. **Send a task**:
```bash
aps a2a tasks send --target <other-profile> --message "Your request here"
```

2. **Check task status**:
```bash
aps a2a tasks list --profile <other-profile>
aps a2a tasks show <task-id> --profile <other-profile>
```

3. **Cancel if needed**:
```bash
aps a2a tasks cancel <task-id> --target <other-profile>
```

---

## Communication Patterns

### Request-Response (Synchronous)

```
You → Send task → Other Agent
You ← Response ← Other Agent
```

**Use when**: You need an immediate answer.

**Example**:
```bash
# Send task
aps a2a tasks send --target worker --message "Get status"

# Get result
aps a2a tasks show <task-id> --profile worker
```

### Fire-and-Forget (Asynchronous)

```
You → Send task → Other Agent
(Task executes in background)
You → Check status later
```

**Use when**: Task takes time to complete.

**Example**:
```bash
# Send task
aps a2a tasks send --target worker --message "Process large dataset"

# Check later
aps a2a tasks list --profile worker --status completed
```

### Streaming (Real-time Updates)

```
You → Send task → Other Agent
You ← Update 1 ← Other Agent
You ← Update 2 ← Other Agent
You ← Final ← Other Agent
```

**Use when**: You want progress updates.

**Example** (when supported):
```bash
aps a2a tasks stream --target worker --message "Long operation"
```

### Push Notifications (Webhooks)

```
You → Subscribe → Other Agent
Other Agent → Webhook → Your server
```

**Use when**: You want to be notified of changes.

**Example**:
```bash
# Subscribe to task updates
aps a2a tasks subscribe <task-id> --target worker --webhook http://your-server/hook
```

---

## Multi-Agent Workflows

### Pattern 1: Orchestrator-Worker

```
Orchestrator → Worker 1 (Task A)
Orchestrator → Worker 2 (Task B)
Orchestrator → Worker 3 (Task C)
Orchestrator ← Results ← Workers
```

**Use case**: Distribute work across multiple agents.

**Implementation**:
```bash
# Orchestrator sends tasks
aps a2a tasks send --target worker-1 --message "Task A"
aps a2a tasks send --target worker-2 --message "Task B"
aps a2a tasks send --target worker-3 --message "Task C"

# Collect results
aps a2a tasks list --profile worker-1
aps a2a tasks list --profile worker-2
aps a2a tasks list --profile worker-3
```

### Pattern 2: Pipeline

```
Agent A → Agent B → Agent C → Result
```

**Use case**: Sequential processing through multiple agents.

**Implementation**:
1. Agent A sends task to Agent B
2. Agent B processes and sends result to Agent C
3. Agent C processes and returns final result

### Pattern 3: Supervisor-Subordinate

```
Supervisor ← Status ← Worker (periodic)
Supervisor → Commands → Worker
```

**Use case**: Monitoring and control.

**Implementation**:
- Worker sends periodic status updates
- Supervisor can cancel or modify tasks

---

## Understanding Task Lifecycle

```
┌──────────┐
│ submitted│  ← Task created
└────┬─────┘
     │
     ▼
┌──────────┐
│  working │  ← Processing in progress
└────┬─────┘
     │
     ├─→ ┌───────────┐
     │   │ completed │  ← Success
     │   └───────────┘
     │
     ├─→ ┌───────────┐
     │   │  failed   │  ← Error occurred
     │   └───────────┘
     │
     └─→ ┌───────────┐
         │ cancelled │  ← User cancelled
         └───────────┘
```

**States**:
- **submitted**: Task received, not yet started
- **working**: Task is being processed
- **completed**: Task finished successfully
- **failed**: Task encountered an error
- **cancelled**: Task was cancelled by user

---

## Discovery and Connection

### Finding Other Agents

1. **Known profiles**: You already know the profile ID
```bash
aps a2a card show --profile known-worker
```

2. **Agent Card URL**: You have the URL
```bash
aps a2a card fetch --url http://remote-agent:8081/.well-known/agent-card
```

3. **Discovery service** (future): Central registry of agents

### Connecting to Remote Agents

For agents on other machines:

1. **Ensure network connectivity**
2. **Use public endpoint** in configuration:
```yaml
a2a:
  public_endpoint: "https://my-agent.example.com:8081"
```
3. **Configure external protection** such as an authenticated tunnel or reverse
   proxy. The current `aps a2a server` path does not enforce API key, mTLS, or
   OAuth/OIDC authentication itself.
4. **Use HTTPS** for production

---

## Security Considerations

### Authentication

API key and mTLS helpers exist as component-level transport code, but they are
not enforced by the current `aps a2a server` listener. Treat those modes as
planned until server/client enforcement is wired and covered end to end.

### Authorization

Control who can send you tasks:

```yaml
a2a:
  allow_tasks:
    - trusted-orchestrator
    - supervisor
  block_tasks:
    - untrusted-agent
```

### Data Privacy

- **Encrypt communications**: Use HTTPS
- **Validate inputs**: Sanitize task messages
- **Limit exposure**: Only expose necessary capabilities

---

## Monitoring and Debugging

### Check Server Status

```bash
# List all tasks
aps a2a tasks list --profile <your-profile>

# Check specific task
aps a2a tasks show <task-id> --profile <your-profile>
```

### View Task History

```bash
# Get task with full history
aps a2a tasks show <task-id> --profile <your-profile>

# Output shows all messages exchanged
```

### Inspect Storage

Tasks are stored in:
```
~/.local/share/aps/a2a/<profile-id>/tasks/<task-id>/
├── meta.json        # Task metadata
├── messages/        # Message history
└── event_*.json    # Event log
```

### Common Issues

**Server won't start**:
- Check if port is already in use
- Verify A2A is enabled in profile config
- Check file permissions on `<data>/`

**Can't send task**:
- Verify target profile has A2A enabled
- Check target server is running
- Verify network connectivity (if remote)
- Check Agent Card URL is correct

**Task stuck in "submitted"**:
- Check target server logs
- Verify executor is processing tasks
- Check for errors in event log

---

## Best Practices

### For Server Agents

1. **Process tasks promptly**: Don't leave tasks in "submitted" state
2. **Emit status updates**: Use streaming to show progress
3. **Handle errors gracefully**: Return meaningful error messages
4. **Set timeouts**: Don't let tasks run forever
5. **Clean up completed tasks**: Archive or delete old tasks

### For Client Agents

1. **Check task status**: Poll or subscribe for updates
2. **Handle failures**: Implement retry logic
3. **Set reasonable timeouts**: Don't wait forever
4. **Cancel when needed**: Clean up abandoned tasks
5. **Validate responses**: Check task results before using

### For All Agents

1. **Use descriptive messages**: Make task intent clear
2. **Structure data properly**: Use appropriate message parts
3. **Respect resource limits**: Don't overload other agents
4. **Secure communications**: Use authentication and encryption
5. **Monitor performance**: Track task completion times

---

## Example Workflows

### Example 1: Data Processing Pipeline

**Goal**: Process data through multiple specialized agents.

```bash
# 1. Orchestrator sends raw data to Parser
aps a2a tasks send --target parser --message "Parse dataset.csv"

# 2. Parser processes and sends to Transformer
# (Parser would programmatically send to next agent)

# 3. Transformer processes and sends to Analyzer
# (Transformer would programmatically send to next agent)

# 4. Orchestrator checks final result
aps a2a tasks list --profile analyzer --status completed
```

### Example 2: Distributed Computation

**Goal**: Parallelize computation across multiple workers.

```bash
# Split work into chunks
aps a2a tasks send --target worker-1 --message "Compute chunk 1"
aps a2a tasks send --target worker-2 --message "Compute chunk 2"
aps a2a tasks send --target worker-3 --message "Compute chunk 3"

# Monitor progress
aps a2a tasks list --profile worker-1
aps a2a tasks list --profile worker-2
aps a2a tasks list --profile worker-3

# Collect results when all complete
```

### Example 3: Agent Supervision

**Goal**: Monitor and control worker agents.

```bash
# Supervisor periodically checks worker status
aps a2a tasks send --target worker --message "Status check"
aps a2a tasks show <task-id> --profile worker

# If worker is overloaded, cancel pending tasks
aps a2a tasks cancel <task-id> --target worker
```

---

## Resources

- **User Documentation**: `docs/user/a2a-quickstart.md`, `docs/user/a2a-examples.md`
- **Developer Documentation**: `docs/dev/a2a-implementation.md`
- **Specification**: `specs/005-a2a-protocol/spec.md`
- **Official A2A Protocol**: https://a2a-protocol.org/latest/

---

## Quick Reference

### Server Commands
```bash
aps a2a server --profile <id>                    # Start server
aps a2a card show --profile <id>                # Show Agent Card
aps a2a tasks list --profile <id>               # List tasks
aps a2a tasks show <task-id> --profile <id>       # Get task details
```

### Client Commands
```bash
aps a2a tasks send -t <target> -m <message>      # Send task
aps a2a tasks cancel <task-id> -t <target>       # Cancel task
aps a2a card fetch --url <url>                   # Fetch Agent Card
aps a2a tasks subscribe <task-id> -t <target> \
  --webhook <url>                                # Subscribe to updates
```

### Configuration
```yaml
a2a:
  enabled: true
  protocol_binding: jsonrpc  # or grpc, http
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://localhost:8081"
```
