# Docker Testing for APS Users

## What is Docker Testing?

Docker testing provides an isolated Linux environment to test APS installation and workflows without affecting your main system. This is especially useful for:

- Testing new APS versions before installing them on your main system
- Learning APS features in a safe, disposable environment
- Testing APS on Linux from macOS or Windows
- Reproducing and debugging issues in a clean environment
- Validating APS setup before deployment

## Quick Start

### Prerequisites

- Docker installed and running
- Basic knowledge of command line
- APS source code (for building binary)

### Option 1: Quick Start Script

The fastest way to get started:

```bash
cd oss-aps-cli
make docker-quick-start
```

This script will:
1. Check if Docker is running
2. Build the APS binary
3. Build the Docker test environment
4. Install APS in the container
5. Run quick verification tests

### Option 2: Manual Setup

For more control over the process:

```bash
# 1. Build APS binary
make build

# 2. Build Docker test image
make docker-build-test

# 3. Start the test container
make docker-test-up

# 4. Install APS in the container
make docker-test-install

# 5. Enter the container
make docker-test-shell
```

You'll now be inside a clean Linux environment with APS installed.

## Common Workflows

### Testing APS Installation

```bash
# Inside the test container
aps --help
aps version

# Verify binary location
which aps
```

### Creating and Managing Profiles

```bash
# Create a test profile
aps profile new test-agent --display-name "Test Agent"

# List all profiles
aps profile list

# Show profile details
aps profile show test-agent

# Update profile
aps profile update test-agent --display-name "Updated Test Agent"

# Delete profile
aps profile delete test-agent --force
```

### Running Commands

```bash
# Run a simple command
aps run test-agent -- echo "Hello from APS!"

# Check environment variables
aps run test-agent -- env | grep APS

# Run multiple commands
aps run test-agent -- sh -c "echo 'First' && echo 'Second'"

# Run in current directory
aps run test-agent -- pwd
```

### Testing Isolation Levels

```bash
# Process isolation (default)
aps profile new process-agent
aps run process-agent -- ps aux | grep $USER

# Platform isolation (Linux only)
aps profile new platform-agent --isolation-level platform
aps run platform-agent -- whoami

# Container isolation (requires Docker)
aps profile new container-agent --isolation-level container
aps run container-agent -- cat /etc/os-release
```

### Testing Capabilities

```bash
# Create profile with capabilities
aps profile new capable-agent \
    --add-capability shell \
    --add-capability execution \
    --add-capability development

# Test capability access
aps run capable-agent -- git --version
aps run capable-agent -- node --version
aps run capable-agent -- python3 --version
```

### Testing Secrets Management

```bash
# Create profile with secrets
aps profile new secret-agent --email "user@example.com"

# Add secrets
aps profile edit secret-agent
# Add your secrets in the editor

# Verify secrets are injected
aps run secret-agent -- env | grep APS_SECRET

# Or use test secrets from fixtures
aps run secret-agent -- cat /test-fixtures/secrets/test-secrets.env
```

### Testing Session Management

```bash
# Start a long-running session
aps run test-agent -- sleep 300 &
SESSION_ID=$(aps session list | tail -1 | cut -d' ' -f1)

# Inspect session
aps session inspect $SESSION_ID

# View session logs
aps session logs $SESSION_ID

# Terminate session
aps session terminate $SESSION_ID
```

### Testing Webhooks

```bash
# Create a simple webhook action
aps action new test-agent webhook-test --trigger "manual"

# Define action
aps action edit test-agent webhook-test
# Add your webhook configuration

# Test webhook trigger
aps action run test-agent webhook-test

# List actions
aps action list test-agent
```

## Interactive Testing

### Start an Interactive Session

```bash
# Enter container
make docker-test-shell

# Start an APS shell
aps test-agent

# You're now in an interactive shell under the profile
# Try commands:
pwd
ls -la
whoami
env

# Exit the profile shell
exit
```

### Testing TUI

```bash
# Launch APS TUI
aps

# Navigate the interface using arrow keys
# Test profile creation, editing, deletion
# Exit with q or Esc
```

### Debugging Issues

```bash
# Check APS logs
aps logs

# Check config directory
ls -la ~/.config/aps/

# Check profile structure
ls -la ~/.agents/profiles/test-agent/

# Verify XDG directories
echo $XDG_CONFIG_HOME
echo $XDG_DATA_HOME
```

## Automated Testing

### Run All User Journey Tests

```bash
# Run complete test suite
make docker-test-e2e-user
```

This will run all test scripts and provide a summary:

```
======================================
APS User Journey Test Suite
======================================

--------------------------------------
Running: 01-installation.sh
--------------------------------------
✓ 01-installation.sh PASSED

--------------------------------------
Running: 02-initial-setup.sh
--------------------------------------
✓ 02-initial-setup.sh PASSED

...

======================================
Test Summary
======================================
Total Tests: 5
Passed: 5
Failed: 0

✓ ALL TESTS PASSED
```

### Run Specific Test

