# APS x402 Integration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement
> this plan task-by-task.

**Goal:** Integrate `hop.top/x402` into APS so profiles can earn (gate endpoints)
and spend (pay 402-gated resources) via non-custodial wallets attached as capability
bundles.

**Architecture:** `internal/core/payment` defines the `PaymentProvider` interface
mirroring how `APSCore` exposes `Store*` today; `internal/adapters/x402` wires
`hop.top/x402` into HTTP, A2A, and MCP protocol adapters; `cap:payment` capability
bundle attaches wallet identity to profiles via the existing capability system.

**Tech Stack:** Go 1.21+, `hop.top/x402` (new), `mark3labs/x402-go`,
`mark3labs/mcp-go-x402`, existing APS adapter registry pattern.

---

## Task List

- [ ] T-01: Add `hop.top/x402` dependency
- [ ] T-02: `internal/core/payment` — interface + types
- [ ] T-03: Extend `APSCore` with `Payment()` method
- [ ] T-04: `cap:payment` capability bundle
- [ ] T-05: `internal/adapters/x402` — HTTP adapter
- [ ] T-06: `internal/adapters/x402` — A2A extension
- [ ] T-07: `internal/adapters/x402` — MCP adapter
- [ ] T-08: Register x402 adapters in adapter registry
- [ ] T-09: CLI commands: `aps wallet` subcommand
- [ ] T-10: Integration tests

---

### Task T-01: Add dependency

**Files:**
- Modify: `go.mod`

**Step 1:**

```
go get hop.top/x402
```

**Step 2: Verify build**

```
make build
```

**Step 3: Commit**

```
git commit -m "chore(deps): add hop.top/x402"
```

---

### Task T-02: `internal/core/payment` — interface + types

**Files:**
- Create: `internal/core/payment/interface.go`
- Create: `internal/core/payment/types.go`
- Test: `internal/core/payment/payment_test.go`

**Note:** Thin re-export layer — types alias `hop.top/x402/payment` so APS
internals never import the adapter package directly.

**Step 1: Write failing test** (`internal/core/payment/payment_test.go`)

```go
func TestPaymentRequestValid(t *testing.T) {
    req := payment.PaymentRequest{
        Amount: "1.00", Currency: "USDC",
        Network: "base", Resource: "/v1/run",
    }
    assert.Equal(t, "base", req.Network)
}
```

**Step 2: Implement `internal/core/payment/interface.go`**

```go
// Re-export hop.top/x402/payment interfaces for APS consumers.
// Nothing in this package imports adapter code.
package payment

import x402payment "hop.top/x402/payment"

type PaymentProvider = x402payment.PaymentProvider
type PaymentGate     = x402payment.PaymentGate
type PaymentRequest  = x402payment.PaymentRequest
type PaymentReceipt  = x402payment.PaymentReceipt
type PaymentChallenge = x402payment.PaymentChallenge
```

**Step 3: Run tests — PASS**

```
go test ./internal/core/payment/...
```

**Step 4: Commit**

```
git commit -m "feat(core/payment): payment interface re-export layer"
```

---

### Task T-03: Extend APSCore with Payment()

**Files:**
- Modify: `internal/core/protocol/interface.go`
- Modify: `internal/core/protocol/core.go`
- Test: `internal/core/protocol/core_adapter_test.go`

**Step 1: Add to `APSCore` interface** (`interface.go`)

```go
// Payment returns the active payment provider (nil if cap:payment not enabled)
Payment() payment.PaymentProvider
PaymentGate() payment.PaymentGate
```

**Step 2: Add to `APSAdapter`** (`core.go`)

```go
func (a *APSAdapter) Payment() payment.PaymentProvider {
    return a.paymentProvider // set during init if cap:payment enabled
}

func (a *APSAdapter) PaymentGate() payment.PaymentGate {
    return a.paymentGate
}
```

**Step 3: Update `APSAdapter` struct** (`core.go`)

