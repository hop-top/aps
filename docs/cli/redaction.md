# Redaction policy and threat model

aps redacts secrets and PII from every stream it writes — log lines,
adapter output, webhook responses, persisted log files, child process
stdio. This document describes what is redacted, how to bypass when
diagnosis genuinely requires raw output, and what redaction does NOT
protect against.

## TL;DR

- Redaction is **ON by default**.
- The default replacement strategy is **Tag**: every match becomes
  `<rule-id>` (e.g. `<openai-api-key>`). The kind of secret is
  diagnosable; the value is gone.
- To bypass for a single command:
  `--no-redact` (persistent flag) or `APS_DEBUG_NO_REDACT=1` env.
- Rules: gitleaks corpus (~211 patterns) + Presidio PII pack (~11) +
  4 aps-domain header rules.

## What's redacted by default

aps adopts `kit/core/redact`'s `Default()` corpus, which loads:

- **Gitleaks content rules** (~211): API keys, access tokens, private
  keys, OAuth bearer tokens for ~150 named providers (OpenAI, AWS,
  GitHub, Stripe, Slack, Twilio, etc.) plus generic patterns
  (`generic-api-key`, `jwt`, `private-key`).
- **Presidio PII pack** (~11): credit cards, IP addresses, IBANs, SSNs,
  phone numbers, email addresses (where the provider class can be
  confidently classified — case-by-case).

aps adds four domain-specific rules on top of the default corpus
(see `internal/logging/redact.go::Redactor`):

| Rule ID                | Matches | Replacement |
|------------------------|---------|-------------|
| `aps-bearer-header`    | `Authorization: Bearer <token>` (any case) | `Authorization: Bearer <aps-bearer-header>` |
| `aps-x-api-key-header` | `X-API-Key: <token>` (any case)            | `X-API-Key: <aps-x-api-key-header>` |
| `aps-aps-signature`    | `X-APS-Signature: sha256=<hex>`            | `X-APS-Signature: sha256=<aps-aps-signature>` |
| `aps-generic-bearer`   | bare `Bearer <token>` outside header context | `<aps-generic-bearer>` |

The header rules use the kit `Custom` replacement strategy to keep
the key name intact. This is the "structured-log key-aware redaction"
property: a debugger can see *that* the operator passed an
Authorization header (and was rejected, accepted, etc.) without ever
seeing the bearer token's bytes.

### Allowlisted placeholders

These doc/test fixtures pass through unchanged:

- `sk-test` — OpenAI dummy key shape used in fixtures.
- `AKIAIOSFODNN7EXAMPLE` — AWS canonical example access key.
- `ghp_test` — GitHub personal-access-token placeholder.

If you need to add a new fixture allowlist entry, edit the
`r.Allow(...)` call in `internal/logging/redact.go`. The kit allowlist
is substring-only — it cannot accept regex (a regex allowlist would
be a ReDoS vector). Pick a substring that is unique to the fixture
and not a prefix of any real-world secret class.

## Where redaction is wired

Four canonical choke points cover every surface in
[redact-inventory.md](redact-inventory.md):

1. **Logger sink** — `internal/logging/logger.go::SetViper` wraps
   the kit logger writer with a redacting `io.Writer`. Every
   `logging.GetLogger().X(...)` line passes through `redact.Apply`
   before reaching `os.Stderr`. Covers all 16 logger callsites
   (L1-L16) and provides defense in depth for header-handling sites
   (H1-H4) that don't log today but might tomorrow.

2. **Stdout/stderr formatter** — `internal/logging/output.go` exposes
   `logging.Print`, `Println`, `Printf`, `Fprint*` helpers. They
   mirror `fmt.*` but route the formatted string through
   `redact.Apply`. Migrated HIGH-severity stdout sites (`aps env`,
   `aps adapter exec`, `aps a2a get-task`, `aps session inspect`).
   LOW sites (curated identity/version/skill output) keep using bare
   `fmt.*` — the indirection adds noise without a leak class to
   defend against.

3. **HTTP response body** — `respondJSON` (webhook) and
   `sendError`/`sendJSON` (agentprotocol) marshal-then-redact-then-
   write the response body. Covers webhook 500s carrying action
   stderr (W1) and agentprotocol error responses (W3).

