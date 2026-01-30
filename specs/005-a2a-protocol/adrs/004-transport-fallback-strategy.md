# ADR-004: Transport Fallback Strategy (WebSocket → HTTP → IPC)

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

The A2A protocol supports three transport mechanisms for communication between profiles. Profiles may be configured with different transports, and some transports may not be available on all systems or may fail at runtime.

**Available Transports**:
- **IPC**: Filesystem-based queues (always available, local only)
- **HTTP**: RESTful endpoints (network capable, requires server)
- **WebSocket**: Real-time streaming (network capable, requires server)

**Requirements**:
- Profiles should be able to communicate regardless of configuration
- Should prefer optimal transport when available
- Should gracefully fallback when preferred transport fails
- Should support per-configuration and per-message transport selection
- Should respect isolation tier constraints

---

## Decision

**Implement fallback strategy: WebSocket → HTTP → IPC**

### Fallback Order

```
Preferred Transport (profile config)
  ↓ WebSocket available and listening?
  ↓ Yes → Use WebSocket
  ↓ No → Fallback to HTTP
  ↓ HTTP available and listening?
  ↓ Yes → Use HTTP
  ↓ No → Fallback to IPC (always available)
```

### Configuration

**Profile Configuration** (`profile.yaml`):
```yaml
a2a:
  enabled: true
  transport_type: "websocket"  # Preferred transport
  # Valid values: "ipc", "http", "websocket"
```

**Per-Message Override**:
```json
{
  "metadata": {
    "preferred_transport": "http"
  }
}
```

### Implementation Logic

```go
func (m *Manager) SelectTransport(msg *Message) (Transport, error) {
    // 1. Check per-message override
    if msg.Metadata["preferred_transport"] != "" {
        preferred := msg.Metadata["preferred_transport"]
        if transport := m.getTransport(preferred); transport.Supported() {
            return transport, nil
        }
    }

    // 2. Check profile preferred transport
    preferred := m.profile.A2A.TransportType

    // 3. Try WebSocket
    if preferred == "websocket" || preferred == "auto" {
        if transport := m.getTransport("websocket"); transport.Supported() {
            return transport, nil
        }
    }

    // 4. Try HTTP
    if preferred == "http" || preferred == "websocket" || preferred == "auto" {
        if transport := m.getTransport("http"); transport.Supported() {
            return transport, nil
        }
    }

    // 5. Fallback to IPC (always available)
    return m.getTransport("ipc"), nil
}
```

---

## Consequences

### Positive

- **Resilience**: Communication always possible (IPC always available)
- **Optimal Performance**: Prefers real-time WebSocket when available
- **Network Capability**: Falls back to HTTP if WebSocket unavailable
- **Local Fallback**: IPC works even without network
- **Flexible**: Per-message override for specific use cases
- **Configuration Control**: Profiles can enforce specific transport

### Negative

- **Complexity**: Fallback logic adds implementation complexity
- **Behavior Variability**: Different profiles may use different transports
- **Debugging**: Harder to debug when fallback happens
- **Performance Variance**: Latency varies by transport (10ms vs. 100ms)
- **Connection Overhead**: Fallback may cause connection attempts to fail first

---

## Alternatives Considered

### 1. Single Transport (No Fallback) ❌

**Approach**: Profiles use only one configured transport

**Pros**:
- **Simple**: No fallback logic
- **Predictable**: Behavior is consistent
- **Easy Debugging**: Single transport to debug

**Cons**:
- **Fragile**: If transport unavailable, communication fails
- **Limited**: Can't support local-only profiles
- **Inflexible**: Can't adapt to network conditions
- **User Burden**: Users must configure correctly for use case

**Example Failure**:
```yaml
# Profile configured for WebSocket
a2a:
  transport_type: "websocket"

# WebSocket server not running → All communication fails
# No automatic fallback → User must reconfigure profile
```

**Rejection**: Too fragile, poor user experience

---

### 2. User-Defined Fallback Order ❌

**Approach**: User defines fallback order in configuration

**Configuration**:
```yaml
a2a:
  transports:
    - "websocket"
    - "http"
    - "ipc"
```

**Pros**:
- **User Control**: Users define priority
- **Transparent**: Clear fallback behavior
- **Flexible**: Any order possible

**Cons**:
- **Complex Configuration**: Users must understand all transports
- **Error-Prone**: Users may configure invalid orders
- **Unnecessary**: Default order is optimal for most cases
- **Documentation Burden**: Need to document each transport

**Example Invalid Configuration**:
```yaml
a2a:
  transports:
    - "http"     # Won't work for local-only
    - "ipc"      # Won't work for network
```

**Rejection**: Too complex for users, default order is optimal

---

### 3. Smart Fallback (Latency-Based) ❌

**Approach**: Measure latency and choose fastest transport

**Implementation**:
```go
func (m *Manager) SelectSmartTransport(targets []string) (Transport, error) {
    for _, transport := range [WebSocket, HTTP, IPC] {
        latency := m.measureLatency(transport, targets)
        if latency < threshold {
            return transport, nil
        }
    }
    return IPC, nil  // Fallback
}
```

**Pros**:
- **Optimal**: Always chooses fastest available
- **Adaptive**: Adapts to network conditions
- **Smart**: Automatically handles congestion

**Cons**:
- **Complex**: Requires latency measurement logic
- **Overhead**: Adds latency measurements
- **Flapping**: May switch transports frequently
- **Unpredictable**: Hard to debug behavior changes
- **Connection Overhead**: Probing adds connections

**Rejection**: Unnecessary complexity for v1.0

---

### 4. Priority-Based Fallback (Quality Score) ❌

