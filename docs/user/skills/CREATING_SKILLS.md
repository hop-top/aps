# Creating Agent Skills

Complete guide to authoring your own Agent Skills.

## Skill Anatomy

A skill is a directory containing:
- **SKILL.md** (required) - Instructions and metadata
- **scripts/** (optional) - Executable code
- **references/** (optional) - Detailed documentation
- **assets/** (optional) - Templates, images, data files

---

## SKILL.md Format

### Minimal Example

```yaml
---
name: my-skill
description: Brief description of what this skill does and when to use it
---

# My Skill

Instructions for the agent go here.
```

### Complete Example

```yaml
---
name: advanced-skill
description: Complete example with all fields
license: MIT
compatibility: Requires Python 3.9+, pandas
metadata:
  author: my-team
  version: "2.1.0"
  protocols: acp,agent-protocol
  required_isolation: platform
allowed-tools: Bash(git:*) Read Write
---

# Advanced Skill

Full instructions with examples...
```

---

## Field Reference

### name (required)
- **Format:** Lowercase letters, numbers, hyphens only
- **Length:** 1-64 characters
- **Rules:** No leading/trailing hyphens, no consecutive hyphens
- **Must match:** Directory name exactly

✅ Valid: `pdf-processing`, `data-analysis-v2`, `code-review`
❌ Invalid: `PDF_Processing`, `data--analysis`, `-my-skill`

### description (required)
- **Length:** 1-1024 characters
- **Content:** What the skill does + when to use it
- **Include:** Keywords the agent can match

✅ Good:
```yaml
description: Extract text and tables from PDF files, fill forms, merge documents. Use when working with PDF documents or when user mentions PDFs, forms, or document extraction.
```

❌ Bad:
```yaml
description: PDF stuff
```

### license (optional)
- License identifier or filename
- Examples: `MIT`, `Apache-2.0`, `Proprietary - see LICENSE.txt`

### compatibility (optional)
- Environment requirements
- Max 500 characters
- Include: Required tools, OS, versions

```yaml
compatibility: Requires macOS 10.15+, Xcode Command Line Tools, and git
```

### metadata (optional)
- Key-value pairs
- Custom fields for your needs

```yaml
metadata:
  author: team-name
  version: "1.2.0"
  protocols: acp,a2a
  required_isolation: container
  category: data-processing
```

### allowed-tools (optional, experimental)
- Pre-approved tools the skill may use
- Space-delimited list

```yaml
allowed-tools: Bash(git:*) Bash(jq:*) Read Write
```

---

## Body Content

After the frontmatter, write instructions in Markdown.

### Structure Recommendations

```markdown
# Skill Name

Brief overview of what this skill does.

## When to Use

- Use case 1
- Use case 2
- Use case 3

## How It Works

1. Step 1
2. Step 2
3. Step 3

## Scripts

### script-name.py
Description of what it does.

**Usage:**
\```bash
python scripts/script-name.py --input file.txt
\```

**Parameters:**
- `--input`: Input file path
- `--output`: Output file path (optional)

## Examples

### Example 1: Basic Usage
\```bash
./scripts/process.sh data.csv
\```

### Example 2: Advanced Usage
\```bash
./scripts/process.sh --format json data.csv output.json
\```

## Troubleshooting

**Problem:** Script fails with X error
**Solution:** Check that Y is installed

## References

See [REFERENCE.md](references/REFERENCE.md) for detailed API documentation.
```

---

## Adding Scripts

### Script Guidelines

1. **Make executable:**
   ```bash
   chmod +x scripts/*.sh
   ```

2. **Add shebang:**
   ```bash
   #!/bin/bash
   # or
   #!/usr/bin/env python3
   ```

3. **Handle errors:**
   ```bash
   set -e  # Exit on error
   ```

4. **Validate inputs:**
   ```bash
   if [ -z "$1" ]; then
       echo "Usage: script.sh <input>"
       exit 1
   fi
   ```

5. **Provide helpful output:**
   ```bash
   echo "Processing $1..."
   # do work
   echo "✓ Complete! Output: $output_file"
   ```

### Script Example

**scripts/process-data.sh:**
```bash
#!/bin/bash
set -e

# Validate input
if [ -z "$1" ]; then
    echo "Usage: process-data.sh <input-file> [output-file]"
    exit 1
fi

INPUT_FILE="$1"
OUTPUT_FILE="${2:-output.txt}"

# Check file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: File not found: $INPUT_FILE"
    exit 1
fi

# Process
echo "Processing $INPUT_FILE..."
cat "$INPUT_FILE" | tr '[:lower:]' '[:upper:]' > "$OUTPUT_FILE"
echo "✓ Done! Output: $OUTPUT_FILE"
```

---

## Using Secrets

### Placeholder Format

```bash
${SECRET:KEY_NAME}
```

### In Scripts

```bash
#!/bin/bash

# Secret will be replaced at execution time
API_KEY="${SECRET:API_KEY}"
DATABASE_URL="${SECRET:DATABASE_URL}"

# Use in commands
curl -H "Authorization: Bearer $API_KEY" \
     https://api.example.com/data
```

### In SKILL.md

```markdown
## Authentication

This skill requires an API key. Store it in your profile secrets:

\```bash
echo "API_KEY=your-key-here" >> ~/.agents/profiles/<profile>/secrets.env
\```

The script will automatically use: `${SECRET:API_KEY}`
```

---

## Adding References

For detailed documentation, create `references/REFERENCE.md`:

```markdown
# Skill Name Reference

## Architecture

Detailed architecture explanation...

## API Documentation

### Function: processData(input)
...

## Advanced Examples

...
```

Link from SKILL.md:
```markdown
See [REFERENCE.md](references/REFERENCE.md) for detailed API documentation.
```

---

## Adding Assets

Store templates, data files, images in `assets/`:

```
assets/
├── template.json
├── schema.yaml
└── example-data.csv
```

Reference in SKILL.md:
```markdown
## Templates

Use the provided template:
\```bash
cp assets/template.json my-config.json
\```
```

---

## Best Practices

### 1. Progressive Disclosure

Keep SKILL.md focused and under 500 lines:
- **SKILL.md:** Overview, quick start, common examples
- **references/:** Detailed docs, API reference, advanced topics
- **Link:** Connect them with relative links

### 2. Clear Instructions

```markdown
✅ Good:
## Usage

Run the extraction script with your PDF file:

\```bash
python scripts/extract.py input.pdf
\```

This will create `input_extracted.txt` with the text content.

❌ Bad:
## Usage
Run the script.
```

### 3. Include Examples

Real examples > abstract descriptions

```markdown
## Examples

### Extract text from a PDF
\```bash
python scripts/extract.py report.pdf
# Output: report_extracted.txt
\```

### Extract tables to CSV
\```bash
python scripts/extract.py --tables report.pdf
# Output: report_tables.csv
\```
```

### 4. Document Dependencies

```markdown
## Requirements

This skill requires:
- Python 3.9+
- pandas library: `pip install pandas`
- jq command-line tool
```

### 5. Error Handling

```markdown
## Troubleshooting

**Error: "Module not found: pandas"**
Install dependencies: `pip install pandas`

**Error: "Permission denied"**
Make scripts executable: `chmod +x scripts/*.sh`
```

---

## Testing Your Skill

### 1. Validate Format

```bash
aps skill validate /path/to/my-skill
```

### 2. Test Scripts Manually

```bash
cd ~/.local/share/aps/skills/my-skill
./scripts/test-script.sh sample-input.txt
```

### 3. Check It Appears

```bash
aps skill list
aps skill show my-skill
```

### 4. Try With Agent

Start a task that should trigger your skill and see if the agent discovers it.

---

## Advanced Topics

### Multi-Protocol Skills

Target specific protocols:

```yaml
metadata:
  protocols: acp,agent-protocol  # Not A2A
```

The skill will only appear when accessed via ACP or Agent Protocol.

### Isolation Requirements

Require specific isolation:

```yaml
metadata:
  required_isolation: container
```

Options: `process`, `platform`, `container`

### Platform-Specific Skills

```yaml
compatibility: macOS only - requires AppleScript
```

Or in metadata:
```yaml
metadata:
  platforms: darwin  # macOS only
```

---

## Skill Templates

### Data Processing Skill

```yaml
---
name: data-processor
description: Process and analyze data files
---

# Data Processor

## Input Formats
- CSV
- JSON
- Excel (.xlsx)

## Operations
- Filter rows
- Aggregate data
- Generate reports

## Scripts

### clean-data.py
Remove duplicates and fix formatting.

### analyze-data.py
Generate statistics and insights.
```

### API Integration Skill

```yaml
---
name: api-client
description: Interact with Company API
---

# Company API Client

## Authentication
Uses API key: `${SECRET:COMPANY_API_KEY}`

## Endpoints

### GET /users
Fetch user list.

### POST /orders
Create new order.

## Scripts

### fetch-users.sh
Downloads all users to users.json

### create-order.sh
Creates an order from a JSON file
```

---

## Publishing Skills

### Share With Team

```bash
# Create team directory
mkdir /team/shared/skills

# Copy your skill
cp -r ~/.local/share/aps/skills/my-skill /team/shared/skills/

# Team members add to config:
# ~/.config/aps/config.yaml
skills:
  skill_sources:
    - /team/shared/skills
```

### Package for Distribution

```bash
# Create archive
cd ~/.local/share/aps/skills
tar -czf my-skill-v1.0.tar.gz my-skill/

# Others can install:
tar -xzf my-skill-v1.0.tar.gz
aps skill install ./my-skill --global
```

---

## Checklist

Before publishing your skill:

- [ ] Name is lowercase with hyphens only
- [ ] Name matches directory name
- [ ] Description is clear and includes keywords
- [ ] Scripts are executable (`chmod +x`)
- [ ] Scripts have proper shebangs
- [ ] Examples are included
- [ ] Dependencies are documented
- [ ] SKILL.md is under 500 lines
- [ ] Detailed docs are in references/
- [ ] `aps skill validate` passes
- [ ] Scripts work when tested manually

---

## Need Help?

- **Spec:** [agentskills.io/specification](https://agentskills.io/specification)
- **Examples:** `examples/skills/` directory
- **Issues:** [GitHub Issues](https://github.com/IdeaCraftersLabs/oss-aps-cli/issues)

---

**Last Updated:** 2026-02-08
