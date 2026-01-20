# Webhooks

Event-driven automation with APS webhooks.

## Overview

APS webhooks allow external services to trigger profile actions via HTTP POST requests. This enables:

- CI/CD pipeline integration
- GitHub/GitLab automation
- Slack notifications
- Custom event-driven workflows

## Quick Start

### 1. Create Profile with Action

```bash
aps profile new webhook-receiver \
  --display-name "Webhook Receiver"
```

Create an action to handle webhooks:

```bash
cat > ~/.agents/profiles/webhook-receiver/actions/handle.sh << 'EOF'
#!/bin/bash
echo "Received webhook event: $APS_WEBHOOK_EVENT"
echo "Delivery ID: $APS_WEBHOOK_DELIVERY_ID"
echo "Source IP: $APS_WEBHOOK_SOURCE_IP"
echo "Payload:"
cat
EOF
chmod +x ~/.agents/profiles/webhook-receiver/actions/handle.sh
```

### 2. Start Webhook Server

```bash
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --event-map github.push=webhook-receiver:handle.sh
```

### 3. Send Test Webhook

```bash
curl -X POST http://127.0.0.1:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-APS-Event: github.push" \
  -d '{"repository": "myrepo", "ref": "main"}'
```

## Command Reference

```bash
aps webhook serve [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--addr` | Bind address (ip:port) | `127.0.0.1:8080` |
| `--profile` | Default profile for unmapped events | - |
| `--secret` | Shared secret for signature validation | - |
| `--event-map` | Map events to profile:action (repeatable) | - |
| `--allow-event` | Allow only specific events (repeatable) | - |
| `--dry-run` | Dry run mode (don't execute actions) | `false` |

## Event Mapping

Map external events to profile actions:

```bash
aps webhook serve \
  --event-map github.push=bot1:deploy.sh \
  --event-map github.issue_comment.created=bot1:respond.py \
  --event-map custom.build=bot2:build.sh
```

**Syntax:** `--event-map <event>=<profile-id>:<action-id>`

### Examples

**GitHub Push:**
```bash
--event-map github.push=deployment-bot:deploy.sh
```

**GitHub Issue Comment:**
```bash
--event-map github.issue_comment.created=triage-bot:comment.py
```

**Custom Events:**
```bash
--event-map build.requested=ci-bot:build.sh
--event-map notify.slack=alert-bot:send-alert.sh
```

## Request Format

### Headers

| Header | Description | Required |
|--------|-------------|----------|
| `X-APS-Event` | Event type (e.g., `github.push`) | Yes |
| `X-APS-Signature` | HMAC SHA256 signature | No |
| `Content-Type` | Request content type | Recommended |

### Body

JSON request body:

```json
{
  "repository": "myrepo",
  "ref": "main",
  "commits": [...]
}
```

## Signature Validation

When a secret is configured, APS validates the request signature.

### Generating Signatures

**Curl:**
```bash
SECRET="my-secret-key"
PAYLOAD='{"test": "data"}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

curl -X POST http://127.0.0.1:8080/webhook \
  -H "X-APS-Event: test.event" \
  -H "X-APS-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

**Bash:**
```bash
#!/bin/bash
SECRET="my-secret-key"
PAYLOAD='{"test": "data"}'
URL="http://127.0.0.1:8080/webhook"

# Generate signature
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')

# Send request
curl -X POST "$URL" \
  -H "Content-Type: application/json" \
  -H "X-APS-Event: test.event" \
  -H "X-APS-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

**Node.js:**
```javascript
const crypto = require('crypto');
const fetch = require('node-fetch');

const SECRET = 'my-secret-key';
const PAYLOAD = { test: 'data' };
const EVENT = 'test.event';

function generateSignature(payload, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(payload);
  return 'sha256=' + hmac.digest('hex');
}

async function sendWebhook() {
  const payloadStr = JSON.stringify(PAYLOAD);
  const signature = generateSignature(payloadStr, SECRET);

  const response = await fetch('http://127.0.0.1:8080/webhook', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-APS-Event': EVENT,
      'X-APS-Signature': signature
    },
    body: payloadStr
  });

  const data = await response.json();
  console.log(data);
}

sendWebhook();
```

**Python:**
```python
import hmac
import hashlib
import json
import requests

SECRET = 'my-secret-key'
PAYLOAD = {'test': 'data'}
EVENT = 'test.event'
URL = 'http://127.0.0.1:8080/webhook'

def generate_signature(payload, secret):
    return 'sha256=' + hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()

def send_webhook():
    payload_str = json.dumps(PAYLOAD)
    signature = generate_signature(payload_str, SECRET)

    response = requests.post(
        URL,
        headers={
            'Content-Type': 'application/json',
            'X-APS-Event': EVENT,
            'X-APS-Signature': signature
        },
        data=payload_str
    )

    print(response.json())

send_webhook()
```

## Event Allowlist

Restrict which events are accepted:

```bash
aps webhook serve \
  --allow-event github.push \
  --allow-event github.issue_comment.created \
  --event-map github.push=bot1:deploy.sh \
  --event-map github.issue_comment.created=bot1:respond.py
```

Any event not in the allowlist will be rejected with `403 Forbidden`.

## Response Format

### Success Response (200 OK)

```json
{
  "delivery_id": "550e8400-e29b-41d4-a716-446655440000",
  "event": "github.push",
  "profile": "deployment-bot",
  "action": "deploy.sh",
  "status": "executed"
}
```

### Dry Run Response (200 OK)

```json
{
  "delivery_id": "550e8400-e29b-41d4-a716-446655440000",
  "event": "github.push",
  "profile": "deployment-bot",
  "action": "deploy.sh",
  "status": "dry_run"
}
```

### Error Responses

**Unauthorized (401):**
```json
{
  "error": "Invalid signature"
}
```

**Not Found (404):**
```json
{
  "event": "custom.event",
  "error": "No handler configured for this event"
}
```

**Forbidden (403):**
```json
{
  "event": "custom.event",
  "error": "Event not in allowlist"
}
```

**Bad Request (400):**
```json
{
  "error": "Invalid request body"
}
```

## Environment Variables

When an action is triggered by a webhook, APS injects these variables:

| Variable | Description |
|----------|-------------|
| `APS_WEBHOOK_EVENT` | Event type from `X-APS-Event` header |
| `APS_WEBHOOK_DELIVERY_ID` | Unique delivery ID (from `X-Request-Id` or generated) |
| `APS_WEBHOOK_SOURCE_IP` | Source IP address of the request |

### Accessing in Actions

**Shell:**
```bash
#!/bin/bash
echo "Event: $APS_WEBHOOK_EVENT"
echo "Delivery ID: $APS_WEBHOOK_DELIVERY_ID"
```

**Python:**
```python
#!/usr/bin/env python3
import os
import json

event = os.environ['APS_WEBHOOK_EVENT']
delivery_id = os.environ['APS_WEBHOOK_DELIVERY_ID']
source_ip = os.environ['APS_WEBHOOK_SOURCE_IP']

print(f"Event: {event}")
print(f"Delivery ID: {delivery_id}")
print(f"Source IP: {source_ip}")
```

**Node.js:**
```javascript
#!/usr/bin/env node
const event = process.env.APS_WEBHOOK_EVENT;
const deliveryId = process.env.APS_WEBHOOK_DELIVERY_ID;
const sourceIp = process.env.APS_WEBHOOK_SOURCE_IP;

console.log(`Event: ${event}`);
console.log(`Delivery ID: ${deliveryId}`);
console.log(`Source IP: ${sourceIp}`);
```

## Common Integrations

### GitHub Webhooks

**Setup:**

1. Go to repository Settings → Webhooks
2. Add webhook URL: `http://your-server:8080/webhook`
3. Content type: `application/json`
4. Secret: (optional) your shared secret
5. Select events: `Push`, `Issues`, `Issue comments`

**Handle Push:**
```bash
aps webhook serve \
  --secret "github-webhook-secret" \
  --event-map github.push=bot:deploy.sh
```

**Handle Issue Comment:**
```bash
aps webhook serve \
  --secret "github-webhook-secret" \
  --event-map github.issue_comment.created=bot:respond.py
```

**Example Action for Issue Comments:**
```python
#!/usr/bin/env python3
import os
import json
from github import Github

# Get webhook payload
payload = json.load(sys.stdin)

# Initialize GitHub client
token = os.environ['GITHUB_TOKEN']
gh = Github(token)

# Get repository and issue
repo = gh.get_repo(payload['repository']['full_name'])
issue = repo.get_issue(payload['issue']['number'])

# Reply
issue.create_comment(
    f"Thanks for the comment from {payload['comment']['user']['login']}!"
)
```

### GitLab Webhooks

**Setup:**

1. Go to project Settings → Webhooks
2. Add webhook URL: `http://your-server:8080/webhook`
3. Secret token: (optional) your shared secret
4. Trigger events: `Push events`, `Comment events`

**Handle Push:**
```bash
aps webhook serve \
  --secret "gitlab-webhook-secret" \
  --event-map Push\ Events=bot:deploy.sh
```

### Slack

**Use Slack Outgoing Webhooks or Slack App with webhook URL.**

**Handle Slack Event:**
```bash
aps webhook serve \
  --event-map slack.command=bot:slack-handler.sh
```

### Custom CI/CD

**Trigger Build:**
```bash
curl -X POST http://127.0.0.1:8080/webhook \
  -H "X-APS-Event: build.requested" \
  -d '{"project": "myapp", "branch": "main"}'
```

**Server:**
```bash
aps webhook serve \
  --event-map build.requested=ci-bot:build.sh
```

## Production Deployment

### Run as Systemd Service

Create `/etc/systemd/system/aps-webhook.service`:

```ini
[Unit]
Description=APS Webhook Server
After=network.target

[Service]
Type=simple
User=aps
WorkingDirectory=/home/aps
ExecStart=/usr/local/bin/aps webhook serve \
  --addr 0.0.0.0:8080 \
  --secret "${WEBHOOK_SECRET}" \
  --event-map github.push=bot:deploy.sh \
  --event-map build.requested=ci-bot:build.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable aps-webhook
sudo systemctl start aps-webhook
```

### Run with Docker

**Dockerfile:**
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk add --no-cache bash
COPY --from=builder /app/bin/aps /usr/local/bin/aps
COPY --from=builder /app/.agents /root/.agents

EXPOSE 8080
CMD ["aps", "webhook", "serve", "--addr", "0.0.0.0:8080"]
```

**Run:**
```bash
docker build -t aps-webhook .
docker run -d \
  -p 8080:8080 \
  -e WEBHOOK_SECRET="my-secret" \
  aps-webhook \
  --secret "$WEBHOOK_SECRET" \
  --event-map github.push=bot:deploy.sh
```

### Reverse Proxy with Nginx

```nginx
server {
    listen 80;
    server_name webhook.example.com;

    location /webhook {
        proxy_pass http://127.0.0.1:8080/webhook;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Testing

### Dry Run Mode

Test webhook server without executing actions:

```bash
aps webhook serve --dry-run \
  --event-map github.push=bot:deploy.sh
```

All webhooks are logged but no actions are executed.

### Health Check

Check if webhook server is running:

```bash
curl http://127.0.0.1:8080/healthz
```

Returns `200 OK` if server is healthy.

### Local Testing with ngrok

Expose local webhook server to internet:

```bash
# In one terminal
ngrok http 8080

# In another terminal
aps webhook serve \
  --event-map github.push=bot:deploy.sh

# Use ngrok URL in GitHub/GitLab webhook settings
```

## Best Practices

1. **Always use secrets** for production webhooks
2. **Validate event types** with allowlist
3. **Log delivery IDs** for debugging and auditing
4. **Use dry run mode** when testing
5. **Rate limit incoming webhooks** at the proxy level
6. **Handle errors gracefully** in action scripts
7. **Test signature validation** before deploying
8. **Monitor webhook server** logs for failures
9. **Keep actions idempotent** - handle duplicate events
10. **Set appropriate timeouts** for long-running actions

## Troubleshooting

### Webhook Not Received

1. Check server logs: `journalctl -u aps-webhook`
2. Test health check: `curl http://127.0.0.1:8080/healthz`
3. Check firewall rules
4. Verify webhook URL is correct

### Signature Validation Failed

1. Ensure secret matches on both sides
2. Check signature generation algorithm (SHA256)
3. Verify signature format: `sha256=<hex>`
4. Ensure payload body is exactly what was signed

### Action Not Executed

1. Check event mapping syntax
2. Verify profile and action exist
3. Check action file is executable
4. Review action logs in server output

### Delivery ID Not Generated

Add `X-Request-Id` header if you want custom delivery IDs:

```bash
curl -X POST http://127.0.0.1:8080/webhook \
  -H "X-APS-Event: test.event" \
  -H "X-Request-Id: my-custom-id" \
  -d '{}'
```
