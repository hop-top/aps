# Adapters: Technical & Integration Guide

Adapters are the integration points that connect APS profiles to external systems. This document covers the internal architecture, supported types, and integration patterns.

## 1. Technical Reference

### Adapter Types
APS categorizes adapters into functional types. These types determine default behaviors and UI grouping (e.g., `aps messenger` alias).

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/adaptermd", "kinds-table"],
    text=True))
]]] -->
| Type | Key | Description |
| :--- | :--- | :--- |
| **Actuator** | `actuator` | Robotics, hardware |
| **Desktop** | `desktop` | Desktop applications |
| **Messenger** | `messenger` | Telegram, Slack, etc. |
| **Mobile** | `mobile` | Mobile devices (via QR linking) |
| **Protocol** | `protocol` | A2A, ACP, webhooks, WebSockets |
| **Scheduler** | `scheduler` | Calendars, schedulers, reminders |
| **Sense** | `sense` | Camera, microphone |
<!-- [[[end]]] -->

### Loading Strategies
Strategies define how APS executes or interacts with the adapter backend.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/adaptermd", "strategies-table"],
    text=True))
]]] -->
| Strategy | Description | Persistence |
| :--- | :--- | :--- |
| **Built-in** | Native Go implementation compiled into the APS binary. | Persistent |
| **Script** | Executes a shell/python/node script on demand per action. | Ephemeral |
| **Subprocess** | Runs a standalone binary as a managed child process. | Persistent |
<!-- [[[end]]] -->

### Which path remembers
The Persistence column above describes process lifetime (long-lived child vs. spawned per
action), not message memory. No strategy records what it handled. Whether a turn is kept
depends on the path the message takes:

| Path | Command shape | Records turns? | Where |
| :--- | :--- | :--- | :--- |
| Adapter device | `aps adapter exec <adapter> <action>` (email, contacts, timer, phone, ...) | No. Output is the backend's (himalaya, cardamum, comodoro); nothing stored, no inbox. | — |
| Message service | `aps service add --type <platform>` + `POST /services/<id>/webhook` via `aps serve` | Yes. Inbound turn on route, outbound turn after a delivered reply. | `<data-dir>/messages/conversations.db`; `aps service conversation list\|show` |

