# ADR-007: Per-Message and Per-Conversation Expiration

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

A2A protocol messages may be temporary or sensitive. Storage of all messages indefinitely has privacy and storage implications.

**Use Cases for Expiration**:
- Temporary status updates (no long-term value)
- Real-time notifications (no history needed)
- Sensitive data (auto-delete after retention period)
- Regulatory compliance (data retention policies)
- Storage optimization (cleanup of old data)

**Requirements**:
- Support optional per-message expiration
- Support conversation-level expiration policies
- Automatic cleanup of expired messages
- Preserve important messages (no forced expiration)
- Configurable expiration times
- Transparent to users

---

## Decision

**Implement optional per-message and per-conversation expiration**

### Implementation Details

#### Per-Message Expiration

**Field**: `expires_at` (nanosecond timestamp)

**Message with Expiration**:
```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "status",
  "payload": {"query": "current"},
  "expires_at": 1705773660000000000,  // 60 seconds from now
  "timestamp": 1705773600000000000
}
```

#### Per-Conversation Expiration

**Field**: `settings.message_ttl_seconds`

**Conversation Metadata**:
```json
{
  "id": "conv-123",
  "settings": {
    "message_ttl_seconds": 86400  // Messages expire after 24 hours
  }
}
```

### Expiration Logic

```go
// Check if message is expired
func IsMessageExpired(msg *Message) bool {
    // 1. Check per-message expiration
    if msg.ExpiresAt > 0 {
        return time.Now().UnixNano() > msg.ExpiresAt
    }

    // 2. Check conversation expiration
    conv, err := LoadConversation(msg.ConversationID)
    if err != nil {
        return false  // Can't determine, don't expire
    }

    if conv.Settings.MessageTTLSeconds > 0 {
        expiration := msg.Timestamp + (conv.Settings.MessageTTLSeconds * 1e9)
        return time.Now().UnixNano() > expiration
    }

    // 3. No expiration configured
    return false
}

// Should deliver message?
func ShouldDeliverMessage(msg *Message) bool {
    if IsMessageExpired(msg) {
        log.Printf("Message expired: %s (expires_at: %d)",
            msg.ID, msg.ExpiresAt)
        return false
    }
    return true
}
```

### Cleanup Process

```go
// Periodic cleanup
type CleanupManager struct {
    interval  time.Duration
    storage   Storage
    stopChan  chan struct{}
}

func (cm *CleanupManager) Start() {
    ticker := time.NewTicker(cm.interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                cm.cleanupExpiredMessages()
            case <-cm.stopChan:
                return
            }
        }
    }()
}

func (cm *CleanupManager) cleanupExpiredMessages() {
    // 1. Get all conversations
    conversations, err := cm.storage.ListConversations()
    if err != nil {
        log.Printf("Cleanup failed: %v", err)
        return
    }

    // 2. Iterate conversations
    for _, conv := range conversations {
        // 3. Get messages
        messages, err := cm.storage.GetMessages(conv.ID, MessageListOptions{})
        if err != nil {
            continue
        }

        // 4. Delete expired messages
        deleted := 0
        for _, msg := range messages {
            if IsMessageExpired(msg) {
                err := cm.storage.DeleteMessage(conv.ID, msg.ID)
                if err == nil {
                    deleted++
                }
            }
        }

        // 5. Update conversation metadata
        if deleted > 0 {
            conv.MessageCount -= deleted
            cm.storage.UpdateConversation(conv)
            log.Printf("Cleaned up %d expired messages from conversation %s",
                deleted, conv.ID)
        }
    }
}
```

---

## Consequences

### Positive

- **Flexible**: Senders choose expiration per message
- **User-Controlled**: Profiles can configure retention policies
- **Automatic Cleanup**: Periodic process removes expired messages
- **Storage Optimization**: Reduces long-term storage requirements
- **Privacy**: Sensitive data auto-deleted
- **Compliance**: Supports regulatory retention requirements

### Negative

- **Data Loss**: Expired messages cannot be recovered
- **Complexity**: Requires expiration checking and cleanup logic
- **Performance**: Periodic cleanup adds overhead
- **Surprise**: Users may not realize messages expire
- **Configuration**: Users must understand TTL settings
- **Storage**: Cleanup may be resource-intensive

