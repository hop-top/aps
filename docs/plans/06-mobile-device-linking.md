# Plan 6: Mobile Device Linking via QR Code

**Date**: 2026-02-12
**Status**: Planning (Architecture)
**Priority**: P2 (Optional but valuable)
**Related Story**: 049 (Mobile Device Linking via QR Code)
**Related Plans**: Plan 4 (Secrets Management), Plan 5 (Webhook Server)

---

## Overview

Mobile Device Linking enables users to securely authorize mobile devices (iOS/Android) to interact with APS profiles via QR code scanning. This feature bridges the gap between the local APS system and mobile clients, allowing agents to be controlled or monitored from smartphones while maintaining isolation and security guarantees.

**Key Principles**:
- **QR Code Simplicity**: One scan to authorize device
- **WebSocket Real-Time**: Persistent connection for live updates
- **Device Registry**: Track authorized devices per profile
- **Expiring Tokens**: Time-limited device credentials
- **Isolation Aware**: Respects APS isolation boundaries (process, platform, container)
- **Mobile-First API**: REST endpoints designed for mobile clients

---

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                APS Mobile Device System                 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────┐         ┌────────────────────┐   │
│  │   Mobile Client  │         │  APS Device Server │   │
│  │  (iOS/Android)   │◄─────►  │   REST + WebSocket │   │
│  │  • Scan QR       │ HTTPS   │   • Auth endpoints │   │
│  │  • Connect       │         │   • Run commands   │   │
│  │  • Monitor       │         │   • Stream results │   │
│  └──────────────────┘         └────────────────────┘   │
│                                         │                │
│                                         ▼                │
│                              ┌──────────────────────┐   │
│                              │  Device Registry     │   │
│                              │  (~/.aps/devices/)   │   │
│                              │  • Device tokens     │   │
│                              │  • Profile mappings  │   │
│                              │  • Expiry times      │   │
│                              └──────────────────────┘   │
│                                         │                │
│                                         ▼                │
│                              ┌──────────────────────┐   │
│                              │  Profile Execution  │   │
│                              │  (existing isolation)│   │
│                              └──────────────────────┘   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Directory Structure

**Global device registry**:
```
~/.aps/devices/
├── registry.json              # Master device list
├── tokens/
│   ├── <device-id>/
│   │   ├── token.jwt          # Encrypted JWT token
│   │   ├── metadata.json      # Device info (name, os, created_at, expires_at)
│   │   ├── sessions.json      # Active sessions
│   │   └── logs.json          # Connection history
└── qr-codes/                  # Cached QR code images
    └── <profile-id>/
        └── <qr-code-id>.png
```

**Profile-scoped device config**:
```
~/.agents/profiles/<profile-id>/
├── devices.yaml               # Profile device settings
├── device-keys/               # Profile-specific device keys
│   ├── <device-id>.pub
│   └── <device-id>.key
```

**Device logging**:
```
~/.aps/logs/devices/
├── <profile-id>/
│   ├── connections.log        # Connection attempts
│   ├── commands.log           # Executed commands
│   └── errors.log             # Authentication/execution errors
```

### QR Code Payload Format

**Encoded in QR Code** (Base64-encoded JSON):
```json
{
  "version": "1.0",
  "profile_id": "myagent",
  "endpoint": "https://localhost:8443/aps/device",
  "pairing_code": "ABC123XYZ789",
  "expires_at": "2026-02-13T10:30:00Z",
  "capabilities": [
    "run:stateless",
    "run:streaming",
    "monitor:sessions"
  ]
}
```

**Decoding Process**:
1. Mobile app scans QR code
2. Extracts Base64 payload
3. Parses JSON structure
4. Validates expiry
5. Connects to endpoint with pairing code

### Device Token Format

**JWT Structure** (signed with profile-specific private key):
```
Header:
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "<profile-id>"  // Key ID points to profile's device key
}

Payload:
{
  "device_id": "iphone-alice-001",
  "device_name": "Alice's iPhone 15",
  "device_os": "iOS",
  "device_version": "17.2.1",
  "profile_id": "myagent",
  "issued_at": "2026-02-12T10:00:00Z",
  "expires_at": "2026-02-26T10:00:00Z",  // 14 days default
  "capabilities": [
    "run:stateless",
    "run:streaming",
    "monitor:sessions"
  ],
  "scope": "aps:device",
  "device_key": "<base64-public-key>"
}

Signature:
HMACSHA256(
  base64UrlEncode(header) + "." +
  base64UrlEncode(payload),
  <profile-private-key>
)
```