**Approach**: Assign quality scores, choose highest available

**Quality Scores**:
- WebSocket: 100 (real-time, low latency)
- HTTP: 80 (network, higher latency)
- IPC: 50 (local, high latency)

**Implementation**:
```go
type Transport struct {
    Type     TransportType
    Priority int
    Available bool
}

func (m *Manager) SelectTransport() Transport {
    available := m.getAvailableTransports()
    return maxByPriority(available)
}
```

**Pros**:
- **Explicit Quality**: Clear quality hierarchy
- **Configurable**: Can adjust priorities
- **Scalable**: Easy to add new transports

**Cons**:
- **Arbitrary Scores**: Quality scores are subjective
- **Same as Order**: Just codifies the order (100 > 80 > 50)
- **Unnecessary**: Order is sufficient, scores don't add value

**Rejection**: Doesn't provide value over simple order

---

### 5. No IPC Fallback (HTTP Only) ❌

**Approach**: Fallback from WebSocket to HTTP only, no IPC

**Pros**:
- **Network-Only**: Simpler, focused on network use case
- **Consistent**: Similar characteristics (both network)

**Cons**:
- **Local-Only Profiles**: Can't support local-only communication
- **Single-Machine**: Can't communicate without network stack
- **Overhead**: Even local communication goes through HTTP

**Example Failure**:
```bash
# Single machine, no network stack available
# Profile A wants to talk to Profile B
# WebSocket → HTTP (requires network) → FAIL
# Should use IPC but disabled → Communication fails
```

**Rejection**: Breaks local communication use case

---

### 6. Random Transport Selection ❌

**Approach**: Randomly select from available transports

**Implementation**:
```go
func (m *Manager) SelectRandomTransport() (Transport, error) {
    available := m.getAvailableTransports()
    return available[rand.Intn(len(available))], nil
}
```

**Pros**:
- **Load Balancing**: Distributes load across transports
- **Simple**: No priority logic

**Cons**:
- **Suboptimal**: May choose slower transport
- **Inconsistent**: Different behavior each time
- **Unpredictable**: Hard to debug
- **Wasteful**: WebSocket available but HTTP selected

**Rejection**: Doesn't optimize for performance

---

## Transport Comparison

| Characteristic | WebSocket | HTTP | IPC |
|----------------|-------------|-------|-----|
| **Latency** | < 10ms | 10-100ms | < 10ms |
| **Throughput** | High (>1000 msg/s) | Medium (<500 msg/s) | Low (<100 msg/s) |
| **Network** | Yes | Yes | No |
| **Real-Time** | Yes (bidirectional) | No (request/response) | No (polling) |
| **Always Available** | No (requires server) | No (requires server) | Yes (always) |
| **Setup Complexity** | Medium | Low | Low |
| **Use Case** | Pub/sub, real-time | Request/response | Simple local |

**Rationale for Order**:
1. **WebSocket**: Best performance, real-time, preferred
2. **HTTP**: Network capability, widely supported
3. **IPC**: Always available, local-only, guaranteed fallback

---

## Fallback Behavior

### Scenario 1: All Transports Available

```
Profile config: transport_type = "websocket"
WebSocket listening → Use WebSocket (optimal)
```

### Scenario 2: WebSocket Down

```
Profile config: transport_type = "websocket"
WebSocket not listening → Fallback to HTTP
HTTP listening → Use HTTP
```

### Scenario 3: WebSocket and HTTP Down

```
Profile config: transport_type = "websocket"
WebSocket not listening → Fallback to HTTP
HTTP not listening → Fallback to IPC
IPC available → Use IPC (guaranteed)
```

### Scenario 4: IPC Only

```
Profile config: transport_type = "ipc"
Use IPC directly (no fallback)
```

### Scenario 5: Per-Message Override

```
Profile config: transport_type = "websocket"
Message: metadata.preferred_transport = "http"
Ignore profile config → Use HTTP (if available)
```

---

## Logging and Monitoring

### Fallback Events

```go
func (m *Manager) logFallback(from, to TransportType) {
    log.Printf("Transport fallback: %s → %s", from, to)
}
```

**Example Logs**:
```
2026-01-20 10:00:00 [A2A] Transport fallback: websocket → http (WebSocket not listening)
2026-01-20 10:00:05 [A2A] Transport fallback: http → ipc (HTTP not listening)
2026-01-20 10:00:10 [A2A] Using ipc transport (fallback complete)
```

### Metrics

```go
type TransportMetrics struct {
    WebSocketUsage    int
    HTTPUsage        int
    IPCUsage         int
    WebSocketFallbacks int
    HTTPFallbacks    int
}
```

---

## Isolation Tier Considerations

### Process Isolation (Tier 1)
- **All Transports Available**: IPC, HTTP, WebSocket all work
- **No Restrictions**: No isolation-specific blocking

### Platform Sandbox (Tier 2)
- **IPC Available**: Yes (shared directory with ACLs)
- **HTTP/WebSocket**: Yes (if network enabled)
- **May Have Restrictions**: Platform-specific (e.g., macOS sandbox)

### Container Isolation (Tier 3)
- **IPC Available**: Yes (via volume mount)
- **HTTP/WebSocket**: Yes (if network enabled)
- **Port Mapping**: May require port mapping for HTTP/WebSocket

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (works with all transports)
- **ADR-005**: Message references (mitigates IPC throughput limitations)

---

## References

- **Specification**: `spec.md` - Transport Selection and Fallback section
- **Decisions Document**: `decisions.md` - Question #3

---

## Revisions

- 2026-01-20: Initial decision - WebSocket → HTTP → IPC fallback
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added isolation tier considerations