---

## Alternatives Considered

### 1. No Expiration (Indefinite Retention) ❌

**Approach**: Never expire messages, store indefinitely

**Pros**:
- **Simplest**: No expiration logic
- **Complete History**: All messages preserved
- **No Data Loss**: No automatic deletion

**Cons**:
- **Unlimited Storage**: Storage grows indefinitely
- **Privacy Risk**: Sensitive data never deleted
- **Compliance Issues**: May violate retention policies
- **Cost**: Storage costs increase over time
- **Performance**: Large conversation directories slow

**Example Impact**:
```
100 messages/day × 365 days = 36,500 messages/year
5 KB average size × 36,500 = 182.5 MB/year
10 years = 1.8 GB per conversation
```

**Rejection**: Storage and privacy implications significant

---

### 2. Global Default Expiration ❌

**Approach**: All messages expire after fixed time (e.g., 30 days)

**Configuration**:
```yaml
# Global APS config
a2a:
  default_ttl_days: 30  # All messages expire after 30 days
```

**Pros**:
- **Simple**: Single configuration
- **Predictable**: Clear behavior
- **Automated**: No user decision needed

**Cons**:
- **Inflexible**: Can't keep important messages longer
- **One Size Fits All**: Doesn't suit all use cases
- **Data Loss**: All messages eventually deleted
- **User Confusion**: Users may not expect deletion

**Example Issues**:
```
Important project conversation:
  - 100 critical messages
  - Expired after 30 days
  - User loses context

Better approach:
  - Per-conversation TTL: 365 days for projects
  - Per-conversation TTL: 7 days for status updates
```

**Rejection**: Too inflexible, doesn't adapt to use cases

---

### 3. Manual Expiration Only ❌

**Approach**: Users manually delete old messages

**Implementation**:
```bash
# CLI command
aps a2a cleanup <conversation-id> --before 2026-01-01
```

**Pros**:
- **User Control**: Users decide what to delete
- **No Surprises**: No automatic deletion
- **Flexible**: Any cleanup strategy

**Cons**:
- **Manual Effort**: Users must remember to cleanup
- **Forgotten**: Users forget, storage grows
- **Compliance Risk**: May violate retention policies
- **Inconsistent**: Different users, different cleanup habits
- **Error-Prone**: Manual cleanup may miss messages

**Example Scenario**:
```
User creates conversation for alerts:
  - 1,000 alert messages
  - Intends to delete after 7 days
  - Forgets to cleanup
  - 6 months later: 26,000 messages stored
  - Storage, performance, privacy issues
```

**Rejection**: Relies on user discipline, not reliable

---

### 4. Hard Delete (Immediate Removal) ❌

**Approach**: When message expires, delete immediately from storage

**Implementation**:
```go
func DeliverMessage(msg *Message) {
    if IsMessageExpired(msg) {
        DeleteMessage(msg.ConversationID, msg.ID)
        return  // Don't deliver
    }

    // Deliver message
    handler.Process(msg)
}
```

**Pros**:
- **Clean**: Expired messages removed immediately
- **No Confusion**: Expired messages never seen
- **Storage Efficient**: No expired messages stored

**Cons**:
- **Race Condition**: Check-expire-then-delete not atomic
- **Performance**: Delete on every message delivery
- **Overhead**: Frequent file deletions
- **Complexity**: More complex delivery logic

**Example Race Condition**:
```
Time T0: Check if expired → No
Time T1: Deliver message to handler
Time T2: Message expires
Time T3: Handler processes (shouldn't have processed)
Time T4: Delete message (already processed, confusing)
```

**Rejection**: Race conditions, performance overhead

---

### 5. Soft Delete + Periodic Cleanup ✅ (CHOSEN)

**Approach**: Mark expired messages, cleanup periodically

**Implementation**:
```go
// Mark expired (doesn't delete)
func MarkExpired(msg *Message) {
    msg.Status = "expired"
    storage.UpdateMessage(msg)
}

// Periodic cleanup
func PeriodicCleanup() {
    for _, msg := range storage.GetExpiredMessages() {
        storage.DeleteMessage(msg.ID)
    }
}
```

