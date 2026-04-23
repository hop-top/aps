# Data Model

## Profile (`profile.yaml`)

Stored at `~/.agents/profiles/<id>/profile.yaml`.

```yaml
id: string          # Unique identifier (matches directory name)
display_name: string # Human-readable name

persona:            # Optional: Shaping AI behavior
  tone: string
  style: string
  risk: string

capabilities:       # Optional: List of allowed tools/scopes
  - string

accounts:           # Optional: Identity mapping
  [service: string]: 
    username: string

preferences:        # Optional: Runtime settings
  language: string
  timezone: string

limits:             # Optional: Resource constraints
  max_concurrency: int
  max_runtime_minutes: int

git:                # Optional: Git module config
  enabled: boolean

ssh:                # Optional: SSH module config
  enabled: boolean
  key_path: string

webhooks:           # Optional: Webhook config
  enabled: boolean
  allowed_events: [string]
```

## Action

Stored as executable files in `~/.agents/profiles/<id>/actions/` or defined in `actions.yaml`.

### Manifest (`actions.yaml`) - Optional

```yaml
actions:
  - id: string        # Unique ID within profile
    title: string     # Description
    entrypoint: string # Filename in actions/ dir
    accepts_stdin: bool
```

### Discovery Model

If `actions.yaml` is missing, files in `actions/` are scanned:
- `script.sh` -> ID: `script`
- `task.py` -> ID: `task`

## Environment Context

Injected into every execution (`aps run` or `aps action run`):

| Variable | Description |
| :--- | :--- |
| `AGENT_PROFILE_ID` | Profile ID |
| `AGENT_PROFILE_DIR` | Absolute path to profile root |
| `AGENT_PROFILE_YAML` | Absolute path to profile.yaml |
| `AGENT_PROFILE_SECRETS` | Absolute path to secrets.env |
| `AGENT_PROFILE_DOCS_DIR` | Absolute path to global docs |

## Webhook Event

Internal representation of a received hook.

```go
type WebhookEvent struct {
    ID        string            // Delivery ID (UUID or header)
    Type      string            // Event type (e.g. github.push)
    Payload   []byte            // Raw body
    Headers   map[string]string // Relevant headers
    ReceivedAt time.Time
}
```