Email sits on both: `aps adapter exec email send` keeps nothing, while
`aps service add --type message --adapter email` records every routed turn. Identity keys,
store schema, retention, and the `prior_turns` stdin contract:
[message-conversation-policy.md](message-conversation-policy.md). User view:
[messengers.md#conversation-history](../user/messengers.md#conversation-history).

### Script Execution Environment
When using the `script` strategy, APS injects the following environment variables. The input prefix comes from the manifest's `env_prefix` field (`EMAIL` for the email adapter, `CAL` for calendar, `CONTACT` for contacts); manifests that omit `env_prefix` fall back to the default.

| Env Var | Value Source |
| :--- | :--- |
| `APS_EMAIL_FROM` | `email` field from `profile.yaml` |
| `APS_EMAIL_ACCOUNT` | `config.account` from the adapter manifest |
| `<PREFIX>_<INPUT_NAME>` | Action inputs (e.g. `--input id=123` on the email adapter -> `EMAIL_ID`) |

Input names are uppercased and hyphens become underscores, so `--input event-id=7` yields `CAL_EVENT_ID` on the calendar adapter.

The inputs that reach the script are the caller-supplied ones plus any manifest defaults the caller omitted — see [Input Contract](#input-contract) below.

---

## 2. Integration & Usage

### The Manifest (`manifest.yaml`)
Every adapter must have a manifest defining its actions.

```yaml
api_version: adapter.aps.dev/v1
kind: Adapter
name: my-adapter
type: actuator
strategy: script
config:
  backend: my-tool
  actions:
    - name: run
      script: backends/{{backend}}/run.sh
      input:
        - name: target
          required: true
        - name: retries
          required: false
          default: "3"
```

Each entry under an action's `input:` list accepts:

| Field | Meaning |
| :--- | :--- |
| `name` | Input key, as passed to `--input <name>=<value>`. |
| `required` | `true` rejects calls that omit the key. Defaults to `false` when absent. Must be an **unquoted** YAML boolean — see [Marker typing](#marker-typing). |
| `default` | Value used when the caller omits the key. Unquoted scalars (`default: 10`, `default: true`) are rendered as strings. |
| `description` | Human-readable note; not enforced. |

### Input Contract

The `input:` list is enforced, not merely descriptive. An adapter manifest written loosely against an earlier, permissive runtime may now be rejected — the rules below are what an action's declared inputs actually buy.

#### Required inputs

An input declared `required: true` must be present in the caller-supplied inputs. Enforcement runs **before** the script is spawned, so a rejected call has no side effects — the backend never executes.

The diagnostic names every missing input at once, in manifest order, so a wrong invocation is fixed in one pass rather than one round-trip per input:

```
missing required input 'to', 'subject', 'body' for action 'send': expected '--input to=<value>'
```

Presence, not emptiness, is the test: `--input to=` supplies the key with an empty value and satisfies `required: true`. Validation of the *value* is the backend script's job.

An action whose schema cannot be resolved — an unparseable manifest, or an action absent from the list — declares nothing and therefore rejects nothing.

#### Marker typing

`required:` must be an unquoted YAML boolean. `required: true` and `required: false` are the only two spellings that mean what they look like.

Quoting changes the type: `required: "true"` is the *string* `true`, not the boolean. So is `required: yes`, and `required: 1` is an int. None of these is a boolean, and a manifest that uses one is a manifest bug.

An unreadable marker resolves to **required**, and says so on stderr:

```
warn: adapter "email" manifest: action "send": input "to" has non-boolean required: true (string); treating the input as REQUIRED — write an unquoted `required: true` or `required: false`
```

The asymmetry is deliberate. Resolving an unreadable marker to *optional* would silently retire the requirement — the manifest still reads `required`, `aps` reports nothing, and the action runs with the input missing. Resolving it to *required* is loud: the next call that omits the input is rejected by name, which is the nudge that gets the manifest fixed.

This applies to `required: "false"` too. It is not special-cased into a string-to-boolean parse, because doing so would re-open the same hole for `required: "true"` in any manifest that quotes one marker but not the other. Write the marker unquoted or leave it out.

#### Duplicate input names

An input name declared more than once within one action collapses to the **first** declaration; later entries are dropped and the collapse is reported:

```
warn: adapter "email" manifest: action "list": input "folder" declared more than once; keeping the first declaration and ignoring the rest
```

Collapsing at parse time is what keeps the schema self-consistent — every reader of the action resolves a name to the same single entry, rather than each scanning the list on its own terms.

#### Defaults

A declared `default:` fills in an input the caller omitted. Merged defaults reach the script as ordinary `<PREFIX>_<KEY>` env vars, indistinguishable from caller-supplied ones.

Precedence is by **key presence**: a caller-supplied value always wins, including an explicitly empty one. `--input cc=` means "empty", not "use the default". Only a key the caller left out entirely is filled from the manifest. An empty `default:` declares nothing and is skipped.

A `default:` must be a scalar — string, number or boolean. Sequences and mappings cannot be rendered into a single env var, so they are dropped with a warning:

```
warn: adapter "email" manifest: action "list": input "folders" has a default of unsupported type []interface {}; dropping it — declare a string, number or boolean scalar
```

Unlike an unreadable `required` marker, a dropped default is not resolved in a stricter direction. It degrades to "input not supplied", which the script's own fallback or a `required` marker already covers — so it warns rather than failing the exec.

#### Ordering: defaults never satisfy `required`

Required-checking runs against the caller-supplied inputs, before defaults are merged. A `default:` on a `required: true` input therefore does not excuse its absence — the call is still rejected. Declaring both on one input is a manifest smell: pick one.

#### Undeclared inputs

Keys the manifest does not declare are still forwarded to the script — backend scripts may legitimately read env vars the manifest does not enumerate — but they are now reported on stderr, naming each key:

```
warn: action "send": undeclared input(s) bdy; not declared in manifest, forwarded to script anyway
```

This is advisory: the exit code is unchanged and the action still runs. It exists so a typo (`bdy=` for `body=`) surfaces instead of silently arriving as an input the operator believes was delivered. The warning goes to stderr, keeping stdout machine-parseable as action output.

An action that declares no inputs at all is exempt — with no declared vocabulary there is nothing to deviate from, so nothing is warned about.

#### Malformed `--input`

A `--input` token without `=`, or with an empty key, is a user error naming the offending token rather than a silently dropped argument:

```
invalid input format 'subject': expected 'key=value'
```

Splitting is on the **first** `=` only, so later separators stay in the value: `--input body=a=b` yields `body` = `a=b`.

#### Worked example

Given the email adapter's manifest (`adapters/email/manifest.yaml`), `send` declares `to`, `subject`, and `body` as required with `cc` optional, and `list` declares `limit` (default `"10"`) and `folder` (default `INBOX`), both optional. The adapter's `env_prefix` is `EMAIL`.

```bash
# Accepted. Defaults fill both omitted inputs.
aps adapter exec email list --profile noor
#   -> script sees EMAIL_LIMIT=10, EMAIL_FOLDER=INBOX

# Accepted. Caller-supplied folder wins; limit still defaults.
aps adapter exec email list --profile noor --input folder=Archive
#   -> script sees EMAIL_LIMIT=10, EMAIL_FOLDER=Archive

# Accepted. An explicitly empty value beats the default.
aps adapter exec email list --profile noor --input folder=
#   -> script sees EMAIL_LIMIT=10, EMAIL_FOLDER=

# Rejected before the script runs; no mail is sent.
aps adapter exec email send --profile noor --input to=user@example.com
#   -> missing required input 'subject', 'body' for action 'send': ...

# Accepted, with a warning. `bdy` is a typo for `body`, so `body` is
# still missing -- this call is rejected first, at the required check.
aps adapter exec email send --profile noor \
  --input to=user@example.com --input subject=Hi --input bdy=Hello
#   -> missing required input 'body' for action 'send': ...

# Accepted, with a warning. All required inputs present; the
# undeclared key is forwarded anyway.
aps adapter exec email send --profile noor \
  --input to=user@example.com --input subject=Hi --input body=Hello \
  --input priority=high
#   -> warn: action "send": undeclared input(s) priority; ...
#   -> script sees EMAIL_TO, EMAIL_SUBJECT, EMAIL_BODY, EMAIL_PRIORITY
```

Note the fifth case: because required-checking precedes the undeclared-key warning path in outcome terms, a typo that leaves a required input missing surfaces as a rejection, not just a warning.

### Execution Patterns

#### Generic Execution
All script adapters can be invoked via the generic adapter CLI:
```bash
aps adapter exec <adapter_name> <action> --profile <profile_id> --input <key>=<value>
```

#### Adapter Promotion (Top-Level Commands)
For frequently used adapters (like `contacts`), APS "promotes" them to top-level commands to provide a better UX.

**How to Promote:**
1. Define a script adapter in `APS_DATA_PATH/devices/`.
2. Create `internal/cli/<name>.go` in the APS source.
3. Register the command in the root: `rootCmd.AddCommand(new<Name>Cmd())`.
4. Delegate the logic to `coreadapter.NewManager().ExecAction()`.

### Discovery & Linking
*   **Discovery**: Adapters are discovered by scanning `APS_DATA_PATH/devices/`.
*   **Linking**: Use `aps adapter link <name> -p <profile>` to enable an adapter for a specific profile.
