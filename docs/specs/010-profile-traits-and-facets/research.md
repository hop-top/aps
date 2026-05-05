# Research: Profile Traits, Facets, Perspectives, and Stance

**Spec**: `010-profile-traits-and-facets`
**Created**: 2026-05-05
**Status**: Living document — captures the brainstorm trail and rejected alternatives so future readers don't relitigate them without new information.

## How this design got here

The brainstorm started from a downstream prompt: a user (Jad) asked to stand up a `~/.fam` workspace mirroring `~/.ops`'s structure, hosting a household-flavored DPKMS, three human profiles (jad, rania, karim), and a set of named agent profiles (personal-assistant, wealth-planner, accountant, coach, dietitian, researcher, enforcer).

Walking the requirements surfaced increasingly load-bearing questions, each of which refined the substrate-level design instead of getting answered downstream:

1. **DPKMS topology** — single-instance multi-tenant (B) evolving to federated (A). Locked.
2. **Protocol surface** — CalDAV/CardDAV/IMAP via adapter daemons backed by DPKMS, with one-platform-per-protocol per DPKMS and federation as the multi-platform answer. Locked.
3. **Adapter substrate** — pluggable adapters belong in DPKMS (not `~/.fam`-specific); `~/.fam` becomes a configuration. Locked.
4. **Profile schema** — must adapt to any worldview; *not* limited to small-business roles. This is what this spec addresses.
5. **Trait taxonomy** — must cover memory, personality, experience, embodiment, schedule, economy, safety, goals, and an open-ended tail of worldview-specific families.
6. **Identity invariance** — name, email, DID never vary by context.
7. **Personality/memory in context** — humans modulate, not replace. Same primitive must cover.

Items 4–7 are the substrate; this spec is their consolidation. Items 1–3 became downstream `~/.fam` and DPKMS specs that consume this one.

## Key design rules and why

### Identity is invariant

Source: explicit user instruction ("profile name, email, identity doesn't change by context").

Implication: all worldview-shaped data lives outside `core.Profile`. The profile schema stays minimal; the substrate grows new layers for everything else.

Concrete consequence: the existing `~/.ops/docs/glossary.md` Facet definition (*"Context-specific presentation (email, avatar, handles)"*) is wrong under this rule. Email/avatar/handles are profile-level. The Facet definition needed correcting; this spec drives that correction.

### Profile is human-or-agent-symmetric

Source: explicit user instruction ("a profile must be applicable to a human or an agent seamlessly").

Implication: no special-case Profile types for humans vs. agents. The discriminator is a trait field (`nature.kind`), not a schema branch.

This rules out the otherwise-tempting move of having `core.HumanProfile` and `core.AgentProfile` as sister types. It also means a Perspective registered for "kid-sidecar" coordinating a human child and an autonomous companion agent uses the *same* facet machinery for both, not parallel pipelines.

### Personality and memory are context-modulated, not context-replaced

Source: explicit user instruction ("personality and memory could still change based on activity profile is doing — aren't humans like that?").

This is a non-trivial shift from the earlier glossary anti-pattern (*"facets adjust presentation, not personality"*). The rule is now:

- Trait baselines are profile-owned and don't change per context.
- Facets declare *modulators* (deltas/weights/filters/gates) that bias trait expression while active.
- Modulator scope ends when the facet deactivates; no leak to baseline.
- Promoting facet-scoped state to baseline (memory consolidation, learned dispositions) is a separate, explicit operation.

The glossary anti-pattern is sharpened in this spec: facets modulate trait *expression*; identity (name, email, DID) is never modulated; trait *baselines* are never rewritten by facets.

### Open taxonomy, blessed core

Source: explicit user instruction ("not limited to just the ones I listed").

Implication: the substrate provides the registration mechanism; worldviews populate it. Same shape applies to traits, facets, runtime parameters. The blessed core is the minimum needed for cross-Perspective interop (e.g. the safety trait is read by the runtime parameter resolver across all Perspectives).

### Substrate computes runtime parameters from situated state

Source: explicit user instruction ("some properties or mix thereof will be used to alter default tokens, temperature, thinking mode, models available, etc. — which could still be overridden but their default values are dynamically calculated based on profile").

