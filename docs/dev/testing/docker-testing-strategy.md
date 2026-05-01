# Docker Testing Strategy

## Overview

Docker testing provides an isolated Linux environment for testing APS installation, setup, and workflows from a user's perspective. This approach ensures that APS works correctly on a clean Linux machine without affecting the developer's local environment.

## Why Docker Testing?

### Benefits

- **Isolation**: Clean environment for every test run
- **Realism**: Simulates actual user machine setup
- **Reproducibility**: Consistent test conditions across all developers
- **Linux Testing**: Tests on target Linux platform regardless of host OS
- **No Local Impact**: Keeps local development environment clean
- **Easy Cleanup**: One-command cleanup of all test artifacts

### Use Cases

1. **User Journey Testing**: Simulate complete user workflows
2. **Installation Testing**: Verify binary installation and setup
3. **Cross-Platform Testing**: Test Linux behavior from macOS/Windows hosts
4. **CI/CD Integration**: Automated testing in GitHub Actions
5. **Manual Testing**: Interactive environment for debugging

## Architecture

### Test Environment Structure

```
Docker Container (aps-test-env)
├── /home/testuser/              # User home directory (persistent volume)
│   ├── .config/aps/            # APS config (persistent volume)
│   └── .local/                 # Local data
├── /host-src/                  # Project source (read-only mount)
│   ├── bin/aps                 # Built binary
│   └── tests/fixtures/         # Test fixtures
└── /test-fixtures/             # Test fixtures (read-only mount)
```

### Key Components

1. **Base Image**: Ubuntu 22.04 (matches default container isolation base)
2. **User**: Non-root user `testuser` for realistic environment
3. **Volumes**: Three persistent volumes for state management
4. **Mounts**: Read-only mounts for project source and fixtures

## Test Strategy

### Test Categories

#### 1. Installation Tests

**Objective**: Verify binary installation and basic availability

**Tests**:
- Binary exists in PATH
- Version command works
- Help command works
- Binary is executable

**Script**: `tests/fixtures/scripts/01-installation.sh`

#### 2. Initial Setup Tests

**Objective**: Verify APS first-time setup and profile creation

**Tests**:
- XDG directory structure creation
- Profile creation with various options
- Profile listing
- Profile details display
- Profile deletion

**Script**: `tests/fixtures/scripts/02-initial-setup.sh`

#### 3. Workflow Execution Tests

**Objective**: Verify command execution and environment isolation

**Tests**:
- Basic command execution
- Environment variable injection (APS_PROFILE_ID)
- Multiple command chaining
- Output verification

**Script**: `tests/fixtures/scripts/03-workflow-execution.sh`

#### 4. Configuration Management Tests

**Objective**: Verify profile configuration updates

**Tests**:
- Profile updates (capabilities, settings)
- Configuration persistence
- Profile listing with filters

**Script**: `tests/fixtures/scripts/04-configuration-management.sh`

#### 5. Cleanup Operations Tests

**Objective**: Verify profile cleanup and deletion

**Tests**:
- Multiple profile creation and deletion
- Profile verification after deletion
- Config directory cleanup

**Script**: `tests/fixtures/scripts/05-cleanup-operations.sh`

### Test Execution

#### Automated

```bash
# Run complete test suite
make docker-test-e2e-user
```

This executes all test scripts in sequence with a summary report.

#### Manual

```bash
# Start test container
make docker-test-up

# Install binary
make docker-test-install

# Enter container
make docker-test-shell

# Run manual tests
aps --help
aps profile create test-agent
aps run test-agent -- echo "Hello"

# Exit and cleanup
exit
make docker-test-down
```

## Makefile Integration

### Available Targets

| Target | Description |
|--------|-------------|
| `docker-build-test` | Build Docker test image |
| `docker-test-up` | Start test container in background |
| `docker-test-down` | Stop and remove test containers |
| `docker-test-shell` | Start interactive shell in test container |
| `docker-test-install` | Install built binary in test container |
| `docker-test-e2e-user` | Run full user journey test suite |
| `docker-test-cleanup` | Clean up all Docker test artifacts |
| `docker-quick-start` | Quick start guide for first-time users |

### Workflow

Typical testing workflow:

```bash
# 1. Build local binary
make build

# 2. Build test image (first time only)
make docker-build-test

# 3. Install binary in container
make docker-test-install

# 4. Run tests
make docker-test-e2e-user

# 5. Cleanup (optional)
make docker-test-cleanup
```

## Test Fixtures

### Profile Configurations

Location: `tests/fixtures/profiles/`

