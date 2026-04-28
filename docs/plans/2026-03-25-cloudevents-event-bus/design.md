# CloudEvents Event Bus Integration — aps

**Author:** $USER
**Task:** T-0083
**Ref:** `aok/docs/aok-architecture.md#7`
**Status:** Draft

---

## Task list

- [ ] [Define `events.Event` + `Bus` interface](#bus-interface) — `internal/events/`
- [ ] [Implement in-process bus (noop/sync)](#in-process-bus) — `internal/events/bus.go`
- [ ] [Implement config-driven transport adapter](#transport) — `internal/events/transport.go`
- [ ] [Wire publisher calls into core subsystems](#publisher-integration) — core/*
- [ ] [Subscribe to AOK event stream](#subscriber-integration) — `internal/events/subscriber.go`
- [ ] [Register event types at startup](#registration) — `cmd/aps/main.go`
- [ ] [Config schema + validation](#config) — `internal/core/config.go`
- [ ] [Unit + integration tests](#testing)

---

## Context

AOK (Operational Kernel Layer) is the shared graph + event bus for the hop-top ecosystem.
All peer systems are **publisher + subscriber**. aps's domain: agent identity and isolation.

Per `aok-architecture.md §7`:

- All events: CloudEvents v1.0 (`github.com/cloudevents/sdk-go/v2`)
- Source URI: AOK instance URI (configured)
- Transport: config-driven — HTTP | Kafka | NATS | AMQP
- Delivery: at-least-once, ordered per entity
- Peer systems register published event types in AOK's Events registry at startup

**aps role: publisher + subscriber**

---

## Events

### Published by aps

| Type | Trigger |
|------|---------|
| `agent.profile.created` | new profile initialised |
| `agent.profile.updated` | profile config/metadata changed |
| `agent.capability.added` | capability added to profile |
| `agent.capability.revoked` | capability removed from profile |
| `agent.session.started` | session bootstrapped |
| `agent.session.terminated` | session ended |

### Subscribed from AOK

| Type | aps reaction |
|------|-------------|
| `entity.created` (actor/agent) | sync actor record into profile |
| `capability.assigned` | update profile capabilities |
| `action.performed` (actor=agent) | update session state / audit log |

---

## Design

### Bus interface

```
// internal/events/bus.go
type Bus interface {
    Publish(ctx, Event) error
    Subscribe(eventType string, handler func(ctx, Event) error)
    Close() error
}
```

Mirrors `ctxt/pkg/pluginapi.Bus` — consistent across the ecosystem.

### Event struct

```
// internal/events/event.go
type Event struct {
    ID              string
    Source          string
    SpecVersion     string          // "1.0"
    Type            string
    DataContentType string          // "application/json"
    SchemaURL       string          // ontology version URL
    Time            time.Time
    Data            json.RawMessage
}
```

Constructor: `events.New(source, eventType string, data any) (Event, error)`

### In-process bus

Default impl (AOK not configured): sync, in-process — no external dep. Used in tests
and standalone mode. Diagram: [bus-v1.mmd](bus-v1.mmd)

### Transport

Config block in `aps.yaml`:

```
event_bus:
  enabled: false               # off by default
  source: "aps://hostname"     # CloudEvents `source`
  transport: http              # http | kafka | nats | amqp
  http:
    endpoint: https://aok.example.com/events
  kafka:
    brokers: [...]
    topic: okl.events
  nats:
    url: nats://...
    subject: okl.events
```

Transport adapter wraps `github.com/cloudevents/sdk-go/v2` protocol bindings.
Falls back to in-process bus when `enabled: false`.

### Publisher integration

Core subsystems emit events via `Bus.Publish` at mutation points:

| Subsystem | File | Hook point |
|-----------|------|-----------|
| Profile create/update | `internal/core/profile.go` | after persist |
| Capability add/revoke | `internal/core/capability/registry.go` | after persist |
| Session start/end | `internal/core/session/registry.go` | after state change |

Bus injected via constructor (DI) — no global state.

### Subscriber integration

`internal/events/subscriber.go` — background goroutine, subscribes at startup:

```
subscriber.Subscribe("entity.created", handlers.OnActorCreated)
subscriber.Subscribe("capability.assigned", handlers.OnCapabilityAssigned)
subscriber.Subscribe("action.performed", handlers.OnActionPerformed)
```

Dead letter: failed handlers log + write to
`$APS_DATA_PATH/events/dlq/<timestamp>-<id>.json` for manual replay.

### Registration

At startup (`cmd/aps/main.go`), register published event types with AOK:

```
bus.RegisterEventTypes(ctx, []EventTypeRegistration{
    {Type: "agent.profile.created", Schema: ..., Version: ontologyVersion},
    ...
})
```

No-op when `event_bus.enabled: false`.

---

## Dependencies

- `github.com/cloudevents/sdk-go/v2` — SDK (new dep)
- `github.com/google/uuid` — already in go.mod

---

## Testing

- Unit: in-process bus publish/subscribe, event construction, DLQ write
- Integration: HTTP transport (testcontainers or mock server)
- E2E: not required for initial impl (no user-facing CLI surface)

---

## References

- `aok/docs/aok-architecture.md#7` — full event bus spec
- `ctxt/internal/events/cloudevent.go` — ecosystem reference impl
- `ctxt/pkg/pluginapi/pluginapi.go` — Bus interface reference
- `wsm` T-0035 description — peer system context