**Pros**:
- **Atomic**: Mark and delete separate phases
- **Efficient**: Cleanup batched, not per-message
- **Reliable**: Periodic job ensures cleanup
- **Performance**: Less frequent file operations

**Cons**:
- **Delay**: Expired messages visible until cleanup runs
- **Complexity**: Need periodic job scheduler

**Delay Trade-off**:
```
Cleanup Interval: 1 hour
Message Expires: T0
Marked Expired: T0
Deleted: T0 + 1 hour
Visible Duration: 1 hour
```

**Acceptable**: 1-hour delay reasonable for cleanup

---

### 6. Per-Topic Expiration ❌

**Approach**: All messages in a topic expire after fixed time

**Configuration**:
```yaml
# Topic-specific expiration
topics:
  deployments:
    ttl_days: 30
  alerts:
    ttl_days: 7
```

**Pros**:
- **Topic-Aware**: Different TTL for different topics
- **Configurable**: Fine-grained control

**Cons**:
- **Topic-Only**: Doesn't work for conversations
- **Complex Configuration**: Need per-topic config
- **Not Universal**: Only works for pub/sub

**Example Issues**:
```
Topic: alerts (TTL: 7 days)
  - Important alert on Day 0
  - Expired on Day 7
  - User needed it for longer

Better: Per-message expiration
  - Alert on Day 0
  - expires_at: Day 30 (user configurable)
```

**Rejection**: Limited to topics, not flexible enough

---

## Expiration Comparison

| Approach | Flexibility | Automation | Complexity | User Control |
|----------|-------------|-------------|------------|--------------|
| **No Expiration** ❌ | N/A | N/A | Low |
| **Global Default** ❌ | Low | High | None |
| **Manual Only** ❌ | High | None | High |
| **Hard Delete** ❌ | Medium | High | Medium |
| **Soft Delete + Cleanup** ✅ | High | High | High |
| **Per-Topic** ❌ | Low | High | Medium |

---

## Expiration Priority

**Priority Order** (highest to lowest):
1. **Per-Message Expiration**: `expires_at` field
2. **Per-Conversation Expiration**: `settings.message_ttl_seconds`
3. **Global Default**: APS configuration (optional)
4. **Indefinite**: No expiration configured

**Implementation**:
```go
func GetExpiration(msg *Message, conv *Conversation) int64 {
    // 1. Per-message
    if msg.ExpiresAt > 0 {
        return msg.ExpiresAt
    }

    // 2. Per-conversation
    if conv.Settings.MessageTTLSeconds > 0 {
        return msg.Timestamp + (conv.Settings.MessageTTLSeconds * 1e9)
    }

    // 3. Global default (optional)
    if globalDefaultTTL > 0 {
        return msg.Timestamp + (globalDefaultTTL * 1e9)
    }

    // 4. Indefinite
    return 0  // No expiration
}
```

---

## Cleanup Configuration

### Cleanup Interval

**Default**: 1 hour

**Configurable**:
```yaml
# APS global config
a2a:
  cleanup_interval_hours: 1  # Run cleanup every hour
```

### Cleanup Throttling

**Per-Cleanup Limits**:
- Max messages: 10,000 per cleanup run
- Max time: 5 minutes per cleanup run

**Rationale**: Prevent cleanup from overwhelming system

---

## Recommended Expiration Times

| Use Case | Recommended TTL | Rationale |
|-----------|------------------|------------|
| **Status Updates** | 1 hour | Temporary, no history needed |
| **Notifications** | 24 hours | Short-term relevance |
| **Alerts** | 7 days | Medium-term relevance |
| **Project Conversations** | 365 days | Long-term project work |
| **Critical Decisions** | Indefinite | Important, keep forever |
| **Compliance Data** | Per policy | Legal/regulatory requirements |

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (expiration in JSON metadata)
- **ADR-006**: Message references (complementary approach for large data)

---

## References

- **Specification**: `spec.md` - Message Expiration section
- **Decisions Document**: `decisions.md` - Question #6

---

## Revisions

- 2026-01-20: Initial decision - Per-message and per-conversation expiration
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added cleanup configuration and recommended TTLs
