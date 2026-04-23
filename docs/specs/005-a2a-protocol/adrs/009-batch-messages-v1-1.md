# ADR-009: Batch Messages for v1.1

**Status**: Accepted (Future)
**Date**: 2026-01-20
**Related**: ADR-001 (JSON for v1.0)
**Target**: Protocol v1.1

---

## Context

After v1.0 implementation, high-throughput scenarios may benefit from batching multiple small messages into a single transport operation.

**Use Cases for Batching**:
- **Multiple Deployments**: Deploy 10 applications at once
- **Bulk Queries**: Query status of 100 services
- **Batch Operations**: Apply configuration changes to multiple targets
- **Reduce Round-Trips**: Minimize network latency overhead

**Current v1.0 Limitations**:
- **1-to-1 Mapping**: Each transport operation = 1 message
- **High Latency**: Multiple small messages = multiple round-trips
- **Overhead**: Repeated transport handshake/headers per message

**Requirements for v1.1**:
- Reduce round-trip overhead
- Support atomic batch processing (all succeed or all fail)
- Maintain backward compatibility with v1.0 clients
- Preserve message integrity within batches
- Support mixed patterns (request, response, publish) within batch
- Keep v1.0 simple (defer complexity)

---

## Decision

**Add batch message pattern for A2A Protocol v1.1**

### Implementation Details

#### Batch Message Format

```json
{
  "id": "batch-001",
  "version": "1.1",
  "from": "orchestrator",
  "to": ["worker"],
  "pattern": "batch",
  "type": "multiple_requests",
  "payload": {
    "messages": [
      {
        "id": "req-001",
        "pattern": "request",
        "type": "task",
        "correl_id": "corr-001",
        "payload": {
          "command": "deploy",
          "target": "app1"
        }
      },
      {
        "id": "req-002",
        "pattern": "request",
        "type": "task",
        "correl_id": "corr-002",
        "payload": {
          "command": "deploy",
          "target": "app2"
        }
      },
      {
        "id": "req-003",
        "pattern": "request",
        "type": "query",
        "correl_id": "corr-003",
        "payload": {
          "query": "status"
        }
      }
    ]
  },
  "metadata": {
    "batch_size": 3,
    "atomic": true
  },
  "timestamp": 1705773600000000000
}
```

#### Batch Response Format

```json
{
  "id": "batch-resp-001",
  "version": "1.1",
  "from": "worker",
  "to": ["orchestrator"],
  "pattern": "batch",
  "type": "multiple_responses",
  "payload": {
    "messages": [
      {
        "correl_id": "corr-001",
        "pattern": "response",
        "type": "result",
        "payload": {
          "status": "success",
          "output": "Deployed app1"
        }
      },
      {
        "correl_id": "corr-002",
        "pattern": "response",
        "type": "result",
        "payload": {
          "status": "success",
          "output": "Deployed app2"
        }
      },
      {
        "correl_id": "corr-003",
        "pattern": "response",
        "type": "result",
        "payload": {
          "status": "running",
          "output": "Worker is running"
        }
      }
    ]
  },
  "metadata": {
    "batch_size": 3,
    "all_succeeded": true
  },
  "timestamp": 1705773601000000000
}
```

#### Batch Pattern

```go
const PatternBatch MessagePattern = "batch"
```

#### Batch Payload Schema

```go
type BatchPayload struct {
    Messages []BatchMessage `json:"messages"`  // Required
}

type BatchMessage struct {
    ID        string                 `json:"id"`         // Required
    Pattern   MessagePattern         `json:"pattern"`    // Required
    Type      string                 `json:"type"`       // Required
    CorrelID  string                 `json:"correl_id,omitempty"`  // For request/response
    Payload   map[string]interface{} `json:"payload"`    // Required
    Metadata  map[string]string      `json:"metadata,omitempty"`  // Optional
}
```

#### Batch Processing Logic