### WebSocket Authentication Flow

**Sequence Diagram**:
```
Mobile Client                APS Device Server         Profile Executor
     │                              │                         │
     │  1. Scan QR Code             │                         │
     ├─────────────────────────────►│                         │
     │                              │                         │
     │  2. POST /device/pair        │                         │
     │     {pairing_code, device_info}                        │
     │                              ├────────────────────────►│
     │                              │   Verify pairing code   │
     │                              │   Check profile access  │
     │                              │◄────────────────────────┤
     │                              │                         │
     │◄─────────────────────────────┤                         │
     │  3. Return device token      │                         │
     │     {jwt, ws_endpoint}       │                         │
     │                              │                         │
     │  4. WebSocket CONNECT        │                         │
     │     (Bearer token in header) │                         │
     ├────────────────────────────►│                         │
     │                              ├────────────────────────►│
     │                              │   Validate JWT          │
     │                              │   Check expiry          │
     │                              │   Verify signature      │
     │                              │◄────────────────────────┤
     │◄─────────────────────────────┤                         │
     │  5. Connection ACK           │                         │
     │                              │                         │
     │  6. Send command via         │                         │
     │     WebSocket message        │                         │
     ├────────────────────────────►│                         │
     │                              ├────────────────────────►│
     │                              │   Execute command       │
     │                              │   Return output         │
     │                              │◄────────────────────────┤
     │◄─────────────────────────────┤                         │
     │  7. Stream response          │                         │
     │                              │                         │
     │  8. WebSocket DISCONNECT     │                         │
     ├────────────────────────────►│                         │
     │                              │                         │
```

### Device Approval Workflow

**New Device Pairing** (automatic on first connection):

```
┌─────────────────────────┐
│  QR Code Generated      │
│  (by CLI or web UI)     │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  Device Scans QR        │
│  Extracts payload       │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  Device Registration    │
│  POST /device/pair      │
│  {pairing_code, info}   │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  Validate Pairing Code  │
│  • Check expiry         │
│  • Verify uniqueness    │
└────────────┬────────────┘
             │
             ▼
         ✓ VALID
             │
             ▼
┌─────────────────────────┐
│  Generate Device Token  │
│  • Create JWT           │
│  • Set expiry (14 days) │
│  • Store device record  │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  Return Auth Response   │
│  {jwt, ws_endpoint, ..} │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  WebSocket Connected    │
│  (persistent session)   │
└─────────────────────────┘
```

**Manual Device Approval** (optional, for security-sensitive profiles):

```
Device Registers → Pending State → User Approves (via TUI) → Active State
```

### Message Format (WebSocket)

**Client Request**:
```json
{
  "id": "req_12345",
  "type": "command",
  "action": "run",
  "payload": {
    "command": "echo 'Hello from mobile'",
    "args": ["--verbose"],
    "timeout": 30,
    "stream": true
  }
}
```

**Server Response Stream**:
```json
[
  {
    "id": "req_12345",
    "type": "status",
    "status": "running",
    "started_at": "2026-02-12T10:30:00Z"
  },
  {
    "id": "req_12345",
    "type": "output",
    "stream": "stdout",
    "data": "Hello from mobile\n"
  },
  {
    "id": "req_12345",
    "type": "status",
    "status": "completed",
    "exit_code": 0,
    "duration_ms": 250
  }
]
```

### Device Registry Schema

**registry.json**:
```json
{
  "version": "1.0",
  "devices": [
    {
      "device_id": "iphone-alice-001",
      "profile_id": "myagent",
      "device_name": "Alice's iPhone 15",
      "device_os": "iOS",
      "device_version": "17.2.1",
      "device_model": "iPhone15,2",
      "registered_at": "2026-02-12T10:00:00Z",
      "last_seen_at": "2026-02-12T10:30:00Z",
      "expires_at": "2026-02-26T10:00:00Z",
      "token_hash": "sha256:abc123...",  // Hash of JWT token
      "status": "active",  // or "pending", "revoked", "expired"
      "capabilities": ["run:stateless", "run:streaming", "monitor:sessions"],
      "approval_required": false,
      "approved_at": "2026-02-12T10:05:00Z",
      "approved_by": "user"
    }
  ]
}
```

---

## Implementation Plan

### Phase 1: Infrastructure & API (Weeks 1-2)

**Objective**: Build the core device server, QR code generation, and basic authentication.

#### 1.1: Device Server Core

**File**: `internal/device/server.go`

