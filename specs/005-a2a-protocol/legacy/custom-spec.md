# A2A Protocol Specification

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-20

---

## Overview

A2A (Agent-to-Agent) is a communication protocol enabling isolated APS profiles to communicate through defined message patterns while maintaining strict isolation boundaries.

### Key Properties
- **Transport-Agnostic**: Works over IPC, HTTP, and WebSocket
- **Pattern-Rich**: Supports request/response, pub/sub, and conversations
- **History-Aware**: Full conversation tracking
- **Isolation-Respecting**: Enforces strict separation between profiles
- **Extensible**: Versioned protocol for evolution

---

## Message Format

### Message Serialization

**Protocol Version 1.0**: JSON encoding
- **Rationale**: Human-readable, widely supported, easy debugging, no external dependencies
- **Format**: UTF-8 encoded JSON with canonical sorting
- **Limitations**: Verbose, slower parsing than binary formats

**Protocol Version 2.0 (Planned)**: Protocol Buffers
- **Rationale**: High performance, compact size, strong typing, code generation
- **Migration Path**: Dual support for v1.0 (JSON) and v2.0 (Protobuf) during transition
- **Backward Compatibility**: Version 1.0 clients continue to use JSON; v2.0+ use Protobuf

### Base Message Structure

All A2A messages are JSON objects with the following structure:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "task",
  "payload": {"command": "deploy"},
  "metadata": {"priority": "high"},
  "timestamp": 1705773600000000000,
  "expires_at": null,
  "signature": null
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|-----------|-------------|
| `id` | string (UUID) | Yes | Unique message identifier (UUID v4) |
| `version` | string | Yes | Protocol version (semantic versioning) |
| `from` | string | Yes | Source profile ID |
| `to` | array of strings | Yes | Target profile IDs (1 or more) |
| `pattern` | string | Yes | Message pattern (see Patterns section) |
| `type` | string | Yes | Message type (application-defined) |
| `payload` | any | Yes | Message content (any JSON value) |
| `metadata` | object | No | Additional key-value metadata |
| `timestamp` | integer (nanoseconds) | Yes | Unix timestamp (UTC) |
| `expires_at` | integer (nanoseconds) | No | Optional expiration timestamp |
| `correl_id` | string | No | Correlation ID for request/response pairs |
| `topic` | string | No | Topic for pub/sub messages |
| `signature` | string | No | HMAC-SHA256 signature (hex encoded) |

### Message Size and Compression

**Maximum Message Size**: 1 MB (1,048,576 bytes)

**Recommendations**:
- Keep messages small for optimal performance
- Use message references for large data (see below)

**Large Payloads (> 1 MB)**:

For messages exceeding 1 MB, use message references:

**Example - Large File Transfer**:
```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "file_transfer",
  "payload": {
    "reference": true,
    "uri": "file:///shared/large-file.zip",
    "size": 52428800,
    "checksum": "sha256:abc123..."
  },
  "timestamp": 1705773600000000000
}
```

**Reference Schemes**:
- `file://<path>` - Shared filesystem path
- `http://<url>` - HTTP URL to download
- `s3://<bucket>/<key>` - S3 object reference
- `data://<base64>` - Inline base64 data (for medium-sized payloads)

**Compression**:

Compression is optional and specified per-message:

**Request with Compression**:
```json
{
  "id": "msg-002",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "log_data",
  "metadata": {
    "compression": "gzip",
    "uncompressed_size": 524288
  },
  "payload": "H4sIAAAAAAAA//8z+3KMTIyND...", // Compressed base64
  "timestamp": 1705773600000000000
}
```

**Supported Compression**:
- `gzip` - RFC 1952 compression
- `deflate` - RFC 1951 compression
- `identity` - No compression (default)

**Compression Requirements**:
- `metadata.compression`: Compression algorithm (if used)
- `metadata.uncompressed_size`: Size before compression (required if compressed)
- Payload: Base64-encoded compressed data

**When to Use Compression**:
- Text-heavy payloads (logs, JSON documents)
- Repetitive data
- Messages > 100 KB (recommended threshold)

---

## Message Patterns

### Pattern: `request`

A synchronous request from one profile to another.

**Required Fields**:
- `from`: Sender profile ID
- `to`: Target profile ID (single)
- `pattern`: `"request"`
- `type`: Request type (application-defined)
- `payload`: Request data