```go
func (m *Manager) ProcessBatch(batch *Message) (*Message, error) {
    // 1. Extract batch messages
    batchPayload, ok := batch.Payload.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid batch payload")
    }

    messagesData, ok := batchPayload["messages"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("batch messages missing")
    }

    // 2. Check atomic flag
    atomic := false
    if atom, ok := batch.Metadata["atomic"].(string); ok {
        atomic = (atom == "true")
    }

    // 3. Process each message
    var responses []BatchMessage
    var firstError error

    for _, msgData := range messagesData {
        // Convert to BatchMessage
        msgMap, _ := msgData.(map[string]interface{})
        batchMsg := parseBatchMessage(msgMap)

        // Process message
        response, err := m.processMessage(batchMsg)

        // Store response
        responses = append(responses, BatchMessage{
            CorrelID: batchMsg.CorrelID,
            Pattern:  "response",
            Type:    batchMsg.Type,
            Payload: response,
        })

        // Handle atomic mode
        if atomic && err != nil {
            firstError = err
            break  // Stop processing on first error
        }
    }

    // 4. Construct batch response
    if firstError != nil && atomic {
        // Atomic mode: Return error for all
        errorResponses := make([]BatchMessage, len(responses))
        for i := range responses {
            errorResponses[i] = BatchMessage{
                CorrelID: responses[i].CorrelID,
                Pattern:  "response",
                Type:    "error",
                Payload: map[string]interface{}{
                    "error": firstError.Error(),
                },
            }
        }
        responses = errorResponses
    }

    batchResp := &Message{
        ID:      generateID(),
        Version: "1.1",
        From:    m.profileID,
        To:      batch.From,
        Pattern: PatternBatch,
        Type:    "multiple_responses",
        Payload: map[string]interface{}{
            "messages": responses,
        },
        Metadata: map[string]string{
            "batch_size":      strconv.Itoa(len(responses)),
            "all_succeeded":  strconv.FormatBool(firstError == nil),
        },
        Timestamp: time.Now().UnixNano(),
    }

    return batchResp, nil
}
```

---

## Consequences

### Positive

- **Reduced Round-Trips**: Multiple messages in 1 transport operation
- **Lower Latency**: Network overhead amortized across batch
- **Atomic Processing**: All succeed or all fail (optional)
- **Flexible**: Supports mixed patterns within batch
- **Efficient**: Better for high-throughput scenarios
- **Backward Compatible**: v1.0 clients unaffected

### Negative

- **Complexity**: Additional batch processing logic
- **Memory**: Must hold all batch messages in memory
- **Partial Failure**: Non-atomic mode: some succeed, some fail
- **Debugging**: Harder to debug batch vs. individual messages
- **Ordering**: Batch ordering doesn't guarantee per-message order
- **Error Handling**: Must correlate batch responses to batch requests

---

## Alternatives Considered

### 1. Continue v1.0 (No Batching) ❌

**Approach**: Don't add batching, keep v1.0 simple

**Pros**:
- **Simplest**: No new features
- **Consistent**: No protocol changes
- **No Complexity**: Existing v1.0 code unchanged

**Cons**:
- **High Latency**: Multiple round-trips for multiple messages
- **Overhead**: Repeated transport handshakes
- **Inefficient**: Network bandwidth underutilized

**Example Inefficiency**:
```
v1.0: Deploy 10 applications
  Message 1: Deploy app1
  Network: 10ms latency
  Message 2: Deploy app2
  Network: 10ms latency
  ...
  Message 10: Deploy app10
  Network: 10ms latency

Total: 10 messages × 10ms = 100ms latency
Total overhead: 10 × TCP handshake + HTTP headers

v1.1: Batch deploy 10 applications
  Batch message (10 deployments)
  Network: 10ms latency
  Total: 10ms latency (10x faster!)
```

**Rejection**: High latency, inefficient for use case

---

### 2. Client-Side Batching (No Protocol Support) ❌

**Approach**: Clients batch messages, send as array (not standard)

**Non-Standard Implementation**:
```json
{
  "id": "client-batch-001",
  "version": "1.0",
  "from": "orchestrator",
  "to": ["worker"],
  "pattern": "request",
  "type": "client_batch",  // Custom, non-standard
  "payload": {
    "messages": [...]  // Hack: using payload for batch
  }
}
```

**Pros**:
- **Immediate**: Can use in v1.0 without protocol change
- **Client Control**: Client decides how to batch

**Cons**:
- **Non-Standard**: No protocol-level batch support
- **Inconsistent**: Different clients may batch differently
- **No Atomicity**: Server processes individually, not atomically
- **Type Hacking**: Misuses `type` field for batch
- **No Standard Response**: Response format unclear

**Example Issues**:
```
Client sends "client_batch" (non-standard):
  Server: Doesn't understand "client_batch" type
  Server: Rejects message (400 Bad Request)
  Result: Batching doesn't work

Alternative: Server accepts but processes individually:
  - No atomicity
  - Responses: How to return? Array? Multiple messages?
  - Inconsistent: Each client has different approach
```

**Rejection**: Non-standard, inconsistent, no atomicity

---

### 3. Pipeline Batching (Over HTTP/2) ❌

**Approach**: Use HTTP/2 multiplexing for batching

**Implementation**:
```
HTTP/2 Request:
  Stream multiple messages in single connection
  Server processes each message individually
  Responses: Stream back over same connection
```

**Pros**:
- **Built-In**: HTTP/2 supports multiplexing
- **Standard**: No custom protocol changes
- **Efficient**: Single TCP connection for multiple requests

**Cons**:
- **Transport-Specific**: Only works for HTTP transport
- **No Atomicity**: Server processes individually
- **Protocol Hidden**: Batching at transport layer, not protocol level
- **IPC/WebSocket**: Doesn't benefit (HTTP/2 only)
- **Implementation**: Requires HTTP/2 support