- **basic-profile.yaml**: Simple profile with basic capabilities
- **dev-profile.yaml**: Development profile with additional tools

### Secrets

Location: `tests/fixtures/secrets/`

- **test-secrets.env**: Non-sensitive test secrets for testing secret management

### Scripts

Location: `tests/fixtures/scripts/`

All scripts are executable and follow these conventions:
- `set -e` for error handling
- Clear output with section markers
- Return 0 on success, non-zero on failure
- Cleanup of test artifacts

## CI/CD Integration

### GitHub Actions Workflow

File: `.github/workflows/docker-user-journey.yml`

**Job**: `docker-user-journey`

**Platform**: Ubuntu latest

**Steps**:
1. Checkout code
2. Set up Go
3. Download dependencies
4. Build binary
5. Build test image
6. Install binary in container
7. Run user journey tests
8. Cleanup (always)

### Triggering

Automatic:
- Push to main/develop branches
- Pull requests to main/develop branches

Manual:
- `gh workflow run docker-user-journey.yml`

## Troubleshooting

### Common Issues

#### Docker Daemon Not Running

**Error**: `Cannot connect to Docker daemon`

**Solution**:
```bash
# macOS/Windows: Start Docker Desktop
# Linux: sudo systemctl start docker
docker info
```

#### Binary Not Found

**Error**: `aps: command not found`

**Solution**:
```bash
# Build binary locally
make build

# Install in container
make docker-test-install
```

#### Container Won't Start

**Error**: Container exists or conflicts

**Solution**:
```bash
# Stop and remove existing container
make docker-test-down

# Try again
make docker-test-up
```

#### Volume Persistence Issues

**Issue**: Old test data interfering

**Solution**:
```bash
# Complete cleanup with volume removal
make docker-test-down -v

# Or full cleanup
make docker-test-cleanup
```

### Debug Mode

For debugging test failures:

1. Run individual test script:
   ```bash
   docker compose -f docker-compose.test.yml run --rm test-env \
       /test-fixtures/scripts/02-initial-setup.sh
   ```

2. Enter container and run manually:
   ```bash
   make docker-test-shell
   bash /test-fixtures/scripts/02-initial-setup.sh
   ```

3. Check container logs:
   ```bash
   docker logs aps-test-container
   ```

## Performance Considerations

### Build Times

- **First build**: ~2-3 minutes (includes image build)
- **Subsequent builds**: ~30 seconds (binary only)
- **Test execution**: ~10-20 seconds for full suite

### Resource Usage

- **Memory**: ~500MB (container overhead + APS)
- **Disk**: ~2GB (Ubuntu base + APS + dependencies)
- **CPU**: Minimal during idle, moderate during builds

### Optimization Tips

1. **Reuse volumes**: Keep containers running for multiple test runs
2. **Parallel testing**: Use `t.Parallel()` in test scripts (with unique IDs)
3. **Minimal base**: Use Alpine Linux if Ubuntu isn't required
4. **Layer caching**: Optimize Dockerfile for layer reuse

## Maintenance

### Updating Test Environment

When APS requirements change:

1. Update `Dockerfile.test` if dependencies change
2. Update test fixtures if profile schema changes
3. Add new test scripts for new features
4. Update this documentation

### Version Management

- Tag test images: `docker build -t aps-test-env:v1.0.0`
- Use specific versions: Update `docker-compose.test.yml`
- Track changes in release notes

## Best Practices

### For Developers

1. **Always test locally** before pushing
2. **Run full suite** after significant changes
3. **Clean up** between major test iterations
4. **Document** any new test scenarios

### For Test Scripts

1. **Use unique IDs** to avoid conflicts in parallel tests
2. **Cleanup after** each test
3. **Provide clear output** for debugging
4. **Return proper exit codes** for automation

### For CI/CD

1. **Cache dependencies** to speed up builds
2. **Run on all platforms** (Linux, macOS, Windows)
3. **Cleanup artifacts** to save space
4. **Fail fast** on first error

## Related Documentation

- [Docker Testing for Users](../../agent/docker-testing.md) - User-facing testing guide
- [Container Platform Overview](../platforms/container/overview.md) - Container isolation with testing section
- [CI/CD Setup](../operations/cicd/ci-cd-setup.md) - CI/CD integration

## Future Enhancements

Potential improvements:

1. **Multi-container testing**: Test agent-to-agent communication
2. **Service mocking**: Test webhooks and external services
3. **Performance testing**: Benchmark within Docker environment
4. **Security testing**: Test isolation boundaries
5. **Network testing**: Test network configurations and DNS

---

**Last Updated**: January 30, 2026
**Version**: 1.0.0
