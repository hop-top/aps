# AGNTCY Integration Gap Analysis

Internal engineering roadmap for making APS a first-class AGNTCY citizen.

## Context

AGNTCY is a Linux Foundation project (Cisco, Google Cloud, Oracle, Dell, Red Hat, 75+ companies) that standardizes multi-agent interoperability through six components:

1. **OASF** — Open Agentic Schema Framework (agent description)
2. **Agent Directory** — distributed discovery and announcement
3. **SLIM** — Secure Low-Latency Interactive Messaging
4. **Identity** — DIDs and Verifiable Credentials
5. **Observability** — OpenTelemetry-based agent telemetry
6. **Security** — MLS encryption and trust verification

APS already implements protocols that overlap with AGNTCY's scope: A2A (via `a2aproject/a2a-go`), Agent Protocol (ACP HTTP adapter), webhooks, and IPC. These are self-contained: profiles are discoverable only locally, messages travel over plain HTTP/gRPC without end-to-end encryption, and there is no agent identity or trust verification.

**Goal**: Make APS profiles discoverable, identifiable, messageable, observable, and trust-verified through AGNTCY's open standards — without losing APS's local-first, profile-centric architecture.

**Constraint**: APS must remain functional without AGNTCY. All integration is opt-in per profile, following the existing protocol toggle pattern (`aps a2a toggle`, `aps webhook toggle`).

---

## Layer 1: Discovery (OASF + Agent Directory)

### What AGNTCY Provides

**OASF** is an OCI-based schema for describing agents. It is a superset of A2A Agent Cards and supports A2A agents, MCP servers, and extensible formats. Records are stored as OCI artifacts via ORAS.

**Agent Directory Service (ADS)** provides distributed discovery over a Kademlia DHT. Agents register OASF records; consumers query by capability taxonomy. A Go SDK is available at `github.com/agntcy/dir/client`. Runtime Discovery auto-detects labeled containers in Docker and Kubernetes.

### What APS Has Today

- `GenerateAgentCardFromProfile()` in `internal/a2a/agentcard.go` produces an A2A `AgentCard` from a profile (name, URL, skills, capabilities, transport)
- Local agent card cache at `<data>/a2a/agent-cards/`
- `internal/a2a/resolver.go` resolves remote agent addresses (local only)
- Skills system populates `AgentSkill` entries from discovered skills

### Gaps

1. ~~**No OASF record generation.**~~ **Implemented.** `GenerateOASFRecord()` in `internal/agntcy/discovery/oasf.go` produces full records with capabilities, transport endpoints, and DID identity when configured.
2. ~~**No Directory registration.**~~ **Implemented.** `internal/agntcy/discovery/client.go` wraps Register/Deregister/Discover/Show. gRPC calls are stubbed pending `github.com/agntcy/dir/client` adoption.
3. ~~**No Directory-based discovery.**~~ **Implemented.** `aps directory discover --capability <query>` provides capability-based search (results depend on Directory service availability).
4. **No Runtime Discovery labels.** APS does not emit Docker or Kubernetes labels for auto-discovery. (Future work.)

### Implementation (Shipped)

- Package: `internal/agntcy/discovery/` — `types.go`, `oasf.go`, `client.go`
- Profile config: `Directory *DirectoryConfig` (Endpoint, AutoRegister, AutoRefresh)
- Capability toggle: `"agntcy-directory"`
- CLI: `aps directory register|discover|deregister|show` (alias: `aps dir`)
- Tests: 13 passing (`oasf_test.go`, `client_test.go`)

---

## Layer 2: Messaging (SLIM)

### What AGNTCY Provides

**SLIM** extends gRPC with pub/sub, streaming, and fire-and-forget patterns. It has three layers: data plane (Rust, message routing), session layer (MLS end-to-end encryption, group management), and control plane (Go, node orchestration).

Language bindings exist for Python, C#, JS/TS, and Kotlin. Go is listed as "coming soon." SLIM acts as transport for A2A, MCP, and custom protocols via SLIMRPC.

### What APS Has Today