```go
// Core device server with HTTP + WebSocket support
type DeviceServer struct {
  profile      *Profile
  tokenManager *TokenManager
  registry     *DeviceRegistry
  wsManager    *WebSocketManager
}

func (s *DeviceServer) StartServer(addr string) error { }
func (s *DeviceServer) HandlePairingRequest(req *PairingRequest) (*PairingResponse, error) { }
func (s *DeviceServer) HandleWebSocket(ws *websocket.Conn) error { }
func (s *DeviceServer) ValidateToken(token string) (*DeviceClaims, error) { }
```

**Deliverables**:
- [ ] HTTP server on configurable port
- [ ] HTTPS support with self-signed certs (profile-specific)
- [ ] Graceful shutdown handling
- [ ] Health check endpoint

#### 1.2: QR Code Generation

**File**: `internal/device/qrcode.go`

```go
type QRCodeGenerator struct {
  profile     *Profile
  tokenMgr    *TokenManager
}

type QRPayload struct {
  Version      string   `json:"version"`
  ProfileID    string   `json:"profile_id"`
  Endpoint     string   `json:"endpoint"`
  PairingCode  string   `json:"pairing_code"`
  ExpiresAt    time.Time `json:"expires_at"`
  Capabilities []string `json:"capabilities"`
}

func (g *QRCodeGenerator) Generate(profile *Profile, opts GenerateOpts) (*QRCode, error) { }
func (g *QRCodeGenerator) EncodePayload(p *QRPayload) (string, error) { }
func (g *QRCodeGenerator) GeneratePairingCode() (string, error) { }
```

**Deliverables**:
- [ ] QR code generation using `github.com/skip2/go-qrcode`
- [ ] JSON payload encoding (Base64 compressed)
- [ ] Pairing code generation (short alphanumeric)
- [ ] PNG image export
- [ ] Cache strategy for generated codes

#### 1.3: Device Token Management

**File**: `internal/device/token.go`

```go
type TokenManager struct {
  profile      *Profile
  signingKey   *rsa.PrivateKey
  registry     *DeviceRegistry
}

type DeviceClaims struct {
  DeviceID     string    `json:"device_id"`
  DeviceName   string    `json:"device_name"`
  DeviceOS     string    `json:"device_os"`
  ProfileID    string    `json:"profile_id"`
  IssuedAt     time.Time `json:"iat"`
  ExpiresAt    time.Time `json:"exp"`
  Capabilities []string  `json:"capabilities"`
}

func (tm *TokenManager) CreateToken(device *Device) (string, error) { }
func (tm *TokenManager) ValidateToken(tokenString string) (*DeviceClaims, error) { }
func (tm *TokenManager) RevokeToken(deviceID string) error { }
func (tm *TokenManager) RefreshToken(oldToken string) (string, error) { }
```

**Deliverables**:
- [ ] JWT token creation with RS256 signing
- [ ] Token validation with expiry checks
- [ ] Token storage in registry
- [ ] Token refresh mechanism
- [ ] Key rotation support

#### 1.4: Device Registry

**File**: `internal/device/registry.go`

```go
type DeviceRegistry struct {
  profileID string
  path      string
}

type Device struct {
  DeviceID       string    `json:"device_id"`
  ProfileID      string    `json:"profile_id"`
  DeviceName     string    `json:"device_name"`
  DeviceOS       string    `json:"device_os"`
  DeviceVersion  string    `json:"device_version"`
  RegisteredAt   time.Time `json:"registered_at"`
  LastSeenAt     time.Time `json:"last_seen_at"`
  ExpiresAt      time.Time `json:"expires_at"`
  TokenHash      string    `json:"token_hash"`
  Status         string    `json:"status"` // active, pending, revoked
  Capabilities   []string  `json:"capabilities"`
}

func (r *DeviceRegistry) RegisterDevice(device *Device) error { }
func (r *DeviceRegistry) GetDevice(deviceID string) (*Device, error) { }
func (r *DeviceRegistry) ListDevices() ([]*Device, error) { }
func (r *DeviceRegistry) UpdateDevice(device *Device) error { }
func (r *DeviceRegistry) RevokeDevice(deviceID string) error { }
func (r *DeviceRegistry) CleanupExpiredDevices() error { }
```

**Deliverables**:
- [ ] JSON registry file management
- [ ] Device CRUD operations
- [ ] Expiry time tracking
- [ ] Cleanup/garbage collection
- [ ] Atomic writes with locking

#### 1.5: WebSocket Manager

**File**: `internal/device/websocket.go`

