# Quickstart: Shell Integration

## 1. Setup Completion
Add to your shell config (e.g., `~/.zshrc`):
```bash
source <(aps completion zsh)
```

## 2. Setup Aliases
Add to your shell config:
```bash
eval "$(aps alias)"
```

## 3. Use Shorthands
```bash
# Start a session
aps agent-a

# Run a command
aps agent-a git status
```