**Optional Fields**:
- `correl_id`: Correlation ID (auto-generated if not provided)
- `expires_at`: Request timeout

**Example**:
```json
{
  "id": "req-123",
  "version": "1.0",
  "from": "orchestrator",
  "to": ["worker"],
  "pattern": "request",
  "type": "task",
  "correl_id": "corr-456",
  "payload": {
    "command": "deploy",
    "target": "app1"
  },
  "metadata": {"priority": "high"},
  "timestamp": 1705773600000000000
}
```

**Expected Response**:
- Pattern: `"response"`
- `correl_id`: Must match request's `correl_id`
- `to`: Original sender (`from` from request)

**Timeout Handling**:
- If `expires_at` is provided and time elapses without response, request fails
- Default timeout: 30 seconds (configurable per profile)

---

### Pattern: `response`

A response to a previous request.

**Required Fields**:
- `from`: Responder profile ID
- `to`: Original request sender
- `pattern`: `"response"`
- `correl_id`: Must match original request's `correl_id`
- `payload`: Response data

**Optional Fields**:
- `metadata.error`: Error object if response indicates failure

**Example** (Success):
```json
{
  "id": "resp-789",
  "version": "1.0",
  "from": "worker",
  "to": ["orchestrator"],
  "pattern": "response",
  "type": "result",
  "correl_id": "corr-456",
  "payload": {
    "status": "success",
    "output": "Deployment complete"
  },
  "timestamp": 1705773605000000000
}
```

**Example** (Error):
```json
{
  "id": "resp-790",
  "version": "1.0",
  "from": "worker",
  "to": ["orchestrator"],
  "pattern": "response",
  "type": "error",
  "correl_id": "corr-456",
  "payload": null,
  "metadata": {
    "error": {
      "code": "DEPLOY_FAILED",
      "message": "Deployment failed: invalid config"
    }
  },
  "timestamp": 1705773605000000000
}
```

---

### Pattern: `publish`

Publish a message to a topic. All subscribers receive the message.

**Required Fields**:
- `from`: Publisher profile ID
- `to`: `["*"]` (wildcard, ignored)
- `pattern`: `"publish"`
- `topic`: Topic name
- `payload`: Event data

**Optional Fields**:
- `metadata`: Additional event metadata

**Example**:
```json
{
  "id": "pub-456",
  "version": "1.0",
  "from": "ci-cd",
  "to": ["*"],
  "pattern": "publish",
  "type": "deployment",
  "topic": "deployments",
  "payload": {
    "app": "myapp",
    "version": "v1.0",
    "status": "started"
  },
  "timestamp": 1705773600000000000
}
```

**Delivery**:
- All profiles subscribed to `topic` receive the message
- No acknowledgment required (fire-and-forget)
- Ordering: Best-effort per topic (no global ordering guarantee)

---

### Pattern: `subscribe`

Subscribe to a topic to receive published messages.

**Required Fields**:
- `from`: Subscriber profile ID
- `to`: `["*"]` (wildcard, ignored)
- `pattern`: `"subscribe"`
- `topic`: Topic name

**Optional Fields**:
- `metadata.wildcard`: Subscribe to topic pattern (e.g., `deployments.*`)

**Example**:
```json
{
  "id": "sub-789",
  "version": "1.0",
  "from": "monitor",
  "to": ["*"],
  "pattern": "subscribe",
  "type": "subscription",
  "topic": "deployments",
  "metadata": {"wildcard": false},
  "timestamp": 1705773600000000000
}
```

**Topic Naming**:
- Use dot notation: `deployments.started`, `alerts.critical`, etc.
- Wildcard subscription: `deployments.*` (receives all `deployments.*` topics)

---

## Transport Specifications

### Transport Selection and Fallback

Profiles can be configured with a preferred transport. The A2A manager automatically falls back to alternative transports if the preferred transport is unavailable.

**Configuration**:
```yaml
a2a:
  transport_type: "websocket"  # Preferred transport
  # Valid values: "ipc", "http", "websocket"
```

**Fallback Priority** (automatic, in order):
1. **WebSocket** (if configured) - Real-time bidirectional
2. **HTTP** (if configured) - Request-response network communication
3. **IPC** (always available) - Filesystem-based local communication

