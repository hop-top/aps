# Agent Skills - Quickstart Guide

Get started with Agent Skills in 5 minutes!

## Step 1: Check If Skills Are Enabled

```bash
aps skill list
```

If this works, skills are enabled! Skip to Step 3.

If not, enable skills:

```bash
# Create config if it doesn't exist
mkdir -p ~/.config/aps
cat > ~/.config/aps/config.yaml << 'EOF'
skills:
  enabled: true
EOF
```

---

## Step 2: Install Example Skill

Let's install the hello-world example:

```bash
# Clone or copy the example
cp -r examples/skills/hello-world ~/.local/share/aps/skills/

# Verify it's there
aps skill list
```

You should see:
```
Found 1 skill(s):

Global (1):
  hello-world                   A simple example skill
```

---

## Step 3: View Skill Details

```bash
aps skill show hello-world
```

**Output:**
```
Name:          hello-world
Description:   A simple example skill for demonstration
License:       MIT

Scripts:
  • hello.sh
  • greet-with-secret.sh

References:
  • REFERENCE.md
```

---

## Step 4: Create Your First Skill

### Create the directory:

```bash
mkdir -p ~/.local/share/aps/skills/my-quick-skill
cd ~/.local/share/aps/skills/my-quick-skill
```

### Create SKILL.md:

```bash
cat > SKILL.md << 'EOF'
---
name: my-quick-skill
description: My first Agent Skill for quick tasks
license: MIT
---

# My Quick Skill

This is my first skill!

## What it does

Helps me with quick text processing tasks.

## Usage

Use this skill when you need to:
- Convert text to uppercase
- Count words in a file
- Find and replace text
EOF
```

### Add a script:

```bash
mkdir scripts

cat > scripts/uppercase.sh << 'EOF'
#!/bin/bash
# Convert text to uppercase

if [ -z "$1" ]; then
    echo "Usage: uppercase.sh <input-file>"
    exit 1
fi

tr '[:lower:]' '[:upper:]' < "$1"
EOF

chmod +x scripts/uppercase.sh
```

---

## Step 5: Validate Your Skill

```bash
aps skill validate ~/.local/share/aps/skills/my-quick-skill
```

**Expected output:**
```
✓ Valid Agent Skill
  Name:        my-quick-skill
  Description: My first Agent Skill for quick tasks
```

---

## Step 6: See It in Action

```bash
aps skill list
```

**Output:**
```
Found 2 skill(s):

Global (2):
  hello-world                   A simple example skill
  my-quick-skill                My first Agent Skill for quick tasks
```

Your skill is now available to your agent!

---

## Step 7: Use It

Now when you work with your agent, it can discover and use your skill automatically!

Example conversation:
```
You: I have a file with lowercase text. Can you convert it to uppercase?

Agent: I'll use the my-quick-skill for this. Let me run the uppercase script.
       [Agent executes scripts/uppercase.sh]
```

---

## Next Steps

### Create Profile-Specific Skills

```bash
# For a specific profile
mkdir -p ~/.agents/profiles/myagent/skills/team-skill

# Create SKILL.md there
# These skills are only available to 'myagent' profile
```

### Add More Advanced Features

1. **Add References:**
   ```bash
   mkdir references
   echo "# Detailed Documentation" > references/REFERENCE.md
   ```

2. **Add Templates:**
   ```bash
   mkdir assets
   echo '{"template": "data"}' > assets/template.json
   ```

3. **Use Secrets:**
   ```bash
   # In your script:
   API_KEY="${SECRET:API_KEY}"
   ```

### Enable IDE Auto-Detection

```bash
# See what IDEs have skills
aps skill suggest

# Enable auto-detection
cat >> ~/.config/aps/config.yaml << 'EOF'
  auto_detect_ide_paths: true
EOF
```

---

## Common Commands

```bash
# List all skills
aps skill list

# List with details
aps skill list --verbose

# Show specific skill
aps skill show <skill-name>

# Validate before installing
aps skill validate <path>

# Install globally
aps skill install <path> --global

# Install to profile
aps skill install <path> --profile myagent

# View usage stats
aps skill stats
```

---

## Tips

### Tip 1: Good Descriptions Matter

✅ **Good:**
```yaml
description: Extract text and tables from PDF files. Use when working with PDF documents or forms.
```

❌ **Bad:**
```yaml
description: PDF stuff
```

### Tip 2: Name Your Skills Clearly

✅ **Good:**
- `pdf-processing`
- `data-analysis`
- `code-review`

❌ **Bad:**
- `tool1`
- `helper`
- `utils`

### Tip 3: Keep SKILL.md Focused

- Main instructions: < 500 lines in SKILL.md
- Detailed docs: Move to `references/`
- Examples: Include in SKILL.md body
- Technical details: Put in REFERENCE.md

### Tip 4: Test Your Scripts

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Test them manually first
./scripts/my-script.sh test-input.txt
```

---

## Troubleshooting

### "Skill not found" Error

```bash
# Check the directory exists
ls ~/.local/share/aps/skills/

# Check SKILL.md exists
ls ~/.local/share/aps/skills/my-skill/SKILL.md

# Validate
aps skill validate ~/.local/share/aps/skills/my-skill
```

### Validation Fails

**Common fixes:**
- Name must be lowercase with hyphens only
- Name must match directory name
- Both name and description are required
- Check for typos in frontmatter

### Skills Not Showing Up

```bash
# Refresh the list
aps skill list

# Check verbose output
aps skill list --verbose

# Verify config
cat ~/.config/aps/config.yaml
```

---

## Example: Complete Skill

Here's a complete, production-ready skill:

**Directory structure:**
```
text-processor/
├── SKILL.md
├── scripts/
│   ├── uppercase.sh
│   ├── lowercase.sh
│   └── word-count.sh
├── references/
│   └── REFERENCE.md
└── assets/
    └── stopwords.txt
```

**SKILL.md:**
```yaml
---
name: text-processor
description: Process text files with common transformations. Use for text conversion, counting, and cleanup tasks.
license: MIT
metadata:
  author: my-team
  version: "1.0.0"
---

# Text Processor

Common text processing operations.

## Available Scripts

### uppercase.sh
Converts text to uppercase.

```bash
./scripts/uppercase.sh input.txt > output.txt
```

### lowercase.sh
Converts text to lowercase.

### word-count.sh
Counts words, lines, and characters.

```bash
./scripts/word-count.sh document.txt
```

## Examples

Convert a file to uppercase:
```bash
./scripts/uppercase.sh report.txt > REPORT.txt
```

Get word count:
```bash
./scripts/word-count.sh article.md
# Output: Words: 1523, Lines: 87, Chars: 9821
```
```

---

## You're Ready!

You've now:
- ✅ Installed skills
- ✅ Created your first skill
- ✅ Validated it
- ✅ Seen it in the skill list

Your agent can now discover and use your skills automatically!

---

## What's Next?

- Read the [full User Guide](README.md) for more details
- Check [Creating Skills Guide](CREATING_SKILLS.md) for advanced features
- Browse [Examples](EXAMPLES.md) for inspiration
- Share your skills with your team!

---

**Need Help?**
- Report issues: [GitHub Issues](https://github.com/IdeaCraftersLabs/oss-aps-cli/issues)
- Documentation: Run `aps docs` to generate local docs

---

**Last Updated:** 2026-02-08
