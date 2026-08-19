# Ticket Services

Ticket services receive work-item events (email, Jira, Linear, GitLab) and
route them to a profile action. They are the inbox-shaped sibling of
[message services](messengers.md): same `aps service` grammar, different
payload shape and route.

## Route

Every ticket service is mounted by `aps serve` (and `aps service start`) at:

```text
POST /services/<service-id>/ticket/<adapter>
```

`aps service status <id>` and `aps service routes <id>` print the same path;
`aps service test <id> --probe` posts a synthetic payload to it.

| Adapter alias | Canonical service | Inbound payload | Maturity |
| --- | --- | --- | --- |
| `email` | `type: ticket`, `adapter: email` | flat email JSON from a relay or poller | ready |
| `jira` | `type: ticket`, `adapter: jira` | Jira issue/comment webhook JSON | ready |
| `linear` | `type: ticket`, `adapter: linear` | Linear issue/comment webhook JSON | ready |
| `gitlab` | `type: ticket`, `adapter: gitlab` | GitLab issue/MR/note webhook JSON | ready |
| `github` | `type: ticket`, `adapter: github` | route mounted; payloads rejected until a normalizer lands | component |

There is no service-less catch-all: a POST to a ticket route whose service
does not exist is `404`, a service of another type is `400`, and an adapter
that does not match the persisted service is `400`.

## Create An Email Ticket Service

```bash
aps profile create inbox

aps service add support-inbox \
  --type email \
  --profile inbox \
  --default-action triage \
  --reply status

aps service show support-inbox
aps service routes support-inbox
aps service status support-inbox --base-url https://hooks.example.com
aps service test support-inbox
```

`--type email` resolves to `type: ticket`, `adapter: email`. The persisted
record lives at the path `aps service add` prints; edit it to set options
that have no flag yet.

### Lock The Route Down

Ticket routes reuse the generic service auth every message service has.
Set one of these options on the service record:

```yaml
options:
  default_action: triage
  # bearer token (Authorization: Bearer <token>); auth_scheme defaults to bearer
  auth_token_env: SUPPORT_INBOX_TOKEN
  # or: HMAC-SHA256 of the raw body in X-APS-Signature (sha256=<hex>)
  # signature_secret_env: SUPPORT_INBOX_SECRET
  # optional replay/timestamp hardening
  # require_timestamp: "true"      # X-APS-Timestamp, 5m tolerance
  # require_replay_check: "true"   # X-APS-Delivery-ID, rejected when seen
  # who may open tickets (exact addresses, IDs, handles, or globs)
  allowed_senders: "*@corp.example, vip@example.com"
```

| Check | Option(s) | Failure |
| --- | --- | --- |
| Request auth | `auth_token` / `auth_token_env` (bearer or `auth_scheme: token` + `auth_header`), `signature_secret` / `signature_secret_env` (HMAC-SHA256), `require_timestamp`, `require_replay_check` | `401` |
| Sender allowlist | `allowed_senders` (CSV; `*`/`?` globs; matched against author email, ID, handle) | `403` |
| Dispatch target | `default_action` or a `routing:` table | `422` when neither claims the ticket |

`aps service test` warns when neither request auth nor `allowed_senders` is
set. `aps service test --probe` sends the configured bearer/token/HMAC
headers and, for email, posts from the first literal `allowed_senders` entry
so a locked-down service accepts its own probe.

## Email Payload

Post one JSON document per message. The same flat shape the messenger email
normalizer accepts works here, so a relay or IMAP poller can target either
service type:

```json
{
  "message_id": "<abc@example.com>",
  "in_reply_to": "<root@example.com>",
  "references": "<root@example.com>",
  "from": "Alice Example <alice@example.com>",
  "to": "support@example.com",
  "cc": "ops@example.com",
  "subject": "Re: Cannot log in",
  "body": "Still broken.",
  "date": "Tue, 19 Aug 2026 10:00:00 +0000",
  "labels": ["inbox"],
  "attachments": [{"type": "file", "url": "https://files.example/1", "mime_type": "application/pdf"}]
}
```

