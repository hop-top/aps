# Examples

Practical examples and common use cases for APS.

## GitHub Issue Triage

**Profile Setup:**
```bash
aps profile new triage-bot \
  --display-name "Issue Triage Bot" \
  --email "triage@example.com" \
  --github "triage-bot"
```

**Add GitHub Token:**
```bash
nano ~/.agents/profiles/triage-bot/secrets.env
# Add: GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

**Create Label Action:**
```bash
cat > ~/.agents/profiles/triage-bot/actions/label.sh << 'EOF'
#!/bin/bash
ISSUE_URL=$1
gh issue edit "$ISSUE_URL" --add-label "triaged"
EOF
chmod +x ~/.agents/profiles/triage-bot/actions/label.sh
```

**Create Comment Action:**
```bash
cat > ~/.agents/profiles/triage-bot/actions/comment.sh << 'EOF'
#!/bin/bash
ISSUE_URL=$1
MESSAGE="$2"
gh issue comment "$ISSUE_URL" --body "$MESSAGE"
EOF
chmod +x ~/.agents/profiles/triage-bot/actions/comment.sh
```

**Usage:**
```bash
# List issues
aps triage-bot -- gh issue list

# Label an issue
aps triage-bot -- gh issue edit 123 --add-label "triaged"

# Comment on issue
aps triage-bot -- gh issue comment 123 --body "Investigating..."
```

## OpenAI Chat Assistant

**Profile Setup:**
```bash
aps profile openai-assistant \
  --display-name "AI Assistant" \
  --email "assistant@example.com"
```

**Add OpenAI API Key:**
```bash
nano ~/.agents/profiles/openai-assistant/secrets.env
# Add: OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx
```

**Create Chat Action:**
```bash
cat > ~/.agents/profiles/openai-assistant/actions/chat.py << 'EOF'
#!/usr/bin/env python3
import os
import sys
import json
from openai import OpenAI

client = OpenAI(api_key=os.environ['OPENAI_API_KEY'])

def main():
    # Read input from stdin
    input_data = json.load(sys.stdin)

    # Make API call
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": input_data.get('message', '')}
        ]
    )

    # Output response
    print(response.choices[0].message.content)

if __name__ == "__main__":
    main()
EOF
chmod +x ~/.agents/profiles/openai-assistant/actions/chat.py
```

**Create actions.yaml:**
```bash
cat > ~/.agents/profiles/openai-assistant/actions.yaml << 'EOF'
actions:
  - id: chat.py
    title: "Chat with AI"
    description: "Send a message to OpenAI GPT-4"
    accepts_stdin: true
EOF
```

**Usage:**
```bash
# Simple chat
echo '{"message": "What is 2+2?"}' | aps action run openai-assistant chat.py --payload-stdin

# With file
cat > message.json << 'EOF'
{"message": "Explain quantum computing"}
EOF
aps action run openai-assistant chat.py --payload-file message.json
```

## Data Processing Pipeline

**Profile Setup:**
```bash
aps profile new data-processor \
  --display-name "Data Processing Agent"
```

**Create Process Action (Node.js):**
```bash
cat > ~/.agents/profiles/data-processor/actions/process.js << 'EOF'
#!/usr/bin/env node
const fs = require('fs');

// Read JSON from stdin
const data = JSON.parse(fs.readFileSync(0, 'utf-8'));

// Process data
const processed = data.map(item => ({
  ...item,
  processed: true,
  timestamp: new Date().toISOString()
}));

// Output result
console.log(JSON.stringify(processed, null, 2));
EOF
chmod +x ~/.agents/profiles/data-processor/actions/process.js
```

**Create Validate Action (Python):**
```bash
cat > ~/.agents/profiles/data-processor/actions/validate.py << 'EOF'
#!/usr/bin/env python3
import sys
import json

def validate(item):
    return 'id' in item and 'name' in item

def main():
    data = json.load(sys.stdin)
    valid = [item for item in data if validate(item)]
    invalid = [item for item in data if not validate(item)]

    print(f"Valid: {len(valid)}")
    print(f"Invalid: {len(invalid)}")

    if invalid:
        print("\nInvalid items:")
        for item in invalid:
            print(f"  - {item}")

if __name__ == "__main__":
    main()
EOF
chmod +x ~/.agents/profiles/data-processor/actions/validate.py
```

**Usage:**
```bash
# Create test data
cat > data.json << 'EOF'
[
  {"id": 1, "name": "Alice"},
  {"id": 2, "name": "Bob"},
  {"name": "Invalid"}
]
EOF

# Process data
aps action run data-processor process.js --payload-file data.json

# Validate data
aps action run data-processor validate.py --payload-file data.json
```

## Multi-Environment Deployment

**Development Profile:**
```bash
aps profile new dev \
  --display-name "Development" \
  --email "dev@example.com"

