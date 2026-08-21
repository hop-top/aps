# Message Routing: Sender Route Tables

A message service dispatches every inbound message to exactly one profile
action. The simple form is option `default_action`. When one number, bot, or
inbox serves many organizations, replace it with a `routing` block: a
sender-keyed route table with an optional contact snapshot and a mandatory
terminal fail-safe.

This document covers the schema, sender key normalization, contact
resolution, evaluation order, fail-safe semantics, and how a resolved route
executes. Conversation identity is defined separately in
[Message conversation and thread policy](message-conversation-policy.md).

## Declaring A Route Table

`aps service add` references two YAML files; nothing is copied into the
service record:

> `secret:NAME` resolves `NAME` from the profile secret store, then the
> environment. See [Credential Bindings](../user/messengers.md#credential-bindings).

```bash
aps service add support-line \
  --type whatsapp \
  --profile triage \
  --provider twilio \
  --from whatsapp:+15550100002 \
  --env TWILIO_ACCOUNT_SID=secret:twilio_sid \
  --env TWILIO_AUTH_TOKEN=secret:twilio_token \
  --route-table routes/support-line.yaml \
  --contacts contacts/support-line.yaml
```

Persisted service (`${XDG_DATA_HOME:-~/.local/share}/aps/services/support-line.yaml`):

```yaml
id: support-line
type: message
adapter: whatsapp
profile: triage
options:
  provider: twilio
  from: whatsapp:+15550100002
routing:
  file: routes/support-line.yaml
  contacts:
    path: contacts/support-line.yaml
```

Relative `file` and `contacts.path` values resolve against the services
directory; absolute paths and `~/` work as expected. A route table declared
in an external file may carry its own `contacts:` block, in which case a
relative contacts path resolves against that file's directory. Inline
`routes:` on the service record are also accepted; `file` and `routes`
together are rejected.

`aps service show support-line` prints the compiled table (or
`routing_error:` when it does not compile).

## Route Table Schema

`routes/support-line.yaml`:

```yaml
# Optional when the service record already declares contacts.
contacts:
  path: ../contacts/support-line.yaml   # relative to this file
  entries:                               # inline entries, appended after path
    - id: oncall
      org: aps
      keys: ["+15550100099"]

routes:
  - match: "+15551230000"        # exact sender key
    profile: vip
    action: concierge
  - match: org:acme              # resolved contact org (glob or exact)
    profile: acme
    action: inbox
  - match: contact:oncall        # resolved contact id
    action: escalate             # profile omitted -> service profile
  - match: "+1555*"              # glob on sender key
    profile: sales
    action: inbox
  - match: unknown               # terminal fail-safe, must be last
    profile: triage
    action: triage
```

| Field | Rule |
| --- | --- |
| `match` | Required. `unknown`, `org:<pattern>`, `contact:<pattern>`, or a sender-key pattern. |
| `profile` | Optional. Defaults to the service profile. |
| `action` | Required plain action name in the target profile. `profile=action` forms are rejected; set `profile` separately. |

A pattern containing `*`, `?`, or `[` is a glob with `path.Match` semantics
(`*` does not cross `/`); anything else compares exactly. `org:` and
`contact:` selectors are case-folded on both sides and only ever match when a
contacts source resolved the sender; declaring one without a contacts source
is a load error.

## Contacts Snapshot

`contacts/support-line.yaml`:

```yaml
contacts:
  - id: jane
    name: Jane Doe
    org: acme
    keys:
      - "whatsapp:+1 (555) 123-4567"   # Twilio form
      - "15551234567"                   # WhatsApp Cloud wa_id
      - "jane@acme.com"
  - id: bob
    org: globex
    keys: ["+15559876543"]
```

Keys are normalized exactly like inbound senders (below), so any provider
form works. Load rejects missing ids, duplicate ids (case-insensitive),
contacts without a usable key, and a key owned by two contacts.

The contact source is an interface (`msgroute.ContactSource`); the YAML
snapshot is the first-class implementation. Address-book adapters can back
the same interface later without touching route evaluation.

## Sender Key Normalization

Both inbound `sender.id` and table patterns/keys pass through the same
function, parameterized by the service adapter (platform):

1. Trim whitespace; strip one leading `whatsapp:`, `sms:`, `tel:`, or
   `mailto:` prefix (case-insensitive).
2. Values containing `@` are lowercased (`Jane@Acme.com` -> `jane@acme.com`).
3. Phone-shaped values (optional `+`, digits, spaces, dashes, dots,
   parentheses) lose their separators. On `sms` and `whatsapp` services, or
   when a phone prefix was stripped, a missing leading `+` is added so Twilio
   `whatsapp:+15551234567` and WhatsApp Cloud `15551234567` share the key
   `+15551234567`.
4. Everything else (Slack `U012ABC`, Telegram `1001`, Discord snowflakes,
   alphanumeric SMS sender ids) is kept verbatim.

Glob patterns keep their metacharacters: `whatsapp:+1 555 *` compiles to
`+1555*`; a pattern starting with a wildcard never gains a `+`.

## Evaluation Order

For each inbound message on a service with a `routing` block:

1. Explicit legacy channel mappings (`messenger-links.json`) still win when
   one matches the channel.
2. The route table is compiled from the referenced files, normalizes
   `sender.id`, and looks the key up in the contacts snapshot.
3. Routes are walked **in declared order; the first match wins**. There is
   no exact-before-glob reordering: put narrow routes above broad ones.
4. The terminal `match: unknown` route matches whatever reached it.

The decision is stamped on the message as `platform_metadata.routing` and
therefore reaches the action payload:

```json
{
  "sender_key": "+15551234567",
  "match": "org:acme",
  "route": 1,
  "terminal": false,
  "profile": "acme",
  "action": "inbox",
  "contact": {"id": "jane", "name": "Jane Doe", "org": "acme"}
}
```

`contact` is absent for unknown senders; `terminal: true` tells the triage
action it received an unrouted sender.

## Fail-Safe Semantics

- Every table must end with `match: unknown`. Load rejects tables without
  it, tables where it is not last (unreachable routes follow), and tables
  with more than one.
- A table that fails to load at message time (missing file, malformed YAML,
  validation error) is a **routing failure**: the webhook answers with an
  error and nothing executes. It never falls back to `default_action` or to
  the service profile silently.
- `aps service add` reports every problem in the table at once, one
  `config_issue: routing: ...` line each; `aps service show` prints a single
  `routing_error:` line for a table that does not compile.

## Relationship To `default_action`

| Config | Behavior |
| --- | --- |
| `options.default_action` only | Unchanged: every message goes to that action in the service profile (or the explicit `profile=action`). |
| `routing` only | Route table decides profile and action. |
| both | Validation issue (`config_valid: false`). At runtime the route table wins; `default_action` is ignored. Remove it. |
| neither | Validation issue; messages report `unknown_channel`. |

`--contacts` without `--route-table` is a CLI error.

## Execution And Isolation

A route resolves to a `profile=action` mapping exactly like a channel
mapping. `MessageRouter` stamps `message.profile_id`, and execution goes
through the same profile-scoped `ExecuteRun(RunInput{ProfileID, ActionID})`
path that `aps action run` uses: the target profile is loaded, its action is
resolved inside that profile, and the action process runs with that
profile's environment and isolation. Chat-mode services (`execution: chat`)
receive the resolved profile in the execution handoff the same way. No shell
re-invocation of `aps` is involved.

## Related

- [Messenger architecture](messenger-architecture.md)
- [Message conversation and thread policy](message-conversation-policy.md)
- [User messenger guide](../user/messengers.md)