**Fallback Behavior**:
- If `transport_type` is set to `websocket` but WebSocket endpoint is not listening, fall back to HTTP
- If HTTP endpoint is not available, fall back to IPC
- If profile explicitly sets `transport_type: "ipc"`, no fallback (IPC only)
- Fallback events are logged to `~/.agents/logs/a2a/transport.log`

**Per-Message Transport Override**:
Messages can specify preferred transport via metadata:
```json
{
  "metadata": {
    "preferred_transport": "http"
  }
}
```

**Transport Comparison**:

| Transport | Latency | Throughput | Network | Use Case |
|-----------|----------|------------|----------|----------|
| IPC | < 10ms | Low (<100 msg/s) | Local only | Simple, same-machine profiles |
| HTTP | 10-100ms | Medium (<500 msg/s) | Yes | Network request/response |
| WebSocket | < 10ms | High (>1000 msg/s) | Yes | Real-time, pub/sub |

---

### Transport: IPC (Filesystem Queue)

**Purpose**: Local communication between profiles on same machine

**Mechanism**:
- Filesystem-based message queues
- Directory: `~/.agents/ipc/queues/<profile-id>/incoming/`
- Message files: `<timestamp>_<uuid>.json`

**Delivery**:
- Polling interval: 100ms (configurable)
- Remove message after processing
- No acknowledgment required

**Limitations**:
- Polling overhead
- Not suitable for high-throughput (> 100 msg/sec)

**Security**:
- Directory permissions: 0700
- File permissions: 0600
- Only owner profile can read messages

**Wire Format**:
```bash
# Write to: ~/.agents/ipc/queues/worker/incoming/1705773600_550e8400.json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "1.0",
  ...
}
```

---

### Transport: HTTP

**Purpose**: Network communication, request/response pattern

**Endpoints**:
- `POST /a2a/message` - Send message to profile
- `GET /health` - Health check

**Headers**:
```
Content-Type: application/json
X-A2A-Version: 1.0
Authorization: Bearer <token> (optional)
```

