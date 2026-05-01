# Profile Management

Complete guide to creating, configuring, and managing APS profiles.

## What is a Profile?

A profile is an isolated environment containing:

- **Identity**: Git config, social media accounts
- **Credentials**: API tokens, SSH keys
- **Preferences**: Language, timezone, shell
- **Capabilities**: Allowed actions/operations
- **Persona**: Tone, style, risk tolerance
- **Actions**: Custom scripts the agent can run

## Creating Profiles

### Basic Profile

```bash
aps profile create myagent
```

Creates:
```
~/.agents/profiles/myagent/
  profile.yaml
  secrets.env
  notes.md
  actions/
```

### Profile with Metadata

```bash
aps profile create openai-agent \
  --display-name "OpenAI Agent" \
  --email "openai@example.com" \
  --github "openai-bot" \
  --twitter "openai_agent"
```

## Profile Configuration

### profile.yaml Structure

```yaml
id: myagent
display_name: "My AI Agent"

# Persona configuration
persona:
  tone: "concise"
  style: "technical"
  risk: "low"

# Capabilities - what this agent can do
capabilities:
  - git
  - github
  - reddit
  - twitter
  - webhooks

# Account configurations
accounts:
  github:
    username: "myagent"
  twitter:
    username: "myagent_ai"
  reddit:
    username: "myagent_research"

# Runtime preferences
preferences:
  language: "en"
  timezone: "America/New_York"
  shell: "/bin/zsh"

# Resource limits
limits:
  max_concurrency: 2
  max_runtime_minutes: 30

# Git module
git:
  enabled: true

# SSH module
ssh:
  enabled: false
  key_path: "~/.ssh/myagent_ed25519"

# Webhook module
webhooks:
  enabled: true
  allowed_events:
    - "github.push"
    - "github.issue_comment.created"
```

### Editing Profiles

Edit the YAML file directly:

```bash
nano ~/.agents/profiles/myagent/profile.yaml
```

Or use your favorite editor.

## Managing Secrets

### Adding Secrets

Edit `secrets.env`:

```bash
nano ~/.agents/profiles/myagent/secrets.env
```

```bash
# Add your secrets here. Format: KEY=VALUE

# GitHub
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx

# OpenAI
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx

# Custom
MY_API_KEY=secret_value_here
```

**Security:** `secrets.env` is created with `0600` permissions (owner read/write only).

### Viewing Secrets

```bash
aps profile show myagent
```

Shows keys but redacts values:
```
secrets:
  GITHUB_TOKEN: ***redacted***
  OPENAI_API_KEY: ***redacted***
  MY_API_KEY: ***redacted***
```

### Accessing Secrets in Actions

Secrets are automatically available as environment variables in all commands and actions:

```bash
# In a shell action
echo "$GITHUB_TOKEN"

# In a Python action
import os
token = os.environ.get('GITHUB_TOKEN')

# In a Node.js action
const token = process.env.GITHUB_TOKEN;
```

## Modules

### Git Module

If `gitconfig` exists and `git.enabled: true`, APS injects:

```bash
GIT_CONFIG_GLOBAL=~/.agents/profiles/myagent/gitconfig
```

**Setup:**

1. Enable in profile.yaml:
```yaml
git:
  enabled: true
```

2. Create gitconfig:
```bash
cat > ~/.agents/profiles/myagent/gitconfig << 'EOF'
[user]
  name = "My AI Agent"
  email = "agent@example.com"

[github]
  user = "myagent"
EOF
```

3. Use it:
```bash
aps myagent -- git config user.name
aps myagent -- git commit -m "Auto-commit"
```

**Or create with email flag:**
```bash
aps profile create myagent --email "agent@example.com"
```

### SSH Module

If `ssh.key` exists and `ssh.enabled: true`, APS may inject:

```bash
GIT_SSH_COMMAND=ssh -i ~/.agents/profiles/myagent/ssh.key -F /dev/null
```

**Setup:**

1. Enable in profile.yaml:
```yaml
ssh:
  enabled: true
```

2. Generate SSH key:
```bash
ssh-keygen -t ed25519 -f ~/.agents/profiles/myagent/ssh.key -N ""
```

3. Use it:
```bash
aps myagent -- git push
```

## Actions

### Creating Actions

Actions live in `~/.agents/profiles/<id>/actions/`:

**Shell Action** (`hello.sh`):
```bash
#!/bin/bash
echo "Hello from $APS_PROFILE_ID!"
echo "Secret: $MY_SECRET"
```

**Python Action** (`greet.py`):
```python
#!/usr/bin/env python3
import os
import sys

def main():
    payload = sys.stdin.read()
    print(f"Hello from {os.environ['APS_PROFILE_ID']}!")
    if payload:
        print(f"Received: {payload}")

if __name__ == "__main__":
    main()
```

