# Adapters: Technical & Integration Guide

Adapters are the integration points that connect APS profiles to external systems. This document covers the internal architecture, supported types, and integration patterns.

## 1. Technical Reference

### Adapter Types
APS categorizes adapters into functional types. These types determine default behaviors and UI grouping (e.g., `aps messenger` alias).

| Type | Key | Description |
| :--- | :--- | :--- |
| **Messenger** | `messenger` | Messaging platforms (Telegram, Discord, Slack, etc.). |
| **Protocol** | `protocol` | Agent communication protocols (A2A, ACP, Webhooks). |
| **Actuator** | `actuator` | Output/Action triggers (Hardware, Timers, local CLIs). |
| **Sense** | `sense` | Input/Sensor data (Camera, Microphone, Web scrapers). |
| **Mobile** | `mobile` | Mobile client pairing via QR and WebSockets. |
| **Desktop** | `desktop` | Native desktop application integration. |

### Loading Strategies
Strategies define how APS executes or interacts with the adapter backend.

| Strategy | Description | Persistence |
| :--- | :--- | :--- |
| **Subprocess** | Runs a standalone binary as a managed child process. | Persistent |
| **Script** | Executes a shell/python/node script on demand per action. | Ephemeral |
| **Built-in** | Native Go implementation compiled into the APS binary. | Persistent |

### Script Execution Environment
When using the `script` strategy, APS injects the following environment variables. **Note:** Currently, these use a legacy `EMAIL` namespace regardless of the adapter type.

| Env Var | Value Source |
| :--- | :--- |
| `APS_EMAIL_FROM` | `email` field from `profile.yaml` |
| `APS_EMAIL_ACCOUNT` | `config.account` from the adapter manifest |
| `EMAIL_<INPUT_NAME>` | Action inputs (e.g., `--input id=123` -> `EMAIL_ID`) |

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
```

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