**Request Body**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "1.0",
  ...
}
```

**Response Codes**:
- `200 OK`: Message accepted
- `202 Accepted`: Message queued (async processing)
- `400 Bad Request`: Invalid message format
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

**Response Body** (200/202):
```json
{
  "accepted": true,
  "message_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response Body** (Error):
```json
{
  "error": {
    "code": "INVALID_MESSAGE",
    "message": "Missing required field: 'from'"
  }
}
```

**TLS**:
- Recommended for network communication
- Optional for local (127.0.0.1) communication

---

### Transport: WebSocket

**Purpose**: Real-time communication, pub/sub pattern

**Endpoints**:
- `WS /a2a/stream` - Bidirectional message stream
- `WS /a2a/topic/<topic>` - Topic subscription stream

**Connection Flow**:
1. Client connects to `ws://<host>/a2a/stream`
2. Client sends authentication message (if required)
3. Server confirms connection
4. Client sends messages, server pushes messages

**Initial Handshake** (optional auth):
```json
{
  "type": "auth",
  "token": "Bearer <token>"
}
```

**Server Response** (auth success):
```json
{
  "type": "auth_success",
  "profile_id": "agent-a"
}
```

**Message Frame**:
```json
{
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "version": "1.0",
    ...
  }
}
```

**Error Frame**:
```json
{
  "error": {
    "code": "AUTH_FAILED",
    "message": "Invalid token"
  }
}
```

**Topic Subscription Flow**:
```
1. Client connects to ws://host/a2a/topic/deployments
2. Server accepts connection
3. Server streams all publish messages for topic
4. Client can send messages to server
```

---

## Conversation Model

### Conversation Metadata

**File**: `meta.json` in conversation directory

```json
{
  "id": "conv-123",
  "type": "duo",
  "participants": ["agent-a", "agent-b"],
  "title": "Deployment Coordination",
  "status": "active",
  "created_at": 1705773600000000000,
  "updated_at": 1705773900000000000,
  "last_message": {
    "id": "msg-456",
    "timestamp": 1705773900000000000,
    "from": "agent-b",
    "preview": "Deployment complete, status: success"
  },
  "settings": {
    "notify_all": true,
    "private": true,
    "auto_archive": true,
    "archive_days": 30,
    "topic": null
  },
  "message_count": 42
}
```

### Conversation Types

#### Type: `duo`

1-to-1 conversation between two profiles.

**Constraints**:
- Exactly 2 participants
- All messages visible to both participants

**Use Cases**:
- Direct collaboration
- Request/response pairs
- Private discussions

---

#### Type: `group`

1-to-n conversation between multiple profiles.

**Constraints**:
- 3 or more participants
- All messages visible to all participants
- Optional notification settings

**Use Cases**:
- Team coordination
- Group decision-making

---

### Message Expiration

Messages can have optional expiration timestamps to automatically remove old data.

**Per-Message Expiration**:
```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "status",
  "payload": {"query": "current"},
  "expires_at": 1705773660000000000, // 60 seconds from now
  "timestamp": 1705773600000000000
}
```

**Expiration Behavior**:
- If `expires_at` is set and time is exceeded, message is ignored
- Expired messages are not delivered to recipients
- Expired messages are cleaned up from storage
- Expired messages are logged but not stored in conversation history

**Conversation-Level Expiration**:
```json
{
  "id": "conv-123",
  "settings": {
    "message_ttl_seconds": 86400  // Messages expire after 24 hours
  }
}
```

**Cleanup Process**:
- Periodic cleanup every hour (configurable)
- Removes expired messages from conversation history
- Updates conversation `message_count`
- Logs cleanup activity

**Use Cases**:
- Temporary status updates (no long-term value)
- Real-time notifications (no history needed)
- Sensitive data (auto-delete after retention period)

---

### Message Ordering

**Per-Conversation Ordering**:
- Messages within a conversation are ordered by timestamp
- Filenames include timestamp: `<timestamp>_<msg-id>.json`
- Retrieval always returns messages in timestamp order

**No Global Ordering**:
- Messages across different conversations are not globally ordered
- No global sequence number (simplifies implementation)
- Each conversation has independent ordering

**Timestamp Resolution**:
- Nanosecond precision (1705773600000000000)
- Sufficient for high-throughput scenarios (millions of messages)
- If timestamps collide (same nanosecond), order by `id` (UUID)

**Example Retrieval**:
```bash
# Retrieve conversation messages (always ordered)
aps a2a show-conversation <conv-id> --history

# Output:
# [1] msg-001 @ 2026-01-20 10:00:00.000000001
# [2] msg-002 @ 2026-01-20 10:00:05.000000003
# [3] msg-003 @ 2026-01-20 10:00:10.000000002
```

**Pub/Sub Ordering**:
- Per-topic best-effort ordering
- No guaranteed global order across subscribers
- Each subscriber receives messages in order they arrived at topic

**Request/Response Ordering**:
- Requests are not guaranteed to be processed in order
- Responses are correlated to requests via `correl_id`
- Client must match responses to requests (not assume FIFO)

---

### Conversation Status

#### Status: `active`

Conversation is active and can receive new messages.

#### Status: `archived`

Conversation is archived (read-only). No new messages allowed.

**Trigger**: Manual archive command or auto-archival (inactivity).

#### Status: `closed`

Conversation is closed and frozen. No modifications allowed.

**Trigger**: Manual close command.

---

## Message Storage

### Message File Format

**Directory**: `conversations/<conv-id>/messages/`

**Filename**: `<timestamp>_<msg-id>.json`

**Content**:
```json
{
  "id": "msg-456",
  "version": "1.0",
  "from": "agent-b",
  "to": ["agent-a"],
  "pattern": "response",
  "type": "result",
  "correl_id": "corr-123",
  "payload": {
    "status": "success",
    "output": "Deployment complete"
  },
  "metadata": {},
  "timestamp": 1705773900000000000,
  "signature": null
}
```

### Storage Rules

1. **Atomic Writes**: Use atomic rename (write to temp, then rename)
2. **Immutable**: Message files never modified after creation
3. **Ordered**: Filenames start with timestamp for sorting
4. **Cleanup**: Archived conversations may be pruned after retention period

---

## Security Specification

### Message Signatures

**Algorithm**: HMAC-SHA256

**Signing Process**:
1. Serialize message to JSON (canonical form)
2. Compute HMAC-SHA256 with profile secret key
3. Hex-encode signature
4. Set `signature` field

**Verification Process**:
1. Extract `signature` field
2. Remove `signature` field from message
3. Serialize message to canonical JSON
4. Compute HMAC-SHA256 with sender's public key
5. Compare with extracted signature

**Key Storage**:
- Keys stored in `~/.agents/profiles/<id>/secrets.env`
- Variable: `A2A_SECRET_KEY`
- Permissions: 0600

**Example Signature**:
```
Signature: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
```

**Optional**: Per-profile setting `require_auth: true`

---

### Access Control

**Allowlist**:
```yaml
a2a:
  allow_messages:
    - orchestrator
    - supervisor
```
**Effect**: Reject messages from all other senders

**Blocklist**:
```yaml
a2a:
  block_messages:
    - untrusted-agent
```
**Effect**: Reject messages from specified senders (if not allowlisted)

**Priority**: Allowlist > Blocklist > Default (allow all)

---

### TLS Configuration

**Server** (HTTP/WebSocket):
```yaml
a2a:
  transport_type: "http"
  tls_enabled: true
  tls_cert: "/path/to/cert.pem"
  tls_key: "/path/to/key.pem"
```

**Client**:
- Validates server certificate
- Optional client certificate (mTLS)

---

## Rate Limiting

### Limits (Configurable)

- **Per-Profile**: 100 messages/minute
- **Per-Topic**: 200 messages/minute
- **Global**: 1000 messages/minute

### Response (Exceeded)

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded: 150 messages/minute (limit: 100)",
    "retry_after": 30
  }
}
```

### Backpressure

- If receiver queue is full, sender receives `429 Too Many Requests`
- Exponential backoff recommended: 1s, 2s, 4s, 8s, 16s

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field": "from",
      "reason": "Required field missing"
    }
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_MESSAGE` | Malformed message JSON |
| `MISSING_FIELD` | Required field missing |
| `INVALID_VERSION` | Unsupported protocol version |
| `UNAUTHORIZED` | Authentication required/failed |
| `FORBIDDEN` | Access denied (ACLs) |
| `NOT_FOUND` | Profile or conversation not found |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| `TIMEOUT` | Request timeout |
| `INTERNAL_ERROR` | Server error |
| `TRANSPORT_ERROR` | Transport-specific error |