```go
type APSAdapter struct {
    // ... existing fields ...
    paymentProvider payment.PaymentProvider
    paymentGate     payment.PaymentGate
}
```

**Step 4: Add test** — assert `Payment()` returns nil by default, non-nil
after `WithPayment(provider, gate)` option applied.

**Step 5: Run tests — PASS**

```
go test ./internal/core/protocol/...
```

**Step 6: Commit**

```
git commit -m "feat(core): extend APSCore with Payment() and PaymentGate()"
```

---

### Task T-04: `cap:payment` capability bundle

**Files:**
- Create: `internal/skills/payment.go`
- Modify: `internal/skills/registry.go` (register new skill)
- Test: `internal/skills/payment_test.go`

**Note:** Follows existing skill/capability bundle pattern in `internal/skills/`.
When a profile has `cap:payment`, `APSAdapter` initialises x402 provider + EVM
wallet for that profile's XDG data dir.

**Step 1: Check existing skill registration pattern**

```
cat internal/skills/registry.go
```

**Step 2: Write failing test**

```go
func TestPaymentCapabilityBundleName(t *testing.T) {
    b := skills.PaymentCapability()
    assert.Equal(t, "cap:payment", b.Name())
}
```

**Step 3: Implement `internal/skills/payment.go`**

```go
package skills

import (
    "hop.top/x402/capability"
)

type PaymentSkill struct {
    bundle capability.PaymentBundle
}

func PaymentCapability() *PaymentSkill {
    return &PaymentSkill{bundle: capability.DefaultPaymentBundle()}
}

func (s *PaymentSkill) Name() string { return "cap:payment" }
```

**Step 4: Run tests — PASS**

```
go test ./internal/skills/...
```

**Step 5: Commit**

```
git commit -m "feat(skills): cap:payment capability bundle"
```

---

### Task T-05: `internal/adapters/x402` — HTTP adapter

**Files:**
- Create: `internal/adapters/x402/adapter.go`
- Create: `internal/adapters/x402/http.go`
- Test: `internal/adapters/x402/adapter_test.go`

**Note:** Implements `protocol.HTTPProtocolAdapter`. Registers a middleware on the
APS HTTP mux that enforces x402 payment on routes tagged with `cap:payment`.

**Step 1: Write failing test**

```go
func TestX402HTTPAdapterName(t *testing.T) {
    a := x402adapter.New(nil, nil)
    assert.Equal(t, "x402", a.Name())
}
```

**Step 2: Implement `internal/adapters/x402/adapter.go`**

```go
package x402adapter

import (
    "net/http"
    "hop.top/aps/internal/core/protocol"
)

type X402Adapter struct {
    provider protocol.PaymentProvider
    gate     protocol.PaymentGate
    status   string
}

var _ protocol.HTTPProtocolAdapter = (*X402Adapter)(nil)

func New(provider protocol.PaymentProvider,
    gate protocol.PaymentGate) *X402Adapter {
    return &X402Adapter{provider: provider, gate: gate, status: "stopped"}
}

func (a *X402Adapter) Name() string { return "x402" }

func (a *X402Adapter) RegisterRoutes(mux *http.ServeMux,
    core protocol.APSCore) error {
    // Wrap existing mux with x402 middleware for /run endpoints
    // Delegate to hop.top/x402/adapters/x402 HTTP middleware
    return nil
}
```

**Step 3: Run tests — PASS**

```
go test ./internal/adapters/x402/...
```

**Step 4: Commit**

```
git commit -m "feat(adapters/x402): HTTP protocol adapter"
```

---

### Task T-06: A2A extension

**Files:**
- Create: `internal/adapters/x402/a2a.go`
- Test: `internal/adapters/x402/a2a_test.go`

**Note:** Extends the existing A2A adapter (`internal/adapters/agentprotocol/`)
with outbound payment headers and inbound payment verification. Does not modify
existing A2A code — composes via middleware.

**Step 1: Write failing test**