**Example Limitation**:
```
IPC transport:
  - No HTTP/2 support
  - Can't pipeline batches
  - Falls back to individual messages

Result: Inconsistent batching across transports
```

**Rejection**: Transport-specific, no protocol-level atomicity

---

### 4. Transactional Batching (Two-Phase Commit) ❌

**Approach**: Use two-phase commit for atomic batches

**Implementation**:
```
Phase 1: Prepare
  Client: Send batch with "prepare" flag
  Server: Validate all messages, lock resources

Phase 2: Commit
  Client: Send "commit" message
  Server: Execute all messages
  Or Rollback: Send "rollback" message
```

**Pros**:
- **True Atomicity**: All or nothing (two-phase commit)
- **Distributed**: Works across multiple servers

**Cons**:
- **Most Complex**: Two-phase commit is very complex
- **State Management**: Server must hold locks between prepare/commit
- **Latency**: Requires 2 round-trips (prepare + commit)
- **Timeout**: Locks must timeout if commit never arrives
- **Rollback Complexity**: Need to rollback partial state

**Example Complexity**:
```
Batch: Deploy 10 apps (atomic)
Phase 1: Prepare
  - Server validates all 10 deployments
  - Server locks all 10 targets
  - If any invalid → Return error, unlock

Phase 2: Commit
  - Client receives prepare success
  - Client sends commit
  - Server executes all 10 deployments
  - Server unlocks all 10 targets

Failure Scenario:
  - Phase 1: Prepare success, locks held
  - Client crashes (never sends commit)
  - Server timeout: 30 seconds → Rollback, unlock
  - Complexity: Timeout management, rollback logic
```

**Rejection**: Overkill for v1.1, two-phase commit very complex

---

### 5. Per-Message Batching (No Atomicity) ❌

**Approach**: Batch multiple messages, process independently (no atomicity)

**Implementation**:
```go
func (m *Manager) ProcessBatch(batch *Message) (*Message, error) {
    // Process each message independently
    var responses []BatchMessage

    for _, batchMsg := range batch.Payload.Messages {
        response, err := m.processMessage(batchMsg)
        responses = append(responses, response)
        // Continue even if error (no atomicity)
    }

    return buildBatchResponse(responses), nil
}
```

**Pros**:
- **Simple**: No atomicity logic
- **Partial Success**: Some messages succeed even if others fail
- **Flexible**: Independent message processing

**Cons**:
- **No Atomicity**: Can't ensure all succeed together
- **Error Handling**: Caller must check each response
- **Use Case Limitations**: Can't use for transactions

**Example Issue**:
```
Batch: Deploy 10 apps (transaction)
Message 1: Deploy app1 → Success
Message 2: Deploy app2 → Failure (config error)
Message 3: Deploy app3 → Success
...
Message 10: Deploy app10 → Success

Result: Partial success (9/10 deployed)
Problem: Intended transaction (all or nothing) failed
Use case: Can't use batching for transactions
```

**Rejection**: No atomicity, doesn't meet transaction use case

---

### 6. Per-Conversation Batching ❌

**Approach**: Batch all messages for a conversation into single transport operation

**Implementation**:
```go
func (m *Manager) SendConversationMessages(convID string) error {
    // 1. Get all pending messages for conversation
    messages := m.storage.GetPendingMessages(convID)

    // 2. Batch send
    batch := buildBatch(messages)
    return m.transport.Send(batch)
}
```

**Pros**:
- **Efficient**: All conversation messages sent together
- **Simpler**: No per-batch logic, per-conversation only

**Cons**:
- **Loss of Granularity**: Can't control batching per message
- **Delay**: Must wait for multiple messages to accumulate
- **Inconsistent**: Different batch sizes per conversation
- **Hard to Debug**: Conversation-level batching harder to debug

**Example Delay**:
```
Conversation messages:
  - 10:00:00 - Message 1 (send now?)
  - 10:00:05 - Message 2 (wait for more?)
  - 10:00:10 - Message 3 (wait for more?)
  ...
  - 10:01:00 - Message 10 (send batch now?)

Trade-off: Wait for batch vs. send immediately
Result: Uncertain batch timing, hard to predict
```

**Rejection**: Unpredictable batching, loses control

---

## Batch Pattern Comparison

| Approach | Atomicity | Complexity | Latency | Use Case |
|----------|-----------|-------------|---------|----------|
| **Per-Message Batching** ❌ | No | Low | Reduced | Fire-and-forget |
| **Atomic Batching** ✅ | Yes | Medium | Reduced | Transactions |
| **Transactional Batching** ❌ | Yes (2PC) | High | Increased | Distributed |
| **Per-Conversation Batching** ❌ | No | Medium | Variable | Conversation-level |
| **HTTP/2 Pipelining** ❌ | No | Low | Reduced | HTTP-only |