```go
type WebSocketManager struct {
  registry *DeviceRegistry
  executor *ProfileExecutor
}

type WSConnection struct {
  conn      *websocket.Conn
  device    *Device
  sessionID string
  closed    chan bool
}

type WSMessage struct {
  ID      string      `json:"id"`
  Type    string      `json:"type"`    // command, status, output
  Action  string      `json:"action"`  // run, cancel, list
  Payload interface{} `json:"payload"`
}

func (m *WebSocketManager) HandleConnection(ws *websocket.Conn, claims *DeviceClaims) error { }
func (m *WebSocketManager) HandleMessage(wsConn *WSConnection, msg *WSMessage) error { }
func (m *WebSocketManager) BroadcastStatus(deviceID string, status string) error { }
```

**Deliverables**:
- [ ] WebSocket connection handling
- [ ] Message marshaling/unmarshaling
- [ ] Connection heartbeat
- [ ] Graceful disconnect
- [ ] Concurrent message handling

### Phase 2: CLI Commands (Week 2)

**Objective**: Add CLI commands for device pairing, management, and monitoring.

#### 2.1: Device Link Command

**File**: `cmd/device.go` + `internal/cli/device_link.go`

```bash
aps device link --profile=myagent [--expires=14d] [--capabilities=...]
# Output:
#   QR Code (in terminal as ASCII art or PNG saved to file)
#   Pairing Code: ABC123XYZ789
#   Endpoint: https://localhost:8443/aps/device
#   Expires in: 14 days
```

**Sub-command**: `aps device link --qr-output=qrcode.png`

**Deliverables**:
- [ ] QR code generation via CLI
- [ ] ASCII QR display in terminal
- [ ] PNG QR file export
- [ ] Customizable expiry
- [ ] Capability selection
- [ ] Multiple device support

#### 2.2: Device List Command

**File**: `internal/cli/device_list.go`

```bash
aps device list [--profile=myagent] [--json]
# Output table:
#   DEVICE_ID           | DEVICE_NAME         | OS      | STATUS   | EXPIRES_AT
#   iphone-alice-001    | Alice's iPhone 15   | iOS     | active   | 2026-02-26
#   android-bob-001     | Bob's Android       | Android | pending  | 2026-02-19
```

**Deliverables**:
- [ ] Formatted table output
- [ ] JSON output for automation
- [ ] Filtering by status
- [ ] Sorting options
- [ ] Last seen timestamps

#### 2.3: Device Revoke Command

**File**: `internal/cli/device_revoke.go`

```bash
aps device revoke iphone-alice-001 [--profile=myagent]
# Output: Device revoked successfully
```

**Deliverables**:
- [ ] Device revocation
- [ ] Confirmation prompt
- [ ] Immediate disconnection of WebSocket
- [ ] Token blacklisting
- [ ] Audit logging

#### 2.4: Device Logs Command

**File**: `internal/cli/device_logs.go`

```bash
aps device logs [--profile=myagent] [--device=iphone-alice-001] [--tail=100]
# Output: Connection/command log entries
```

**Deliverables**:
- [ ] Connection history
- [ ] Command execution log
- [ ] Error tracking
- [ ] Real-time tail mode
- [ ] Filtering options

#### 2.5: Device Approve Command (Optional)

**File**: `internal/cli/device_approve.go`

```bash
aps device approve iphone-alice-001 [--profile=myagent]
# Required if approval_required=true
```

**Deliverables**:
- [ ] Manual device approval
- [ ] TUI approval interface
- [ ] Audit trail

### Phase 3: Profile Integration (Week 3)

**Objective**: Integrate device linking with existing profile system and isolation boundaries.

#### 3.1: Profile Configuration

**File**: `internal/core/profile.go` (extend)

```go
type Profile struct {
  // ... existing fields ...
  DeviceConfig *DeviceConfig `yaml:"device_config"`
}

type DeviceConfig struct {
  Enabled             bool      `yaml:"enabled"`
  Port                int       `yaml:"port"`
  MaxDevices          int       `yaml:"max_devices"`
  DefaultExpiry       string    `yaml:"default_expiry"`      // "14d"
  ApprovalRequired    bool      `yaml:"approval_required"`
  AllowedCapabilities []string  `yaml:"allowed_capabilities"`
}
```

**Deliverables**:
- [ ] Device config in profile YAML
- [ ] Port configuration (auto-assigned)
- [ ] Device limit enforcement
- [ ] Approval workflow toggle
- [ ] Capability restrictions

#### 3.2: Isolation Integration

**File**: `internal/device/isolation.go`