```go
func TestA2APaymentMiddlewareAddsHeader(t *testing.T) {
    p := &mockProvider{}
    mw := x402adapter.NewA2AMiddleware(p)
    req := httptest.NewRequest("POST", "/tasks/send", nil)
    mw.AddPaymentHeaders(req, "1.00", "USDC", "base")
    assert.NotEmpty(t, req.Header.Get("X-Payment"))
}
```

**Step 2: Implement `internal/adapters/x402/a2a.go`**

> Wrap `hop.top/x402/adapters/a2a`. `AddPaymentHeaders` for outbound;
> `VerifyPaymentHeaders` for inbound.

**Step 3: Run tests — PASS**

```
go test ./internal/adapters/x402/... -run TestA2A
```

**Step 4: Commit**

```
git commit -m "feat(adapters/x402): A2A payment header extension"
```

---

### Task T-07: MCP adapter

**Files:**
- Create: `internal/adapters/x402/mcp.go`
- Test: `internal/adapters/x402/mcp_test.go`

**Note:** Wraps `hop.top/x402/adapters/mcp` which itself wraps `mcp-go-x402`.
Injects `x402_pay` tool into MCP sessions managed by APS.

**Step 1: Write failing test**

```go
func TestMCPAdapterProvidesPayTool(t *testing.T) {
    a := x402adapter.NewMCPAdapter(nil)
    tools := a.MCPTools()
    names := toolNames(tools)
    assert.Contains(t, names, "x402_pay")
}
```

**Step 2: Implement `internal/adapters/x402/mcp.go`**

> Delegate to `hop.top/x402/adapters/mcp`. Expose `MCPTools() []mcp.Tool` for
> registration with the MCP server managed by APS.

**Step 3: Run tests — PASS**

```
go test ./internal/adapters/x402/... -run TestMCP
```

**Step 4: Commit**

```
git commit -m "feat(adapters/x402): MCP payment tool adapter"
```

---

### Task T-08: Register x402 in adapter registry

**Files:**
- Modify: `internal/adapters/registry.go`

**Step 1: Add to `RegisterDefaults()`**

```go
if core.HasCapability("cap:payment") {
    x402 := x402adapter.New(core.Payment(), core.PaymentGate())
    reg.Register("x402", x402)
}
```

**Step 2: Run full test suite**

```
make test
```

**Step 3: Commit**

```
git commit -m "feat(adapters): register x402 adapter when cap:payment enabled"
```

---

### Task T-09: CLI — `aps wallet` subcommand

**Files:**
- Create: `internal/cli/wallet.go`
- Modify: `cmd/root.go` (register subcommand)
- Test: `internal/cli/wallet_test.go`

**Commands to implement:**

```
aps wallet create --network base       # generate new EVM wallet for profile
aps wallet show                        # display address + network
aps wallet balance                     # query on-chain balance
```

**Step 1: Write failing test**

```go
func TestWalletCreateCmd(t *testing.T) {
    cmd := cli.NewWalletCmd(mockCore)
    assert.Equal(t, "wallet", cmd.Use)
    _, _, err := cmd.ExecuteC()
    require.NoError(t, err)
}
```

**Step 2: Implement `internal/cli/wallet.go`**

> Follow existing CLI pattern in `internal/cli/`. Use cobra subcommands.
> Delegate to `APSCore.Payment()` for wallet operations.

**Step 3: Run tests — PASS**

```
go test ./internal/cli/... -run TestWallet
```

**Step 4: Commit**

```
git commit -m "feat(cli): aps wallet subcommand"
```

---

### Task T-10: Integration tests

**Files:**
- Create: `tests/x402_integration_test.go`

**Note:** Uses x402 test facilitator (no real chain). Tests:
1. Profile with `cap:payment` gets wallet on first run
2. HTTP endpoint returns 402 without payment token
3. Payment token accepted, request proceeds

**Step 1: Write test (build tag `integration`)**

**Step 2: Run**

```
go test -tags integration ./tests/...
```

**Step 3: Commit**

```
git commit -m "test(integration): x402 payment flow in APS"
```