---

## Protocol Versioning

### Semantic Versioning

Format: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes
- **MINOR**: New features, backward-compatible
- **PATCH**: Bug fixes, backward-compatible

### Version Negotiation

**Request**:
```json
{
  "version": "1.0",
  ...
}
```

**Response** (if version unsupported):
```json
{
  "error": {
    "code": "INVALID_VERSION",
    "message": "Unsupported version: 2.0",
    "supported_versions": ["1.0", "1.1"]
  }
}
```

### Backward Compatibility

- **1.0.x**: Always backward compatible within patch versions
- **1.1.0**: New fields are optional, clients ignore unknown fields
- **2.0.0**: Breaking changes, requires migration

### Future Versions

**Version 2.0 - Protocol Buffers Support** (Planned)

**Breaking Changes**:
- Message serialization changes from JSON to Protocol Buffers
- Requires new `.proto` schema definition
- Wire format becomes binary

**Migration Path**:
- Dual-support period (v1.0 + v2.0) for 6 months
- v1.0 clients continue to use JSON encoding
- v2.0+ clients use Protocol Buffers encoding
- Automatic negotiation via `Accept` and `Content-Type` headers

**Advantages of v2.0**:
- **Performance**: 5-10x faster serialization/deserialization
- **Size**: 3-5x smaller message size
- **Type Safety**: Compile-time schema validation
- **Code Generation**: Auto-generated client/server stubs

**Example v2.0 Schema (.proto)**:
```protobuf
syntax = "proto3";
package a2a.v2;

message Message {
  string id = 1;                           // UUID v4
  string version = 2;                        // "2.0"
  string from = 3;
  repeated string to = 4;
  Pattern pattern = 5;
  string type = 6;
  oneof payload {
    TaskPayload task = 10;
    QueryPayload query = 11;
    EventPayload event = 12;
  }
  map<string, string> metadata = 7;
  int64 timestamp = 8;                      // Nanoseconds
  int64 expires_at = 9;                     // Optional
  string correl_id = 13;                     // For request/response
  string topic = 14;                         // For pub/sub
  string signature = 15;                     // HMAC-SHA256
}

enum Pattern {
  REQUEST = 0;
  RESPONSE = 1;
  PUBLISH = 2;
  SUBSCRIBE = 3;
}

message TaskPayload {
  string command = 1;
  map<string, string> args = 2;
}
```