- `internal/a2a/transport/` with a pluggable `Transport` interface: `Type()`, `Send()`, `Receive()`, `Close()`, `IsHealthy()`
- Four implementations: HTTP, gRPC, IPC (filesystem), plus a selector
- A2A server runs standalone HTTP at a configurable address
- Collaboration system (`internal/core/collaboration/`) routes tasks between agents locally

### Gaps

1. ~~**No SLIM transport.**~~ **Partially implemented.** `internal/a2a/transport/slim.go` implements the `Transport` interface as a stub on `main`. A full implementation using `github.com/agntcy/slim-bindings-go` (CGO, App/Session/Publish API) exists on branch `feat/agntcy-slim-transport`.
2. **No end-to-end encryption.** MLS encryption is available via the SLIM transport's `EnableMLS` session config when using the full implementation.
3. **No pub/sub for collaboration.** The collab system uses direct task dispatch. SLIM's pub/sub would enable workspace-level event streaming without polling. (Future work.)
4. **No SLIMRPC.** APS uses standard gRPC. SLIMRPC adds agent-aware routing. (Future work.)

### Implementation (Partial)

- `internal/a2a/transport/slim.go` — stub on `main`, full implementation on `feat/agntcy-slim-transport`
- `A2AConfig.ProtocolBinding` accepts `"slim"` alongside `jsonrpc`, `grpc`, `http`
- Selector: `TransportSLIM` added to `TransportPriority`
- Agent card: `mapProtocolBindingToTransport()` handles `"slim"` binding
- **Blocker for `main` merge**: `github.com/agntcy/slim-bindings-go` not yet on pkg.go.dev; branch uses local shim at `third_party/slim-bindings-go/` with `replace` directive

---

## Layer 3: Identity (DIDs + Verifiable Credentials)

### What AGNTCY Provides

**Decentralized Identifiers (DIDs)** give each agent a universally unique, cryptographically verifiable ID with no central authority. **Verifiable Credentials (VCs)**, called "Agent Badges," contain agent metadata, schema definitions, and capability attestations. Issuers are organizations; anyone can verify.

Identity assignment follows three principles: open (no central authority), collision-free (universally unique), and verifiable (backed by VCs). Supported identity providers include Okta, Microsoft AD, Entra ID, Auth0, and Google ID alongside DIDs. Go modules are at `github.com/agntcy/identity`.

### What APS Has Today

- Profile `ID` is a user-chosen string (e.g., "invoice-bot") — not globally unique, not cryptographically backed
- `secrets.env` holds API keys per profile as flat key=value pairs
- A2A security schemes: `apikey`, `mtls`, `openid` — configured but basic
- SSH keys exist per profile but serve only git operations
- No cryptographic identity, message signing, or credential verification

### Gaps

1. ~~**No agent identity.**~~ **Implemented.** `aps identity init` generates a DID (`did:key` or `did:web`) and Ed25519 key pair per profile.
2. ~~**No message signing.**~~ **Implemented.** `internal/agntcy/identity/signer.go` provides `SignMessage()`/`VerifySignature()` with Ed25519. Trust verifier checks `X-Agent-Signature` headers on inbound A2A.
3. ~~**No credential issuance or verification.**~~ **Implemented.** `internal/agntcy/identity/badge.go` issues W3C Verifiable Credentials with `Ed25519Signature2020` proofs. `VerifyBadge()` validates signature against issuer DID.
4. ~~**No DID resolution.**~~ **Implemented.** `ResolveDID()` resolves `did:key` to DID Documents with verification methods. `ExtractPublicKeyFromDID()` extracts Ed25519 public keys.

### Implementation (Shipped)

- Package: `internal/agntcy/identity/` — `signer.go`, `did.go`, `badge.go`
- Profile config: `Identity *IdentityConfig` (DID, KeyPath, Badges)
- Capability toggle: `"agntcy-identity"`
- CLI: `aps identity init|show|verify` + `aps identity badge issue|verify` (alias: `aps id`)
- DID embedded in OASF records via `discovery/oasf.go`
- Tests: 18 passing (`signer_test.go`, `did_test.go`, `badge_test.go`)