4. **Persisted log files** — adapter subprocess `stdout.log` /
   `stderr.log` writers (in `internal/core/adapter/manager.go::startSubprocess`)
   are wrapped with a redacting writer. Covers O4. The on-disk file
   is `0644` and may persist indefinitely, so this is the highest-
   consequence surface — log retention is the implicit attacker
   here, not just a momentary console viewer.

5. **Child process stdio** — `runCommandWithProcessIsolation` and
   `runActionWithProcessIsolation` (in `internal/core/execution.go`)
   wrap `cmd.Stdout` / `cmd.Stderr` with redacting writers. This is
   the surface that produced the 2026-05-02 incident: a user ran
   `aps run noor -- env` and the child process's `env` listed
   `OPENAI_API_KEY=sk-…` to the parent terminal, which a tailing
   agent then absorbed into chat context.

## How to bypass

### Persistent flag

```sh
aps --no-redact run noor -- env
```

Flag is registered as a kitcli Global on the root command, so it
works on every subcommand and is visible in `--help`.

### Env var

```sh
APS_DEBUG_NO_REDACT=1 aps run noor -- env
```

Convenient for one-shot CI debugging or container-internal probes
where flag plumbing is awkward. The check accepts the standard
truthy strings (`1`, `true`, `yes`, `on`, case-insensitive variants).

### Viper config key

`redact.enabled` (default `true`). The flag inverts into this key in
`PersistentPreRun`. Operators can pin redaction ON in
`$XDG_CONFIG_HOME/aps/config.yaml` even when contributors pass
`--no-redact` ad-hoc:

```yaml
redact:
  enabled: true
```

…but note: `--no-redact` is set by `PersistentPreRun` AFTER config
load, so the flag wins. If you need true config-side enforcement,
use the env-var inversion guard pattern in your wrapping shell.

### When it's safe to bypass

- **Local dev with no log shipping** — your terminal is a closed
  channel; nothing is forwarding the bytes anywhere persistent.
- **A `tmpfs` mount with a self-clean budget** — diagnosis sessions
  where the output dies with the shell.
- **HMAC-signed verifier replay** — when you genuinely need to see
  the raw signature header to debug a verifier mismatch and you've
  already verified the signing secret is present in the same env
  you're running aps in (no information leak relative to the
  starting state).

### When it is NOT safe to bypass

- **Any session where Claude / a coding agent / a tailing assistant
  is reading the terminal**. The 2026-05-02 incident is the
  canonical case. Agents absorb the bytes into their context, which
  is then prompt-cached, written to transcript, and persisted. The
  incident specifically required clearing the cache + rotating the
  key.
- **CI logs** — GitHub Actions, GitLab CI, CircleCI, Buildkite, all
  retain logs by default and surface them to anyone with read access
  to the repo. Even on private repos, the retention window matters
  more than the moment.
- **`tee` to a file or `> out.log`** — unlike the agent case,
  retention is voluntary here, but the file mode is whatever your
  umask says. World-readable on most macOS configs.
- **`script(1)` / `asciinema rec`** — recording sessions are
  durably persistable and easy to share. Bypass redaction only if
  you intend to re-redact the recording before sharing.
- **Any adapter that ships logs to a remote sink** — you may not
  even know the sink is on; check the adapter manifest.

## Threat model

Redaction's job is to remove secret bytes from streams that may be
absorbed into a sink the operator does not directly control. The
assumed adversaries:

| Adversary | Capability | What redaction protects |
|-----------|------------|-------------------------|
| Log-storage operator | Reads any file aps writes (CI logs, persisted adapter logs, syslog) | Yes — secrets on disk become tags |
| Log-shipping pipeline (Loki, Splunk, ELK) | Reads stderr/stdout of any aps process via stdout-tailer/journald-shipper | Yes — secrets in transit become tags |
| Tailing agent (Claude/Cursor/etc.) | Reads the terminal in real time; absorbs into LLM context, transcript, cache | Yes — same mechanism as log shipping |
| Accidental sharing | A teammate runs `aps profile show` and pastes the output in Slack | Yes — `***redacted***` for known fields, redact tags for unknown/leaked-via-warning fields |
| Webhook caller | Receives the HTTP response body | Yes — both 200 and 5xx response bodies pass through redact |
| Adapter peer (subprocess) | Receives the env via `cmd.Env` | **NO** — redaction is an *egress* filter on streams aps writes; secrets in `cmd.Env` are intended (the whole point of `aps run` is to pass them to the child). The threat boundary is the parent stream, not the env table. |
| Co-tenant on the box | Reads `/proc/<pid>/environ`, `ps eww`, `lsof` | **NO** — redaction does not touch the process table. Use a different isolation boundary (containers, user separation). |
| Memory dump / coredump | Reads the address space | **NO** — redaction operates on serialized form. The `*Redactor` and the rule corpus are in memory. The original secret values are in `cmd.Env`, in the secrets-store, in any goroutine local. |
| Kernel keyring / syscall tracer | `strace -f` or similar | **NO** — redaction is a userspace string filter applied before `Write`. A tracer reading the syscall args before the filter runs sees nothing; a tracer reading the bytes after the writer does sees the redacted form. But a tracer that reads `read()` syscalls on the secrets-store path sees raw bytes — that is the secret store's threat boundary, not redaction's. |
| Disk forensics on the secrets file | `cat ~/.aps/profiles/noor/secrets.env` | **NO** — the file is the source of truth; redaction is downstream of it. Use `chmod 600` (which aps warns about in `core.LoadSecrets`) and consider `keyring` or `env` backends instead of `file`. |

Performance note: `kit/core/redact`'s default `Apply` is currently
~44ms on a 4KB clean payload with the full ~250-rule policy
(see `kit/go/core/redact/PERF.md`). aps wires redact onto
low-volume CLI paths and infrequent webhook events; we do NOT wire
it onto high-frequency hot paths (e.g. session reaper tick, adapter
log-file `cmd.Wait()` loop). The reaper logs an integer count, so
redact cost there is N/A.

## What's NOT a leak surface (and why)

These are NOT covered because they are not stream egress:

- **`cmd.Env` for the child process** — the subprocess receives the
  full secrets.env environment. That is the point of `aps run`.
  Redaction does not (and cannot) intercept the env-table handoff.
- **The secrets file on disk** — `~/.aps/profiles/<id>/secrets.env`
  is the source of truth. aps warns when its mode is loose (`!= 0600`)
  but does not encrypt at rest in the `file` backend. For the
  `keyring` backend, the OS keyring's encryption applies.
- **Memory** — secret values are in goroutine locals, the secrets
  store, and `cmd.Env`. A coredump or memory dump captures all of
  it. Use macOS Touch ID-gated keyring + `RLIMIT_CORE=0` if this is
  in scope.

## Failure modes

- **Catastrophic regex failure** — kit/core/redact uses RE2 (stdlib
  `regexp`). RE2 is linear-time matching by construction, so there
  is no ReDoS surface. Patterns from upstream gitleaks that use
  PCRE-only features (backreferences, lookaround) are silently
  skipped at load time rather than rejected.
- **Custom-formatter panic** — the kit Custom strategy recovers
  panics and degrades to Mask. aps's `apsCustomReplacement`
  function is panic-free (no map writes, no nil dereferences) but
  the safety net is there regardless.
- **Match against allowlist fixture** — `Allow("sk-test")` causes
  any match containing `sk-test` to pass through. This is by design
  for docs/test fixtures. Production secrets are statistically
  unlikely to contain that exact substring; if you find one that
  does, narrow the allowlist.

## Cross-references

- Source: `internal/logging/redact.go`, `internal/logging/output.go`
- Inventory of leak surfaces: [redact-inventory.md](redact-inventory.md)
- kit/core/redact API: `kit/go/core/redact/README.md`
- kit/core/redact perf budget: `kit/go/core/redact/PERF.md`
- kit ADR-0005: `kit/docs/adr/0005-kit-redact-egress-filtering.md`
- Postmortem reference: 2026-05-02 OPENAI_API_KEY exposure during
  diagnosis. The incident drove this track (`aps-redact-logs`).
