# How APS Implements 12-Factor Agents

This document maps the [12-Factor Agents methodology](12-factor-agents/) to APS's implementation.

---

## Overview

APS (Agent Profile System) provides a complete runtime implementation of 12-Factor Agent principles. This document shows exactly how each factor is implemented in APS.

---

## Factor 1: Natural Language to Tool Calls

**Principle:** Convert natural language to structured tool calls.

**APS Implementation:**

### A2A Protocol Integration
```bash
# Natural language task → Structured A2A message
aps myagent a2a task create \
  --agent "payment-processor" \
  --description "Create payment link for $750 to Terri for February AI Tinkerers meetup"
```

**Under the hood:**
```json
{
  "type": "task",
  "id": "task_123",
  "agent": "payment-processor",
  "message": {
    "role": "user",
    "content": "Create payment link for $750 to Terri..."
  },
  "context": {
    "profile": "myagent",
    "isolation": "platform"
  }
}
```

**Files:**
- `internal/a2a/client.go` - A2A client implementation
- `internal/a2a/server.go` - A2A server implementation
- `cmd/a2a.go` - CLI commands for A2A

**Docs:**
- [A2A Protocol Spec](../specs/005-a2a-protocol/spec.md)

---

## Factor 2: Own Your Prompts

**Principle:** Don't outsource prompt engineering to frameworks. Treat prompts as first-class code.

**APS Implementation:**

### Profile Actions
```yaml
# ~/.local/share/aps/profiles/myagent/profile.yaml
actions:
  triage-issue:
    command: |
      gh issue view {{.issue_number}} | \
      analyze-issue | \
      gh issue comment {{.issue_number}} --body-file -
    env:
      ANALYZE_PROMPT: |
        You are a helpful GitHub issue triage assistant.
        Analyze this issue and provide:
        1. Issue type (bug, feature, question)
        2. Priority (high, medium, low)
        3. Recommended assignee
        4. Suggested labels
```

**Key Point:** Prompts are stored in `profile.yaml` and versioned with your profile. You own the exact text, not a framework.

**Files:**
- `<data>/profiles/*/profile.yaml` - Profile configuration
- `internal/profile/action.go` - Action execution logic

**Docs:**
- [Profile Configuration](../user/PROFILES.md)

---

## Factor 3: Own Your Context Window

**Principle:** Manage context explicitly, don't rely on implicit framework behavior.

**APS Implementation:**

### Environment Variables
```yaml
# ~/.local/share/aps/profiles/myagent/profile.yaml
env:
  CONTEXT_DOCS: "/path/to/docs"
  PROJECT_ROOT: "/workspace/myproject"

# Auto-injected by APS
# APS_PROFILE_ID=myagent
# APS_PROFILE_PATH=~/.local/share/aps/profiles/myagent
# APS_SESSION_ID=sess_abc123
```

### Explicit Context Mounting (Container Isolation)
```yaml
isolation:
  level: container
  container:
    volumes:
      - source: /workspace/docs
        target: /context/docs
        readonly: true
      - source: /workspace/code
        target: /context/code
```

**Files:**
- `internal/isolation/container/volume.go` - Volume mounting
- `internal/profile/env.go` - Environment variable injection

**Docs:**
- [Isolation Levels](../user/ISOLATION.md)

---

## Factor 4: Tools are Structured Outputs

**Principle:** Tool calls are just structured outputs that deterministic code can execute.

**APS Implementation:**

### Actions Return Structured Data
```yaml
# Action outputs JSON for downstream processing
actions:
  analyze-logs:
    command: |
      tail -n 100 app.log | \
      analyze-errors --format json
    output_format: json

# Downstream action consumes structured output
actions:
  alert-on-errors:
    depends_on: analyze-logs
    command: |
      jq -r '.critical_errors[]' | \
      send-alert
```

### A2A Task Results
```bash
# Task execution returns structured result
aps myagent a2a task get task_123
```

```json
{
  "id": "task_123",
  "status": "completed",
  "result": {
    "payment_link": "https://buy.stripe.com/...",
    "amount": 750,
    "recipient": "terri@example.com"
  }
}
```

