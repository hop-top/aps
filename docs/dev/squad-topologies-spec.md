# Squad Topologies — Structured Spec
## Generic Framework v1.0

> *From Team Topologies (teamtopologies.com) adapted for autonomous agent systems.*
> *The unit shifts from humans to agents. The constraint shifts from cognitive load to context load.*

---

## Core Premise

Squad Topologies applies the Team Topologies mental model to agent system design. It answers: **how do you organize autonomous agents to maximize value flow and minimize context overhead?**

The three foundational shifts from the human version:

| Dimension | Team Topologies | Squad Topologies |
|---|---|---|
| Primary unit | Human team | Agent squad |
| Primary constraint | Cognitive load | Context load (working memory) |
| Communication default | Ad-hoc | Declared interaction contracts |
| Shape evolution | Slow (org change) | Fast (measured, deliberate) |
| Conway's Law direction | Communication → architecture | Topology → agent comm architecture |

---

## The Four Squad Types

### 1. Stream-Aligned Squad

**What it is:** The primary value-producing unit. Aligned to a continuous flow of work in a specific domain — billing, content extraction, market monitoring, etc.

**Design criteria:**
- A single agent within the squad must be able to complete a meaningful unit of work without calling out to another squad
- Context boundary is the domain: the squad holds all knowledge needed to act
- Owns its pipeline end-to-end: intake → processing → output

**Key question:** *If we cut this squad off from all other squads, does it degrade gracefully or collapse immediately?*

**Target interaction mode:** Consumes platform X-as-a-Service. Occasionally collaborates with other stream squads (time-boxed). Receives facilitation from enabling squads until self-sufficient.

**Warning signs:**
- Squad regularly reaches into platform internals instead of consuming the API
- Squad replicates logic owned by a complicated subsystem squad
- Collaboration with another squad has been running for >2 sprint cycles without converging

---

### 2. Enabling Squad

**What it is:** A capability-delivery unit that unlocks other squads. Does not own a domain — it closes a capability gap in a stream squad, then moves on.

**Design criteria:**
- Has no permanent domain — it borrows context temporarily
- Success is measured by the capability it leaves behind, not output it produces
- Must have a defined exit condition before attachment begins

**Key question:** *What is the specific capability this squad embeds, and what is our definition of "done" for the facilitation engagement?*

**Target interaction mode:** Facilitating. Single direction. Time-boxed by design.

**Warning signs:**
- Enabling squad becomes permanently attached to a stream squad (dependency captured)
- No clear exit condition was defined at engagement start
- Stream squad requests the enabling squad instead of using the embedded capability

---

### 3. Complicated Subsystem Squad

**What it is:** Wraps a hard, specialized problem behind a clean interface. Legal reasoning, financial modeling, graph traversal, domain-specific classification. The complexity lives inside; the surface is narrow.

**Design criteria:**
- The interface (inputs/outputs) must be stable and documented
- Internal implementation is opaque to consumers
- No other squad should be able to replicate this capability economically

**Key question:** *Is the interface narrow enough that a stream squad never needs to understand what happens inside?*

**Target interaction mode:** X-as-a-Service exclusively. No collaboration, no facilitation. Consumed like a function call.

**Warning signs:**
- Consumers are reading implementation details to work around the interface
- More than one squad is replicating logic that should live here
- Interface keeps changing, forcing cascade updates in consuming squads

---

### 4. Platform Squad

**What it is:** The substrate. Provides the shared infra all other squads consume: tool registries, memory primitives, session state, prompt versioning, cost telemetry, observability. Exists so stream squads never have to rebuild foundational capabilities.

**Design criteria:**
- Self-service by design — consuming squads should never need to file a ticket
- Versioned contracts — breaking changes are communicated, not discovered
- Measures its own health by the friction it removes from stream squads
- Has an explicit API contract; internal implementation is its own concern

**Key question:** *Can a new stream squad reach production-ready state using only the platform's golden path, without help from a platform team member?*

**Target interaction mode:** Consumed X-as-a-Service by all other squad types. Never collaborates — if a collaboration is needed, the platform's interface is wrong.

**Warning signs:**
- Stream squads are reaching into platform internals instead of consuming the API
- New squads require hand-holding from the platform team to get started
- No cost/latency telemetry per consuming squad
- Platform squad is a bottleneck on any stream squad's critical path

---

## The Three Interaction Modes

### X-as-a-Service

**Pattern:** One squad exposes a stable, well-documented capability. Another consumes it without coordination.

**When to use:** Steady-state relationship between any squads. The default target for all squad pairs.

**Contract requirements:**
- Input/output schema is versioned and documented
- SLA is defined (latency, availability)
- Breaking changes require deprecation window
- Consumer never needs to understand the provider's implementation

**Failure mode:** Provider keeps changing the interface. Consumer works around the interface. Either signals the contract is wrong, not the mode.

---