---

## Layer 4: Observability

### What AGNTCY Provides

**Observe SDK** provides OpenTelemetry-based instrumentation for multi-agent systems. Traces span agent boundaries (Agent A calls Agent B, the trace follows). Evaluation tooling assesses agent performance and efficiency. Integrates with SLIM and Directory.

### What APS Has Today

- `internal/logging/` uses Charmbracelet `log` for structured key-value logging to stderr
- A2A task storage records state but not timing or call chains
- Collaboration audit log records events in local SQLite but produces no exportable telemetry
- No tracing, no metrics, no spans

### Gaps

1. ~~**No OpenTelemetry.**~~ **Implemented.** `internal/agntcy/observability/middleware.go` propagates W3C Trace Context and instruments A2A method calls as spans.
2. ~~**No metrics.**~~ **Implemented.** `internal/agntcy/observability/metrics.go` registers counters (TasksProcessed, MessagesRouted, WebhookDeliveries) and a histogram (TaskDuration).
3. ~~**No distributed tracing.**~~ **Implemented.** A2A `server.go` `Before()`/`After()` hooks create and end spans per request when `agntcy-observability` capability is enabled.
4. ~~**No telemetry export.**~~ **Implemented.** `provider.go` supports OTLP (gRPC) and stdout exporters with configurable sampling rate.

### Implementation (Shipped)

- Package: `internal/agntcy/observability/` — `provider.go`, `metrics.go`, `middleware.go`
- Profile config: `Observability *ObservabilityConfig` (Exporter, Endpoint, SamplingRate)
- Capability toggle: `"agntcy-observability"`
- CLI: `aps observability toggle` (aliases: `aps otel`, `aps o11y`)
- Wired into: `internal/a2a/server.go` Before/After, `internal/cli/a2a/server.go` init/shutdown
- Tests: 9 passing (`provider_test.go`, `metrics_test.go`)

---

## Layer 5: Security (MLS + Trust)

### What AGNTCY Provides

**MLS** (Message Layer Security) is built into SLIM's session layer. Messages stay encrypted through intermediaries. Supports post-quantum algorithms. MLS groups enable encrypted multi-party communication.

**Trust verification** checks sender identity (DIDs + VCs) before accepting messages.

### What APS Has Today

- Webhook HMAC-SHA256 signature verification (`X-APS-Signature` header)
- A2A security schemes declared in agent card (`apikey`, `mtls`, `openid`) but no enforcement logic
- Isolation system (process/platform/container sandboxing) protects the local machine, not the network
- No payload encryption, no trust model for inbound tasks

### Gaps

1. **No payload encryption.** Agent messages are plaintext over TLS at best. MLS encryption is available only via the SLIM transport (branch `feat/agntcy-slim-transport`).
2. ~~**No trust verification.**~~ **Implemented.** `internal/agntcy/trust/verifier.go` verifies sender DID (`X-Agent-DID` header) and Ed25519 signatures (`X-Agent-Signature` header) on inbound A2A requests.
3. ~~**No trust policy.**~~ **Implemented.** `TrustConfig` supports `RequireIdentity` and `AllowedIssuers` allowlist. Wired into `server.go` `Before()` hook.
4. **No group encryption.** Collaboration workspaces communicate in the clear. (Requires SLIM transport.)

### Implementation (Shipped)

- Package: `internal/agntcy/trust/` — `policy.go`, `verifier.go`
- Profile config: `Trust *TrustConfig` (RequireIdentity, AllowedIssuers)
- Capability toggle: `"agntcy-trust"`
- CLI: `aps policy trust set|show`
- Wired into: `internal/a2a/server.go` Before() — rejects requests when trust policy requires identity
- Tests: 8 passing (`verifier_test.go`)

---

## Dependencies and Implementation Order