```go
// Map device operations through isolation boundaries
type IsolationBridge struct {
  profile  *Profile
  executor *ProfileExecutor
}

func (b *IsolationBridge) ExecuteOnIsolated(cmd string, env map[string]string) (string, error) {
  // Route through appropriate isolation tier (process, platform, container)
}

func (b *IsolationBridge) GetProfileEnvironment() map[string]string {
  // Get isolated environment for device token operations
}
```

**Deliverables**:
- [ ] Process isolation support
- [ ] Platform isolation support (with auth bridge)
- [ ] Container isolation support
- [ ] Environment isolation
- [ ] Credential injection
- [ ] Resource limit respecting

#### 3.3: Secrets Integration

**File**: `internal/device/secrets.go`

```go
// Use existing secrets management for device keys
type DeviceSecrets struct {
  profile *Profile
  secrets *SecretManager
}

func (ds *DeviceSecrets) StoreDeviceKey(deviceID string, key []byte) error {
  // Store in secrets manager
}

func (ds *DeviceSecrets) RetrieveDeviceKey(deviceID string) ([]byte, error) {
  // Retrieve from secrets manager
}
```

**Deliverables**:
- [ ] Device key storage via secrets manager
- [ ] Encryption at rest
- [ ] Secure key rotation
- [ ] Access auditing

### Phase 4: Mobile Client Libraries (Week 3-4)

**Objective**: Publish mobile client libraries for iOS and Android.

#### 4.1: Swift SDK (iOS)

**Repository**: `ios-aps-client` (separate repo)

```swift
import Foundation

class APSDeviceClient {
    let profileEndpoint: URL
    let deviceToken: String

    func connect() async throws -> AsyncSequence<APSMessage>
    func executeCommand(_ cmd: String) async throws -> String
    func cancel() async throws
}

enum APSMessage {
    case output(String)
    case error(String)
    case status(APSStatus)
}
```

**Deliverables**:
- [ ] QR code scanning integration
- [ ] WebSocket connection management
- [ ] Message streaming
- [ ] Token persistence (Keychain)
- [ ] Error handling
- [ ] SPM package distribution

#### 4.2: Kotlin SDK (Android)

**Repository**: `android-aps-client` (separate repo)

```kotlin
class APSDeviceClient(
    val profileEndpoint: String,
    val deviceToken: String
) {
    suspend fun connect(): Flow<APSMessage>
    suspend fun executeCommand(cmd: String): String
    suspend fun cancel()
}
```

**Deliverables**:
- [ ] QR code scanning integration
- [ ] WebSocket connection management
- [ ] Message streaming
- [ ] Token persistence (Android Keystore)
- [ ] Error handling
- [ ] Maven/Gradle distribution

#### 4.3: JavaScript/React Native SDK

**Repository**: `js-aps-client` (separate repo)

```typescript
class APSDeviceClient {
    profileEndpoint: string;
    deviceToken: string;

    async connect(): Promise<AsyncGenerator<APSMessage>>;
    async executeCommand(cmd: string): Promise<string>;
    async cancel(): Promise<void>;
}
```

**Deliverables**:
- [ ] QR code scanning
- [ ] WebSocket management
- [ ] React hooks (useAPSDevice)
- [ ] Token persistence
- [ ] npm distribution

### Phase 5: Testing & Security (Week 4)

**Objective**: Comprehensive testing, security audit, and performance validation.

#### 5.1: Unit Tests

**Files**:
- `internal/device/token_test.go` - JWT creation, validation, refresh
- `internal/device/registry_test.go` - CRUD operations, cleanup
- `internal/device/qrcode_test.go` - Encoding, generation
- `internal/device/websocket_test.go` - Connection handling

**Test Coverage**:
- [ ] Token generation and validation
- [ ] Pairing code generation
- [ ] Registry operations
- [ ] QR code encoding/decoding
- [ ] WebSocket message handling
- [ ] Expiry enforcement
- [ ] Device revocation

#### 5.2: Integration Tests

**Files**:
- `tests/integration/device_linking_test.go`

**Test Scenarios**:
- [ ] Full pairing flow (QR → register → connect)
- [ ] Command execution via WebSocket
- [ ] Streaming responses
- [ ] Device revocation
- [ ] Token refresh
- [ ] Concurrent connections
- [ ] Timeout handling
- [ ] Isolation boundary crossing

#### 5.3: End-to-End Tests

**Files**:
- `tests/e2e/device_linking_e2e_test.go`

**Test Scenarios**:
- [ ] Real mobile client pairing
- [ ] Multi-profile device linking
- [ ] Long-running session
- [ ] Network interruption recovery
- [ ] Device approval workflow
- [ ] Concurrent devices