### Collaboration

**Pattern:** Two squads work closely together, sharing context actively, to navigate unknowns or build something neither could alone.

**When to use:** Discovery phases. Novel problems. Short-lived integration work. Never as a default state.

**Contract requirements:**
- Time-boxed before it begins (e.g., "2 weeks to validate the integration boundary")
- Exit criteria defined: what does success look like, and what interaction mode does this graduate to?
- One squad owns the final artifact; collaboration doesn't mean co-ownership forever

**Failure mode:** Collaboration persists indefinitely. Both squads become context-entangled. Neither can operate independently. This is captured dependency, not collaboration.

---

### Facilitating

**Pattern:** An enabling squad works alongside a stream squad to build up a capability, then withdraws.

**When to use:** Stream squad has a gap that prevents it from operating independently. Enabling squad closes that gap, then exits.

**Contract requirements:**
- Capability definition: what exactly is being embedded?
- Exit condition: what does "self-sufficient" look like for the stream squad?
- Handoff artifact: documentation, tooling, or embedded prompt that persists after exit

**Failure mode:** Stream squad remains dependent on the enabling squad after the engagement. Either the capability wasn't truly embedded, or the exit condition was never enforced.

---

## Context Load: The Primary Design Constraint

In human systems, cognitive load limits how much complexity a team can handle before quality degrades. In agent systems, the equivalent is **context load**: how much an agent must hold in working context to complete a unit of work.

**Context load includes:**
- Tool schemas and usage patterns
- Domain knowledge (e.g., the full CA product taxonomy for a matching squad)
- Interaction protocols with other squads
- Memory of prior state within a session

**Design rule:** A well-scoped squad should be able to operate with a context window that is primarily domain knowledge, not coordination overhead.

**If a squad's context window is dominated by coordination (what other squads are doing, what they return, how to interpret their outputs), the topology is wrong.** Either the squad is too broad, or the interaction mode is wrong (collaboration where X-as-a-Service should exist).

---

## Topology-Aware Routing

Unlike human systems where routing is handled by management structure, agent squad systems require explicit routing logic at the orchestration layer.

**Routing principles:**
1. Tasks must be classified by squad type before assignment — a task that belongs to a complicated subsystem squad should not be routed to a stream squad even if the stream squad could theoretically attempt it
2. Interaction mode is declared in the routing contract, not inferred at runtime
3. The orchestrator understands squad boundaries — not just tool availability

**Anti-pattern:** Routing by capability alone. Just because a squad has the tool doesn't mean it's the right topological assignment. Capability-only routing erodes squad boundaries over time.

---

## Evolutionary Pressure

Squad shape is not fixed. The topology should evolve based on measured interaction patterns.

**Signals to watch:**

| Signal | Likely action |
|---|---|
| Two squads in collaboration for >N cycles | Evaluate merge, or force graduation to X-as-a-Service |
| Enabling squad attached for >N cycles | Exit condition was wrong; redefine and enforce |
| Stream squad making N+ calls per task to complicated subsystem | Interface may be too narrow; consider enriching output |
| Platform squad is on critical path of any stream squad | Golden path is broken; fix before adding features |
| Subsystem interface changing frequently | Decouple versioning from internal evolution |

**Inverse Conway Maneuver:** Design the squad topology first, then build the agent communication architecture to match it. Don't let emergent agent communication define the topology retroactively.

---

## Quick Reference Card

```
SQUAD TYPE          OWNS DOMAIN?   INTERACTION MODE    EXIT CONDITION?
─────────────────────────────────────────────────────────────────────
Stream-Aligned      YES            Consumes XaaS       —
Enabling            NO             Facilitating        YES — required
Complicated Sub.    YES (narrow)   Provides XaaS       —
Platform            YES (infra)    Provides XaaS       —

INTERACTION MODE    DIRECTION      DURATION            TARGET STATE?
─────────────────────────────────────────────────────────────────────
X-as-a-Service      One-way        Indefinite          YES — default
Collaboration       Bidirectional  Time-boxed          Graduates to XaaS
Facilitating        One-way        Time-boxed          Stream becomes self-sufficient
```

---

## Design Checklist

Before finalizing a squad topology:

- [ ] Every stream squad can complete a meaningful unit of work without cross-squad coordination
- [ ] All complicated subsystem logic is behind a clean, versioned interface
- [ ] Every enabling squad engagement has a defined exit condition
- [ ] Platform squad has a self-service golden path for new squads
- [ ] All collaboration engagements are time-boxed with a defined graduation target
- [ ] No stream squad is reaching into platform internals
- [ ] Routing layer understands squad types, not just tool availability
- [ ] Topology was designed before agent communication architecture (not retroactively)

---

*Squad Topologies v1.0 — adapted from Team Topologies by Matthew Skelton & Manuel Pais*
*teamtopologies.com/key-concepts · teamtopologies.com/platform-manifesto*