```
                    ┌─────────────┐
                    │ Observability│  (independent)
                    │   Layer 4    │
                    └─────────────┘

┌─────────────┐     ┌─────────────┐
│  Discovery   │◄────│  Identity   │
│   Layer 1    │     │   Layer 3   │
└──────┬───────┘     └──────┬──────┘
       │                    │
       │              ┌─────▼──────┐
       │              │  Security  │
       │              │  Layer 5   │
       │              └─────┬──────┘
       │                    │
       └────────┬───────────┘
                │
         ┌──────▼──────┐
         │    SLIM     │
         │  Layer 2    │  (blocked: Go binding unreleased)
         └─────────────┘
```

| Phase | Layer | Status | Branch |
|-------|-------|--------|--------|
| **1** | Observability (L4) | **Shipped** | `main` |
| **2** | Discovery (L1) | **Shipped** (gRPC stubs pending `dir/client`) | `main` |
| **3** | Identity (L3) | **Shipped** | `main` |
| **4** | Security (L5) | **Shipped** | `main` |
| **5** | SLIM (L2) | **Implemented** (stub on main, full on branch) | `feat/agntcy-slim-transport` |

Phases 1–4 shipped on `main`. Phase 5 has a full implementation on `feat/agntcy-slim-transport` using a local shim of `github.com/agntcy/slim-bindings-go`; merge to `main` when upstream publishes to pkg.go.dev.

---

## Profile Config (Unified)

All new config fields are pointer types (nil = not configured, opt-in per profile):

```yaml
# Layer 1: Discovery
directory:
  endpoint: "https://dir.agntcy.org"
  auto_register: false
  auto_refresh: true

# Layer 3: Identity
identity:
  did: "did:web:example.com:agents:invoice-bot"
  key_path: "~/.local/share/aps/profiles/invoice-bot/identity.key"
  badges: ["~/.local/share/aps/profiles/invoice-bot/badges/"]

# Layer 4: Observability
observability:
  exporter: "otlp"            # otlp | stdout | none
  endpoint: "http://localhost:4317"
  sampling_rate: 1.0

# Layer 5: Trust
trust:
  require_identity: false
  allowed_issuers: []
```

SLIM adds `"slim"` as a value for the existing `a2a.protocol_binding` field.

## Capability Toggles

Consistent with existing toggles (`a2a`, `webhooks`, `agent-protocol`):

- `"agntcy-directory"` — enables Directory registration
- `"agntcy-identity"` — enables DID-based identity
- `"agntcy-observability"` — enables telemetry export
- `"agntcy-trust"` — enables inbound trust verification

## CLI Commands (New)

```
aps directory register [--profile <id>]
aps directory discover --capability <query>
aps directory deregister [--profile <id>]
aps directory show [--profile <id>]

aps identity init [--profile <id>] [--method did:web|did:key]
aps identity show [--profile <id>]
aps identity verify <did>
aps identity badge issue [--profile <id>] --capability <name>
aps identity badge verify <badge-file>

aps observability toggle [--profile <id>] [--enabled on|off]

aps policy trust set [--profile <id>] --require-identity [--allowed-issuers <did>,...]
aps policy trust show [--profile <id>]
```

## Package Structure

```
internal/agntcy/
  discovery/     # Layer 1: OASF record generation + Directory client
  identity/      # Layer 3: DID management + VC issuance/verification
  trust/         # Layer 5: Inbound trust verification middleware
  observability/ # Layer 4: OTel instrumentation + export

internal/a2a/transport/
  slim.go        # Layer 2: SLIM transport implementation
```

## References

- [AGNTCY Documentation](https://docs.agntcy.org/)
- [AGNTCY GitHub Organization](https://github.com/agntcy)
- [SLIM Specification](https://spec.slim.agntcy.org/draft-mpsb-agntcy-slim.html)
- [SLIM Repository](https://github.com/agntcy/slim)
- [Agent Directory Repository](https://github.com/agntcy/dir)
- [Identity Repository](https://github.com/agntcy/identity)
- [OASF Repository](https://github.com/agntcy/oasf)
- [Linux Foundation Announcement](https://www.linuxfoundation.org/press/linux-foundation-welcomes-the-agntcy-project-to-standardize-open-multi-agent-system-infrastructure-and-break-down-ai-agent-silos)