#### 5.4: Security Audit

**Focus Areas**:
- [ ] Token signing verification (RS256)
- [ ] JWT expiry enforcement
- [ ] Pairing code entropy (>128 bits)
- [ ] WebSocket authentication
- [ ] HTTPS enforcement
- [ ] TLS certificate validation
- [ ] Secrets storage encryption
- [ ] Cross-isolation authentication
- [ ] DoS protection (rate limiting)
- [ ] Input validation (command injection)

**Deliverables**:
- [ ] Security audit report
- [ ] Vulnerability assessment
- [ ] Recommendations
- [ ] Mitigation strategies

#### 5.5: Performance Tests

**Benchmarks**:
- [ ] Token generation: <100ms
- [ ] WebSocket connect: <500ms
- [ ] Command execution: <1s (+ command time)
- [ ] Concurrent devices: 100+ simultaneous
- [ ] Memory per connection: <10MB
- [ ] Token refresh: <50ms

**Deliverables**:
- [ ] Performance benchmark report
- [ ] Optimization recommendations
- [ ] Load testing results

---

## Data Model & Schema

### Device Registry (JSON)

```json
{
  "version": "1.0",
  "devices": [
    {
      "device_id": "iphone-alice-001",
      "profile_id": "myagent",
      "device_name": "Alice's iPhone 15",
      "device_os": "iOS",
      "device_version": "17.2.1",
      "device_model": "iPhone15,2",
      "registered_at": "2026-02-12T10:00:00Z",
      "last_seen_at": "2026-02-12T10:30:00Z",
      "expires_at": "2026-02-26T10:00:00Z",
      "token_hash": "sha256:...",
      "status": "active",
      "capabilities": ["run:stateless", "run:streaming"],
      "approval_required": false,
      "approved_at": "2026-02-12T10:05:00Z"
    }
  ]
}
```

### Profile Device Configuration (YAML)

```yaml
# ~/.agents/profiles/<profile-id>/profile.yaml
device_config:
  enabled: true
  port: 8443
  max_devices: 10
  default_expiry: "14d"
  approval_required: false
  allowed_capabilities:
    - "run:stateless"
    - "run:streaming"
    - "monitor:sessions"
```

### Connection Log

```
2026-02-12T10:00:00Z [REGISTER] device_id=iphone-alice-001 device_os=iOS
2026-02-12T10:05:00Z [APPROVE] device_id=iphone-alice-001 approved_by=user
2026-02-12T10:30:00Z [CONNECT] device_id=iphone-alice-001 session_id=sess_123
2026-02-12T10:31:00Z [EXECUTE] device_id=iphone-alice-001 command="echo hello"
2026-02-12T10:31:05Z [DISCONNECT] device_id=iphone-alice-001 session_id=sess_123
```

---

## Critical Files to Create/Modify

### New Files (Core)

1. **`internal/device/server.go`** - Device server with HTTP + WebSocket
2. **`internal/device/qrcode.go`** - QR code generation
3. **`internal/device/token.go`** - JWT token management
4. **`internal/device/registry.go`** - Device registry (CRUD)
5. **`internal/device/websocket.go`** - WebSocket connection handling
6. **`internal/device/isolation.go`** - Isolation boundary integration
7. **`internal/device/secrets.go`** - Secrets manager integration
8. **`internal/device/types.go`** - Shared types and interfaces
9. **`internal/device/errors.go`** - Error types
10. **`internal/device/logger.go`** - Connection and audit logging

### New Files (CLI)

11. **`cmd/device.go`** - Device command root
12. **`internal/cli/device_link.go`** - Link/QR generation command
13. **`internal/cli/device_list.go`** - List devices command
14. **`internal/cli/device_revoke.go`** - Revoke device command
15. **`internal/cli/device_logs.go`** - View device logs command
16. **`internal/cli/device_approve.go`** - Approve device command

### New Files (Testing)

17. **`internal/device/token_test.go`** - Token tests
18. **`internal/device/registry_test.go`** - Registry tests
19. **`internal/device/qrcode_test.go`** - QR code tests
20. **`internal/device/websocket_test.go`** - WebSocket tests
21. **`tests/integration/device_linking_test.go`** - Integration tests
22. **`tests/e2e/device_linking_e2e_test.go`** - E2E tests

### Modified Files

23. **`internal/core/profile.go`** - Add DeviceConfig struct
24. **`cmd/root.go`** - Register device command
25. **`go.mod`** - Add dependencies (websocket, qrcode)
26. **`Makefile`** - Build targets for device server
27. **`docs/user/README.md`** - Link to mobile documentation

