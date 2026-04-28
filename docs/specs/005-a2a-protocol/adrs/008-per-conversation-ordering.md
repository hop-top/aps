# ADR-008: Per-Conversation Message Ordering

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

A2A protocol messages need ordering for conversation history and sequence tracking.

**Use Cases for Ordering**:
- Conversation history (chronological display)
- Debugging (sequence of events)
- Request/response correlation (though done via `correl_id`)
- Replay capabilities (reconstruct conversation timeline)

**Requirements**:
- Support message ordering within conversations
- Maintain ordering across storage and retrieval
- Handle message collisions (same timestamp)
- Balance simplicity with sufficient ordering guarantees
- Support high-throughput scenarios (no bottlenecks)

---

## Decision

**Implement per-conversation message ordering (no global ordering)**

### Implementation Details

#### Ordering Mechanism

1. **Per-Conversation Only**: Messages ordered within each conversation independently
2. **Timestamp-Based Primary**: Ordered by `timestamp` field (nanoseconds)
3. **UUID-Based Secondary**: If timestamps collide, order by `id` (UUID v4)
4. **No Global Ordering**: No guarantee of order across different conversations

#### Message Filenames

**Format**: `<timestamp>_<uuid>.json`

**Examples**:
```
1705773600000000001_550e8400-e29b-41d4-a716-446655440000.json
1705773600000000002_6ba7b810-9dad-11d1-80b4-00c04fd430c8.json
1705773600000000002_7d444860-9dad-11d1-b45c-00c04fd430c8.json  // Same timestamp, different UUID
```

#### Retrieval Logic

```go
// Get conversation messages (always ordered)
func (s *Storage) GetMessages(convID string, opts MessageListOptions) ([]*Message, error) {
    // 1. Get all message files
    msgDir := filepath.Join(s.baseDir, convID, "messages")
    entries, err := os.ReadDir(msgDir)
    if err != nil {
        return nil, err
    }

    // 2. Load all messages
    var messages []*Message
    for _, entry := range entries {
        data, err := os.ReadFile(filepath.Join(msgDir, entry.Name()))
        if err != nil {
            continue
        }

        var msg Message
        if err := json.Unmarshal(data, &msg); err != nil {
            continue
        }

        messages = append(messages, &msg)
    }

    // 3. Sort by timestamp, then by ID
    sort.Slice(messages, func(i, j int) bool {
        if messages[i].Timestamp != messages[j].Timestamp {
            return messages[i].Timestamp < messages[j].Timestamp
        }
        return messages[i].ID < messages[j].ID
    })

    // 4. Apply pagination
    if opts.Limit > 0 {
        start := opts.Offset
        end := start + opts.Limit
        if end > len(messages) {
            end = len(messages)
        }
        if start >= len(messages) {
            return []*Message{}, nil
        }
        return messages[start:end], nil
    }

    return messages, nil
}
```

#### Collision Handling

**Example Collision Scenario**:
```
Time: 1705773600.000000001
Message A: id="abc123", timestamp=1705773600000000001
Message B: id="def456", timestamp=1705773600000000001
Message C: id="ghi789", timestamp=1705773600000000002

Sorted Order:
1. Message A (timestamp: 1705773600000000001, id: abc123)
2. Message B (timestamp: 1705773600000000001, id: def456)  // Same time, sorted by ID
3. Message C (timestamp: 1705773600000000002)
```

**Collision Probability**:
- **Nanosecond precision**: 1 billion timestamps per second
- **High Throughput**: 1000 messages/second = 0.001% collision probability
- **UUID Secondary**: UUID ensures deterministic ordering when collision occurs

---

## Consequences

### Positive

- **Simple**: No complex global coordination
- **Sufficient**: Most use cases only need ordering within conversations
- **Scalable**: No bottlenecks from global ordering
- **Decoupled**: Conversations ordered independently
- **No Coordination**: No need for distributed sequence numbers
- **Fast**: Sorting O(n log n) per conversation (not global)