```bash
# Run installation test only
docker compose -f docker-compose.test.yml run --rm test-env \
    /test-fixtures/scripts/01-installation.sh

# Run workflow execution test only
docker compose -f docker-compose.test.yml run --rm test-env \
    /test-fixtures/scripts/03-workflow-execution.sh
```

## Cleanup

### Stop Test Container

```bash
# Stop but keep volumes (preserves test data)
make docker-test-down

# Start again with same data
make docker-test-up
```

### Complete Cleanup

```bash
# Remove containers and volumes (reset to clean state)
make docker-test-down -v

# Or remove everything including Docker image
make docker-test-cleanup
```

### Cleanup Between Tests

```bash
# Remove all profiles
for profile in $(aps profile list | grep -v "^ID" | awk '{print $1}'); do
    aps profile delete $profile --force
done

# Or use cleanup test
docker compose -f docker-compose.test.yml run --rm test-env \
    /test-fixtures/scripts/05-cleanup-operations.sh
```

## Advanced Usage

### Custom Test Environments

```bash
# Modify Dockerfile.test for custom setup
vim Dockerfile.test

# Rebuild image
make docker-build-test

# Test with custom environment
make docker-test-shell
```

### Test with Real Data

```bash
# Mount your project directory
docker run -it --rm \
    -v $(pwd):/workspace \
    -v $(PWD)/bin/aps:/usr/local/bin/aps \
    aps-test-env \
    bash

# Inside container
cd /workspace
aps run my-agent -- npm install
```

### Network Testing

```bash
# Test network connectivity
aps run test-agent -- ping -c 3 google.com

# Test DNS resolution
aps run test-agent -- nslookup google.com

# Test external API calls
aps run test-agent -- curl https://api.github.com
```

### Performance Testing

```bash
# Measure setup time
time aps profile new perf-test

# Measure execution overhead
time aps run perf-test -- echo "test"

# Measure profile listing
time aps profile list
```

## Troubleshooting

### Container Won't Start

```bash
# Check Docker status
docker ps -a

# Check logs
docker logs aps-test-container

# Remove existing container
make docker-test-down
make docker-test-up
```

### Commands Not Working

```bash
# Verify APS is installed
which aps

# Check APS version
aps version

# Check permissions
ls -la $(which aps)

# Reinstall binary
make docker-test-install
```

### Profile Creation Fails

```bash
# Check XDG directories
ls -la ~/.config/aps/

# Check permissions
ls -la ~/.agents/

# Remove corrupted profile
rm -rf ~/.agents/profiles/test-agent

# Try again
aps profile new test-agent
```

### No Output Expected

```bash
# Some commands may not produce output
aps run test-agent -- true
# Exit code 0, but no output

# Check exit code
aps run test-agent -- false
echo $?  # Exit code 1
```

## Best Practices

### For Testing

1. **Start fresh**: Clean up between test iterations
2. **Use unique IDs**: Avoid conflicts in parallel testing
3. **Test incrementally**: Verify each feature before moving on
4. **Document failures**: Note any issues for debugging

### For Development

1. **Test in Docker first**: Before installing locally
2. **Use version control**: Keep track of changes
3. **Automate tests**: Create custom test scripts
4. **Share findings**: Report issues with reproduction steps

### For Learning

1. **Explore freely**: Docker environment is disposable
2. **Read the docs**: Check `aps docs` for generated documentation
3. **Experiment safely**: Try different isolation levels
4. **Ask questions**: Use help commands when unsure

## Resources

### Documentation

- [APS Documentation](../user/README.md) - Main documentation
- [CLI Reference](../user/CLI.md) - Complete command reference
- [Isolation Guide](../user/ISOLATION.md) - Isolation levels explained
- [Examples](../user/EXAMPLES.md) - Practical examples

### Development

- [Docker Testing Strategy](../dev/testing/docker-testing-strategy.md) - Developer testing strategy guide
- [GitHub Issues](https://github.com/IdeaCraftersLabs/oss-aps-cli/issues) - Report bugs

### Commands

```bash
# Get help
aps --help
aps profile --help
aps run --help

# Generate documentation
aps docs

# Shell integration
aps completion bash  # or zsh
eval "$(aps alias)"
```

## FAQ

**Q: Can I use Docker testing on macOS or Windows?**

A: Yes! The Docker test environment runs Linux regardless of your host OS.

**Q: Does Docker testing require a fast computer?**

A: No, but it may be slower than native execution. Allow extra time for container startup.

**Q: Can I persist data between Docker sessions?**

A: Yes, volumes preserve data until you run `make docker-test-down -v`.

**Q: Is Docker testing suitable for production use?**

A: No, it's for testing only. Use the binary directly in production.

**Q: Can I test APS with my own projects?**

A: Yes, mount your project directory into the container and use APS to run commands.

**Q: How do I report a bug found in Docker testing?**

A: Include your test steps, Docker version, and output from `aps version` in your GitHub issue.

---

**Last Updated**: January 30, 2026
**Version**: 1.0.0
