# Quickstart: Building the Core

## Prerequisites
- Go 1.22+
- `make` (optional)

## Build
```bash
make build-local
```

## Verify CLI
```bash
./bin/aps help
./bin/aps profile list
```

## Run the TUI
```bash
./bin/aps
```
*(Should show Bubble Tea interface)*

## Create a Test Profile
```bash
./aps profile new test-agent --display-name "Test Agent"
```

## Run a Command
```bash
./aps run test-agent -- echo "Hello from Agent"
```
*(Should output "Hello from Agent" with environment injected)*