### Negative

- **No Global Order**: Can't determine order across different conversations
- **Clock Dependency**: Relies on system clock accuracy
- **Clock Skew**: Different machines may have different times
- **Collision Handling**: Rare but possible (nanosecond precision)

---

## Alternatives Considered

### 1. Global Ordering with Sequence Numbers ❌

**Approach**: Single global sequence number across all messages

**Implementation**:
```go
// Global sequence counter (atomic)
var globalSequence uint64

func GenerateMessage() *Message {
    seq := atomic.AddUint64(&globalSequence, 1)
    return &Message{
        Sequence: seq,  // Global ordering
    }
}
```

**Pros**:
- **Global Order**: All messages ordered globally
- **No Clock Dependency**: Doesn't rely on timestamps
- **Simple Comparison**: Compare sequence numbers
- **No Collisions**: Guaranteed unique ordering

**Cons**:
- **Bottleneck**: Single global counter (can't scale)
- **Coordination**: Requires coordination across processes/machines
- **Storage**: Must persist sequence number (for recovery)
- **Single Point of Failure**: If counter lost, ordering breaks
- **Distributed Complexity**: Network profiles need shared counter

**Example Bottleneck**:
```
1000 messages/second
  → Must increment global counter 1000 times/sec
  → Single lock/atomic operation per message
  → CPU overhead
  → Scaling: Can't distribute across machines

Distributed scenario:
  Machine A: Sequence 1-1000
  Machine B: Sequence 1001-2000
  → Conflict: Who manages global counter?
  → Network: Additional latency for sequence allocation
```

**Rejection**: Bottleneck, coordination complexity, doesn't scale

---

### 2. Per-Conversation Sequence Numbers ❌

**Approach**: Each conversation has its own sequence counter

**Implementation**:
```go
type Conversation struct {
    ID      string
    NextSeq uint64  // Next sequence number
}

func (c *Conversation) GenerateMessage() *Message {
    seq := c.NextSeq
    c.NextSeq++  // Increment conversation's sequence
    return &Message{
        Sequence: seq,
        ConvID:   c.ID,
    }
}
```

**Pros**:
- **Per-Conversation Order**: Messages ordered within conversation
- **No Clock**: Doesn't rely on timestamps
- **Simple**: No global coordination
- **No Collisions**: Guaranteed unique per conversation

**Cons**:
- **State Management**: Must persist sequence number per conversation
- **Recovery**: Need to restore sequence from storage after restart
- **Complexity**: Additional field and persistence logic
- **Migration**: Existing messages need sequence numbers backfilled
- **Failure Risk**: If sequence lost, ordering breaks

**Example Recovery Issue**:
```
Conversation state:
  - Sequence: 42
  - Crash/Restart
  - Lost: Sequence number not persisted
  - Result: New messages start from 0 or 1
  - Ordering: Confusing (42 messages, then 0 again)

Recovery: Need to scan storage for max sequence
  - Performance: O(n) scan on startup
  - Complex: Need consistent state
```

**Rejection**: State management complexity, recovery issues

---

### 3. Timestamp-Only Ordering ❌

**Approach**: Order only by timestamp, no secondary key

**Implementation**:
```go
sort.Slice(messages, func(i, j int) bool {
    return messages[i].Timestamp < messages[j].Timestamp
})
```

**Pros**:
- **Simplest**: No secondary key needed
- **Natural**: Chronological ordering

**Cons**:
- **Collision Risk**: Same timestamp = undefined order
- **Non-Deterministic**: Different sort results each time
- **Random**: Messages with same timestamp order randomly

**Example Collision**:
```
Time: 1705773600.000000001
Message A: timestamp=1705773600000000001
Message B: timestamp=1705773600000000001

Sort Attempt 1:
  Order: [A, B]  (random)

Sort Attempt 2:
  Order: [B, A]  (different!)
```

**Rejection**: Non-deterministic ordering, collision issues

---

### 4. Hybrid: Timestamp + Per-Conversation Sequence ❌

**Approach**: Both timestamp and sequence numbers

**Implementation**:
```go
type Message struct {
    Timestamp int64  // Nanoseconds
    Sequence  uint64  // Per-conversation
}

sort.Slice(messages, func(i, j int) bool {
    // Primary: Timestamp
    if messages[i].Timestamp != messages[j].Timestamp {
        return messages[i].Timestamp < messages[j].Timestamp
    }
    // Secondary: Sequence
    return messages[i].Sequence < messages[j].Sequence
})
```

**Pros**:
- **Best of Both**: Timestamp + sequence
- **Clock-Independent**: Sequence handles clock issues
- **Collision-Resistant**: Sequence handles same timestamp

**Cons**:
- **Most Complex**: Both timestamp and sequence management
- **State Management**: Must persist sequence numbers
- **Redundant**: Either timestamp or sequence sufficient
- **Storage Overhead**: Additional field per message
- **Recovery**: Need to restore sequences after restart

**Rejection**: Unnecessary complexity, either approach sufficient

---

### 5. Logical Clock (Vector Clocks) ❌

**Approach**: Use vector clocks for distributed ordering

**Implementation**:
```go
type VectorClock struct {
    Clocks map[string]int64  // Profile ID → logical time
}

func (vc *VectorClock) Increment(profileID string) {
    if _, exists := vc.Clocks[profileID]; !exists {
        vc.Clocks[profileID] = 0
    }
    vc.Clocks[profileID]++
}

func (vc *VectorClock) HappenedBefore(other *VectorClock) bool {
    // Vector clock comparison for happened-before
    // ...
}
```

**Pros**:
- **Distributed**: Works across machines without coordination
- **Causality**: Captures causal relationships
- **Clock-Independent**: No reliance on physical clocks

**Cons**:
- **Most Complex**: Significant complexity overhead
- **Overkill**: A2A doesn't need causal ordering
- **Storage Overhead**: Vector clock per message
- **Comparison Complexity**: Vector clock comparison is non-trivial
- **Hard to Understand**: Developers unfamiliar with vector clocks
- **Not Required**: Per-conversation ordering is sufficient

**Example Complexity**:
```
Message A: {profile-a: 1, profile-b: 0}
Message B: {profile-a: 1, profile-b: 1}

Comparison:
  - A happened before B?
  - Check: profile-a: 1 <= 1 (yes), profile-b: 0 <= 1 (yes)
  - Yes, A happened before B
  - Complexity: O(n) where n = number of profiles
```

**Rejection**: Overkill, too complex for use case

---

### 6. Lamport Clocks ❌

**Approach**: Logical time with Lamport timestamps

**Implementation**:
```go
type LamportClock struct {
    time uint64
}

func (lc *LamportClock) Tick() {
    lc.time++
}

func (lc *LamportClock) Receive(other uint64) {
    lc.time = max(lc.time, other) + 1
}
```

**Pros**:
- **Distributed**: Works across machines
- **Clock-Independent**: No physical clock reliance
- **Simpler**: Than vector clocks

**Cons**:
- **Complex**: More complex than timestamp
- **State Management**: Must persist Lamport clock
- **Causality Only**: Only partial order, no total order
- **Overkill**: Per-conversation ordering is sufficient
- **Not Standard**: Not common in chat/messaging systems

**Example Partial Order**:
```
Message A: Lamport time = 5
Message B: Lamport time = 5

Ordering:
  - Neither A happened before B, nor B happened before A
  - Concurrent (same Lamport time)
  - Need additional tie-breaker
```

**Rejection**: Complex, overkill, not standard

---

## Ordering Comparison

| Approach | Global Order | Clock Dependency | Complexity | Scalability | Collisions |
|----------|---------------|-------------------|------------|--------------|------------|
| **Per-Conversation (Timestamp+UUID)** ✅ | No | Yes (nanosecond) | Low | High | Resolved by UUID |
| **Global Sequence** ❌ | Yes | No | Medium | Low | None |
| **Per-Conversation Sequence** ❌ | No | No | Medium | Medium | None |
| **Timestamp Only** ❌ | No | Yes | Lowest | High | Unresolved |
| **Timestamp+Sequence** ❌ | No | No | High | Medium | Resolved |
| **Vector Clocks** ❌ | Partial | No | Very High | Low | Partial |
| **Lamport Clocks** ❌ | Partial | No | High | Medium | Partial |

---

## Clock Considerations

### Clock Skew

**Issue**: Different machines have different times

**Impact**:
```
Machine A: 2026-01-20 10:00:00.000
Machine B: 2026-01-20 10:00:05.000  (5 seconds ahead)

Conversation on Machine A:
  - Message A (10:00:00.000)
  - Message B (10:00:00.001)
  - Message from Machine B (10:00:05.000 from B's perspective)

Ordering within A's conversation:
  - Message A (10:00:00.000)
  - Message B (10:00:00.001)
  - Message from B (10:00:05.000 on A's clock = 10:00:00.002)
  - Correct order preserved
```

**Conclusion**: Clock skew doesn't break per-conversation ordering (each machine orders its own messages)

### Nanosecond Precision

**Collision Probability**:
```
Nanosecond timestamps: 1e9 per second

High throughput: 1000 messages/second
Collision probability: 1000 / 1e9 = 0.0001%
Expected collisions: 1 per 1 million messages
```

**Mitigation**: UUID secondary key ensures deterministic ordering

---

## Pub/Sub Ordering

### Topic-Level Ordering

**Best-Effort Ordering**:
- Messages delivered to subscribers in order received at topic
- No guarantee of order across different subscribers
- No global topic ordering

**Implementation**:
```go
type Topic struct {
    Queue []*Message  // FIFO queue
}

func (t *Topic) Publish(msg *Message) {
    t.Queue = append(t.Queue, msg)  // Append to queue
}

func (t *Topic) Deliver(handler MessageHandler) {
    for _, msg := range t.Queue {
        handler(msg)  // Deliver in queue order
    }
}
```

**Subscriber Ordering**:
- Each subscriber receives messages in order
- Different subscribers may have different delivery speeds
- No synchronization across subscribers

**Example**:
```
Topic: deployments
Messages: [A, B, C]

Subscriber X receives:
  - A (10:00:00)
  - B (10:00:01)
  - C (10:00:02)

Subscriber Y receives:
  - A (10:00:00)
  - B (10:00:02)  # Delayed
  - C (10:00:03)

Order preserved per subscriber, not globally
```

---

## Request/Response Ordering

**No FIFO Guarantee**:
- Requests not guaranteed to be processed in order
- Responses matched via `correl_id`, not sequence

**Example**:
```
Conversation:
  - Request 1 (correl_id: abc)
  - Request 2 (correl_id: def)
  - Request 3 (correl_id: ghi)

Processing Order (async):
  - Process Request 3 → Response 3 (ghi)
  - Process Request 1 → Response 1 (abc)
  - Process Request 2 → Response 2 (def)

Conversation History (timestamp order):
  - Request 1 (10:00:00.000)
  - Request 2 (10:00:00.001)
  - Request 3 (10:00:00.002)
  - Response 3 (10:00:01.000)
  - Response 1 (10:00:01.001)
  - Response 2 (10:00:01.002)

Ordering preserved by timestamp, not by request order
```

**Conclusion**: Clients must match responses to requests via `correl_id`, not assume FIFO

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (timestamp in JSON)
- **ADR-003**: UUID v4 for message IDs (secondary ordering key)

---

## References

- **Specification**: `spec.md` - Message Ordering section
- **Decisions Document**: `decisions.md` - Question #7

---

## Revisions

- 2026-01-20: Initial decision - Per-conversation message ordering
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added clock considerations