**Files:**
- `internal/a2a/task.go` - Structured task results
- `internal/profile/action.go` - Action result handling

---

## Factor 5: Unify Execution State

**Principle:** Single source of truth for execution state.

**APS Implementation:**

### Session Registry
```json
// ~/.aps/sessions/registry.json
{
  "sessions": {
    "sess_abc123": {
      "id": "sess_abc123",
      "profile_id": "myagent",
      "status": "active",
      "pid": 12345,
      "started_at": "2026-02-01T10:00:00Z",
      "last_heartbeat": "2026-02-01T10:05:00Z",
      "isolation_level": "platform",
      "metadata": {
        "tier": "premium",
        "task_type": "webhook"
      }
    }
  }
}
```

**Commands:**
```bash
# Query unified state
aps session list
aps session inspect sess_abc123
aps session logs sess_abc123
```

**Files:**
- `internal/session/registry.go` - Session registry implementation
- `internal/session/store.go` - Persistent state storage
- `~/.aps/sessions/registry.json` - Single source of truth

**Docs:**
- [Session Management](../user/SESSIONS.md)

---

## Factor 6: Launch, Pause, Resume

**Principle:** Agents should support full lifecycle management.

**APS Implementation:**

### Launch
```bash
# Launch long-running session
aps myagent run-detached -- process-queue
```

### Attach (Resume)
```bash
# Attach to running session
aps session attach sess_abc123

# For platform/container isolation, SSH connection is established
# Connection to myagent-sandbox (macOS/Linux)
# Connection to container sess_abc123 (Docker)
```

### Detach (Pause)
```bash
# Detach from session (keeps running)
aps session detach sess_abc123
```

### Terminate
```bash
# Gracefully terminate session
aps session terminate sess_abc123
```

**Under the hood:**
- **Process isolation:** Direct shell fork/exec
- **Platform isolation:** SSH to sandbox user account + tmux
- **Container isolation:** SSH to container + tmux

**Files:**
- `internal/session/lifecycle.go` - Lifecycle management
- `internal/session/attach.go` - SSH-based attachment
- `internal/isolation/*/session.go` - Isolation-specific session handling