**Backward Compatibility During Migration**:
```
v1.0 Client -> v2.0 Server:
  Accept: application/json
  Server responds with JSON (supports both)

v2.0 Client -> v1.0 Server:
  Content-Type: application/protobuf
  Server returns error (v1.0 doesn't support Protobuf)
  Client falls back to JSON

v2.0 Client -> v2.0 Server:
  Content-Type: application/protobuf
  Accept: application/protobuf
  Server responds with Protobuf (optimal path)
```

---

## Planned Future Features

### Version 1.1 - Batch Messages

**Purpose**: Reduce round-trip overhead for multiple small messages

**Batch Message Format**:
```json
{
  "id": "batch-001",
  "version": "1.1",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "batch",
  "type": "multiple_requests",
  "payload": {
    "messages": [
      {
        "id": "req-001",
        "pattern": "request",
        "type": "task",
        "payload": {"command": "deploy", "target": "app1"}
      },
      {
        "id": "req-002",
        "pattern": "request",
        "type": "task",
        "payload": {"command": "deploy", "target": "app2"}
      },
      {
        "id": "req-003",
        "pattern": "request",
        "type": "query",
        "payload": {"query": "status"}
      }
    ]
  },
  "metadata": {
    "batch_size": 3
  },
  "timestamp": 1705773600000000000
}
```

**Batch Response**:
```json
{
  "id": "batch-resp-001",
  "version": "1.1",
  "from": "agent-b",
  "to": ["agent-a"],
  "pattern": "batch",
  "type": "multiple_responses",
  "payload": {
    "messages": [
      {
        "correl_id": "req-001",
        "pattern": "response",
        "type": "result",
        "payload": {"status": "success"}
      },
      {
        "correl_id": "req-002",
        "pattern": "response",
        "type": "result",
        "payload": {"status": "success"}
      },
      {
        "correl_id": "req-003",
        "pattern": "response",
        "type": "result",
        "payload": {"status": "running"}
      }
    ]
  },
  "metadata": {
    "batch_size": 3
  },
  "timestamp": 1705773601000000000
}
```

**Use Cases**:
- Multiple deployments in batch
- Bulk queries for status
- Bulk notifications

**Backward Compatibility**:
- v1.1 clients send batch messages
- v1.0 servers reject `pattern: "batch"` (unsupported pattern)
- v1.1 servers process batch messages atomically
- Batch messages can include mixed patterns (request, response, publish)

**Benefits**:
- Reduced network round-trips
- Lower transport overhead
- Atomic processing (all messages in batch succeed or fail together)

---

### Version 1.2 - Message Acknowledgments

**Purpose**: Reliable delivery for pub/sub and IPC transport

**Ack Message Format**:
```json
{
  "id": "ack-001",
  "version": "1.2",
  "from": "agent-b",
  "to": ["agent-a"],
  "pattern": "ack",
  "type": "delivery_confirmation",
  "payload": {
    "message_id": "msg-001",
    "status": "received",
    "timestamp": 1705773600000000000
  },
  "timestamp": 1705773600000000100
}
```

**Ack Modes**:
- `received` - Message received and queued for processing
- `processed` - Message processing completed
- `failed` - Message processing failed

---

### Version 1.3 - Message Filtering

**Purpose**: Reduce network traffic by filtering messages before delivery

**Subscription with Filter**:
```json
{
  "id": "sub-001",
  "version": "1.3",
  "from": "agent-b",
  "to": ["*"],
  "pattern": "subscribe",
  "type": "filtered_subscription",
  "topic": "deployments",
  "payload": {
    "filter": {
      "field": "status",
      "operator": "in",
      "values": ["started", "completed"]
    }
  },
  "timestamp": 1705773600000000000
}
```

**Filter Operators**:
- `eq` - Equal to
- `ne` - Not equal to
- `in` - In array
- `not_in` - Not in array
- `gt`, `lt`, `gte`, `lte` - Numeric comparisons
- `contains` - String contains

---

## Compliance Requirements

### Client Requirements

- MUST include all required fields
- MUST validate messages before sending
- MUST handle error responses appropriately
- SHOULD implement retry logic with backoff
- SHOULD respect rate limits
- MAY include optional fields

### Server Requirements

- MUST validate incoming messages
- MUST reject invalid messages (400)
- MUST enforce access controls
- MUST implement rate limiting
- MUST support all transport types
- SHOULD return appropriate error codes
- SHOULD log security events

