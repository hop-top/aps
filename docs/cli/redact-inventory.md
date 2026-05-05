# Redact inventory — secret-leak surfaces in aps logs and output

Audit (`T-0459`) of every site where aps may emit potentially-sensitive
content to a stream (logger, stdout, stderr, network response) without
running it through `kit/core/redact` first.

Inputs: source @ `aps-redact-logs` worktree (`a913ce9`),
audit reference `~/.ops/reviews/aps-kit-integration-audit-2026-05-04.md`
§4 item 3, leak postmortem 2026-05-02 (`OPENAI_API_KEY`).

## Method

Conservative scan (false positives are worse than misses):

- `git grep` for `logging.GetLogger`, `kitlog.`, `fmt.Print*`,
  `fmt.Fprint*`, `json.Marshal*`, header-name string literals.
- For each callsite, classify what flows in: profile fields,
  secrets-derived env, request bodies, response bodies, headers,
  webhook payloads.
- Risk severity:
  - **HIGH** — secret value can land verbatim on a stream by default
    (no opt-in, no flag, normal usage path).
  - **MEDIUM** — can land if a specific code path is exercised, but
    today's call sites bound the input class.
  - **LOW** — structurally possible but the input is curated (label,
    counter, ID).

Out of scope:

- Debug-only paths gated behind a `--debug` flag that is off by default
  (none qualify today; aps does not have a `--debug` toggle).
