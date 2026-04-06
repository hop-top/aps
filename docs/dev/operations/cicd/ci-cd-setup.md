# CI/CD Configuration Guide

This document describes the CI/CD setup for the APS CLI project, including GitHub Actions workflows, runner configuration, and artifact management.

## Workflows Overview

### Main Workflows

| Workflow | Purpose | Triggers |
|----------|---------|----------|
| **ci.yml** | Main CI pipeline with test/build matrix | Push to main/develop, Pull requests |
| **platform-adapter-tests.yml** | Platform-specific adapter tests | Push to main/develop/adapter/*, Pull requests |
| **coverage.yml** | Coverage reporting and artifact collection | Push to main/develop, Pull requests |
| **security.yml** | Security scanning (gosec, trivy, codeql) | Push, PRs, Weekly schedule |

## CI Workflow (ci.yml)

### Platform Matrix

The CI workflow runs on three platforms:

| Platform | Runner Image | Go Version |
|----------|--------------|------------|
| Linux | ubuntu-latest | 1.25.5 |
| macOS | macos-latest | 1.25.5 |
| Windows | windows-latest | 1.25.5 |

### Test Jobs

#### Unit Tests
```bash
go test -v -coverprofile=coverage.out -covermode=atomic ./tests/unit/...
```

#### E2E Tests
```bash
# Linux/macOS
go test -v -coverprofile=coverage-e2e.out -covermode=atomic ./tests/e2e

# Windows: Skipped due to TUI limitations
```

#### Build Jobs
Produces platform-specific binaries:
- `aps_linux_amd64` (Linux)
- `aps_darwin_amd64` (macOS)
- `aps_windows_amd64.exe` (Windows)

## Coverage Workflow (coverage.yml)

### Coverage Collection

1. **Upload Coverage Artifacts**
   - Collects `coverage.out` from each platform
   - Retained for 7 days

2. **Merge Coverage Reports**
   - Uses `gocovmerge` to combine platform coverage
   - Creates unified `coverage-final.out`

3. **Generate Reports**
   - Text report: `coverage-report.txt`
   - HTML report: `coverage.html`

4. **Upload to Codecov**
   - Integration with Codecov for tracking
   - Fails CI if coverage drops below 70%

## Security Workflow (security.yml)

### Security Scans

| Scanner | Purpose | Severity Threshold |
|---------|---------|-------------------|
| **gosec** | Go security vulnerability scanner | All |
| **Trivy** | Container/filesystem vulnerability scanner | CRITICAL, HIGH |
| **CodeQL** | GitHub's semantic code analysis | All |
| **Dependency Review** | Checks dependency vulnerabilities | Moderate+ |

### Scheduled Scans

Security workflows run weekly (Sundays at midnight) to catch newly discovered vulnerabilities.

## Runner Configuration

### GitHub Hosted Runners

The project uses GitHub's hosted runners:

#### Ubuntu Latest
- **OS**: Ubuntu 22.04 LTS
- **Architecture**: x86_64
- **Features**: Full Linux kernel support, cgroups, Docker
- **Best for**: Linux adapter testing, container isolation tests

#### macOS Latest
- **OS**: macOS 13 (Ventura)
- **Architecture**: x86_64
- **Features**: Apple Sandbox, Xcode tools
- **Best for**: macOS adapter testing, sandbox isolation tests

#### Windows Latest
- **OS**: Windows Server 2022
- **Architecture**: x86_64
- **Features**: Job Objects, AppContainer, PowerShell 7
- **Best for**: Windows adapter testing, job object tests

### Self-Hosted Runners (Optional)

For advanced testing scenarios, you can set up self-hosted runners:

#### Linux Self-Hosted Runner

```bash
# Create runner directory
mkdir actions-runner && cd actions-runner

# Download latest runner
curl -o actions-runner-linux-x64-2.311.0.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz

# Extract runner
tar xzf ./actions-runner-linux-x64-2.311.0.tar.gz

# Install dependencies
sudo ./bin/installdependencies.sh

# Configure runner (requires repo PAT)
./config.sh \
  --url https://github.com/IdeaCraftersLabs/oss-aps-cli \
  --token YOUR_RUNNER_TOKEN \
  --labels self-hosted,linux,x64

# Install and start service
sudo ./svc.sh install
sudo ./svc.sh start
```

#### macOS Self-Hosted Runner

```bash
# Create runner directory
mkdir actions-runner && cd actions-runner

# Download latest runner
curl -o actions-runner-osx-x64-2.311.0.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-osx-x64-2.311.0.tar.gz

# Extract runner
tar xzf ./actions-runner-osx-x64-2.311.0.tar.gz

# Configure runner
./config.sh \
  --url https://github.com/IdeaCraftersLabs/oss-aps-cli \
  --token YOUR_RUNNER_TOKEN \
  --labels self-hosted,macos,x64

# Install and start service
sudo ./svc.sh install
sudo ./svc.sh start
```

#### Windows Self-Hosted Runner

```powershell
# Create runner directory
New-Item -ItemType Directory -Force -Path actions-runner
Set-Location actions-runner

# Download latest runner
Invoke-WebRequest -Uri https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-win-x64-2.311.0.zip -OutFile actions-runner-win-x64-2.311.0.zip

# Extract runner
Expand-Archive -LiteralPath actions-runner-win-x64-2.311.0.zip

# Configure runner
.\config.cmd --url https://github.com/IdeaCraftersLabs/oss-aps-cli --token YOUR_RUNNER_TOKEN --labels self-hosted,windows,x64

# Install and start service
.\install.ps1
.\run.cmd
```

### Runner Labels

Use labels to route jobs to specific runners:

```yaml
jobs:
  test:
    runs-on: [self-hosted, linux]  # Only Linux self-hosted runners
    # or
    runs-on: [self-hosted, macos]  # Only macOS self-hosted runners
```

## Artifact Management

### Coverage Artifacts

| Artifact | Retention | Content |
|----------|-----------|---------|
| `coverage-Linux` | 7 days | Linux coverage.out |
| `coverage-macOS` | 7 days | macOS coverage.out |
| `coverage-Windows` | 7 days | Windows coverage.out |
| `coverage-html` | 30 days | Combined HTML report |
| `coverage-report` | 30 days | Combined text report |

### Build Artifacts

| Artifact | Platform | Retention |
|----------|----------|-----------|
| `aps_linux_amd64` | Linux | 7 days |
| `aps_darwin_amd64` | macOS | 7 days |
| `aps_windows_amd64.exe` | Windows | 7 days |

## Environment Variables

### GitHub Secrets

Configure these secrets in repository settings:

| Secret | Purpose | Required |
|--------|---------|----------|
| `CODECOV_TOKEN` | Codecov upload token | Yes |
| `RUNNER_TOKEN` | Self-hosted runner token | No |

### GitHub Environment Variables

Available in all workflows:

| Variable | Description |
|----------|-------------|
| `GITHUB_RUN_ID` | Unique run identifier |
| `GITHUB_SHA` | Commit SHA |
| `GITHUB_REF` | Branch or tag reference |
| `RUNNER_OS` | Operating system of runner |
| `RUNNER_ARCH` | Architecture of runner |

## Caching Strategy

### Go Module Cache

Caches `~/go/pkg/mod` across runs:

```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}
```

Cache keys are invalidated when `go.sum` changes.

## Troubleshooting

### Common Issues

#### Tests Fail on Windows

**Issue**: E2E tests fail on Windows

**Solution**: Tests are skipped on Windows due to TUI limitations. Check if TUI tests should be skipped.

```yaml
- name: Run E2E tests
  shell: bash
  run: |
    if [[ "${{ runner.os }}" == "Windows" ]]; then
      echo "Skipping E2E tests on Windows due to TUI limitations"
    else
      go test -v ./tests/e2e
    fi
```

#### Coverage Not Uploaded

**Issue**: Coverage report not uploaded to Codecov

**Solution**: Ensure `CODECOV_TOKEN` is set in repository secrets.

#### Lint Errors

**Issue**: golangci-lint fails

**Solution**: Run locally to reproduce:

```bash
golangci-lint run
```

### Debug Mode

Enable debug logging in workflows:

```yaml
- name: Run tests
  env:
    DEBUG: "1"
  run: go test -v ./tests/unit/...
```

## Performance Optimization

### Parallel Jobs

All jobs run in parallel where possible:

```yaml
strategy:
  fail-fast: false  # Continue even if one job fails
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
```

### Fail-Fast

Set `fail-fast: false` to ensure all platform tests complete:

```yaml
strategy:
  fail-fast: false  # Don't cancel other jobs if one fails
```

### Job Dependencies

Use `needs` to control job execution order:

```yaml
build:
  needs: test  # Only run after test job completes
  runs-on: ${{ matrix.os }}
```

## Maintenance

### Update GitHub Actions

Regularly update action versions:

```yaml
- uses: actions/checkout@v4  # Keep updated
- uses: actions/setup-go@v5
- uses: golangci/golangci-lint-action@v6
```

### Monitor Usage

Check Actions usage in repository settings:

- Usage limits: Minutes per month
- Storage limits: Artifacts and caches
- Self-hosted runner status

## Best Practices

1. **Matrix Strategy**: Use matrix for cross-platform testing
2. **Fail-Fast**: Set to `false` to get results from all platforms
3. **Caching**: Cache Go modules to speed up builds
4. **Artifacts**: Keep artifact retention periods short (7 days default)
5. **Security**: Run security scans on every commit
6. **Coverage**: Aim for 70%+ coverage, set lower threshold
7. **Self-Hosted Runners**: Use only when necessary (e.g., specific hardware)

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Actions Self-Hosted Runners](https://docs.github.com/en/actions/hosting-your-own-runners)
- [Codecov Documentation](https://docs.codecov.com/)
- [gosec Security Scanner](https://github.com/securego/gosec)
- [Trivy Scanner](https://github.com/aquasecurity/trivy)