Implication: the resolver is part of the substrate, not a downstream concern. Each consumer doesn't reinvent the math. Perspectives declare derivation functions; the substrate runs them; explicit overrides are first-class with audit.

## Vocabulary trail

Vocabulary churned through several rounds during the brainstorm. Recording rejected names so future readers don't reach for them again unaware of what was already considered.

### "Worldview" → "POV" → "Perspective"

- **Worldview**: too philosophical, implied a totality of meaning. Reads heavy for what's actually a registered schema bundle.
- **POV (PointOfView)**: read more naturally as a vantage/projection. Closer fit. But "POV" both as the registered definition and as the active runtime projection conflated two senses.
- **Perspective**: settled. Registered definition; the runtime-active projection got its own name (Stance).

### Runtime-active projection: "Stance"

Considered: Stance, Posture, Vantage, Perspective (rejected — already taken for the registered definition).

Stance won on brevity, neutrality, and natural implication of a temporary positioning that can shift. *"Karim's stance right now is `home-evening`."* reads cleanly.

### "Facet" alternatives considered

The semantics shifted from "presentation" (the original glossary definition) to "context-bound activation with trait modulation." That made "Facet" — a word whose everyday meaning is *one face of a gem*, i.e. presentation — semantically off. Several alternatives were proposed:

- **Mode** — operating-state, switchable, modulates behavior. Generic; collision risk with future tooling.
- **Context** — literal but heavily overloaded. High collision risk with the `ctxt` tool itself.
- **Situation** — semantically precise; clunky as a code identifier.
- **Activation** — mechanically accurate; verby as a noun.
- **Engagement** — overloaded with business meaning ("user engagement"); also collides with "engagement pack" in `~/.agents/AGENTS.md`.
- **Role** — collision with collaboration roles (owner/assignee/evaluator) on `core.Profile`. Rejected.
- **Hat** — too cute; doesn't extend.
- **Posture** — would conflict with Stance if both were used.
- **Frame** — heavy reuse in software (stack frame, iframe).
- **Setting** — overloaded with config.
- **Scene** — theatrical metaphor cohering across Profile/Trait/Perspective/Stance/Scene; lower collision in software vocab. Was the runner-up.
- **Aspect** — already loaded in aspect-oriented programming.
- **Disposition** — confuses trait/facet boundary.
- **Boundary** — actively wrong: implies a wall/edge/limit, opposite of the activation/modulation semantics. Also collides with existing `scope` ("tlc task boundary") and the existing anti-pattern about identity walls. Rejected with discussion.

**Decision**: stick with **Facet**. The word's already on disk in `<APS_DATA_PATH>/profiles/<id>/facets/`. Renaming has migration cost. The *definition* is what was wrong; renaming wouldn't fix that. This spec corrects the definition (FR-031, glossary edits) while keeping the name.

### "Pack" / "bundle" → kept generic

The registered definition was briefly called "Pack" before settling on "Perspective." "Pack" collided with `~/.rlz/<pack>/` engagement packs in the user's environment. Dropped.

## Layering against existing concepts

Pre-existing aps concepts that interact with this spec:

| Concept | Pre-existing | Relationship to this spec |
|---|---|---|
| **Profile** | aps identity record (`internal/core/profile.go`) | Stays minimal. This spec adds Trait and Facet alongside, both attaching to Profile. |
| **Capability** | aps `Profile.Capabilities []string` (a2a, webhooks, agntcy-identity) | **Kept.** Master list lives on Profile; Facets reference subsets via `active capabilities`. No migration. |
| **Persona** (`Profile.Persona`: Tone/Style/Risk struct) | aps `internal/core/profile.go:158` | **Deprecate** in favor of the future `character` Trait family. This spec doesn't migrate; the migration tool ships when `character` Trait spec lands. Persona stays functional in the meantime. |
| **Squads** (`Profile.Squads []string`) | aps `profile.go:108` | **Kept.** Squads represent collaboration archetype membership; distinct from dispositional baseline. May eventually live under a `membership` Trait family but not in this spec. |
| **Roles** (`Profile.Roles []string`: owner/assignee/evaluator/auditor) | aps `profile.go:110` | **Kept.** Task-collaboration roles, already disambiguated in glossary. Distinct from Trait taxonomy. |
| **ScopeConfig** (`Profile.Scope`: file_patterns/operations/tools/secrets/networks) | aps `profile.go:114` | **Migrate into Facet definitions.** ScopeConfig describes access boundaries; Facets carry "active capabilities + scope" by definition (FR-030). Implementation plan promotes existing Scope to an explicit Facet under a per-profile default Perspective. |
| **SessionInfo** (`core/session.SessionInfo.ID`) | aps `internal/core/session/registry.go:88` | Referenced by FR-038 as the key for session-scoped facet activations. Not changed by this spec. |
| **DeviceID** (string PK across `WorkspaceDeviceLink`, `DevicePresence`, `DevicePermissions`) | aps `internal/core/multidevice/types.go` | Referenced by FR-038 as the key for device-scoped activations. There is no top-level `Device` struct; `DeviceID` is a field across multidevice's link/presence/permission types. Activation state replication piggybacks on multidevice's event store + `lww_resolver` (no parallel sync path). |
| **Capacity** (Principal/Delegate) | wsm authority mode | Orthogonal to traits/facets. A Stance composes with Capacity at runtime; they don't replace each other. Spec explicitly excludes agency from `nature` trait to avoid duplication. |
| **Grant** (workspace-scoped capability binding) | wsm | Grants are referenced from a Facet's `active capabilities`. This spec doesn't change Grant mechanics. |
| **Lens** | ctxt search filter | Distinct from Perspective (avoids the previously-proposed name "Lens" for what became Perspective). The Lens stays in ctxt; integration with Perspective (e.g. a Perspective declaring a default Lens) is a follow-up integration question, not part of this spec. |

## Why this couldn't be deferred to ctxt or downstream tools

A natural alternative was to leave aps minimal and put the trait/facet machinery in ctxt or in each downstream consumer (`~/.fam`, agr, kid-sidecar). Rejected because:

1. **Profile identity lives in aps.** Traits are profile-owned. Putting them in ctxt or downstream means duplicating profile state across stores, which violates aps's role as the canonical identity record.
2. **Multiple consumers need the same machinery.** agr, fam-workspace, ctxt, and any future runtime all need to ask "what's this profile's situated state?" Building it once in aps lets them all share.
3. **The runtime parameter resolver is the wedge.** If even *one* downstream consumer expects to call `aps profile resolve-params <id>` and get a deterministic answer, the resolver has to live where profile state lives. That's aps.

## What this spec deliberately does *not* settle

Listing these so the implementation plan knows what it owns:

- **Derivation-function and predicate language** (THE biggest implementation question, per spec § Open Questions): CEL vs. embedded scripting (Lua/Starlark/JS) vs. JSON-Logic vs. WASM. Used by derivation functions (FR-061), facet preconditions (FR-037), gate predicates (FR-032), and Perspective composition overrides (FR-041). Must be sandboxable, deterministic, inspectable.
- **Concrete schema language** for trait/facet schemas (JSON Schema vs. CUE vs. Pkl vs. custom). Should align with ctxt's plugin schema choice.
- **Storage backend** for trait values beyond "human-readable on disk" (FR-016).
- **Specific blessed trait schemas**. Each gets its own follow-up spec (`character`, `memory`, `experience`, `embodiment`, …).
- **First Perspective definitions** (`ic`, `family`, `kid-sidecar`, `elder-monitor`). Downstream specs.
- **Memory consolidation mechanics**. Its own follow-up spec; FR-014 establishes the positive invariant (baselines change only via explicit `trait set`) which consolidation must respect.
- **Migration tooling** for existing freeform `facets/<name>.yaml` files and for legacy fields (Persona, ScopeConfig). Tools land in the implementation plan; deprecation timeline (one-release-warning before removal) is named but not scheduled.

## Decisions made during review (Round 1 and Round 2 edits)

The first draft of this spec received a structured review that landed 16 issues. Decisions made in response, in case future readers wonder how they got there:

- **InvocationParams split** (issue #11): renamed `RuntimeParams` and split into `model` and `runner` sub-structs. The first draft mixed `temperature` (model concern) with `available_tools` (runner concern). Sub-structs separate consumers; mixing is rejected at registration. Single resolver call from caller's POV.
- **Composition default disambiguated** (issue #2): two-level rule. Level 1 is unconditional and per-trait-class (safety/permission classes use most-restrictive-wins, ignoring priority). Level 2 is per-modulator priority within trait families that have no Level 1 rule. Earlier draft had both at the same level, which was ambiguous.
- **Modulator vocabulary inline-defined** (issue #6): rather than punt to implementation plan, the spec defines delta/weight/filter/gate semantics. Without this, the composition rules would be abstract.
- **Facet preconditions added** (issue #7): facets can declare prerequisites; activation fails fail-closed by default; `degrade` policy is opt-in. Without this, the kid-sidecar use case (requires `nature.kind = human`) would have nowhere to express its preconditions.
- **Stance wire format** (issue #9): FR-072 specifies a normative YAML/JSON schema for `--format json` stance output. Without this, every consumer parses ad-hoc.
- **Trait write authorization** (issue #10): FR-079/080/081 lock down the "runtime executing inside a facet" question — runtimes get situated reads but not baseline writes. The sandbox excludes write APIs regardless of the language chosen.
- **`nature` trait expanded** (issue #12): FR-017 now declares `kind` + `lifecycle` + `embodiment`. Agency explicitly deferred to wsm Capacity (don't duplicate authority logic).
- **Migration & Deprecation section added** (issue #15): not all existing fields deprecate. Persona deprecates → character trait. Squads/Roles/Capabilities are kept (they serve different purposes than traits). ScopeConfig migrates into Facets (the real overlap point).
- **Glossary moved out of Requirements** (issue #13): documentation tasks (D-001 — D-005) are tracked separately from substrate requirements (FR-NNN).
- **CLI section trimmed** (issue #14): substrate operations described; concrete CLI surface deferred to implementation plan.
- **Baseline-vs-modulator precedence** (issue #5): FR-042 makes "baseline is seed; modulators apply on top" explicit. Computation order spelled out.
- **FR-014 phrased as positive invariant** (issue #1): replaced "deferred to consolidation spec" with "trait baselines are mutated only by explicit `trait set` operations." Future learning behaviors are additional explicit operations, not implicit side effects.
- **FR-031 enforcement split** (issue #3): identity exclusion checked at schema-declaration time AND at instance-write time (defense-in-depth).
- **Attach vs. activate clarified** (issue #4): three states — Defined / Attached / Active — with separate storage paths and explicit transitions.
- **FR-038 references real types** (issue #8): instead of abstract "session id, device id," the spec names `core/session.Session.ID` and `core/multidevice.Device.ID` and requires multidevice CRDT sync piggyback for cross-device activation.
- **Open Questions §1 names derivation-language** (issue #16): explicitly called out as the biggest implementation question, not buried.

## Source references

- This spec emerged from a brainstorming conversation in `~/.ops` (workspace: noor) on 2026-05-01 through 2026-05-05.
- Glossary corrections paired with this spec land at `~/.ops/docs/glossary.md` (Documentation Tasks D-001 — D-005).
- Pre-existing identity model documentation: `~/.ops/docs/architecture/identity-model.md` (current state; this spec extends it; D-005 updates it).
- Pre-existing aps Profile schema: `internal/core/profile.go` (specifically: `Persona` at :158, `Squads` at :108, `Roles` at :110, `Scope` at :109/114).
- Pre-existing aps Session model: `internal/core/session/registry.go` (`SessionInfo.ID` at :88), `types.go` (SessionType constants), `tmux.go`, `eventbus.go` (referenced by FR-038).
- Pre-existing aps Multidevice model: `internal/core/multidevice/types.go` (`DeviceID` is a string PK field across `WorkspaceDeviceLink`, `DevicePresence`, `DevicePermissions` — there is no top-level `Device` struct), `presence.go`, `sync_manager.go`, `lww_resolver.go` (referenced by FR-038/FR-039).
- Pre-existing `facets/` on-disk convention: `<APS_DATA_PATH>/profiles/<id>/facets/<name>.yaml` (e.g. `<APS_DATA_PATH>/profiles/noor/facets/ic.yaml`).