- Source files in `tests/`, `examples/`, `internal/testing/`.
- The `***redacted***` line in `aps profile show` (already manually
  redacted; see Excluded #X1).

## Categories

1. Logger callsites (`logging.GetLogger().X`) that include user-supplied
   or env-derived fields.
2. Direct stdout/stderr writes (`fmt.Print*`, `fmt.Fprint*`) that include
   profile state, env vars, request/response bodies.
3. Webhook payload logging (`internal/core/webhook.go`,
   `internal/cli/webhook/*.go`).
4. HTTP request/response paths
   (`internal/cli/serve.go`, `internal/adapters/agentprotocol/*.go`,
   `internal/core/adapter/mobile/server.go`).
5. Adapter exec output (`internal/cli/adapter/exec.go` →
   `internal/core/adapter/manager.go`).
6. Adapter subprocess log files
   (`internal/core/adapter/manager.go::startSubprocess`).
7. Session inspection (`internal/cli/session/inspect.go`).

## Inventory — flagged surfaces

Each row: ID, file:line, what's logged, what could leak, severity,
choke point that should redact it.

| # | Surface | File:line | Sink | What flows in | What could leak | Severity | Choke point |
|---|---------|-----------|------|---------------|-----------------|----------|-------------|
| L1 | webhook event log | `internal/core/webhook.go:112` | `kitlog.Info` (stderr) | event name, delivery_id, profile, action — but the prior step on line 76 reads `X-APS-Event` from the request and other request-derived strings can ride along in error paths | low directly; high indirectly because the next entry (L2) shares the same logger | MEDIUM | logger sink (writer wrap) |
| L2 | webhook execution failure | `internal/core/webhook.go:142` | `kitlog.Error` (stderr) | `err.Error()` from `RunAction(...)` — action stdout/stderr is forwarded through `os.Stdout`/`os.Stderr` (see `internal/core/execution.go:313`); when the action fails, an HTTP 500 response also includes `err.Error()` (line 145) which can echo any token the action printed before crashing | HIGH — action error message can include API key strings echoed by the user's script | HIGH | logger sink + HTTP body formatter |
| L3 | bundle warning | `internal/core/execution.go:188` | `kitlog.Warn` | bundle name + warning text returned by `bundle.Resolve` | warning strings are produced by kit/bundle; can include resolved env-var names but values are validated by bundle scope, not embedded in warnings today | LOW | logger sink |
| L4 | always-service start log | `internal/core/execution.go:192` | `kitlog.Info` | bundle, service.Name, service.Adapter | curated config strings; no secrets today | LOW | logger sink |
| L5 | adapter server error | `internal/core/adapter/mobile/server.go:155` | `kitlog.Error` | net error from `http.Server.Serve` | usually a port/listener error; tls handshake errors can include cert serials | LOW | logger sink |
| L6 | websocket error | `internal/core/adapter/mobile/server.go:444` | `kitlog.Error` | adapter ID + ws error | curated adapter ID + ws error string (no headers logged); but the adapter authenticates with a JWT and `Authorization` header (line 366) — if the JWT parser returns an error message that includes the raw token (charm/jwt does not, but kit/jwt could), it lands here | MEDIUM | logger sink |
| L7 | session reaper cleanup | `internal/core/session/registry.go:159,163` | `kitlog.Error/Info` | session-count integers, error string from `CleanupInactive` | low — session IDs are uuids, not secrets | LOW | logger sink |
| L8 | adapter close errors | `internal/cli/serve.go:106` | `kitlog.Error` | error slice from `mgr.CloseAll()` | adapter close errors are infra-level; low risk | LOW | logger sink |
| L9 | protocol server starting | `internal/cli/serve.go:125` | `kitlog.Info` | listen addr, health URL | curated; low | LOW | logger sink |
| L10 | protocol server error/shutdown | `internal/cli/serve.go:137,150` | `kitlog.Error` | net err from `server.Serve` | low | LOW | logger sink |
| L11 | adapter event publish failed | `internal/cli/adapter/events.go:21` | `kitlog.Warn` | topic name + error | event publish errors come from kit/runtime/bus; low | LOW | logger sink |
| L12 | acp auto-enable / start | `internal/cli/acp/server.go:47,56,81,91` | `kitlog.Info` | profile ID, transport, signal, protocol | curated; low | LOW | logger sink |
| L13 | session ended with error | `internal/cli/root.go:108` | `kitlog.Error` | run error | the error is returned from `core.Run*`/`a2a` paths and can wrap the action stdout/stderr or the inbound webhook body — see L2 chain | MEDIUM | logger sink |
| L14 | command failed | `internal/cli/root.go:120` | `kitlog.Error` | unknown error from cobra command | error strings vary by command; the `aps run` path produces `failed to setup environment: %w` whose wrapped err can include a secrets-file path but not the secret values; the `aps a2a tasks send` path can wrap a JSON-RPC error with peer-supplied detail | MEDIUM | logger sink |
| L15 | unknown command or profile | `internal/cli/root.go:126,129` | `kitlog.Error` | command args | could include a bare token if a user pastes `aps <token>` by mistake | MEDIUM | logger sink |
| L16 | agentprotocol handlers | `internal/adapters/agentprotocol/handlers.go:106,114,120` | `kitlog.Error` | profile ID, action ID, run-input field name, error | the run input payload is not directly logged — only the field name and the wrapped error; risk depends on what the core layer puts in the error string | MEDIUM | logger sink |
| O1 | `aps env` export | `internal/cli/env.go:25` | stdout (`fmt.Println`) | `export KEY=VALUE` lines from `capability.GenerateEnvExports()` — these are environment variables for capabilities, not the secrets file values, but capabilities can resolve secret-bearing env (see `capability.GenerateEnvExports` in `internal/core/capability/`) | HIGH if a capability env-resolver pulls a secret; the command's stated purpose is "eval $(aps env)" so values must be plain text | HIGH | output formatter at the `fmt.Println(export)` callsite |
| O2 | `aps run` child stdio | `internal/core/execution.go:248-250,313-314` | child `os.Stdout`/`os.Stderr` are wired directly to parent stdio | child process arbitrary output; the leak postmortem 2026-05-02 was an `OPENAI_API_KEY` echoed by a user script via `aps run noor -- env` | HIGH | parent stdio capture wrapper (per `aps run` invocation); requires `--no-redact` bypass |
| O3 | `aps adapter exec` output | `internal/cli/adapter/exec.go:72` | stdout (`fmt.Print(out)`) | result string from `mgr.ExecAction(...)` — the email/messenger adapters return arbitrary action output that may include the action's own debug strings, request bodies, response bodies | HIGH (action output is by definition external content) | output formatter at the `fmt.Print(out)` callsite |
| O4 | adapter subprocess stdout/stderr files | `internal/core/adapter/manager.go:201-214` | `device.Path/{stdout,stderr}.log` files (0644) | child process output — the child has been handed every secret in the profile's `secrets.env` via `cmd.Env` (see `manager.go:471-481` then through cxr ProcessHandler) | HIGH — log files are persisted on disk, world-readable per default umask | log-file writer wrap |
| O5 | a2a get-task printed messages | `internal/cli/a2a/get_task.go:91-101` | stdout | task message parts as text/file/data — these arrive over the wire from a peer agent and can carry tokens, prompts, file contents | HIGH | output formatter |
| O6 | a2a send-task printed status | `internal/cli/a2a/send_task.go:89-93` | stdout | task ID, last message ID, status — all peer-controlled strings | LOW for ID-shape values; MEDIUM if the peer chooses long IDs that embed text | LOW |
| O7 | a2a fetch-card printed body | `internal/cli/a2a/fetch_card.go:85-89` | stdout | URL, transport, description from peer's well-known card | description is peer-controlled free text; could embed anything | MEDIUM | output formatter |
| O8 | session inspect | `internal/cli/session/inspect.go:69-74` | stdout (tabwriter) | full `sess.Environment` map iterated key/value | HIGH — environment can include `OPENAI_API_KEY`, `GITHUB_TOKEN`, anything propagated by `buildEnvVars`. This is the closest analog to the 2026-05-02 leak surface (the user did `aps run noor -- env` which is just O2; `aps session inspect` is the persisted-state equivalent) | HIGH | output formatter at the `fmt.Fprintf(w, ...)` for environment rows |
| O9 | session inspect JSON | `internal/cli/session/inspect.go:85-87` | stdout | `json.Marshal(sess)` includes `Environment` field (`registry.go:102`) | HIGH — same surface as O8, JSON shape | HIGH | output formatter (`json.Marshal` value) |
| O10 | profile show env-var key list | `internal/cli/profile.go:376-381` | stdout | bundle resolved env-var KEYS (no values) | LOW — keys only; allowlisted | LOW (no redact needed) |
| W1 | webhook 500 response | `internal/core/webhook.go:144-148` | HTTP 500 response body (JSON) | `err.Error()` from `RunAction(...)` — chained from L2; sent to the webhook caller, which may persist it | HIGH (peer log retention) | HTTP response body formatter (`respondJSON` value) |
| W2 | webhook 400 response | `internal/core/webhook.go:92-96` | HTTP 400 response body (JSON) | event name from `X-APS-Event` header — attacker-chosen string but not secret unless attacker is also victim | LOW | HTTP response body formatter |
| W3 | agentprotocol error responses | `internal/adapters/agentprotocol/handlers.go:108,116,121` | `a.sendError(w, ..., err.Error())` — HTTP response to AP client | wrapped error from `core.ExecuteRun`; can echo action stdout/stderr (chain from L13) | MEDIUM-HIGH | HTTP response body formatter (`sendError` body) |
| H1 | webhook signature header | `internal/core/webhook.go:57-72` | not directly logged today, but the bytes are in `r.Header` and will land on a logger if anyone adds `r.Header` to a log line in this file | the signature is HMAC-SHA256 of the body; not a secret per se but reveals that `config.Secret` value (the shared HMAC key) was used | LOW | logger sink (defensive — protects against future regressions) |
| H2 | adapter mobile JWT | `internal/core/adapter/mobile/server.go:366-372` | not directly logged today — extracted, validated, never written to a stream | future regression risk: a future "log auth failures with reason" change would echo the bearer token | LOW | logger sink (defensive) |
| H3 | a2a transport API key | `internal/a2a/transport/auth.go:91`, `internal/a2a/transport/http.go:100` | sent to peer in `X-API-Key` header; if the kit/cli HTTP middleware logs request headers, the key lands there | depends on kit/cli — `RequestID` middleware does not log headers; but `Recovery` panics could surface them in stack traces | LOW | logger sink |
| H4 | bus auth token | `internal/cli/bus.go:52-57` | only logged when ABSENT (`warn: bus auth: BUS_TOKEN... not set`) | LOW (only the absence is logged) | LOW | (no redact needed; defensive logger sink covers it) |

## Inventory — chains worth calling out

- **L2 → W1 chain** (webhook): action stdout/stderr → `RunAction` error
  string → `kitlog.Error` (line 142) AND HTTP 500 body (line 144-148).
  Same content fans out to two sinks. Redacting at the logger only
  fixes half the problem; webhook caller still gets the raw bytes
  unless `respondJSON` filters its `data any` argument.

- **O2 / O8 / O9 chain** (env-bearing surfaces): `aps run` lets the
  child process see secrets and write them to parent stdio.
  `aps session inspect` reads `Environment` from the persisted session
  registry, which was populated from the same `buildEnvVars` source.
  These are three views of the same underlying data class
  (profile-resolved env). Redacting all three needs the same patterns;
  a single `redact.Default()` instance covers it.

- **O3 chain** (adapter exec): `mgr.ExecAction(...)` returns whatever
  the script wrote to stdout. The script has all the profile env in
  scope. `fmt.Print(out)` ships it to the user's terminal and to any
  shell redirect (`> out.log`).

- **O4 chain** (adapter subprocess persisted logs): the leak window
  here is unbounded — files persist with `0644` perms (world-readable
  on most umask configs) until manually cleaned. Higher consequence
  than O2/O3 because retention is involuntary.

## Inventory — Excluded (NOT a leak surface)

- **X1** `internal/cli/profile.go:245` — `fmt.Printf("  - %s: ***redacted***\n", k)`
  already redacts the value side; only the key prints, which is the
  intended audit signal.
- **X2** `internal/cli/skill/skill.go:*` — skill metadata (name,
  description, scripts list). No secret class involved.
- **X3** `internal/cli/identity/*.go` — DID + badge content. The DID
  is a public identifier and the badge is a verifiable credential, not
  a secret. Key path is printed (line 39 of `show.go`) but that is the
  filesystem path, not the key material.
- **X4** `internal/cli/version.go` — version strings only.
- **X5** `internal/cli/adapter/create.go`, `start.go`, `pair.go`,
  `approve.go`, `presence_cmd.go`, `test_messenger.go` — adapter
  metadata, names, types, Q codes; no secrets in the printed values
  (note: the `Set it: aps secrets set %s "your-token"` lines print
  the env-var NAME, not the value).
- **X6** Bus `warn:` lines in `internal/cli/bus.go` — log token
  ABSENCE, never the value.
- **X7** `internal/cli/env.go:20` `# Error generating envs: %v` — the
  error wraps a config-load failure; values are not in scope.

## Summary by severity

| Severity | Count | Surfaces |
|----------|-------|----------|
| HIGH     | 7     | L2, O1, O2, O3, O4, O5, O8, O9, W1 (some chained) |
| MEDIUM   | 8     | L1, L6, L13, L14, L15, L16, O7, W3 |
| LOW      | 13    | L3-L5, L7-L12, O6, O10, W2, H1-H4 |

Counts by sink:

- Logger callsites flagged: 16
- Stdout/stderr direct writes flagged: 10
- HTTP response bodies flagged: 3
- Persisted log files flagged: 1

## Choke points (to be implemented in T-0460)

The inventory groups around **four** boundary points where a single
wrap covers most surfaces:

1. **kit logger writer** — wrap `os.Stderr` with a redacting
   `io.Writer` inside `internal/logging/logger.go::SetViper` so every
   `logging.GetLogger().X(...)` line passes through `redact.Apply`.
   Covers L1-L16 (16 sites) and H1-H4 defensively.

2. **stdout output formatter** — provide a thin
   `internal/logging/output.go` helper (`Print`, `Println`, `Printf`,
   `Fprint*`) that applies `redact.Apply` to its formatted string,
   honoring the global `--no-redact` toggle. Migrate the **HIGH**
   stdout sites (O1, O2, O3, O5, O8, O9). Leave LOW sites
   (`identity show`, `version`, `profile show`) on raw `fmt.*` since
   they print curated data.

3. **HTTP response body** — wrap `respondJSON` (`webhook.go:180`) and
   `sendError`/`sendJSON` (agentprotocol handlers) so the JSON body
   flows through `redact.ApplyBytes` before `w.Write`. Covers W1, W3.

4. **persisted log-file writer** — wrap the `os.OpenFile` for
   `stdout.log` / `stderr.log` in `manager.go::startSubprocess` with
   a redacting writer. Covers O4.

`aps run`'s parent stdio path (O2) is special: the child process owns
its own stdout/stderr; aps cannot retrofit a writer without copying
through a pipe. Phase 2 introduces the wrapper at the CLI layer
(`internal/cli/run.go`) by setting `cmd.Stdout` / `cmd.Stderr` to a
redacting writer over the real terminal, with the `--no-redact` flag
short-circuiting back to bare `os.Stdout` / `os.Stderr`.

## Default-vs-bypass policy

- **Default**: redact ON, `Tag` strategy
  (`<openai-api-key>`, `<bearer-token>`, …) — diagnosable, value-free.
- **Bypass**: `--no-redact` flag (root command, persistent) and
  `APS_DEBUG_NO_REDACT=1` env. Both must be explicit; never default.
- **Inversion**: a viper key `redact.enabled` (default true) lets
  ops force-on at the config layer even if the user passes
  `--no-redact`. Standard kit pattern.

## kit/core/redact gaps observed

None blocking. Everything the inventory needs is supported by today's
kit/core/redact API:

- `redact.Default()` — gitleaks + Presidio defaults: covers
  `OPENAI_API_KEY`, `sk-…`, `AKIA…`, `gh[ps]_…`, JWT, x-api-key.
- `Redactor.Apply(string)` / `ApplyBytes([]byte)` — covers
  io.Writer wrap and JSON body wrap.
- `Replacement.Tag` — diagnosable output without leaking the value.
- `Redactor.Allow(...)` — pass-through for known-safe placeholders
  (`sk-test`, `AKIA…EXAMPLE`).

Key-aware redaction (the e2e requirement: `Authorization: Bearer xyz`
must redact `xyz` while keeping the key) is satisfied by the
`bearer-token` + `jwt` patterns in the gitleaks corpus, which match
the *value* shape regardless of the surrounding key. No structural
key-aware feature needed in kit; the regex match catches the value.

The README does flag a perf caveat (~44ms per 4KB clean line vs the
50µs design budget). For aps's traffic shape (low-volume CLI output,
infrequent webhook events) this is fine. We do NOT wire redact onto
the high-frequency hot paths (e.g. session reaper tick, adapter
log-file `cmd.Wait()` loop) without first measuring; the reaper logs
an integer count, so redact cost there is N/A.
