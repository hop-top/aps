# APS Helper Scripts

Utility scripts to help with common APS tasks.

## Setup Scripts

### `setup-telegram.sh`
Interactive setup for Telegram messenger integration with APS profiles.

```bash
./scripts/setup-telegram.sh
```

**What it does:**
1. Creates or selects an agent profile
2. Sets up a Telegram messenger (bot)
3. Configures channel-to-action mappings
4. Stores your Telegram token securely
5. Starts the messenger service

**No arguments** - fully interactive and guided.

**See also:** `docs/TELEGRAM_SETUP.md` for detailed documentation.

---

### `setup-messenger.sh`
Generic setup script for any messenger platform (Telegram, Slack, GitHub, Email).

```bash
# Interactive mode (asks for all details)
./scripts/setup-messenger.sh

# With specific messenger type
./scripts/setup-messenger.sh --type=telegram

# Quick setup (predefined values)
./scripts/setup-messenger.sh --type=slack --profile=my-bot --messenger=slack-bot --yes
```

**Options:**
- `--type=TYPE` — Messenger type: `telegram`, `slack`, `github`, `email`
- `--profile=ID` — Skip profile selection, use this profile
- `--messenger=NAME` — Skip naming, use this messenger name
- `--yes, -y` — Skip confirmation prompts
- `--help` — Show full help with examples

**Supports:**
- ✓ Telegram
- ✓ Slack
- ✓ Discord
- ✓ GitHub (webhooks)
- ✓ Email

**See also:** `docs/MESSENGER_SETUP_QUICK_REF.md` for quick reference.

---

## Testing Scripts

### `test-stories.sh`
Run all Go tests referenced in the user story documentation.

```bash
# Run all story tests
./scripts/test-stories.sh

# Run tests for specific stories
./scripts/test-stories.sh 001 003 005

# List tests without running
./scripts/test-stories.sh --list
```

**What it does:**
- Parses all files in `docs/stories/*.md`
- Extracts test function names from the "## Tests" sections
- Groups tests by Go package
- Runs them with `go test -run`
- Provides a summary of pass/fail

**Useful for:**
- Verifying story implementations
- Running subset of tests by story
- CI/CD integration

---

### `check-links.sh`
Validate documentation links and test file references.

```bash
./scripts/check-links.sh
```

**Checks:**
1. Markdown links (using `lychee`)
2. Referenced test files exist

**Useful for:**
- Documentation maintenance
- Pre-commit validation
- CI/CD checks

---

## Quick Messenger Setups

### Telegram (Recommended for First-Time Users)
```bash
./scripts/setup-telegram.sh
```
Guided setup with detailed explanations for each step.

### Quick Slack
```bash
./scripts/setup-messenger.sh --type=slack
```

### Quick Discord
```bash
./scripts/setup-messenger.sh --type=discord
```

### Quick GitHub
```bash
./scripts/setup-messenger.sh --type=github
```

### All Options (One Command)
```bash
./scripts/setup-messenger.sh \
  --type=telegram \
  --profile=my-agent \
  --messenger=my-telegram \
  --yes
```

---

## Common Workflows

### Setup a New Telegram Integration
```bash
./scripts/setup-telegram.sh
```
Then:
```bash
aps messengers logs my-telegram -f
```

### Migrate from Manual Setup
If you've already created profiles/messengers manually:
```bash
# Reset and redo with script
aps messengers delete my-old-telegram
./scripts/setup-messenger.sh --type=telegram --profile=existing-profile
```

### Test a Story Implementation
```bash
./scripts/test-stories.sh 050  # Test story 050
```

### Pre-commit Documentation Check
```bash
./scripts/check-links.sh  # Verify all docs are correct
```

---

## Script Development Guide

All scripts follow consistent patterns:

1. **Colors** — Terminal output uses colors (disabled for non-TTY)
2. **Error Handling** — `set -euo pipefail` for safety
3. **Helper Functions** — Reusable `print_*` and `prompt*` functions
4. **Logging** — Clear progress messages for users
5. **Flexibility** — Support both interactive and non-interactive modes

### Adding a New Setup Script

If you need to create a new setup script:

1. Copy pattern from `setup-telegram.sh` or `setup-messenger.sh`
2. Use the helper functions (`print_*`, `prompt*`)
3. Add `--help` option for documentation
4. Test both interactive and flag-based modes
5. Add to this README

Example skeleton:
```bash
#!/usr/bin/env bash
set -euo pipefail

# Setup colors
if [ -t 1 ]; then
  GREEN='\033[0;32m' RED='\033[0;31m' RESET='\033[0m'
else
  GREEN='' RED='' RESET=''
fi

# Helper functions (copy from existing scripts)
print_success() { echo -e "${GREEN}✓${RESET}  $1"; }
print_error() { echo -e "${RED}✗${RESET}  $1" >&2; }

# Implementation
main() {
  print_success "Setup complete"
}

main "$@"
```

---

## Troubleshooting Scripts

### "Permission denied"
```bash
chmod +x scripts/*.sh
```

### Script hangs on prompt
Press `Ctrl+C` to cancel and start over.

### "Command not found" errors
```bash
# Ensure aps CLI is installed and in PATH
go install ./cmd/aps
export PATH=$PATH:$(go env GOPATH)/bin

# Then retry
./scripts/setup-messenger.sh
```

### Messenger setup fails silently
```bash
# Run with verbose output
bash -x ./scripts/setup-messenger.sh

# Or check prerequisites manually
aps --version
aps profile list
```

---

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Validate Docs
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: ./scripts/check-links.sh
      - run: ./scripts/test-stories.sh
```

### Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit
./scripts/check-links.sh || exit 1
```

---

## Performance Notes

- **`setup-telegram.sh`** — <10 seconds (depends on download speed for first run)
- **`setup-messenger.sh`** — <10 seconds (generic version, slightly faster)
- **`test-stories.sh`** — Varies by test count (~30 seconds for ~50 tests)
- **`check-links.sh`** — <5 seconds (faster if `lychee` installed)

---

## Contributing

When adding new scripts:
1. Follow the existing style and patterns
2. Add comprehensive help text (`--help`)
3. Support both interactive and CLI flag modes
4. Test on both Linux and macOS
5. Update this README with usage instructions