---

## Atomic Mode

### Atomic Flag

```json
{
  "metadata": {
    "atomic": "true"  // All succeed or all fail
  }
}
```

### Atomic Behavior

**Success**:
```
Batch: 3 messages
  Message 1: Success
  Message 2: Success
  Message 3: Success
Result: All succeed, return all responses
```

**Failure**:
```
Batch: 3 messages
  Message 1: Success
  Message 2: Failure (error)
  Message 3: Not processed (stopped after first error)
Result: All responses have error from Message 2
```

### Non-Atomic Mode

```json
{
  "metadata": {
    "atomic": "false"  // Process independently
  }
}
```

### Non-Atomic Behavior

```
Batch: 3 messages
  Message 1: Success → Response 1: Success
  Message 2: Failure → Response 2: Error
  Message 3: Success → Response 3: Success
Result: Mixed success/failure, caller handles
```

---

## Batch Size Limits

### Max Batch Size

**Default**: 100 messages per batch

**Configurable**:
```yaml
# APS global config
a2a:
  max_batch_size: 100
```

### Rationale

- **Memory**: Must hold all batch messages in memory
- **Latency**: Large batches increase processing time
- **Network**: Large batches may timeout

**Enforcement**:
```go
func (m *Manager) ValidateBatch(batch *Message) error {
    batchPayload, _ := batch.Payload.(map[string]interface{})
    messagesData, _ := batchPayload["messages"].([]interface{})

    if len(messagesData) > m.maxBatchSize {
        return fmt.Errorf("batch too large: %d > %d",
            len(messagesData), m.maxBatchSize)
    }

    return nil
}
```

---

## Backward Compatibility

### v1.0 Client → v1.1 Server

```
v1.0 Client: Sends single message
  {
    "pattern": "request",
    ...
  }

v1.1 Server: Receives v1.0 message
  - Processes normally (no batch)
  - Returns v1.0 response
Result: Compatible
```

### v1.1 Client → v1.0 Server

```
v1.1 Client: Sends batch message
  {
    "pattern": "batch",
    ...
  }

v1.0 Server: Receives v1.1 batch message
  - Doesn't understand "batch" pattern
  - Returns error: 400 Bad Request
  - Error: "Unsupported pattern: batch"
Result: v1.1 client falls back to v1.0 (individual messages)
```

### Version Negotiation

```go
func (c *Client) SupportsBatching(serverVersion string) bool {
    server := parseVersion(serverVersion)
    if server.Major > 1 {
        return true  // v2.0+ supports batching
    }
    if server.Major == 1 && server.Minor >= 1 {
        return true  // v1.1+ supports batching
    }
    return false  // v1.0 doesn't support batching
}
```

---

## Performance Benefits

### Latency Reduction

**Scenario**: Send 10 messages

**v1.0 (No Batching)**:
```
10 messages × 10ms latency = 100ms
10 × TCP handshake = 50ms overhead
10 × HTTP headers = 100ms overhead
Total: 250ms
```

**v1.1 (Batching)**:
```
1 batch × 10ms latency = 10ms
1 × TCP handshake = 5ms overhead
1 × HTTP headers = 10ms overhead
Total: 25ms (10x faster)
```

### Throughput Improvement

**Scenario**: Process 1000 messages/second

**v1.0**:
```
1000 messages/second
Network: 10ms/message
CPU: 1ms/message
Throughput: ~90 messages/second (bottleneck: network)
```

**v1.1 (Batch Size: 10)**:
```
100 batches/second (1000/10)
Network: 10ms/batch
CPU: 10ms/batch (10×1ms)
Throughput: ~500 messages/second (5x improvement)
```

---

## Migration Path

### v1.0 → v1.1

**Phase 1: Development** (Months 1-2)
- Implement batch pattern
- Add batch processing logic
- Write tests for atomic/non-atomic modes

**Phase 2: Beta** (Months 3-4)
- Release v1.1-beta
- Deploy to test environment
- Gather performance metrics
- Benchmark v1.0 vs. v1.1

**Phase 3: GA** (Month 5)
- Release v1.1-GA
- Update documentation
- Provide migration guide
- v1.0 clients continue working

**Phase 4: Deprecation** (Months 6-12)
- Encourage migration to v1.1+
- v1.0 clients still supported
- Deprecate v1.0 in v1.2

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (compatible with v1.1)
- **ADR-003**: UUID v4 for message IDs (used in batch messages)

---

## References

- **Specification**: `spec.md` - Planned Future Features section
- **Decisions Document**: `decisions.md` - Question #8

---

## Revisions

- 2026-01-20: Initial decision - Batch messages for v1.1
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added performance benefits and migration path