nano ~/.agents/profiles/dev/secrets.env
# Add: API_KEY=dev_12345
# Add: DB_HOST=localhost
```

**Staging Profile:**
```bash
aps profile new staging \
  --display-name "Staging" \
  --email "staging@example.com"

nano ~/.agents/profiles/staging/secrets.env
# Add: API_KEY=staging_67890
# Add: DB_HOST=staging.example.com
```

**Production Profile:**
```bash
aps profile new prod \
  --display-name "Production" \
  --email "prod@example.com"

nano ~/.agents/profiles/prod/secrets.env
# Add: API_KEY=prod_abcde
# Add: DB_HOST=prod.example.com
```

**Create Deploy Action:**
```bash
cat > ~/.agents/profiles/dev/actions/deploy.sh << 'EOF'
#!/bin/bash
echo "Deploying from $APS_PROFILE_ID"
echo "API Key: $API_KEY"
echo "DB Host: $DB_HOST"
npm run deploy
EOF
chmod +x ~/.agents/profiles/dev/actions/deploy.sh
```

**Copy to other profiles:**
```bash
cp ~/.agents/profiles/dev/actions/deploy.sh ~/.agents/profiles/staging/actions/
cp ~/.agents/profiles/dev/actions/deploy.sh ~/.agents/profiles/prod/actions/
```

**Usage:**
```bash
# Deploy to each environment
aps dev -- npm run deploy
aps staging -- npm run deploy
aps prod -- npm run deploy
```

## Reddit Bot

**Profile Setup:**
```bash
aps profile new reddit-bot \
  --display-name "Reddit Bot" \
  --email "reddit@example.com" \
  --reddit "mybot"
```

**Add Reddit Credentials:**
```bash
nano ~/.agents/profiles/reddit-bot/secrets.env
# Add: REDDIT_CLIENT_ID=your_client_id
# Add: REDDIT_CLIENT_SECRET=your_client_secret
# Add: REDDIT_USERNAME=your_username
# Add: REDDIT_PASSWORD=your_password
```

**Create Post Action (Python):**
```bash
cat > ~/.agents/profiles/reddit-bot/actions/post.py << 'EOF'
#!/usr/bin/env python3
import os
import sys
import json
import praw

def main():
    data = json.load(sys.stdin)

    reddit = praw.Reddit(
        client_id=os.environ['REDDIT_CLIENT_ID'],
        client_secret=os.environ['REDDIT_CLIENT_SECRET'],
        username=os.environ['REDDIT_USERNAME'],
        password=os.environ['REDDIT_PASSWORD'],
        user_agent='aps/1.0'
    )

    subreddit = reddit.subreddit(data['subreddit'])
    subreddit.submit(
        title=data['title'],
        selftext=data.get('body', '')
    )

    print(f"Posted to r/{data['subreddit']}")

if __name__ == "__main__":
    main()
EOF
chmod +x ~/.agents/profiles/reddit-bot/actions/post.py
```

**Usage:**
```bash
cat > post.json << 'EOF'
{
  "subreddit": "test",
  "title": "Test post from APS",
  "body": "This is an automated post."
}
EOF

aps action run reddit-bot post.py --payload-file post.json
```

## Git Workflow Automation

**Profile Setup:**
```bash
aps profile new git-automator \
  --display-name "Git Automator" \
  --email "bot@example.com"
```

**Create Commit Action:**
```bash
cat > ~/.agents/profiles/git-automator/actions/commit.sh << 'EOF'
#!/bin/bash
MESSAGE="$1"

git add .
git commit -m "$MESSAGE"
git push
EOF
chmod +x ~/.agents/profiles/git-automator/actions/commit.sh
```

**Create Branch Action:**
```bash
cat > ~/.agents/profiles/git-automator/actions/branch.sh << 'EOF'
#!/bin/bash
BRANCH="$1"

git checkout -b "$BRANCH"
git push -u origin "$BRANCH"
EOF
chmod +x ~/.agents/profiles/git-automator/actions/branch.sh
```

**Create PR Action:**
```bash
cat > ~/.agents/profiles/git-automator/actions/pr.sh << 'EOF'
#!/bin/bash
TITLE="$1"
BODY="$2"

gh pr create --title "$TITLE" --body "$BODY"
EOF
chmod +x ~/.agents/profiles/git-automator/actions/pr.sh
```

**Usage:**
```bash
cd /path/to/repo

# Create feature branch
aps action run git-automator branch.sh -- feature/new-feature

# Make changes...

# Commit changes
aps action run git-automator commit.sh -- "Add new feature"

# Create PR
aps action run git-automator pr.sh -- "New feature" "Implements XYZ"
```

## Webhook Integration

**Profile Setup:**
```bash
aps profile new webhook-handler \
  --display-name "Webhook Handler" \
  --email "webhook@example.com"