**Node.js Action** (`process.js`):
```javascript
#!/usr/bin/env node
const fs = require('fs');

const payload = fs.readFileSync(0, 'utf-8');
const data = JSON.parse(payload);
console.log(`Processing: ${JSON.stringify(data)}`);
```

### Action Manifest

Create `actions.yaml` for better UX:

```yaml
actions:
  - id: hello.sh
    title: "Say hello"
    description: "Prints a greeting message"
    accepts_stdin: true

  - id: greet.py
    title: "Greet user"
    description: "Greets the user with payload"
    accepts_stdin: true

  - id: process.js
    title: "Process JSON data"
    description: "Processes incoming JSON data"
    accepts_stdin: true
```

### Running Actions

```bash
# List actions
aps action list myagent

# Show action details
aps action show myagent hello.sh

# Run action
aps action run myagent hello.sh

# Run with file payload
aps action run myagent process.js --payload-file data.json

# Run with stdin payload
echo '{"name": "John"}' | aps action run myagent greet.py --payload-stdin
```

## Notes

Each profile has a `notes.md` for documentation:

```markdown
# My Agent Notes

## Purpose
This agent handles GitHub issues.

## Setup
1. Add GITHUB_TOKEN to secrets.env
2. Enable git module for commits

## Known Issues
- Rate limits apply for GitHub API
```

## Listing Profiles

```bash
aps profile list
```

Output:
```
myagent
openai-agent
test-agent
dev-agent
```

## Viewing Profiles

```bash
aps profile show myagent
```

Shows complete configuration with redacted secrets.

## Deleting Profiles

Delete the profile directory:

```bash
rm -rf ~/.agents/profiles/myagent
```

## Profiles Directory Structure

```
~/.agents/profiles/
  myagent/
    profile.yaml           # Profile configuration
    secrets.env           # Environment variables (chmod 0600)
    gitconfig             # Git configuration (optional)
    ssh.key               # SSH key (optional)
    actions/              # Action scripts
      hello.sh
      greet.py
      process.js
    actions.yaml          # Action manifest (optional)
    notes.md              # Profile notes
```

## Common Use Cases

### GitHub Bot

```bash
aps profile create github-bot \
  --display-name "GitHub Bot" \
  --email "bot@example.com" \
  --github "mybot"

# Add secrets
nano ~/.agents/profiles/github-bot/secrets.env
# Add: GITHUB_TOKEN=ghp_xxxxxxxxxx

# Create action
cat > ~/.agents/profiles/github-bot/actions/label.sh << 'EOF'
#!/bin/bash
gh issue edit "$1" --add-label "triaged"
EOF
chmod +x ~/.agents/profiles/github-bot/actions/label.sh

# Use
aps github-bot -- gh issue list
aps action run github-bot label.sh
```

### OpenAI Agent

```bash
aps profile create openai-agent \
  --display-name "OpenAI Agent"

# Add secrets
nano ~/.agents/profiles/openai-agent/secrets.env
# Add: OPENAI_API_KEY=sk-xxxxxxxxx

# Create Python action
cat > ~/.agents/profiles/openai-agent/actions/chat.py << 'EOF'
#!/usr/bin/env python3
import os
import sys
import json
from openai import OpenAI

client = OpenAI(api_key=os.environ['OPENAI_API_KEY'])

def main():
    data = json.load(sys.stdin)
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": data['message']}]
    )
    print(response.choices[0].message.content)

if __name__ == "__main__":
    main()
EOF
chmod +x ~/.agents/profiles/openai-agent/actions/chat.py

# Use
echo '{"message": "Hello!"}' | aps action run openai-agent chat.py --payload-stdin
```

### Multi-Environment Profile

```bash
# Development profile
aps profile create dev-agent \
  --display-name "Dev Agent" \
  --email "dev@example.com"

# Add dev secrets
nano ~/.agents/profiles/dev-agent/secrets.env
# Add: API_KEY=dev_xxxxxxxxxx

# Production profile
aps profile create prod-agent \
  --display-name "Prod Agent" \
  --email "prod@example.com"

# Add prod secrets
nano ~/.agents/profiles/prod-agent/secrets.env
# Add: API_KEY=prod_xxxxxxxxxx

# Use different profiles
aps dev-agent -- npm test
aps prod-agent -- npm run deploy
```

## Best Practices

1. **One Profile Per Environment**: Use separate profiles for dev/staging/prod
2. **Minimal Secrets**: Only add secrets that profile needs
3. **Descriptive Names**: Use clear profile IDs and display names
4. **Version Control**: Don't commit secrets.env (add to .gitignore)
5. **Document Notes**: Use notes.md to document profile purpose and setup
6. **Secure Permissions**: Ensure secrets.env has 0600 permissions
7. **Test Actions**: Test actions with --dry-run first
8. **Use Manifests**: Create actions.yaml for better action discoverability