Normalization:

| Ticket field | From |
| --- | --- |
| `id` | `message_id` (synthesized when absent) |
| `kind` | `comment` when `in_reply_to`/`references`/`thread_id` is set, else `issue` |
| `channel_id` | recipient mailbox (`to`, lowercased address) |
| `thread_id` | `thread_id`, else `in_reply_to`, else first `references` entry, else the message itself |
| `title` / `body` | `subject` / `body` (or `text`) |
| `author` | `from` parsed as `Name <addr>`; `id`, `email`, `handle` are the lowercased address |
| `created_at` | `date` (RFC 3339 or RFC 5322), else receive time |
| `metadata` | `raw`, `mailbox`, `to`, `cc`, `references`, `attachments`, ... |

Jira, Linear, and GitLab payloads are the providers' own webhook bodies.

## What The Action Receives

The routed action reads one JSON document on stdin: the normalized ticket
fields at the top level plus `service_id` and `route_key`
(`<channel_id>#<thread_id>`). Its stdout is the reply body.

```json
{
  "id": "<abc@example.com>",
  "adapter": "email",
  "kind": "comment",
  "channel_id": "support@example.com",
  "thread_id": "<root@example.com>",
  "title": "Re: Cannot log in",
  "body": "Still broken.",
  "author": {"id": "alice@example.com", "name": "Alice Example", "email": "alice@example.com"},
  "service_id": "support-inbox",
  "route_key": "support@example.com#<root@example.com>",
  "metadata": {"raw": {"...": "..."}}
}
```

The webhook response is status metadata plus the action output:

```json
{"status": "success", "body": "<action stdout>", "ticket_id": "<abc@example.com>", "thread_id": "<root@example.com>", "target": "email_reply", "timestamp": "..."}
```

`status` is `failed` when the action exited non-zero; the HTTP status stays
`200` because the ticket was received and dispatched. `500` means the action
could not be started (unknown profile/action, executor error).

## Route By Sender

One service has one `--default-action`. To dispatch by author instead, give
the service a route table (and optionally a contacts snapshot), exactly as
for message services; the author's email address is the sender key:

```bash
aps service add support-inbox \
  --type email \
  --profile inbox \
  --route-table routes/support-inbox.yaml \
  --contacts contacts/support-inbox.yaml
```

```yaml
# routes/support-inbox.yaml
routes:
  - match: "*@vip.example"
    profile: concierge
    action: escalate
  - match: org:acme
    action: acme-triage
  - match: unknown
    action: triage
```

The decision is stamped on the ticket as `metadata.routing`. Schema and rules:
[Message routing](../dev/message-routing.md).

## Feeding Email In

Anything that can turn a mailbox into HTTP POSTs works: a provider inbound
webhook (Mailgun/SendGrid/Postmark style relays mapped to the flat shape
above), or a poller that emits one JSON event per new message:

```bash
# poller emits one flat email JSON object per line on stdout
your-imap-poller --mailbox support@example.com \
| while read -r event; do
    curl -fsS -X POST "$APS_BASE_URL/services/support-inbox/ticket/email" \
      -H "Authorization: Bearer $SUPPORT_INBOX_TOKEN" \
      -H "Content-Type: application/json" \
      --data "$event"
  done
```

Post to the ticket route directly; there is no need to reshape mail into a
messenger-style event for a message service.

## Testing

```bash
aps service test support-inbox                 # validate config, print route
aps service test support-inbox --probe \
  --base-url https://hooks.example.com         # POST synthetic email, print status
aps service status support-inbox               # last inbound/outbound event
```

Inbound and outbound events (status, sender, ticket ID) are recorded on the
service record; `aps service status` shows the latest of each.