```

**Create Handler Action:**
```bash
cat > ~/.agents/profiles/webhook-handler/actions/handle.sh << 'EOF'
#!/bin/bash
echo "Webhook event: $APS_WEBHOOK_EVENT"
echo "Delivery ID: $APS_WEBHOOK_DELIVERY_ID"
echo "Source IP: $APS_WEBHOOK_SOURCE_IP"

# Read payload
PAYLOAD=$(cat)
echo "Payload:"
echo "$PAYLOAD"
EOF
chmod +x ~/.agents/profiles/webhook-handler/actions/handle.sh
```

**Start Webhook Server:**
```bash
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --event-map github.push=webhook-handler:handle.sh \
  --event-map github.issue_comment.created=webhook-handler:handle.sh
```

**Send Test Webhook:**
```bash
curl -X POST http://127.0.0.1:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-APS-Event: github.push" \
  -d '{"repository": "myrepo", "ref": "main"}'
```

## Database Migration

**Profile Setup:**
```bash
aps profile new db-migrator \
  --display-name "Database Migrator"
```

**Add Database Credentials:**
```bash
nano ~/.agents/profiles/db-migrator/secrets.env
# Add: DB_HOST=localhost
# Add: DB_PORT=5432
# Add: DB_USER=migration_user
# Add: DB_PASS=migration_password
# Add: DB_NAME=mydb
```

**Create Migrate Action:**
```bash
cat > ~/.agents/profiles/db-migrator/actions/migrate.sh << 'EOF'
#!/bin/bash
MIGRATION="$1"

psql \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  -f "migrations/$MIGRATION.sql"

echo "Migration $MIGRATION completed"
EOF
chmod +x ~/.agents/profiles/db-migrator/actions/migrate.sh
```

**Usage:**
```bash
# Run migration
aps action run db-migrator migrate.sh -- 001_initial.sql
```

## Custom Shell Environment

**Profile Setup:**
```bash
aps profile new custom-shell \
  --display-name "Custom Shell" \
  --email "shell@example.com"
```

**Configure Profile:**
```bash
nano ~/.agents/profiles/custom-shell/profile.yaml
```
```yaml
preferences:
  shell: /bin/bash
```

**Create Setup Action:**
```bash
cat > ~/.agents/profiles/custom-shell/actions/setup.sh << 'EOF'
#!/bin/bash
export MY_VAR="custom_value"
export ANOTHER_VAR="another_value"
export NODE_ENV=development

echo "Environment variables set:"
echo "  MY_VAR=$MY_VAR"
echo "  ANOTHER_VAR=$ANOTHER_VAR"
echo "  NODE_ENV=$NODE_ENV"

# Keep shell open
exec $SHELL
EOF
chmod +x ~/.agents/profiles/custom-shell/actions/setup.sh
```

**Usage:**
```bash
# Run setup and enter shell
aps action run custom-shell setup.sh

# The shell will have the custom environment
```

## Running Tests with Different Environments

**Test Profiles:**
```bash
# Unit tests profile
aps profile new unit-tests \
  --display-name "Unit Tests"

nano ~/.agents/profiles/unit-tests/secrets.env
# Add: TEST_DB=localhost/test_unit
# Add: ENV=test

# Integration tests profile
aps profile new integration-tests \
  --display-name "Integration Tests"

nano ~/.agents/profiles/integration-tests/secrets.env
# Add: TEST_DB=localhost/test_integration
# Add: ENV=integration

# E2E tests profile
aps profile new e2e-tests \
  --display-name "E2E Tests"

nano ~/.agents/profiles/e2e-tests/secrets.env
# Add: TEST_DB=localhost/test_e2e
# Add: ENV=e2e
```

**Usage:**
```bash
# Run unit tests
aps unit-tests -- npm run test:unit

# Run integration tests
aps integration-tests -- npm run test:integration

# Run E2E tests
aps e2e-tests -- npm run test:e2e
```

## Tips and Tricks

### Quick Command Execution

```bash
# Single command (no -- separator needed for shorthands)
aps myagent -- ls -la

# Interactive shell
aps myagent

# Command with arguments
aps myagent -- npm test -- --coverage
```

### Chaining Commands

```bash
# Multiple commands with shell
aps myagent -- sh -c "cd /tmp && ls && pwd"
```

### Debugging Profile Environment

```bash
# See all environment variables
aps myagent -- env | grep APS
```

### Running Scripts from Repository

```bash
# Run a script from your repo with profile environment
cd /path/to/repo
aps myagent -- ./scripts/deploy.sh
```

### Working with Stdin

```bash
# Pipe data to command
echo "test" | aps myagent -- wc -l

# Read from file
aps myagent -- wc -l < file.txt
```
