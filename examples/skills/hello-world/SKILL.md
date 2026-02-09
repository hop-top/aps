---
name: hello-world
description: A simple example skill that demonstrates the Agent Skills format. Use when you want to test skill execution or learn the skill structure.
license: MIT
metadata:
  author: APS Team
  version: "1.0.0"
  category: example
---

# Hello World Skill

This is a minimal Agent Skills example demonstrating the basic structure.

## Usage

This skill provides a simple greeting script that can be used to test skill execution.

### Scripts

#### hello.sh

Prints a greeting message.

**Usage:**
```bash
./scripts/hello.sh [name]
```

**Example:**
```bash
./scripts/hello.sh "Alice"
# Output: Hello, Alice!
```

#### greet-with-secret.sh

Demonstrates secret placeholder replacement.

**Usage:**
```bash
./scripts/greet-with-secret.sh
```

This script expects an API_KEY secret to be available via `${SECRET:API_KEY}`.

## Files

- `scripts/hello.sh` - Basic greeting script
- `scripts/greet-with-secret.sh` - Demonstrates secret injection
- `references/REFERENCE.md` - Additional documentation

## Testing

Run the validation:
```bash
aps skill validate examples/skills/hello-world
```

Install and test:
```bash
aps skill install examples/skills/hello-world --global
aps skill run hello-world -- hello.sh "World"
```