### New Documentation

28. **`docs/user/mobile-device-linking.md`** - User guide
29. **`docs/dev/mobile/api-reference.md`** - Device API reference
30. **`docs/dev/mobile/mobile-client-guide.md`** - Client dev guide

---

## Dependencies & Integration

### External Dependencies

1. **`github.com/gorilla/websocket`** - WebSocket protocol
2. **`github.com/skip2/go-qrcode`** - QR code generation
3. **`github.com/golang-jwt/jwt/v5`** - JWT token handling
4. **`github.com/google/uuid`** - Device ID generation

### Internal Dependencies

1. **Profile System** (`internal/core/profile.go`)
   - Profile lookup
   - Profile configuration
   - Profile executor

2. **Secrets Manager** (`internal/core/secrets/`)
   - Store device keys
   - Retrieve credentials
   - Encrypt at rest

3. **Isolation System** (`internal/core/isolation/`)
   - Process isolation
   - Platform isolation
   - Container isolation

4. **Webhook Server** (Plan 5 - `internal/webhook/`)
   - Already provides REST endpoint capability
   - Can extend for device endpoints

5. **Logging System** (`internal/core/logger.go`)
   - Connection logging
   - Command execution logging
   - Error tracking

### Plan Dependencies

**Plan 4: Secrets Management**
- Device key storage
- Token encryption
- Credential injection

**Plan 5: Webhook Server**
- HTTP endpoint infrastructure
- HTTPS/TLS support
- Route registration

---

## Success Criteria

### Functional Requirements

- ✓ QR code generation (PNG + ASCII)
- ✓ Device pairing via pairing code
- ✓ WebSocket connection with JWT auth
- ✓ Command execution via mobile client
- ✓ Streaming responses
- ✓ Device registry persistence
- ✓ Token expiry enforcement
- ✓ Device revocation
- ✓ Multi-device support per profile
- ✓ Isolation boundary respect

### Non-Functional Requirements

- ✓ Token generation: <100ms
- ✓ WebSocket connect: <500ms
- ✓ Support 100+ concurrent devices
- ✓ Memory per connection: <10MB
- ✓ 99% uptime for device connections
- ✓ Comprehensive test coverage (>85%)

### Security Requirements

- ✓ RS256 JWT signing
- ✓ Pairing code entropy (>128 bits)
- ✓ HTTPS only for network communication
- ✓ Token expiry enforcement
- ✓ WebSocket authentication
- ✓ Secrets encryption at rest
- ✓ No plaintext credentials in registry
- ✓ Audit logging

---

## Timeline & Phases

### Phase 1: Infrastructure & API (Weeks 1-2)
- Device server core
- QR code generation
- Token management
- Device registry
- WebSocket handling

**Deliverables**: Working device server + token system

### Phase 2: CLI Commands (Week 2)
- `aps device link`
- `aps device list`
- `aps device revoke`
- `aps device logs`
- `aps device approve`

**Deliverables**: Full CLI device management

### Phase 3: Profile Integration (Week 3)
- Profile configuration
- Isolation integration
- Secrets integration
- Device lifecycle

**Deliverables**: Device linking works across all isolation tiers

### Phase 4: Mobile Libraries (Week 3-4)
- Swift SDK (iOS)
- Kotlin SDK (Android)
- JavaScript SDK (React Native)

**Deliverables**: Published mobile SDKs

### Phase 5: Testing & Security (Week 4)
- Unit tests
- Integration tests
- E2E tests
- Security audit
- Performance testing

**Deliverables**: Comprehensive test coverage + security audit

**Total Timeline**: 4 weeks (1 month)

---

## Risk Mitigation

### Technical Risks

1. **WebSocket Connection Stability**
   - Risk: Mobile network interruptions cause disconnections
   - Mitigation: Automatic reconnection, session resumption
   - Implementation: Heartbeat + exponential backoff

2. **Token Expiry During Operations**
   - Risk: Token expires mid-command execution
   - Mitigation: Token refresh before expiry, long default expiry
   - Implementation: 14-day default, 30-second refresh buffer

3. **Cross-Isolation Authentication**
   - Risk: Device auth doesn't work across isolation boundaries
   - Mitigation: Isolation-aware token validation
   - Implementation: Tests for each isolation tier

4. **Performance Under Load**
   - Risk: 100+ concurrent devices cause slowdowns
   - Mitigation: Connection pooling, async message handling
   - Implementation: Benchmarking, optimization phase

### Security Risks

