# Hello World Skill Reference

## Overview

This is a reference document that provides additional details about the hello-world skill.

## Architecture

The skill consists of two simple bash scripts that demonstrate:
1. Basic script execution
2. Secret placeholder replacement

## Environment Variables

The skill expects the following environment variables:

- `SECRET:API_KEY` - An API key for demonstration purposes

## Exit Codes

All scripts use standard exit codes:
- `0` - Success
- `1` - Error

## Examples

### Example 1: Basic Greeting

```bash
aps skill run hello-world -- hello.sh "Alice"
```

**Output:**
```
Hello, Alice!
This is the hello-world skill from APS.
```

### Example 2: Greeting with Secret

```bash
# Ensure API_KEY is set in profile secrets
aps skill run hello-world -- greet-with-secret.sh
```

**Output:**
```
Greeting with authentication!
API Key (first 5 chars): sk-12...
Secret replacement working!
```

## Troubleshooting

**Q: Script permission denied**

A: Make scripts executable:
```bash
chmod +x examples/skills/hello-world/scripts/*.sh
```

**Q: Secret not found**

A: Add the secret to your profile:
```bash
echo "API_KEY=sk-1234567890abcdef" >> ~/.agents/profiles/<profile>/secrets.env
```
