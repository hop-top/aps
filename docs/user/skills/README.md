# Agent Skills - User Guide

Welcome to Agent Skills! This guide will help you understand, use, and create skills for APS.

## What are Agent Skills?

**Agent Skills** are packages of instructions, scripts, and resources that extend your AI agent's capabilities with specialized knowledge and workflows.

Think of skills as:
- 📚 **Expertise modules** - Specialized knowledge for specific domains
- 🔧 **Tool extensions** - New capabilities for your agent
- 📋 **Workflow templates** - Repeatable processes for common tasks
- 🔄 **Cross-platform** - Works across Claude Code, Cursor, VS Code, and more

---

## Quick Start

### 1. Check Available Skills

```bash
aps skill list
```

**Output:**
```
Found 3 skill(s):

Global (2):
  pdf-processing                Extract and manipulate PDF documents
  data-analysis                 Analyze datasets with Python/pandas

Profile (1):
  custom-workflow               Team-specific automation
```

### 2. View Skill Details

```bash
aps skill show pdf-processing
```

**Output:**
```
Name:          pdf-processing
Description:   Extract text and tables from PDF files
License:       MIT
Location:      ~/.local/share/aps/skills/pdf-processing

Scripts:
  • extract.py
  • merge.py

References:
  • REFERENCE.md
```

### 3. Install a Skill

```bash
# Install globally (available to all profiles)
aps skill install ./my-skill --global

# Install to specific profile
aps skill install ./my-skill --profile myagent
```

### 4. Use Skills

Skills are automatically available to your agent! When you start a task, your agent can discover and use relevant skills.

---

## How Skills Work

### Discovery & Activation

1. **Discovery** - APS scans configured directories for skills
2. **Metadata Loading** - Agent sees skill names and descriptions
3. **Matching** - Agent identifies relevant skills for your task
4. **Activation** - Agent loads full skill instructions
5. **Execution** - Agent follows skill workflows and runs scripts

### Skill Structure

```
my-skill/
├── SKILL.md              # Required: Instructions for the agent
├── scripts/              # Optional: Executable scripts
│   ├── process.py
│   └── analyze.sh
├── references/           # Optional: Additional documentation
│   └── REFERENCE.md
└── assets/               # Optional: Templates, data files
    └── template.json
```

---

## Where Skills Live

APS discovers skills from multiple locations (in priority order):

### 1. Profile Skills (Highest Priority)
```
~/.local/share/aps/profiles/<profile-id>/skills/
```
Skills specific to one profile.

### 2. Global APS Skills
```
~/.local/share/aps/skills/           # Linux
~/Library/Application Support/aps/skills/  # macOS
%LOCALAPPDATA%/aps/skills/           # Windows
```
Skills shared across all profiles.

### 3. User-Configured Paths
Add custom paths in `~/.config/aps/config.yaml`:
```yaml
skills:
  skill_sources:
    - /team/shared/skills
    - ~/custom-skills
```

### 4. IDE Skills (Optional)
Auto-detect skills from other tools:
```
~/.claude/skills/        # Claude Code
~/.cursor/skills/        # Cursor
~/.vscode/skills/        # VS Code
~/.gemini/skills/        # Gemini CLI
```

**Enable auto-detection:**
```bash
aps skill suggest    # Shows detected paths
```

Then add to config:
```yaml
skills:
  auto_detect_ide_paths: true
```

---

## Creating Your First Skill

### Step 1: Create Directory

```bash
mkdir -p ~/.local/share/aps/skills/my-first-skill
cd ~/.local/share/aps/skills/my-first-skill
```

### Step 2: Create SKILL.md

```yaml
---
name: my-first-skill
description: A simple example skill for processing data
license: MIT
---

# My First Skill

This skill helps process data files.

## Usage

Run the processing script:

```bash
./scripts/process.sh input.txt
```

## What it does

1. Reads the input file
2. Processes the data
3. Outputs results
```

### Step 3: Add Scripts (Optional)

```bash
mkdir scripts
cat > scripts/process.sh << 'EOF'
#!/bin/bash
echo "Processing $1..."
cat "$1" | tr '[:lower:]' '[:upper:]' > output.txt
echo "Done! Check output.txt"
EOF

chmod +x scripts/process.sh
```

### Step 4: Validate

```bash
aps skill validate ~/.local/share/aps/skills/my-first-skill
```

**Output:**
```
✓ Valid Agent Skill
  Name:        my-first-skill
  Description: A simple example skill for processing data
```

### Step 5: Use It!

```bash
aps skill list
```

Your skill is now available to your agent!

---

## Common Use Cases

### Use Case 1: Team Workflows

Create a skill for your team's standard processes:

```yaml
---
name: deploy-workflow
description: Standard deployment process for our team
---

# Deployment Workflow

## Pre-deployment Checklist
1. Run tests
2. Update changelog
3. Create release tag

## Deployment Steps
1. Build production bundle
2. Deploy to staging
3. Run smoke tests
4. Deploy to production

## Post-deployment
1. Monitor logs
2. Check metrics
3. Update team channel
```

### Use Case 2: Domain Expertise

Package specialized knowledge:

```yaml
---
name: legal-review
description: Legal review checklist for contracts
---

# Legal Review Process

## Key Areas to Review
1. Liability clauses
2. Indemnification
3. Termination conditions
4. IP ownership

## Red Flags
- Unlimited liability
- Auto-renewal without notice
- Restrictive non-compete clauses
```

### Use Case 3: Tool Integration

Integrate with external tools:

```yaml
---
name: api-integration
description: Integrate with our CRM API
metadata:
  api_endpoint: https://api.example.com
---

# CRM API Integration

## Authentication
Use API key from secrets: ${SECRET:CRM_API_KEY}

## Common Operations
- Get customer: `GET /customers/:id`
- Create lead: `POST /leads`
- Update status: `PATCH /deals/:id`
```

---

## Advanced Features

### Secret Management

Use secret placeholders in scripts:

```bash
#!/bin/bash
API_KEY="${SECRET:API_KEY}"
curl -H "Authorization: Bearer $API_KEY" https://api.example.com
```

APS automatically replaces `${SECRET:API_KEY}` with the real value from your profile's secrets at execution time.

### Protocol Targeting

Specify which protocols support your skill:

```yaml
---
name: editor-skill
description: VS Code specific skill
metadata:
  protocols: acp
  required_isolation: platform
---
```

### Platform Requirements

Indicate platform compatibility:

```yaml
---
name: macos-automation
description: Automate macOS tasks
compatibility: Requires macOS 10.15+
---
```

---

## CLI Command Reference

### List Skills
```bash
aps skill list                    # All skills
aps skill list --profile myagent  # Profile-specific
aps skill list --verbose          # Detailed info
```

### Show Skill Details
```bash
aps skill show <skill-name>
```

### Install Skill
```bash
aps skill install <path> --global              # Global install
aps skill install <path> --profile <id>        # Profile install
```

### Validate Skill
```bash
aps skill validate <path>
```

### Usage Statistics
```bash
aps skill stats                   # All profiles
aps skill stats --profile myagent # Specific profile
```

### Suggest IDE Paths
```bash
aps skill suggest                 # Show detected IDE paths
```

---

## Configuration

### Global Configuration

Edit `~/.config/aps/config.yaml`:

```yaml
skills:
  enabled: true

  # Additional search paths
  skill_sources:
    - /team/shared/skills

  # Auto-detect IDE skills
  auto_detect_ide_paths: false

  # Secret replacement
  secret_replacement:
    enabled: true
    local_models:
      - llama3.2:3b
    local_only: false

  # Usage tracking
  telemetry:
    enabled: true
    event_log: ~/.local/share/aps/skills/usage.jsonl
```

### Profile Configuration

Edit `<data>/profiles/<profile>/profile.yaml`:

```yaml
skills:
  enabled: true

  # Profile-specific paths
  skill_sources:
    - ~/myprofile-skills

  # Isolation requirements
  isolation_requirements:
    secure-skill: container
```

---

## Skill Metadata Reference

### Required Fields

```yaml
name: skill-name          # Lowercase, hyphens only, max 64 chars
description: Description  # Max 1024 chars, describes what & when
```

### Optional Fields

```yaml
license: MIT                                 # License identifier
compatibility: Requires Python 3.9+          # Environment requirements
metadata:
  author: your-team
  version: "1.0.0"
  protocols: acp,agent-protocol              # Supported protocols
  required_isolation: container              # Isolation requirement
allowed-tools: Bash(git:*) Read Write       # Pre-approved tools
```

---

## Troubleshooting

### Skill Not Found

**Problem:** `aps skill show my-skill` says "skill not found"

**Solutions:**
1. Check if skill directory exists:
   ```bash
   ls -la ~/.local/share/aps/skills/my-skill
   ```

2. Validate SKILL.md:
   ```bash
   aps skill validate ~/.local/share/aps/skills/my-skill
   ```

3. Check discovery paths:
   ```bash
   aps skill list --verbose
   ```

### Validation Errors

**Problem:** `aps skill validate` shows errors

**Common Issues:**
- Name contains uppercase or underscores (use lowercase + hyphens)
- Name doesn't match directory name
- Missing required fields (name, description)
- Description too long (max 1024 chars)

### Skills Not Auto-Discovered

**Problem:** IDE skills not showing up

**Solution:**
1. Check if paths exist:
   ```bash
   aps skill suggest
   ```

2. Enable auto-detection:
   ```yaml
   # ~/.config/aps/config.yaml
   skills:
     auto_detect_ide_paths: true
   ```

---

## Best Practices

### 1. Clear Naming
- Use descriptive names: `pdf-processing` not `tool1`
- Include domain: `legal-review`, `data-analysis`
- Avoid generic names: `helper`, `utils`

### 2. Good Descriptions
- Describe **what** it does: "Extract text from PDF files"
- Describe **when** to use: "Use when working with PDF documents"
- Include keywords: "PDF, extract, tables, forms"

### 3. Progressive Disclosure
- Keep SKILL.md under 500 lines
- Move detailed docs to `references/`
- Link to references in main SKILL.md

### 4. Reusability
- Avoid hardcoded paths
- Use environment variables
- Support multiple platforms when possible

### 5. Documentation
- Include usage examples
- Document script parameters
- List dependencies

---

## Examples

See `examples/skills/hello-world/` for a complete working example.

More examples:
- [Official Agent Skills Examples](https://github.com/anthropics/skills)
- [Community Skills](https://github.com/skillmatic-ai/awesome-agent-skills)

---

## Getting Help

- **Issues:** Report at [GitHub Issues](https://github.com/IdeaCraftersLabs/oss-aps-cli/issues)
- **Documentation:** `aps docs` (generates local docs)
- **Examples:** `examples/skills/` directory

---

## Next Steps

1. **Quick Start:** Follow the [Quickstart Guide](QUICKSTART.md)
2. **Create Skills:** Read [Creating Skills](CREATING_SKILLS.md)
3. **See Examples:** Check [Examples](EXAMPLES.md)

---

**Last Updated:** 2026-02-08
**Version:** 1.0
**Spec:** [agentskills.io](https://agentskills.io)