1. **Token Theft**
   - Risk: JWT token intercepted over network
   - Mitigation: HTTPS only, token expiry
   - Implementation: Enforce TLS, short-lived tokens

2. **Pairing Code Brute Force**
   - Risk: Attacker guesses pairing code
   - Mitigation: Rate limiting, entropy, expiry
   - Implementation: 6+ character code, 15-minute expiry, 3-attempt limit

3. **Device Impersonation**
   - Risk: Attacker uses revoked device token
   - Mitigation: Token blacklist, signature verification
   - Implementation: Maintain revoked token list

4. **Command Injection**
   - Risk: Mobile app sends malicious commands
   - Mitigation: Input validation, capability restrictions
   - Implementation: Whitelist allowed commands/capabilities

### Operational Risks

1. **Device Registry Corruption**
   - Risk: Concurrent write causes data loss
   - Mitigation: Atomic file operations, locking
   - Implementation: `sync.Mutex` + rename-on-write

2. **Orphaned WebSocket Connections**
   - Risk: Zombie connections consume resources
   - Mitigation: Heartbeat, timeout, cleanup
   - Implementation: 5-minute idle timeout

3. **Key Rotation Complexity**
   - Risk: Device key rotation breaks active tokens
   - Mitigation: Support multiple keys, grace period
   - Implementation: Key versioning (KID in JWT)

---

## References & Resources

### Official Standards

- **JWT (RFC 7519)**: https://tools.ietf.org/html/rfc7519
- **WebSocket (RFC 6455)**: https://tools.ietf.org/html/rfc6455
- **HTTPS/TLS**: https://tools.ietf.org/html/rfc8446

### Go Libraries

- **Gorilla WebSocket**: https://github.com/gorilla/websocket
- **go-qrcode**: https://github.com/skip2/go-qrcode
- **golang-jwt**: https://github.com/golang-jwt/jwt
- **Google UUID**: https://github.com/google/uuid

### APS Components

- **Profile System**: `internal/core/profile.go`
- **Isolation Architecture**: `specs/001-build-cli-core/isolation-architecture.md`
- **Secrets Management**: Plan 4
- **Webhook Server**: Plan 5

### Mobile Development

- **Swift Concurrency**: https://developer.apple.com/documentation/swift/concurrency
- **Kotlin Coroutines**: https://kotlinlang.org/docs/coroutines-overview.html
- **React Native**: https://reactnative.dev/

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **QR Code for Pairing** | Simple, human-friendly, no manual token entry required |
| **JWT Tokens** | Industry standard, stateless, built-in expiry support |
| **WebSocket for Streaming** | Low-latency, bidirectional, built-in message framing |
| **14-Day Default Expiry** | Balance between security and usability |
| **Per-Profile Keys** | Isolation: one compromised profile doesn't affect others |
| **Device Registry (JSON)** | Human-readable, version-controllable, no DB dependency |
| **Optional Device Approval** | Trust-by-default for personal use, approval for high-security profiles |
| **Separate Mobile SDKs** | Language-agnostic, easier to maintain, independent release cycle |
| **Capability-Based Auth** | Fine-grained control, future extensibility |
| **HTTPS Required** | Protects tokens in transit, standard security practice |

---

## Future Enhancements (Out of Scope)

1. **Biometric Authentication** - Add fingerprint/face unlock for token access
2. **Device Geofencing** - Restrict device access by location
3. **Command Templates** - Pre-approved command sequences
4. **Mobile UI** - Native iOS/Android apps (beyond SDK)
5. **Device Groups** - Manage multiple devices as one unit
6. **Push Notifications** - Notify device of profile status changes
7. **Offline Mode** - Queue commands when offline, sync on reconnect
8. **Analytics** - Device usage stats, command frequency tracking

---

## Summary

**Plan 6: Mobile Device Linking** enables secure, user-friendly mobile access to APS profiles via QR code-based pairing. The architecture leverages JWT tokens for authentication, WebSocket for real-time communication, and respects all existing isolation boundaries. Implementation follows a phased approach with comprehensive testing and security hardening, delivering a complete system in 4 weeks.

**Key Deliverables**:
- Device server with QR code generation
- Mobile-friendly REST + WebSocket API
- Device registry and management CLI commands
- Isolation-aware authentication
- Mobile client libraries (iOS, Android, React Native)
- Comprehensive test suite and security audit

**Success Criteria**:
- Functional QR-based pairing flow
- Command execution with streaming responses
- 100+ concurrent device support
- Sub-second latency
- Production-grade security
- 85%+ test coverage
