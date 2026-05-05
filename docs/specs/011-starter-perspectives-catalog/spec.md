# Feature Specification: Starter Perspectives Catalog

**Feature Branch**: `011-starter-perspectives-catalog`
**Created**: 2026-05-05
**Status**: Draft
**Input**: aps spec 010 (Profile/Trait/Facet/Perspective/Stance) defines the substrate but explicitly defers actual Perspective definitions to follow-up specs. Multiple downstream consumers (`~/.fam` household workspace, three signed-up early-adopter businesses, IC's FIR portfolio) all need a starter set of registered Perspectives so they can configure DPKMS deployments without each authoring the same definitions from scratch. This spec is the *catalog* — one paragraph per Perspective sketching what trait families and facets it will declare, what worldview it serves, and what its load-bearing distinctions are. Each catalog entry expands into a full per-Perspective spec later (012, 013, …).

## Overview

Spec 010 introduced Perspectives as the registered worldview definition that bundles trait schemas + facet schemas + invocation-param derivations + composition policy. Spec 010 deliberately did NOT define any Perspectives; it left that to "first Perspective definitions (`ic`, `family`, `kid-sidecar`, `elder-monitor`) — downstream specs."

This spec is the *intermediate artifact* between "Perspectives exist as a concept" and "every Perspective is fully specified." It enumerates a starter set of twelve Perspectives, each in one paragraph, with:

- The worldview the Perspective serves (who uses it, in what context).
- The trait families it will declare (with namespace prefix).
- The facets it will declare.
- Load-bearing distinctions vs. sibling Perspectives.
- A flag for whether the Perspective is `~/.fam`-relevant (as a forcing function for that workspace's design).

Each catalog entry is a sketch, not a full spec. Field-level schemas, derivation functions, modulator rules — all those land in per-Perspective follow-up specs (012, 013, …). The catalog's job is to (a) name what's coming, (b) draw load-bearing boundaries between sibling Perspectives so they don't accidentally collide, and (c) give downstream consumers (`~/.fam`, early-adopter businesses, IC's FIR portfolio) a list to pull from when they configure their DPKMS deployments.

### Why now

- aps spec 010 is committed (`bff05c9`); the substrate exists in spec but no concrete Perspectives have been authored.
- `~/.fam` workspace design (separate spec, `~/.ops`) names `family` and `kid-sidecar` as the Perspectives it registers; the catalog gives them a canonical home in aps.
- Three early-adopter businesses are signed up and will need at least `business-operations`, `consulting`, `professional-services`, possibly `agency` in their deployments.
- IC's FIR portfolio reuses these Perspectives; `founder-in-residence` is a Perspective in its own right.
- Authoring twelve full Perspective specs in parallel would scatter design attention. The catalog forces the boundaries to be drawn first; full specs follow once boundaries are stable.

### Out of scope

- Full schema definitions for any catalog entry (each is its own follow-up spec 012, 013, …).
- Implementation of any Perspective.
- Modulator math, derivation function bodies, composition policy details.
- Catalog entries beyond the initial twelve. Adding a Perspective is a future spec; this catalog is a starting set, not a closed list.

## User Scenarios & Testing

### User Story 1 — Operator Picks Perspectives for a Deployment (Priority: P1)

As an operator standing up a new DPKMS deployment (`~/.fam`, an early-adopter business, an FIR engagement), I want a list of available registered-or-soon-to-be-registered Perspectives so that I can pick which ones to install for my deployment without authoring them myself.

**Acceptance Scenarios**:

1. **Given** an operator is configuring `~/.fam`, **When** they consult this catalog, **Then** they identify `family` and `kid-sidecar` as the Perspectives to register, with each catalog entry pointing them at the (eventually-existing) full Perspective spec.
2. **Given** an operator is configuring an early-adopter consulting business deployment, **When** they consult this catalog, **Then** they identify `business-operations` + `consulting` + `professional-services` (depending on the engagement shape) as candidates.
3. **Given** an operator surveys this catalog and finds none of the twelve match their use case, **Then** the catalog's open-list note tells them they can author their own Perspective and propose adding it to the catalog in a follow-up spec.

### User Story 2 — Distinguishing Sibling Perspectives (Priority: P1)

As a Perspective author writing a full follow-up spec for one catalog entry, I want each catalog entry's distinctions vs. its siblings clearly stated so that I don't author overlapping schemas that would then need merging.

**Acceptance Scenarios**:

1. **Given** an author is writing the full spec for `professional-services`, **When** they consult the catalog entry, **Then** the entry tells them what makes professional-services distinct from `consulting`, `business-operations`, `legal-services`, and `medical-practice`.
2. **Given** a catalog entry collides with a sibling on a load-bearing concept, **Then** this catalog spec is updated to draw the boundary explicitly, before either follow-up spec lands.

## Catalog

Each entry below is a one-paragraph sketch in the same shape: **Worldview / Traits declared / Facets declared / Load-bearing distinctions / `~/.fam` relevance**.

### `family`

**Worldview**: a household — a set of related humans (and possibly assistive agents and embodied devices) coordinating shared logistics: meals, schedules, bills, school/work life, travel, finances, health.

**Traits declared**: `family.relationship` (parent-of, child-of, spouse-of, sibling-of, … with directional pairs), `family.role-in-household` (primary-caretaker, breadwinner, dependent, …), `family.contribution-to-budget` (numeric or band).

**Facets declared**: `family-member` (always-on activation when a profile carries the family Perspective; modulates presentation and active capabilities for the household context), `home-evening` (evening / weekend mode with playfulness, lower formality), `school-day` (weekday morning/afternoon mode for kids and supporting parents).

**Load-bearing distinctions**: distinct from `business-operations` (different stakeholder structure, no formal hierarchy, different policy guards). The relationship trait family is bidirectional and rich (kinship structure matters); business Perspectives have flatter relationship semantics (manager-of, report-of).

**`~/.fam` relevance**: ✓ — `~/.fam` registers `family` as its primary Perspective.

### `kid-sidecar`

**Worldview**: a child or teen in a household, with assistive agents that help them while respecting age-appropriate autonomy and parental oversight. Maturity bands distinguish a 6-year-old from a 13-year-old from a 17-year-old; same Perspective, different modulators.

**Traits declared**: `kid-sidecar.maturity` (with `band` field: early-childhood, middle-childhood, tween, teen, late-teen), `kid-sidecar.literacy` (reading-comprehension level, content-appropriateness threshold), `kid-sidecar.parental-permissions` (what household-shared capabilities are gated by parental approval).

**Facets declared**: `child-companion` (the assistive-agent-mode for sidecar interaction), `school-context` (active during school hours, modulates safety strictness up and playfulness down), `personal-channel-private` (active when the child is using personal-shared-only channels — modulates household-assistant read access to deny content reads while permitting metadata-only).

**Load-bearing distinctions**: distinct from `family` — `family` describes the household structure; `kid-sidecar` describes a child's specific protections and developmental considerations. A child carries BOTH `family` (with `relationship.child-of` set) AND `kid-sidecar` (with `maturity` set). They compose; they're not alternatives.

**`~/.fam` relevance**: ✓ — `~/.fam` registers `kid-sidecar` for Karim's profile, with `maturity.band: teen`.

### `elder-monitor`

**Worldview**: an elderly person whose profile is bound to monitoring devices (pendant, smart speaker, fall-detection sensors) for safety + health surveillance, with escalation policies for emergencies. The profile is the elderly person's; the device is its embodiment.

**Traits declared**: `elder-monitor.health-conditions` (catalog-of-conditions for context-aware alerting), `elder-monitor.escalation-contacts` (who-to-call-in-which-priority), `elder-monitor.cognitive-baseline` (for detecting deviation), `elder-monitor.medication-schedule` (with reminder cadence).

**Facets declared**: `monitoring-active` (the always-on activation for an elderly profile bound to monitoring devices), `emergency` (active when an alert triggers — modulates almost everything: bypass DND, enable rapid escalation, suspend privacy gates), `caregiver-visit` (active during scheduled visits — modulates privacy gates differently, allows caregiver read access to medical context).

**Load-bearing distinctions**: distinct from `family` — `family` is a coordination Perspective; `elder-monitor` is a safety-and-health Perspective. They compose: an elderly family member carries `family` AND `elder-monitor`. The `nature` Trait `embodiment.surface` is `pendant` or `voice-only` for many `elder-monitor` profiles; this is the strongest case for a non-screen embodiment.

**`~/.fam` relevance**: ✗ — not registered for `~/.fam` v1; named in the catalog because the substrate must support it for future deployments.

### `business-operations`

**Worldview**: a small or mid-sized business — administrators, managers, and executives coordinating the day-to-day running of an organization. Combines what would otherwise be three sub-Perspectives (admin, management, executive) into one with sub-role facets, because the trait surface is largely shared.

**Traits declared**: `business-operations.org-role` (admin / coordinator / manager / director / executive — flat enum, no hierarchy assumed), `business-operations.department` (free-form string), `business-operations.budget-authority` (numeric ceiling), `business-operations.signature-authority` (for contracts / approvals).

**Facets declared**: `administrator` (sub-role facet for admin-flavored operations: scheduling, document handling, intake), `manager` (decision-routing, report management, hiring participation), `executive` (strategy, board-level signaling, signing authority), `working-hours` (default working window per profile; modulates DND outside hours).

**Load-bearing distinctions**: distinct from `consulting` (consulting is engagement-bound to a specific client; business-operations is intra-org). Distinct from `professional-services` (professional-services has a fiduciary/confidentiality dimension absent in plain business-operations). Distinct from `agency` (agency is service-delivery to a client with retainer; business-operations is internal). Three signed-up early-adopter businesses are the primary load-bearing consumer.

**`~/.fam` relevance**: ✗ — but this Perspective is the business-shaped sibling that defines what `family` is NOT, so the boundary helps both.

### `consulting`

**Worldview**: an advisory engagement — a consultant (single or team) advises a client on a problem within a bounded scope, with deliverables and time-bounded engagement lifecycle. Distinct from agency in that consulting is advisory; the client retains execution responsibility.

**Traits declared**: `consulting.specializations` (list of expertise domains), `consulting.engagement-style` (advisory / coaching / assessment / strategy / implementation-light), `consulting.client-confidentiality-tier` (default / NDA / strict).

**Facets declared**: `engagement-active` (activated for the duration of an engagement; declares which client profile/Perspective is engaged with, what scope of data is in-scope), `discovery-mode` (early phase, broader read access for assessment), `delivery-mode` (deliverable-producing phase, narrower context, deeper focus).

**Load-bearing distinctions**: distinct from `agency` — consulting advises, doesn't execute. Distinct from `professional-services` — consulting is bounded to a project/engagement; professional-services is an ongoing fiduciary relationship (lawyer with continuing retainer, accountant on annual cycle). Distinct from `business-operations` — consulting is client-facing-advisory, business-operations is intra-org.

**`~/.fam` relevance**: ✗.

### `professional-services`

**Worldview**: an ongoing professional relationship under a fiduciary or expert duty (lawyer, accountant, architect, financial advisor, doctor's office not under medical-practice, …). Distinct from consulting in continuity; distinct from business-operations in client-facing fiduciary role.

**Traits declared**: `professional-services.profession` (lawyer / accountant / architect / financial-advisor / …), `professional-services.licenses` (jurisdiction-specific licenses with expiration dates), `professional-services.fiduciary-tier` (informational / advisory / fiduciary / …).

**Facets declared**: `client-engaged` (active for a specific client; declares confidentiality scope and cross-client conflict-of-interest checks), `general-practice` (default mode, no specific client active).

**Load-bearing distinctions**: distinct from `legal-services` and `medical-practice` — those carry distinct privilege/confidentiality regimes that warrant their own Perspectives. `professional-services` is the catch-all for fiduciary professionals not covered by those two. Distinct from `consulting` — ongoing fiduciary, not engagement-bounded advisory.

**`~/.fam` relevance**: ✗.

### `legal-services`

**Worldview**: a law practice — attorney-client privilege, conflict-of-interest checks, jurisdiction-specific bar admission and ethical rules. Privilege is THE load-bearing concern that warrants a separate Perspective from `professional-services`.

**Traits declared**: `legal-services.bar-admissions` (jurisdiction-keyed list with admission and good-standing dates), `legal-services.privilege-tier` (attorney-client / work-product / common-interest / waived), `legal-services.specialization` (corporate / litigation / IP / family / immigration / …).

**Facets declared**: `client-matter-active` (activated for a specific matter on a specific client; declares privilege scope, conflict-checked counterparties, in-scope and out-of-scope data), `non-engaged-research` (research not yet bound to a client; not yet privileged), `cross-client-firewall` (when a profile is in one matter, this facet's modulators block read access to other clients' data even when normally available).

**Load-bearing distinctions**: distinct from `professional-services` — privilege rules are unique to legal; cross-client firewalls are unique to legal practice ethics; document retention requirements are jurisdiction-specific.

**`~/.fam` relevance**: ✗.

### `medical-practice`

**Worldview**: a medical practice — patient confidentiality (HIPAA in US, similar elsewhere), clinical workflows, prescription authority, mandatory reporting. Confidentiality regime is unique enough to warrant its own Perspective.

**Traits declared**: `medical-practice.specialization` (general / specialty / sub-specialty), `medical-practice.licenses` (jurisdiction-keyed), `medical-practice.privilege-tier` (patient-confidential / public-health-reportable / consent-released / waived), `medical-practice.dea-authority` (controlled-substance prescription authority where applicable).

**Facets declared**: `patient-encounter-active` (active for a specific patient encounter; declares which patient's records are in scope, which clinical context is loaded), `peer-consultation` (active when consulting another provider on a patient case; modulates which patient data is shared), `mandatory-report-pending` (active when a mandatory-report condition has been observed but not yet reported; specific escalation/audit modulators).

**Load-bearing distinctions**: distinct from `professional-services` and `legal-services` — patient confidentiality and mandatory reporting are unique. HIPAA-shaped audit requirements affect every operation.

**`~/.fam` relevance**: ✗ — but "elder-monitor" Perspective interfaces with this in deployments where an elderly family member's monitoring data is shared with a treating physician under explicit consent. Cross-Perspective integration is a follow-up question.

### `agency`

**Worldview**: an agency or service-delivery business that produces deliverables for clients on a retainer or per-engagement basis (LesExperts, design agencies, marketing agencies, dev shops). Distinct from consulting in execution responsibility; distinct from `professional-services` in being non-fiduciary.

**Traits declared**: `agency.service-lines` (what the agency offers), `agency.retainer-tiers` (yearly-plan, monthly-retainer, project-based), `agency.client-portfolio` (current client list with engagement state).

**Facets declared**: `client-engagement-active` (per-client engagement facet), `delivery-sprint` (intensive delivery period), `account-management` (relationship management mode), `intake` (new-client onboarding mode).

**Load-bearing distinctions**: distinct from `consulting` — agency executes; consulting advises. Distinct from `business-operations` — agency is client-facing; business-operations is intra-org. LesExperts is the load-bearing example here.

**`~/.fam` relevance**: ✗.

### `receptionist`

**Worldview**: front-of-house / first-point-of-contact for an organization — handles intake, routing, scheduling, screening. May be a human role or an autonomous agent role. The Perspective covers both; the discriminator is the `nature.kind` Trait.

**Traits declared**: `receptionist.intake-flows` (which incoming-channel-and-disposition flows are configured), `receptionist.routing-table` (recipient by topic/urgency/contact), `receptionist.screening-rules` (what gets blocked or escalated).

**Facets declared**: `intake-active` (the always-on receptionist mode), `vip-handling` (active for a recognized VIP contact; modulates routing and tone), `crisis-mode` (active when escalation conditions trigger — bypass DND, escalate fast).

**Load-bearing distinctions**: distinct from `business-operations.administrator` — receptionist is a specialized role that may exist within a larger business-operations Perspective (so a profile can carry both, with receptionist's facets active during front-desk hours). Standalone, it covers small-business or single-person-with-receptionist-agent deployments. The "personal-assistant" agent the user named in earlier brainstorming is closely related but distinct: personal-assistant is one-to-one (one principal), receptionist is one-to-many (one front-desk, many incoming).

**Catalog status (under review)**: receptionist's standalone Perspective status is open. Editorial review flagged that its declared Traits (intake-flows, routing-table, screening-rules) read more like configuration of an *agent-shaped* profile than a worldview a human would carry, and its load-bearing distinctions argue it could ride as facets on `business-operations` and `agency` rather than as a peer Perspective. The catalog keeps the entry pending the full receptionist follow-up spec, which is the right place to make the keep-or-merge call (with concrete schema in hand). If the full spec confirms the merge case, this entry is removed and the facet patterns migrate to `business-operations` and `agency` entries.

**`~/.fam` relevance**: ✗ — but the household equivalent "head of household coordination" function maps to `family-member` facet within the `family` Perspective, not to `receptionist`.

### `broker`

**Worldview**: an intermediary between two or more parties in a transaction (real estate, finance/securities, insurance, M&A, talent). Fiduciary duty to the client; sometimes regulated; transaction-bound rather than ongoing-relationship.

**Traits declared**: `broker.brokerage-type` (real-estate / financial / insurance / m-and-a / talent / …), `broker.licenses` (regulator-keyed list), `broker.commission-structure` (per-side / per-transaction / retainer + commission).

**Facets declared**: `transaction-active` (per-transaction facet declaring all parties, fiduciary alignment, conflict-check status), `pre-engagement` (cultivation/lead-gen mode, no fiduciary duty yet), `post-close` (transaction complete, residual obligations).

**Load-bearing distinctions**: distinct from `agency` — broker has fiduciary alignment to specific parties in a specific transaction; agency has retainer-style ongoing service delivery. Distinct from `professional-services` — broker is transaction-bounded; professional-services is relationship-bounded.

**`~/.fam` relevance**: ✗.

### `founder-in-residence`

**Worldview**: an external founder leading a venture incubated by IC's FIR portfolio. IC studio supplies tech, design, GTM, legal, finance support; takes 20–50% equity; founder leads execution. The Perspective covers the founder's profile when they're operating within the IC ecosystem.

**Traits declared**: `founder-in-residence.venture` (the FIR venture name and stage), `founder-in-residence.equity-arrangement` (IC equity percentage, vesting schedule, governance), `founder-in-residence.support-mix` (which IC support areas are active for this venture: tech-by-IC, design-by-IC, gtm-by-IC, legal-by-IC, finance-by-IC).

**Facets declared**: `fir-active` (always-on for an FIR-engaged founder; modulates which IC capabilities the founder has access to), `board-meeting-mode` (active during scheduled IC board interactions for this venture), `venture-only` (active when the founder is operating purely on their venture's behalf, without IC support context).

**Load-bearing distinctions**: distinct from `business-operations` — FIR has IC-specific support and equity arrangements; business-operations is generic. Distinct from `consulting` — FIR is an equity-holding ongoing relationship, not advisory. The "Our" portfolio (IC fully-owned products) does NOT use this Perspective; FIR is specifically the incubation case.

**`~/.fam` relevance**: ✗ — but IC operators' profiles carry `founder-in-residence` when active in FIR ventures, and may simultaneously carry `family` (for their household) — so the catalog's job is to draw the boundary so an operator's stance can compose both cleanly.

## Requirements

### Functional Requirements

- **FR-001**: This catalog MUST include the twelve Perspectives listed in § Catalog above. Each catalog entry follows the same shape: Worldview / Traits declared / Facets declared / Load-bearing distinctions / `~/.fam` relevance.
- **FR-002**: This catalog is **a living document**. New Perspectives MAY be added in follow-up specs that update this catalog. Removing a Perspective from the catalog requires explicit deprecation + migration plan if any deployment registered it.
- **FR-003**: Each catalog entry MUST point to its eventual full per-Perspective spec when authored. Until the full spec exists, the catalog entry is the authoritative sketch.
- **FR-004**: Boundaries between sibling Perspectives MUST be drawn at catalog level, before per-Perspective full specs are authored. Mergers and splits are catalog-level decisions, not full-spec-level.
- **FR-005**: A downstream consumer registering a Perspective from this catalog MUST do so by pointing aps's Perspective registration (per spec 010 FR-050) at the eventual full spec's `perspective.yaml`. Until the full spec exists, registration of a catalog Perspective fails with "spec not yet finalized."
- **FR-006**: When a catalog entry's full per-Perspective spec is authored, it MUST adopt kit primitives where applicable rather than redescribing substrate. In particular: Perspective-declared policy modulators land as CEL rules consumed by `kit/runtime/policy` (kit ADR-0008); derivation functions, facet preconditions, and gate predicates use kit's pluggable Evaluator interface (CEL default per spec 010 Open Questions resolution); cross-instance/cross-device replication of Perspective state piggybacks on `kit/runtime/sync`; bus topic names follow kit's 4-segment past-tense convention. See `~/.ops/docs/architecture/kit-primitive-map.md` for the canonical mapping.

### Key Entities

- **Catalog entry**: a one-paragraph Perspective sketch with Worldview / Traits / Facets / Distinctions / `~/.fam` relevance fields.
- **Perspective (existing aps concept, spec 010)**: registered worldview definition. Catalog entries become full Perspectives via per-Perspective follow-up specs.
- **Sibling distinction**: explicit boundary statement between two catalog entries that share concept space (e.g. consulting vs. agency, professional-services vs. legal-services).

## Documentation Tasks

- **D-001**: When each per-Perspective full spec lands, this catalog's corresponding entry MUST be updated to link the full spec (`docs/specs/0XX-<perspective-name>/spec.md`).
- **D-002**: Update aps's `docs/specs/` README (or top-level index) to point at this catalog as the entry-point for "what Perspectives exist or are planned."

## Success Criteria

- **SC-001**: A downstream consumer (`~/.fam`, an early-adopter business, an FIR engagement) can read this catalog and identify which Perspectives apply to their deployment without needing to author them.
- **SC-002**: Per-Perspective full specs (012, 013, …) authored from these catalog entries do NOT have load-bearing collisions with their siblings — the catalog drew the boundaries clearly enough.
- **SC-003**: Adding a thirteenth Perspective is a documented process: a follow-up spec proposes the addition, this catalog is updated, the new full spec follows.
- **SC-004**: No two catalog entries declare a Trait or Facet name that would collide under aps spec 010's namespacing rule (FR-015 / FR-040).

## Open Questions

- **Catalog structure beyond twelve**: as more Perspectives are added, the catalog may need section grouping (Family-shaped, Business-shaped, Specialty-shaped). v1 catalog is flat; revisit if it grows past ~20.
- **Cross-Perspective composition rules**: a single profile may carry traits and facets from multiple Perspectives (an operator with `family` + `business-operations` + `consulting`). Spec 010's composition rules (FR-040) govern modulator composition within a trait field; cross-Perspective interaction at the *workflow* level (e.g. policy guards from `family` interacting with policy guards from `business-operations`) is a follow-up concern.
- **Sub-Perspective vs. sibling distinction**: `business-operations` merges admin/management/executive into sub-role facets within one Perspective. The same logic could apply to others (e.g. should `kid-sidecar` be sub-modes of `family`?). Each catalog entry's boundary draws this call; the principle isn't fully systematized.
- **Locale and regulatory variation**: `legal-services`, `medical-practice`, `broker` all have jurisdiction-specific regimes. Whether to fork them per-jurisdiction (e.g. `legal-services.us-bar-admitted` vs. `legal-services.eu-bar-admitted`) or keep them flat with jurisdiction Traits is a per-Perspective full-spec decision.

---

**Related**:

- aps spec 010 (Profile/Trait/Facet/Perspective/Stance) — `docs/specs/010-profile-traits-and-facets/spec.md`. The substrate this catalog rides on.
- `~/.fam` workspace design — `~/.ops/docs/superpowers/specs/2026-05-05-fam-workspace-design.md`. Consumer of `family` and `kid-sidecar` from this catalog.
- Future per-Perspective specs (each catalog entry → its own follow-up spec): 012-family, 013-kid-sidecar, 014-elder-monitor, 015-business-operations, 016-consulting, 017-professional-services, 018-legal-services, 019-medical-practice, 020-agency, 021-receptionist, 022-broker, 023-founder-in-residence (numbering provisional).