### Transport Requirements

**IPC**:
- MUST maintain filesystem permissions
- MUST use atomic file operations
- SHOULD poll at reasonable intervals

**HTTP**:
- MUST support `Content-Type: application/json`
- MUST support `X-A2A-Version` header
- SHOULD support TLS
- SHOULD implement connection pooling

**WebSocket**:
- MUST support bidirectional messaging
- MUST handle connection failures gracefully
- SHOULD implement heartbeats

---

## Examples

### Example 1: Simple Request/Response

**Request (Agent A → Agent B)**:
```json
{
  "id": "req-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "query",
  "correl_id": "corr-001",
  "payload": {"query": "status"},
  "timestamp": 1705773600000000000
}
```

**Response (Agent B → Agent A)**:
```json
{
  "id": "resp-001",
  "version": "1.0",
  "from": "agent-b",
  "to": ["agent-a"],
  "pattern": "response",
  "type": "result",
  "correl_id": "corr-001",
  "payload": {"status": "running"},
  "timestamp": 1705773601000000000
}
```

---

### Example 2: Pub/Sub

**Subscribe (Agent B)**:
```json
{
  "id": "sub-001",
  "version": "1.0",
  "from": "agent-b",
  "to": ["*"],
  "pattern": "subscribe",
  "type": "subscription",
  "topic": "deployments",
  "timestamp": 1705773600000000000
}
```

**Publish (Agent A)**:
```json
{
  "id": "pub-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["*"],
  "pattern": "publish",
  "type": "deployment",
  "topic": "deployments",
  "payload": {"app": "myapp", "version": "v1.0"},
  "timestamp": 1705773600000000000
}
```

**Received (Agent B)**:
```json
{
  "id": "pub-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["*"],
  "pattern": "publish",
  "type": "deployment",
  "topic": "deployments",
  "payload": {"app": "myapp", "version": "v1.0"},
  "timestamp": 1705773600000000000
}
```

---

### Example 3: Signed Message

**Unsigned Message**:
```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "task",
  "payload": {"command": "deploy"},
  "timestamp": 1705773600000000000,
  "signature": null
}
```

**Signed Message**:
```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "task",
  "payload": {"command": "deploy"},
  "timestamp": 1705773600000000000,
  "signature": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

---

## Appendix

### A. JSON Canonicalization

To compute signatures, messages must be canonicalized:

1. Remove whitespace
2. Sort object keys alphabetically
3. Use UTF-8 encoding
4. Omit `signature` field

**Example**:
```json
// Original
{
  "id": "123",
  "from": "agent-a",
  "to": ["agent-b"],
  "payload": {"z": 1, "a": 2}
}

// Canonical
{"from":"agent-a","id":"123","payload":{"a":2,"z":1},"to":["agent-b"]}
```

---

### B. Timestamp Format

Unix timestamps in **nanoseconds** since epoch (UTC).

**Example**:
- `1705773600000000000` = 2026-01-20T10:00:00.000Z

---

### C. UUID Format

UUID v4 (random) format.

**Example**: `550e8400-e29b-41d4-a716-446655440000`

**Generation**: Use standard UUID v4 generator.

---

### D. Topic Naming Convention

Use dot notation for hierarchical topics.

**Examples**:
- `deployments.started`
- `deployments.completed`
- `alerts.critical`
- `alerts.warning`
- `system.health`

**Wildcard**: `deployments.*` matches all `deployments.*` topics.

---

## Change Log

### Version 1.0 (2026-01-20)
- Initial protocol specification
- Define message format and patterns
- Specify IPC, HTTP, and WebSocket transports
- Define conversation model
- Specify security requirements
- **Resolved**: JSON for v1.0 serialization (planned Protobuf for v2.0)
- **Resolved**: UUID v4 for message IDs
- **Resolved**: Transport fallback order: WebSocket → HTTP → IPC
- **Resolved**: Per-message compression support (gzip, deflate)
- **Resolved**: Message references for large payloads (> 1 MB)
- **Resolved**: Optional message expiration (per-message and per-conversation)
- **Resolved**: Per-conversation message ordering (no global ordering)
- **Planned**: Batch messages for v1.1
- **Planned**: Acknowledgments for v1.2
- **Planned**: Message filtering for v1.3

---

## License

This specification is part of the APS (Agent Profile System) project.