**Docs:**
- [Session Management](../user/SESSIONS.md)
- [Platform Isolation Sessions](../dev/platforms/macos/overview.md#sessions)

---

## Factor 7: Contact Humans with Tools

**Principle:** Use tool calls for human interaction, not separate communication channels.

**APS Implementation:**

### Webhook Actions
```yaml
# ~/.local/share/aps/profiles/myagent/profile.yaml
actions:
  request-approval:
    trigger: webhook
    command: |
      echo "Approval requested: {{.payload.description}}" | \
      send-notification --channel slack --wait-for-response
    approval_required: true
```

### Webhook Integration
```bash
# Configure webhook endpoint
aps webhook register myagent request-approval \
  --url https://example.com/webhook/myagent/request-approval \
  --secret mysecret
```

**Flow:**
1. External system POSTs to webhook
2. APS validates signature
3. APS triggers action with `approval_required: true`
4. Action sends notification to human
5. Human responds via Slack/email
6. Action continues with approval result

**Files:**
- `internal/webhook/handler.go` - Webhook handling
- `internal/profile/action.go` - Action approval flow
- `cmd/webhook.go` - Webhook CLI commands

**Docs:**
- [Webhooks Guide](../user/WEBHOOKS.md)

---

## Factor 8: Own Your Control Flow

**Principle:** Explicit control flow, avoid framework magic.

**APS Implementation:**

### Profile Actions Define Flow
```yaml
# ~/.local/share/aps/profiles/myagent/profile.yaml
actions:
  deploy:
    command: |
      # Explicit control flow in bash
      if [ "$ENVIRONMENT" = "production" ]; then
        aps myagent action run request-approval --env ENVIRONMENT=production
        if [ $? -ne 0 ]; then
          echo "Approval denied, aborting deployment"
          exit 1
        fi
      fi

      # Check deployment status
      check-deployment-status || exit 1

      # Deploy
      deploy-app --env $ENVIRONMENT
    env:
      ENVIRONMENT: staging
```

**Key Point:** Control flow is YOUR code (bash, Python, etc.), not framework abstractions.

**Files:**
- `internal/profile/action.go` - Action execution (executes YOUR commands)

---

## Factor 9: Compact Errors into Context Window

**Principle:** When tool calls fail, add errors to the context window for self-healing. The LLM can read the error and retry with corrections.

**APS Implementation:**

### Error Recovery in A2A Tasks
```yaml
# Profile action with error recovery
actions:
  process-with-retry:
    command: |
      #!/bin/bash
      attempt=0
      max_attempts=3

      while [ $attempt -lt $max_attempts ]; do
        # Try the operation
        if process-data input.json > output.json 2> error.log; then
          echo "Success on attempt $((attempt + 1))"
          exit 0
        fi

        # Add error to context for next attempt
        attempt=$((attempt + 1))
        if [ $attempt -lt $max_attempts ]; then
          echo "Attempt $attempt failed. Error context:"
          cat error.log
          # Error is now in session logs (context window)
          sleep 2
        fi
      done

      echo "Failed after $max_attempts attempts"
      exit 1
```

### A2A Task Error Context
```bash
# Agent receives task
aps myagent a2a task create \
  --agent "data-processor" \
  --description "Process invoice data"

# Task fails, error added to conversation context
# Agent can retry with corrections based on error message
```

**Under the hood (session logs):**
```
2026-02-01 10:00:00 [ATTEMPT 1] Processing invoice data
2026-02-01 10:00:01 [ERROR] Missing required field: customer_id
2026-02-01 10:00:02 [ATTEMPT 2] Processing with customer_id added
2026-02-01 10:00:03 [SUCCESS] Invoice processed
```

**Key Points:**
- Errors are **added to context**, not hidden
- Enables **self-healing** through retry with corrections
- **Limit retries** to prevent error spin-outs (typically 3 attempts)
- **Compact format** - clear what failed, why, and what to try next

### Error Escalation
```yaml
actions:
  process-with-escalation:
    command: |
      if ! process-data --retry 3 input.json; then
        # After 3 failures, escalate to human
        aps myagent action run request-approval \
          --env ERROR_LOG="$(cat error.log)" \
          --env MESSAGE="Data processing failed after 3 attempts"
      fi
```

**Files:**
- `internal/session/logger.go` - Error context logging
- `internal/a2a/conversation.go` - Error in conversation history
- `internal/profile/action.go` - Retry logic support

**Docs:**
- [Session Logs](../user/SESSIONS.md#logs)
- [A2A Protocol](../specs/005-a2a-protocol/spec.md)

---

## Factor 10: Small, Focused Agents

**Principle:** Decompose into small, single-purpose agents.

**APS Implementation:**

### One Profile Per Agent
```bash
# Create focused profiles
aps profile create github-triage --display-name "GitHub Issue Triage"
aps profile create slack-responder --display-name "Slack Auto-Responder"
aps profile create payment-processor --display-name "Payment Link Generator"
```

**Each profile:**
- Has one clear responsibility
- Isolated environment
- Dedicated secrets
- Focused actions

**Anti-pattern (don't do this):**
```yaml
# ❌ Bad: "do-everything" profile
profile:
  id: super-agent
  actions:
    - github-triage
    - slack-responder
    - payment-processor
    - deploy-prod
    - send-emails
    # ... 50 more actions
```

**Best practice:**
```yaml
# ✅ Good: Focused profiles
profiles:
  - github-triage    # Only GitHub issues
  - slack-responder  # Only Slack
  - payment-processor # Only payments
```

**Files:**
- `<data>/profiles/*/profile.yaml` - One profile = one agent

**Docs:**
- [Profile Management](../user/PROFILES.md)

---

## Factor 11: Trigger from Anywhere

**Principle:** Agents should be callable from anywhere (CLI, API, webhooks).

**APS Implementation:**

### CLI
```bash
aps myagent -- echo "Hello from CLI"
```

### Webhooks
```bash
curl -X POST https://example.com/webhook/myagent/deploy \
  -H "X-Webhook-Secret: mysecret" \
  -d '{"environment": "staging"}'
```

### A2A Protocol (Agent-to-Agent)
```bash
# Other agents can invoke via A2A
aps otheragent a2a task create \
  --agent myagent \
  --description "Process payment for invoice #123"
```

### Interactive Shell
```bash
# Start interactive session
aps myagent
```

**All entry points use the same underlying execution:**
- Same isolation levels
- Same environment injection
- Same session tracking
- Same logging

**Files:**
- `cmd/run.go` - CLI entry point
- `internal/webhook/handler.go` - Webhook entry point
- `internal/a2a/server.go` - A2A entry point

**Docs:**
- [CLI Reference](../user/CLI.md)
- [Webhooks](../user/WEBHOOKS.md)
- [A2A Protocol](../specs/005-a2a-protocol/spec.md)

---

## Factor 12: Stateless Reducer

**Principle:** Agents should be stateless, deterministic.

**APS Implementation:**

### Stateless Profile Execution
```yaml
# Profile is stateless configuration
# ~/.local/share/aps/profiles/myagent/profile.yaml
env:
  API_KEY: "{{secrets.api_key}}"

actions:
  process:
    command: |
      # Pure function: same input → same output
      cat input.json | transform | output
```

**Key Points:**
- Profiles are configuration (stateless)
- State lives in sessions (transient)
- Actions are pure transformations
- Secrets are injected, not stored in actions

### Session State is Separate
```json
// Session = ephemeral execution state
{
  "id": "sess_123",
  "profile_id": "myagent",  // References stateless profile
  "status": "active",        // Execution state
  "pid": 12345
}
```

**Files:**
- `<data>/profiles/*/profile.yaml` - Stateless configuration
- `~/.aps/sessions/registry.json` - Ephemeral execution state

---

## Summary: Full 12-Factor Coverage

| Factor | APS Implementation | Files | Docs |
|--------|-------------------|-------|------|
| 1. NL → Tool Calls | A2A Protocol | `internal/a2a/` | [A2A Spec](../specs/005-a2a-protocol/spec.md) |
| 2. Own Prompts | Profile YAML | `profile.yaml` | [Profiles](../user/PROFILES.md) |
| 3. Own Context | Env vars, volumes | `internal/profile/env.go` | [Isolation](../user/ISOLATION.md) |
| 4. Structured Outputs | Action results, A2A | `internal/a2a/task.go` | [CLI](../user/CLI.md) |
| 5. Unified State | Session registry | `internal/session/registry.go` | [Sessions](../user/SESSIONS.md) |
| 6. Launch/Pause/Resume | Session lifecycle | `internal/session/lifecycle.go` | [Sessions](../user/SESSIONS.md) |
| 7. Contact Humans | Webhooks, actions | `internal/webhook/` | [Webhooks](../user/WEBHOOKS.md) |
| 8. Own Control Flow | Profile actions | `internal/profile/action.go` | [Profiles](../user/PROFILES.md) |
| 9. Compact Errors → Context | Error recovery, retry | `internal/session/logger.go`, `internal/a2a/conversation.go` | [Sessions](../user/SESSIONS.md) |
| 10. Small Agents | One profile = one agent | `<data>/profiles/` | [Profiles](../user/PROFILES.md) |
| 11. Trigger Anywhere | CLI/webhooks/A2A | `cmd/`, `internal/webhook/`, `internal/a2a/` | [CLI](../user/CLI.md) |
| 12. Stateless | Profile config | `profile.yaml` | [Profiles](../user/PROFILES.md) |

---

## Next Steps

1. **Read the full 12-factor methodology:** [12-factor-agents/](12-factor-agents/)
2. **Try the examples:** [EXAMPLES.md](../user/EXAMPLES.md)
3. **Design your agent profiles** following these principles
4. **Join the discussion:** https://github.com/ideacrafterslabs/oss-aps-cli/discussions

---

## Related

- **12-Factor Agents (Full Methodology):** [12-factor-agents/README.md](12-factor-agents/README.md)
- **APS Architecture:** [AGENTS.md](../../AGENTS.md)
- **A2A Protocol:** [specs/005-a2a-protocol/](../specs/005-a2a-protocol/)
