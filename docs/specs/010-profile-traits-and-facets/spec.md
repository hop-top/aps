# Feature Specification: Profile Traits, Facets, Perspectives, and Stance

**Feature Branch**: `010-profile-traits-and-facets`
**Created**: 2026-05-05
**Status**: Draft
**Input**: aps profiles need to adapt to any worldview. The same `core.Profile` shape must serve a small business, a kid's storytelling sidecar, an elderly's health-monitoring pendant, a household coordinator, a research lab — all without baking any one worldview into the substrate. Today's profiles only carry identity (id, did, email, key, capabilities) plus an unstructured `facets/<name>.yaml` convention that no aps code reads or validates. This spec promotes the substrate to a typed, pluggable trait + facet system, defines context-bound activation with trait modulation, and adds runtime parameter resolution so model/temperature/thinking-mode/tools can be derived from situated profile state.

## Overview

A profile in aps today is identity-only: id, display_name, email, DID, key, capabilities, isolation, a2a, preferences, git. Worldview-specific shape (a job role, a family relationship, a kid's reading-comprehension level, an elderly user's escalation contact) has nowhere to live except a free-form `facets/` directory the aps binary doesn't read.

This spec introduces four substrate concepts that close the gap without coupling aps to any one worldview:

1. **Trait** — a profile-owned property family. Open-ended taxonomy. Substrate ships blessed schemas for a small core; downstream code registers additional trait families per worldview. Traits hold the dispositional baseline.
2. **Facet** — a context-bound activation. Bundles scope + active capabilities + active connections + presentation surface + trait modulators. Facets do *not* alter identity (name, email, DID) and do *not* rewrite trait baselines; they modulate trait expression while active.
3. **Perspective** — a registered definition (schema bundle): declares which traits exist, which facets exist, which runtime parameters are derivable, and what the composition policy is. Installing the `family` Perspective registers the `relationship` trait and the `family-member` facet.
4. **Stance** — the runtime-active state: which facets are currently active, which traits they're modulating, which parameters resolve to what values right now. Read by callers needing situated parameters.

The substrate is opinion-free about which worldviews exist. It provides the registration mechanism, the modulation pipeline, and the resolver. Worldviews ride on top.

### Why now

- aps is gaining downstream consumers (agr, fam-workspace, ctxt) that need profile state beyond identity. Without typed traits/facets, each consumer reinvents the schema and they collide on disk in `facets/`.
- A growing list of named-but-unimplemented use cases (kid-sidecar, elder-monitor, family-member, accountant agent, dietitian agent, …) all need the same substrate machinery. Building them piecemeal would either bake business-shaped assumptions into core or duplicate the resolver per worldview.
- The personality-vs-presentation question keeps re-litigating in design conversations. A typed boundary — *traits are profile-owned baselines, facets modulate expression in context, identity is never modulated* — makes the rule enforceable, not just a convention.

### Out of scope (named so dependencies are visible)

- The blessed trait schemas themselves (memory, character, experience, capability, embodiment, relationship, schedule, economy, safety, goals, nature). Each is its own follow-up spec.
- Memory consolidation and trait evolution over time. Its own follow-up spec.
- The first Perspective definitions (`ic`, `family`, `kid-sidecar`, `elder-monitor`). Downstream consumers, separate specs.
- Migration of existing on-disk `facets/<name>.yaml` files. Addressed in a transition note (§ Transition) — actual migration tooling lives in the implementation plan that follows this spec's approval.

## User Scenarios & Testing

### User Story 1 — Worldview Registration (Priority: P1)

As a downstream tool author (e.g. someone building a household-management workspace), I want to register a Perspective declaring my custom traits and facets so that profiles consumed by my tool can carry my domain shape without my needing to fork aps.

**Acceptance Scenarios**:

1. **Given** a Perspective definition file declaring a `relationship` trait schema and a `family-member` facet schema, **When** the operator runs `aps perspective register ./family.perspective.yaml`, **Then** the Perspective is added to the registry, its trait and facet schemas become available for use, and `aps perspective list` shows it as installed.
2. **Given** a registered Perspective, **When** an operator attaches a value of one of its traits to a profile (e.g. setting `relationship.spouse_of = jad` on Rania's profile), **Then** the trait value is validated against the Perspective's schema and rejected if invalid.
3. **Given** two Perspectives that both declare a trait named `tone`, **When** they're both registered, **Then** the substrate keeps them distinct via Perspective-namespacing (`ic.tone`, `family.tone`) and no value silently overwrites the other.

### User Story 2 — Trait Read/Write (Priority: P1)

As a profile owner (or an automated process operating on a profile's behalf), I want to read and update typed trait values on a profile so that downstream consumers can reason about the profile's dispositional state.

**Acceptance Scenarios**:

1. **Given** a profile with no `character` trait set, **When** the operator runs `aps profile trait set noor character.formality=high character.humor=dry`, **Then** the values are stored, validated against the (eventually registered) `character` schema, and surfaced in `aps profile show noor`.
2. **Given** a profile with a `memory` trait whose schema is registered, **When** the operator queries it via `aps profile trait get noor memory`, **Then** the substrate returns the structured value (not a raw YAML blob).
3. **Given** a profile with a trait family that has *no* registered schema (worldview-defined, schema-loose), **When** values are written to it, **Then** the substrate accepts them and stores them with a `schema: unregistered` marker so consumers know to defensively parse.

### User Story 3 — Facet Activation & Trait Modulation (Priority: P1)

As a runtime consuming a profile (e.g. an LLM-backed agent runner, a CLI session), I want to activate one or more facets on a profile and have trait values automatically modulated for the duration of the activation, so that the profile expresses itself appropriately in context without consumers writing modulation logic per call site.

**Acceptance Scenarios**:

1. **Given** a profile with `character.openness = 0.6` and a `home-evening` facet whose modulator declares `character.openness += 0.2`, **When** the runtime activates the `home-evening` facet and queries `character.openness` via the situated read API, **Then** it gets `0.8` (modulated), and **When** it queries via the baseline read API, **Then** it gets `0.6` (unchanged).
2. **Given** a profile with two facets active that both modulate the same trait field, **When** the runtime queries the situated value, **Then** the substrate composes the modulators per FR-040: safety/permission-class fields use most-restrictive-wins (Level 1, priority ignored); other fields use explicit priority (Level 2). The returned value is deterministic and independent of facet activation order.
3. **Given** an active facet, **When** the facet is deactivated, **Then** subsequent situated reads return baseline values (no leak: facet-scoped state does not write back to the trait baseline by default).

### User Story 4 — Invocation Parameter Resolution (Priority: P1)

As an agent runner invoking a profile, I want to ask the substrate "what model and runner parameters should I use for this profile right now?" and have the answer derived from active traits + facets, with explicit overrides supported, so that I don't hard-code parameters per call site.

**Acceptance Scenarios**:

1. **Given** a profile with `character.formality = high`, `safety.strictness = 0.8`, `memory.working_size = small`, and a `kid-sidecar` facet active, **When** the runner calls `aps profile resolve-params noor`, **Then** the substrate returns an `InvocationParams` struct with two sub-structs: `model` (model_id, max_tokens, temperature, thinking_mode, response_register) and `runner` (available_tools, mcp_servers, retrieval_filters, timeout_budget) — each field derived from the active stance via the registered Perspective's derivation functions.
2. **Given** a resolution call with `--override model.temperature=0.7`, **When** the substrate runs the resolver, **Then** `InvocationParams.model.temperature = 0.7` and the per-field audit shows source = `overridden`.
3. **Given** a resolution call with a bare `--override temperature=0.7` (no sub-struct qualifier), **When** the substrate runs the resolver, **Then** the call fails with an error pointing to the ambiguity (model.temperature vs. runner.temperature), per FR-062.
4. **Given** a Perspective declares a derivation function for a parameter, **When** the resolver runs, **Then** the function receives the current stance (active facets + their modulated trait values) and returns a value within the parameter's declared bounds; values outside bounds are clamped and the clamp is logged in the audit.

### User Story 5 — Symmetric Profile Treatment (Priority: P1)

As an operator, I want the same profile schema and trait/facet machinery to serve a human profile (Karim, age 8) and an agent profile (Noor, autonomous ops lead) without distinguishing them at the type level, so that worldviews remain free to compose human and agent profiles however they need.

**Acceptance Scenarios**:

1. **Given** two profiles where one is a human and one is an agent, **When** an operator inspects them via `aps profile show`, **Then** both surface the same fields and trait families; the only difference is the value of the `nature` trait (e.g. `nature.kind = human` vs. `nature.kind = autonomous-agent`).
2. **Given** a Perspective registers a facet that combines a human profile and an agent profile in a relationship (e.g. `kid-sidecar` linking Karim's profile to a sidecar-agent profile), **When** the facet is activated, **Then** trait modulators apply uniformly to both profiles per the facet's declaration.

### User Story 6 — Stance Inspection (Priority: P2)

As an operator debugging unexpected agent behavior, I want to inspect the current stance of a profile — which facets are active, which trait values are modulated, which invocation parameters are derived — so I can reason about why the agent acted the way it did.

**Acceptance Scenarios**:

1. **Given** a running session with a profile in a particular stance, **When** the operator runs `aps profile stance show <scope-key>`, **Then** the substrate emits the active facets, the modulator deltas being applied per trait field, the resolved invocation parameters, and the override audit — in the structured format declared by FR-072.
2. **Given** the same scope-key, **When** the operator runs `aps profile stance diff <scope-key>`, **Then** the substrate emits the diff between baseline trait values and situated values, scoped to the active stance.
3. **Given** an operator attempts to activate a facet whose preconditions don't match the target profile's traits, **When** the activation fails fail-closed (per FR-037), **Then** the error message names the failing precondition and the offending trait values; no partial state is left in the stance.

## Requirements

### Functional Requirements

#### Profile shape

- **FR-001**: `core.Profile` MUST remain identity-only: id, display_name, email, DID, keys, color, avatar, capabilities, isolation, a2a, preferences, git, identity. No worldview-specific fields.
- **FR-002**: `core.Profile` MUST be human-or-agent-symmetric: no type-level distinction between profiles representing humans, agents, devices, or hybrid embodiments. Discrimination, when needed, lives in the `nature` trait (FR-018 below).

#### Trait system

- **FR-010**: The substrate MUST support a typed Trait registration mechanism. A Trait family is identified by a name (e.g. `memory`, `character`) and an optional Perspective namespace (e.g. `family.relationship`).
- **FR-011**: The substrate MUST support schema-strict and schema-loose Trait registration. Schema-strict Traits validate values against a JSON Schema (or equivalent) on write. Schema-loose Traits accept arbitrary values with a `schema: unregistered` marker stored at the top of the trait's YAML file (per FR-012's storage path). The marker is read on file load; when a previously-unregistered trait family is later registered (via `aps perspective register`), the substrate MUST locate all profiles holding values for that family, validate each against the newly-registered schema, and convert validated values to schema-strict (clearing the marker). Validation failures during this re-validation MUST be surfaced (not silently dropped) so operators can repair them before the trait is consumed by runtimes.
- **FR-012**: Trait values MUST be stored at `<APS_DATA_PATH>/profiles/<id>/traits/<namespace>.<name>.yaml` (or equivalent typed store; the on-disk format is the authoritative source for backups).
- **FR-013**: The substrate MUST provide read APIs for traits at two granularities: **baseline** (the stored value, ignoring active facets) and **situated** (the value as projected through currently-active facets' modulators).
- **FR-014**: Trait writes MUST go to baseline; situated reads MUST never write back. The substrate guarantees a positive invariant: **trait baselines are mutated only by explicit `trait set` operations**. No facet activation, deactivation, modulation, or runtime read path may modify a baseline trait value. Future learning-style behaviors (memory consolidation, learned dispositions) will be additional explicit operations that respect this invariant — they will not be implicit side effects of facet lifecycle.
- **FR-015**: The substrate MUST support trait-family namespacing to prevent collision when multiple Perspectives declare a trait with the same name. Bare unqualified names MUST be reserved for the blessed core (FR-018–FR-020).
- **FR-016**: Trait values MUST be human-readable on disk (YAML or equivalent) so backups, audits, and manual recovery are practical.

#### Blessed core trait families (substrate-shipped)

- **FR-017**: The substrate MUST ship a blessed `nature` Trait family with at least the following fields:
    - **`kind`** (enum, required): `human`, `autonomous-agent`, `assistive-agent`, `embodied-device`, `hybrid`. Discriminator that lets all profile shapes share the same schema.
    - **`lifecycle`** (enum, optional, default `active`): `active`, `paused`, `retired`, `archived`. Lets downstream consumers filter / surface profiles appropriately (a retired family-member profile shouldn't appear in current calendars). Lifecycle transitions are explicit operator actions, not auto-derived.
    - **`embodiment`** (struct, optional): describes physical/device manifestation when applicable. At minimum: `device_ref` (reference to a `core/multidevice.Device`), `surface` (e.g. `tablet`, `pendant`, `phone`, `voice-only`, `none`). Profiles where embodiment doesn't apply (e.g. pure cloud-resident agents) omit this struct.
    - Additional `nature` fields MAY be added in follow-up specs, but consumers MUST NOT add bare-namespace fields (e.g. `nature.foo`) ad-hoc; all additions go through the blessed-trait promotion process.
    Explicitly NOT in `nature`: agency / authority. Who-can-act-on-whose-behalf is `Capacity` (Principal/Delegate, wsm-resolved) and stays there. Mixing the two would duplicate authority logic.
- **FR-018**: The substrate MUST ship a blessed `safety` Trait family. Cross-Perspective consumers (especially the runtime parameter resolver) read this trait to enforce policy uniformly. Schema includes at minimum: `strictness`, `content_policy`, `escalation_triggers`.
- **FR-019**: Additional blessed Trait families (memory, character, experience, capability, embodiment, relationship, schedule, economy, goals) MAY be added in follow-up specs. Each is independent; this spec does NOT define them.
- **FR-020**: The blessed Trait list is open-ended by design. Promoting a worldview-defined Trait to blessed status is a deliberate decision (its own spec), not a fast path.

#### Facet system

- **FR-030**: The substrate MUST support a typed Facet registration mechanism. A Facet is identified by name + Perspective namespace and declares: scope reference (which tlc scope is active), active capabilities (which Grants apply), active connections (which related profiles are addressable), presentation surface (which avatar/voice/UI shell), and trait modulators.
- **FR-031**: Facets MUST NOT contain identity fields (name, email, DID). Identity-field exclusion is enforced at two points:
    - **(a) Schema declaration**: a Perspective registering a facet *schema* that declares an identity field MUST be rejected at Perspective registration time (before any facet instance exists).
    - **(b) Instance write**: an attempt to write an identity-named field to an attached facet *instance* MUST be rejected at write time, even if the schema somehow permits it (defense-in-depth against misregistered schemas).
    Identity for any profile is invariant and lives only on `core.Profile`.
- **FR-032**: Trait modulators in a Facet MUST be expressed as one of four kinds — never as full replacements of trait baselines. Each kind has defined semantics:
    - **Delta** (`op: delta, value: <scalar>`): additive bias on numeric fields. Situated value = baseline + delta. Applies to fields whose trait schema declares a numeric type.
    - **Weight** (`op: weight, value: <scalar>`): multiplicative bias on numeric fields. Situated value = baseline × weight. Weight = 1.0 is identity. Applies only to numeric fields with a declared range; result is clamped to that range.
    - **Filter** (`op: filter, predicate: <expr>`): membership constraint on set/enum/list fields. Situated value = baseline ∩ filter (for sets), or baseline if predicate matches (for scalars). Predicate evaluation is the same kind of expression used by derivation functions (see § Open Questions).
    - **Gate** (`op: gate, condition: <expr>, on_pass: <action>, on_fail: <action>`): conditional pass-through. If `condition` evaluates true on the baseline, the modulator yields `on_pass`; otherwise `on_fail`. Actions are themselves modulators (delta/weight/filter), enabling conditional modulation. No infinite recursion: gates may not nest more than one level.
    Modulators MUST declare which trait field they target (`field: <namespace>.<field-path>`) and SHOULD declare a `priority` integer (see FR-040). A modulator targeting a field whose trait family is not registered, or whose declared kind doesn't fit the field's type (e.g. `weight` on an enum field), MUST be rejected at facet registration.
- **FR-033**: Facets MUST be activatable and deactivatable per session. The substrate MUST track which facets are active for a given runtime context (typically a session, but the binding is left to the runtime).
- **FR-034**: Multiple Facets MAY be active simultaneously on the same profile. The substrate MUST compose their modulators per the registered composition policy (FR-040).
- **FR-035**: Facet *definitions* (templates) MUST be stored at `<APS_DATA_PATH>/perspectives/<perspective>/facets/<name>.yaml` and reference-able from any profile. Per-profile facet *attachments* (instances bound to a specific profile, with per-profile overrides if any) MUST be stored at `<APS_DATA_PATH>/profiles/<id>/facets/<perspective>.<name>.yaml`. Note that the new attachment files always carry a `<perspective>.` prefix (with the dot separator) in their filename, while legacy freeform facet files (preserved during migration per § Migration & Deprecation) do NOT have such a prefix. This filename-shape distinction guarantees the new and legacy files cannot collide in the same directory during the transition window.
- **FR-036**: The substrate distinguishes three states for a facet:
    - **Defined**: a facet schema exists in a registered Perspective. No profile owns it yet.
    - **Attached**: the facet is bound to a specific profile (an instance file exists at the per-profile path from FR-035). Attachment may include profile-specific modulator overrides; if no overrides, the attachment is a thin reference to the definition.
    - **Active**: the attached facet is currently shaping the profile's stance for one or more runtime contexts. Activation is per-context; the same attached facet may be active in one session and not another.
    Operators attach facets explicitly; runtimes activate them. Attachment without activation is valid (a profile can carry a facet ready to be activated by some future context). Activation requires prior attachment.
- **FR-037**: A Perspective MAY declare **preconditions** on a facet: a list of trait predicates that must be true on the target profile for activation to succeed. Examples: `kid-sidecar` requiring `nature.kind = human` and `experience.age < 18`. Preconditions are evaluated against baseline trait values at activation time.
    - When preconditions fail, the substrate MUST refuse activation (fail-closed default) and report which precondition failed.
    - Perspectives MAY declare a `degrade` policy on a facet to allow partial activation when non-critical preconditions fail; the substrate MUST log the degraded state and surface it in stance inspection.
- **FR-038**: Per-profile facet activation state MUST be persistable so it survives process restarts. Activation state is keyed by **(profile-id, scope-key)** where `scope-key` identifies the runtime context. The substrate MUST support three scope-key kinds, each grounded in an existing aps primitive where one exists:
    - **Session-scoped** (default): activation lives for the lifetime of an aps session, keyed by `core/session.SessionInfo.ID` (the existing session registry; see `internal/core/session/registry.go:88`). Forgotten when the session ends.
    - **Device-scoped**: activation is bound to a specific device, keyed by the `DeviceID` string used as primary key across multidevice's link, presence, and permission types (`WorkspaceDeviceLink.DeviceID`, `DevicePresence.DeviceID`, `DevicePermissions.DeviceID` in `internal/core/multidevice/types.go`). Persists across sessions on that device. Used for cases like "this household pendant is always in elder-monitor stance."
    - **Operator-pinned**: activation persists until an operator explicitly deactivates it. Keyed by an operator-supplied label (e.g. `pinned:elder-monitor`). This is a new concept introduced by this spec; FR-039 governs pinned-activation policy.
    Persistence storage details are an implementation-plan decision; the on-disk shape MUST be inspectable for debugging. Multi-device replication for activation state is REQUIRED for device-scoped and operator-pinned activations (so a user transitioning between devices retains pinned stances). Replication MUST piggyback on the existing multidevice infrastructure: activation transitions emit events into multidevice's event store, and cross-device reconciliation runs through the existing `lww_resolver` (see `internal/core/multidevice/lww_resolver.go`), which itself rides on `kit/runtime/sync` (the kit-level CRDT-shaped replicator with clock + diff + merge + transports). Implementations MAY NOT introduce a parallel sync path; the substrate hierarchy is `kit/runtime/sync` → `aps/internal/core/multidevice` → activation state.
- **FR-039**: Operator-pinned activations (FR-038, third kind) follow these rules:
    - **Authority**: any operator with OS-level access to `<APS_DATA_PATH>` may pin or unpin activations on profiles whose data dir they can read/write. This matches the substrate's existing OS-level authorization model (FR-079) and does NOT introduce a new authority primitive. Tighter authorization is an implementation-plan concern.
    - **Label namespacing**: pinned-activation labels are **profile-scoped**. Two operators on different profiles may both use the label `pinned:elder-monitor` without collision; the storage key is (profile-id, label).
    - **Lifecycle on Perspective unregister**: when a Perspective is unregistered (FR-051), pinned activations referencing that Perspective's facets MUST be flagged but not auto-deactivated. They surface in stance inspection with a `degraded: true, degraded_reasons: [perspective-unregistered]` marker. Operators clear them explicitly. This matches FR-051's flag-don't-purge posture for trait values.
    - **Lifecycle on facet definition removal**: when a facet definition is removed from a still-registered Perspective (e.g. via Perspective update), pinned activations referencing it follow the same flag-not-purge rule.

#### Composition rules

- **FR-040**: When multiple modulators target the same trait field, the substrate MUST compose them deterministically using a two-level rule:
    - **Level 1 (unconditional, per-family)**: the trait family's declared composition class wins. The substrate ships defaults for the blessed core:
        - **Safety-class fields** (anything in `safety.*` or trait fields tagged `class: safety` in their schema): most-restrictive wins (lowest permissiveness, highest strictness). Priority is ignored.
        - **Capability/permission-class fields** (anything tagged `class: permission`): most-restrictive wins. Priority is ignored.
    - **Level 2 (within a family without a Level 1 rule, by priority)**: when the trait family has no composition class declaration, modulators compose by **explicit per-modulator priority**:
        - Higher integer priority wins for delta/weight/filter on scalar fields.
        - Ties: deltas sum, weights multiply, filters intersect.
        - For set/list fields: union of all modulator outputs, with priority used only to resolve conflicting filters on the same element.
    - Modulators that omit a priority field default to priority = 0.
    - Composition is order-independent within a priority tier (same-priority modulators commute).
    - **Ordering-undefined fields**: a field tagged `class: safety` or `class: permission` whose value type has no defined total ordering (e.g. an enum like `content_policy` whose schema enumerates strings without rank, a set/list of escalation triggers without a comparison rule) cannot use most-restrictive-wins; the substrate falls back to Level 2 (priority-ordered) for such fields. Schemas MAY declare an ordering hint (e.g. `order: [strict, moderate, lenient]` for an enum) to opt in to Level 1; without an ordering hint, Level 2 applies.
- **FR-041**: A Perspective MAY override Level 1 by tagging its traits with a different composition class, OR override both levels by declaring a custom composition function on a specific trait field. Overrides are scoped to traits the Perspective declares; one Perspective cannot redefine composition for traits owned by another Perspective. Custom composition functions MUST be **pure**:
    - No side effects (no logging, no I/O, no mutation of any state outside the function's return value).
    - No mutation of the modulator list passed in; the input is read-only by contract.
    - Deterministic: same inputs always produce the same output.
    - Output type is exactly the composed trait field value. The function does NOT return audit metadata, modulator lists, or any tuple wrapping the value.
    Violations of purity (detectable via the sandbox chosen in § Open Questions) MUST cause composition to fail with an error rather than silently mutate state.
- **FR-042**: Baseline-vs-modulator precedence: **the baseline is the seed; modulators apply on top of baseline**. Computation order for a situated read of trait field `T`:
    1. Start with `baseline(T)`.
    2. Collect all active modulators targeting `T`.
    3. **Resolve gates first**: for each gate modulator, evaluate its condition against `baseline(T)` (per FR-032 — gate predicates always read baseline, never an in-progress composition value). Replace the gate in the modulator pool with whichever sub-modulator (`on_pass` or `on_fail`) was selected. Gates that select no action (e.g. condition false with no `on_fail` declared) are removed from the pool.
    4. Apply Level 1 / Level 2 composition (FR-040) to the post-gate modulator pool to produce a single composite modulator (or class-winning modulator).
    5. Apply that composite to the baseline per the modulator's kind (delta adds, weight multiplies, filter intersects).
    6. Clamp to the trait field's declared bounds (if any) and return.
    Gate-first evaluation guarantees SC-010 ("regardless of activation order") holds for gated modulators: gates always see the same baseline, never a partially-modulated value. A modulator may not "set" a baseline value directly. Modulators that resolve to nonsense given the current baseline (e.g. weight × baseline outside declared bounds) are clamped per step 6; the clamp is logged in the stance audit.

#### Perspective registration

- **FR-050**: The substrate MUST support Perspective registration. A Perspective is a YAML (or equivalent) bundle declaring: trait schemas, facet schemas, runtime parameter declarations, derivation functions, composition policy overrides.
- **FR-051**: Registration MUST be reversible. `aps perspective unregister <name>` removes a Perspective; profiles holding traits or active facets from that Perspective are flagged but not auto-purged. Operator confirms purge separately.
- **FR-052**: Perspective registration MUST validate internal consistency (no self-referential facets, no traits declared without schemas if marked schema-strict, etc.) and report errors at registration time, not at first use.

#### Invocation parameter resolution

- **FR-060**: The substrate MUST provide an `InvocationParams` resolver. Inputs: profile id, scope-key (session/device/pinned, per FR-038), optional explicit overrides. Output: a structured `InvocationParams` with two sub-structs that separate concerns by layer:
    - **`model`** — parameters consumed by an LLM call: `model_id`, `max_tokens`, `temperature`, `thinking_mode`, `response_register`, plus any Perspective-declared model-layer parameters.
    - **`runner`** — parameters consumed by the agent runner (which arranges the LLM call): `available_tools`, `mcp_servers`, `retrieval_filters`, `timeout_budget`, plus any Perspective-declared runner-layer parameters.
    Mixing the two is rejected: a Perspective declaring `available_tools` under `model` (or `temperature` under `runner`) MUST fail at registration. The blessed parameter name → sub-struct mapping is part of the substrate, not Perspective-defined.
- **FR-061**: Each `InvocationParams` field (in either sub-struct) MUST be resolvable by a derivation function declared on a Perspective. Derivation functions receive the current stance (active facets + situated trait values) as a read-only input and return **only the parameter value** — never a tuple, struct, or wrapper that includes audit metadata. The substrate (not user code) populates audit fields in the resolved `InvocationParams` (FR-062). This is a type-level discipline: a derivation function whose declared signature returns anything other than the bare parameter value MUST be rejected at Perspective registration. Derivation functions MUST be pure (same purity rules as FR-041): no side effects, deterministic, read-only input.
- **FR-062**: Explicit overrides passed to the resolver MUST take precedence over derivation. Overrides MUST specify the sub-struct (e.g. `--override model.temperature=0.7` or `--override runner.timeout_budget=30s`); a bare `--override temperature=0.7` is ambiguous and MUST be rejected. The `InvocationParams` output MUST include an audit field per parameter indicating whether the value was derived, overridden, or defaulted, and (when derived) which Perspective and which derivation function produced it. Audit fields are **populated exclusively by the substrate**, never by derivation functions or user-supplied code; the substrate observes its own resolution path and records the source. A derivation function cannot forge a `source: overridden` claim because it has no path to write the audit field.
- **FR-063**: Values returned by derivation functions outside the declared bounds for a parameter MUST be clamped to the bound and the clamp MUST be logged in the audit.
- **FR-064**: When no Perspective declares a derivation function for a blessed parameter, the substrate MUST fall back to a built-in default. Defaults are documented per parameter (e.g. `model.temperature` default 0.7, `model.thinking_mode` default off, `runner.timeout_budget` default 60s).

#### Authorization

- **FR-079**: Trait baseline writes (`aps profile trait set`) follow the same authorization model as existing profile edits today: operator-runs-CLI with OS-level access to `<APS_DATA_PATH>`. No in-process auth gate beyond what aps already enforces. This is the substrate's default and the implementation plan owns whether to tighten it (e.g. for multi-user deployments).
- **FR-080**: A runtime executing *under* an active facet (e.g. an LLM agent running inside a `kid-sidecar` facet on Karim's profile) MUST NOT have an automatic write path to baseline traits on that profile. Specifically:
    - Situated trait reads are unrestricted (the runtime needs them to function).
    - Baseline trait writes require an explicit, separately-authorized operation outside the runtime's normal call path. The runtime CANNOT bypass this by using the substrate's read-then-write APIs from within an active facet.
    - The substrate MUST provide a "facet runtime" API surface that exposes situated reads and runtime-parameter resolution, but NOT baseline trait writes. Runtimes that need to suggest baseline updates emit them as proposals (a future consolidation API; deferred per FR-014's positive invariant) — they don't write directly.
- **FR-081**: Perspective-declared derivation functions (FR-061), facet preconditions (FR-037), and gate predicates (FR-032) execute in `kit/runtime/policy`'s CEL-via-pluggable-Evaluator sandbox (per Open Questions resolution). The sandbox MUST NOT expose baseline trait write APIs to executing code. CEL is read-only over the activation map by construction; plugged Evaluators (Cedar, OPA, etc.) MUST preserve the same read-only-over-trait-baseline guarantee.

#### Stance

- **FR-070**: The substrate MUST expose the current Stance for a given runtime context: active facets, situated trait values, resolved invocation parameters, audit metadata.
- **FR-071**: Stance MUST be inspectable by operators (`aps profile stance show <ctx>`) and diff-able against baseline (`aps profile stance diff <ctx>`).
- **FR-072**: Stance output MUST have a stable structured format when `--format json` (or `yaml`) is requested. Each field below is marked **R** (required, always present), **C** (conditional, present only when the named condition holds), or **O** (optional — emitter MAY include for completeness, consumer MUST tolerate absence). Defaults that "match the absence of the field" (e.g. `clamped: false`) MAY be omitted by emitters.
    ```yaml
    stance:
      profile_id: <string>                  # R
      scope_key:                            # R
        kind: session | device | pinned     # R
        id: <string>                        # R
      active_facets:                        # R (may be empty list)
        - perspective: <string>             # R
          name: <string>                    # R
          activated_at: <RFC3339 timestamp> # R
          degraded: <bool>                  # C: present only when true (absence == false)
          degraded_reasons: [<string>]      # C: required iff degraded == true
      traits:                               # R (may be empty map)
        <namespace>.<field>:
          baseline: <value>                 # R
          situated: <value>                 # R
          modulators_applied:               # R (empty list when no modulators apply)
            - facet: <perspective>.<name>   # R
              kind: delta | weight | filter | gate  # R (gate kind appears only if the gate matched and its action was applied; the resulting kind is what landed)
              value: <modulator-specific>   # R
              priority: <int>               # C: present only when non-zero (absence == 0 per FR-040)
              clamped: <bool>               # C: present only when true (absence == false)
      invocation_params:                    # R
        model:                              # R
          <field>:
            value: <value>                  # R
            source: derived | overridden | default  # R
            derived_by:                     # C: required iff source == derived
              perspective: <string>         # R within derived_by
              function: <string>            # R within derived_by (function name as declared in Perspective)
            override_at: <RFC3339 timestamp> # C: required iff source == overridden
            clamped: <bool>                 # C: present only when true (absence == false)
        runner:                             # R (same field shape as model)
          <field>: …
    ```
    The exact field names and required/conditional/optional discipline above are normative; consumers MAY rely on them. Additional fields MAY be added in compatible ways (new keys, never repurposed keys, never tightening optional → required without a contract version bump). When `source: default`, neither `derived_by` nor `override_at` is present; the value is the substrate-built-in fallback (FR-064).

### Key Entities

- **Profile**: identity record (existing). Identity-only, invariant. Human-or-agent-symmetric.
- **Trait**: profile-owned property family. Stored per-profile. Open-ended taxonomy with blessed core + Perspective-namespaced extensions.
- **Facet**: context-bound activation bundle. Declared per-Perspective, *attached* to specific profiles, *activated* per runtime context (session / device / pinned).
- **Perspective**: registered definition (schemas + derivations + composition policy) declaring a coherent set of traits, facets, and invocation-param shapes.
- **Stance**: the runtime-active state for a given (profile, scope-key) — which facets are active, which trait values they're modulating, which invocation parameters resolve to what.
- **Modulator**: a delta/weight/filter/gate declared on a Facet that biases a trait field's situated value while the Facet is active.
- **Scope-key**: the addressing tuple identifying a runtime context for activation persistence — kind ∈ {session, device, pinned} + id.
- **Derivation function**: a Perspective-declared function mapping (stance) → (InvocationParams field value).
- **InvocationParams**: structured output of the resolver, with `model` and `runner` sub-structs separating LLM-layer concerns from agent-runner-layer concerns.

## CLI Surface (sketch)

The substrate MUST support operations covering Perspective lifecycle (register/unregister/list/show), Trait read/write (list/get/set/remove), Facet definition/attachment/activation (define/attach/detach/activate/deactivate/list), Stance inspection (show/diff), and Invocation parameter resolution (with override syntax that requires sub-struct qualification per FR-062).

The concrete CLI surface — exact command tree, flag names, output formatting — is an implementation-plan decision and is NOT prescribed in this spec. The user-facing UX must be settled with attention to existing aps CLI conventions (kit/console patterns, global flags) at implementation-plan time, not bikeshedded here.

## Documentation Tasks

The following documentation updates are side effects of this spec landing. They are not normative requirements on the substrate; they're required-but-separate work items tracked alongside implementation:

- **D-001**: Update `~/.ops/docs/glossary.md` Facet definition. Replace *"Context-specific presentation (email, avatar, handles). Multiple per profile."* with *"Context-bound activation. Bundles scope + active capabilities + active connections + presentation surface + trait modulators. Modulates trait expression in context; does not replace identity (name, email, DID) or rewrite trait baselines."*
- **D-002**: Add glossary entries for Trait, Perspective, Stance, Modulator, Scope-key, InvocationParams.
- **D-003**: Sharpen the existing anti-pattern. *"Don't treat facets as identity walls; facets adjust presentation, not personality"* becomes *"Facets modulate trait expression in context; they don't replace identity (name, email, DID) or rewrite trait baselines. Personality and memory expression CAN vary by facet (modulated); their baselines are profile-owned."*
- **D-004**: Update glossary term-overload row for `facet`: aps definition becomes "context-bound activation" (was "presentation").
- **D-005**: Update `~/.ops/docs/architecture/identity-model.md` to reflect Trait joining as a peer to Facet on Profile, with Perspective as registration mechanism and Stance as runtime projection.

## Migration & Deprecation

This spec adds the Trait/Facet/Perspective/Stance substrate. Existing aps profile fields and conventions interact with it as follows; deprecation is selective, not wholesale:

### Existing fields on `core.Profile` (from `internal/core/profile.go`)

- **`Persona`** (Tone/Style/Risk struct, profile.go:158): **Deprecate in favor of the future `character` Trait family.** Persona's three fields are conceptually a tiny subset of what a `character` trait will carry. Deprecation path:
    - This spec's implementation: Persona stays in place; the `character` blessed Trait is not yet defined (FR-019 names it as a future spec). No migration yet.
    - When `character` Trait spec lands: a migration tool reads each profile's Persona and proposes a `character` baseline (e.g. `Persona.Tone="formal"` → `character.tone=formal`). Operator approves. Persona field becomes deprecated and emits a load-time warning; removal in a later release.
- **`Squads`** ([]string, profile.go:108): **Keep.** Squads represent collaboration archetype membership — a relationship-with-other-profiles concept distinct from dispositional baseline. May eventually live under a `membership` Trait family, but this spec does not migrate it. Squads continue to work unchanged.
- **`Roles`** ([]string with values like `owner, assignee, evaluator, auditor`, profile.go:110): **Keep.** Roles are task-collaboration roles, already disambiguated in the glossary. Distinct from anything in the Trait taxonomy. Roles continue to work unchanged.
- **`Scope`** (`*ScopeConfig` with FilePatterns/Operations/Tools/Secrets/Networks, profile.go:109): **Migrate into Facet definitions.** ScopeConfig describes access boundaries; Facets carry "active capabilities + scope" by definition (FR-030). The implementation plan will:
    - Treat existing `Scope` as the implicit "default Facet" of every profile during a transition window.
    - Provide a migration tool that promotes `Scope` to an explicit Facet under a per-profile default Perspective.
    - After migration, `core.Profile.Scope` becomes deprecated; runtime continues to read it for backwards compat for one release, then removed.
- **`Capabilities`** ([]string, profile.go:91): **Keep, with semantic clarification.** Capabilities (a2a, webhooks, agntcy-identity) are profile-level enabled features. They're referenced by Facets via `active capabilities`, but the master list stays on the Profile. No migration.

### Existing on-disk `facets/` directory

Today's `<APS_DATA_PATH>/profiles/<id>/facets/<name>.yaml` files (e.g. `noor/facets/ic.yaml`) are freeform YAML. They mix what this spec separates: identity-shaped fields (display_name, email, handles) appear alongside context-shaped fields (tone, governance, ctxt.default_lens).

Rather than migrate them as part of this spec's implementation, the implementation plan will:

1. Stand up the new typed Trait + Facet system alongside the existing freeform `facets/` directory.
2. Provide a migration tool (`aps profile migrate-legacy-facets`) that reads existing freeform facet files and proposes a split: identity-shaped fields *rejected* (already on Profile), trait-shaped fields proposed for promotion to typed Traits under a default Perspective, facet-shaped fields proposed for promotion to typed Facets.
3. Operator approves the migration per profile; freeform `facets/` directory is preserved as a `.legacy/` archive after migration.

This keeps existing tooling working during the transition and avoids destructive rewrites.

## Success Criteria

- **SC-001**: An operator can register a custom Perspective from a YAML file and use its traits and facets on profiles without modifying aps source code.
- **SC-002**: A profile carrying both a human-shaped facet (e.g. `family-member`) and an agent-shaped facet (e.g. `assistive-runner`) reads cleanly via the same APIs; no schema fork between human and agent profiles.
- **SC-003**: A runner calling `aps profile resolve-params` for a profile with two active facets gets a deterministic `InvocationParams` answer; the same call with the same state always returns the same answer.
- **SC-004**: Activating and deactivating a facet does not mutate baseline trait values. After deactivation, baseline reads return the values that were stored before activation. Specifically: no facet-lifecycle event ever writes to a baseline trait file.
- **SC-005**: Two Perspectives that both declare a trait named `tone` can coexist without colliding. Each is namespaced (`<perspective>.tone`) and addressable independently.
- **SC-006**: An explicit override passed to the resolver appears in the audit field of the returned `InvocationParams`, distinguishable from a derived value. A bare override without sub-struct qualifier is rejected with a clear error.
- **SC-007**: Stance inspection (`aps profile stance show --format json`) emits the FR-072 structured format completely: active facets (with degraded state if any), per-trait baseline + situated + applied modulators, per-parameter source + audit. An operator reading the output can reconstruct why the runner behaved a particular way.
- **SC-008**: The blessed `nature` trait correctly distinguishes human, agent, and device profiles; downstream consumers can branch on it without inspecting other fields.
- **SC-009**: Activating a facet whose preconditions are unmet fails fail-closed by default; the error names the failing precondition. A Perspective opting into `degrade` policy gets partial activation with the degraded state visible in stance inspection.
- **SC-010**: Safety-class trait modulation under multi-facet composition always yields the most-restrictive value, regardless of activation order or modulator priorities.
- **SC-011**: Existing `core.Profile` fields (Persona, Squads, Roles, Scope, Capabilities) continue to function during the transition window without operator intervention. Persona and Scope migrate to their typed equivalents via explicit operator-driven tools, not auto-rewrite. Squads, Roles, and Capabilities are preserved without migration.
- **SC-012**: A runtime executing under an active facet cannot write to baseline trait values via any substrate API surface available to it. Trait baseline mutations require an authorization path outside the facet runtime.

## Open Questions (to resolve during implementation)

- **Derivation-function and predicate language — RESOLVED**: derivation functions (FR-061), facet preconditions (FR-037), gate conditions in modulators (FR-032), and Perspective composition overrides (FR-041) all use **CEL via `kit/runtime/policy`'s pluggable Evaluator interface** (kit ADR-0008). kit ships CEL as the default backend (`policy/withcel`) and exposes an `Evaluator` interface so consumers MAY plug Cedar / OPA / etc. via the same shape. aps adopts the default CEL Evaluator unless a specific Perspective declares an alternative. The sandbox + determinism + audit-inspectability requirements (FR-062, FR-072) are met by kit's existing engine; aps does not need to choose or build a new runtime.
- **Schema language**: JSON Schema vs. CUE vs. Pkl vs. custom DSL. Decision deferred to implementation plan; should align with whatever ctxt's plugin layer settled on (per ADR-012 in ctxt).
- **Storage backend for trait values**: per-file YAML vs. SQLite vs. embedded KV. Implementation-plan decision; per-file YAML is the default (FR-016) but the substrate MAY add a cache layer.
- **Activation lifecycle defaults**: FR-038 supports session/device/pinned scope-keys; the implementation plan picks the default for `aps profile facet activate` (most likely session-scoped if no `--scope` flag).
- **`aps profile facet activate` UX**: explicit scope flag vs. implied current session via env. Operator UX decision in the implementation plan.

---

**Related**:

- `~/.ops/docs/glossary.md` — terminology canon, updated alongside this spec's approval.
- `~/.ops/docs/architecture/identity-model.md` — current three-layer Profile/Facet/Capacity model. This spec extends it: Trait joins as a peer to Facet (both ride on Profile), Perspective is the registration mechanism, Stance is the runtime projection.
- Follow-up specs (not yet written): blessed trait schemas (`memory`, `character`, `experience`, …), memory consolidation, first Perspective definitions (`ic`, `family`, `kid-sidecar`).
