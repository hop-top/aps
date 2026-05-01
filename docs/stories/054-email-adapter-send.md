---
status: paper
---

# 054 - Email Adapter Send (Founder Approval Workflow)

**ID**: 054
**Feature**: Adapter Subsystem
**Persona**: [User](../personas/user.md)
**Priority**: P2
**Status**: paper
**Author**: jadb
**Task**: T-0111

## Story

As a founder running an ops profile (e.g. Lina, Marketing), I want to send email via
`aps adapter exec email send --profile <id> --input to=... --input subject=...
--input body-file=...` (himalaya backend) so outbound mail uses profile-bound identity
without hand-rolling SMTP config.

Per ops convention: drafts queue in outbox; explicit release step sends. Founder approval
gates outbound sends pre-release.

## Acceptance Scenarios

1. **Given** profile `lina` configured with himalaya backend,
   **When** `aps adapter exec email send --profile lina --input to=foo@x --input
   subject=Hi --input body-file=/tmp/b.txt` runs,
   **Then** exit 0; message delivered via himalaya.

2. **Given** successful send,
   **When** I list mailbox folders post-send,
   **Then** Sent folder contains the message; outbox does not.

3. **Given** profile DKIM/SPF configured at upstream MTA,
   **When** send executes,
   **Then** himalaya delegates signing/policy unchanged; adapter does not re-sign.

4. **Given** transient send failure (network/SMTP),
   **When** caller re-invokes the same command,
   **Then** retry succeeds idempotently; no duplicate left in outbox.

5. **Given** caller adds `--input draft=true`,
   **When** command runs,
   **Then** message lands in outbox (NOT sent); exit 0; draft id surfaced.

6. **Given** queued draft `<id>` awaiting approval,
   **When** founder runs `aps adapter exec email send-draft <id> --profile lina`,
   **Then** draft releases; delivered via himalaya; outbox cleared; Sent updated.

## Implementation Notes

- Adapter action `email/send` wraps `himalaya message send`.
- Adapter action `email/send-draft` wraps `himalaya message send` from outbox path.
- Draft mode: writes EML to `$APS_DATA_PATH/profiles/<id>/outbox/<draft-id>.eml`;
  no network call.
- Founder approval is a workflow convention (drafts pile up; founder reviews; releases
  via send-draft). No in-adapter ACL gate beyond `email:draft-only` capability check
  (see story 053).

## Tests

### E2E (planned)

- `tests/e2e/email/send_test.go`
  - `TestEmail_SendSucceeds`
  - `TestEmail_DraftOnly`
  - `TestEmail_SendDraftReleases`

### Unit (planned)

- `internal/adapter/email/send_test.go` — argument parsing, draft routing,
  outbox path resolution.

## Dependencies

- Builds on: 053-nadia-ea-capabilities (capability bundles include
  `email:draft-only` and `email:send-direct`).
- Related: ops runbook `runbooks/email-send.md` (founder approval flow).
