# Container Isolation

**Isolation Level**: Container (Tier 3)
**Status**: Supported (requires Docker)

## Overview

Container isolation provides the highest level of isolation by executing commands in Docker containers. This is the "nuclear option" for isolation, suitable for untrusted code execution.

## Requirements

### System Requirements
- Docker installed and running
- Docker CLI available in PATH
- Docker daemon must be accessible
- Sufficient disk space for container images
- Admin rights to manage Docker containers

### Install Docker

```bash
# Linux
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# macOS
brew install --cask docker
# Or download from: https://www.docker.com/products/docker-desktop

# Windows
# Download from: https://www.docker.com/products/docker-desktop
```

### Verify Docker Installation

```bash
docker --version
docker info
```

## Setup

### 1. Admin SSH Key Generation

Generate admin SSH key for container access:

```bash
# Create keys directory
mkdir -p ~/.aps/keys

# Generate admin key pair
ssh-keygen -t ed25519 -f ~/.aps/keys/admin_key -N ""

# Copy public key to admin_pub file
cp ~/.aps/keys/admin_key.pub ~/.aps/keys/admin_pub

# Set permissions
chmod 700 ~/.aps/keys
chmod 600 ~/.aps/keys/admin_key
chmod 644 ~/.aps/keys/admin_pub
```

### 2. Create Profile with Container Isolation

```bash
# Create new profile with container isolation
aps profile new container-profile --isolation-level container

# Or create manually
aps profile new container-profile
# Edit: ~/.local/share/aps/profiles/container-profile/profile.yaml
# Add:
#   isolation:
#     level: "container"
```

### Profile Configuration Example

```yaml
id: container-profile
display_name: "Container Isolation Profile"

isolation:
  level: "container"
  strict: false
  fallback: true

  container:
    image: "ubuntu:22.04"
    packages:
      - nodejs
      - python3
      - git
      - curl
    build_steps:
      - type: "shell"
        run: "npm install -g @anthropic-ai/claude-code@latest"
    volumes:
      - "/tmp/aps-shared:/workspace"
      - "/home/user/.aps/profiles/container-profile/secrets.env:/secrets.env:ro"
    network: "bridge"
    resources:
      memory_mb: 1024
      cpu_quota: 100000
```

## Usage

### Running Commands in Containers

```bash
# Run command in container
aps run container-profile -- whoami

# Expected output: appuser

# Run command with args
aps run container-profile -- ls -la /workspace

# Run interactive command
aps run container-profile -- bash
```

### SSH Access to Container

Connect to container via SSH:

```bash
# SSH to container (port-forwarded automatically)
ssh -i ~/.aps/keys/admin_key -p 2222 appuser@localhost

# Container IP and port are managed by APS
```

### Session Management

```bash
# List active sessions
aps session list

# Attach to session
aps session attach <session-id>

# Delete session
aps session delete <session-id>
```

## Architecture

### Container Structure

```
Docker Container (aps-{profileID}:latest)
├── App User: appuser
├── Home Directory: /home/appuser
├── Working Directory: /workspace
├── SSH Server: Running on port 22
├── SSH Keys: Admin public key in authorized_keys
├── tmux: Installed for session management
└── Tools: Node.js, Python3, Git, etc.
```

### Dockerfile Structure

Generated Dockerfiles include:

1. **Base Image**: User-specified (e.g., `ubuntu:22.04`)
2. **Package Installation**: Apt packages from profile
3. **SSH Server**: OpenSSH server installed and configured
4. **tmux**: Terminal multiplexer for sessions
5. **App User**: Non-root user for isolation
6. **Build Steps**: Profile-specified custom commands
7. **Volume Mounts**: Shared workspace, secrets, custom volumes
8. **Resource Limits**: CPU, memory constraints

### Volume Mounting

Volumes are mounted at container creation time:

- **Shared Workspace**: `/tmp/aps-shared:/workspace`
- **Secrets**: `~/.aps/profiles/{profileID}/secrets.env:/secrets.env:ro`
- **Custom Volumes**: User-defined volumes from profile

### Resource Limits

CPU and memory can be limited:

```yaml
resources:
  memory_mb: 1024    # 1GB RAM
  cpu_quota: 100000    # 100% of 1 CPU (100ms per 100ms)
```

### Network Configuration

Three network modes available:

- **bridge** (default): Isolated network, port mapping required
- **host**: Shares host network stack (no isolation)
- **none**: No network access

```yaml
network: "bridge"  # or "host" or "none"
ports:
  - "8080:80"     # Map host port 8080 to container port 80
```

## Security

### Isolation Boundaries

1. **Process Isolation**: Separate container process tree
2. **Filesystem Isolation**: Separate filesystem, volumes mounted explicitly
3. **Network Isolation**: Separate network namespace (unless host mode)
4. **User Isolation**: Non-root user (appuser)
5. **Kernel Isolation**: Container uses separate cgroups

### Security Considerations

- **Docker Daemon**: Requires admin access to manage containers
- **Volume Mounts**: Host filesystem accessible through volumes
- **Docker Socket**: Not mounted by default (security best practice)
- **Privilege Escalation**: No privileged containers by default
- **Resource Limits**: Prevents container from consuming all resources

### Recommended Use Cases

✅ **Suitable for**:
- Untrusted code execution
- High-security requirements
- Production workload isolation
- Multi-tenant environments
- Sandboxing unknown dependencies

❌ **Not suitable for**:
- Graphics/GUI applications (no display)
- Direct hardware access
- Host networking required (use host mode)
- Performance-critical workloads (container overhead)

## Troubleshooting

### Issue: "Docker not available"

**Problem**: Docker CLI not found or daemon not running

**Solution**:
```bash
# Check Docker is installed
which docker

# Check Docker daemon is running
sudo systemctl status docker   # Linux
docker info                       # All platforms

# Start Docker daemon
sudo systemctl start docker  # Linux
# On macOS/Windows, start Docker Desktop
```

### Issue: "Failed to pull image"

**Problem**: Docker registry unreachable or image doesn't exist

**Solution**:
```bash
# Test network connectivity
docker pull ubuntu:22.04

# Use local images
aps profile new local-container
# Edit: ~/.local/share/aps/profiles/local-container/profile.yaml
# Set: container.image: "your-local-image"

# Configure registry mirror
sudo mkdir -p /etc/docker
echo '{"registry-mirrors": ["https://mirror.gcr.io"]}' | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
```

### Issue: "Permission denied"

**Problem**: Cannot manage containers (not in docker group)

**Solution**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in for group changes to take effect
# Or run: newgrp docker

# Test Docker access
docker ps
```

### Issue: "SSH connection failed"

**Problem**: Cannot SSH into container

**Solution**:
```bash
# Check SSH key exists
ls -la ~/.aps/keys/admin_key

# Check SSH key permissions
chmod 600 ~/.aps/keys/admin_key

# Check container is running
aps session list

# Check SSH port mapping
docker port <container-id> 22

# Test SSH manually
ssh -v -i ~/.aps/keys/admin_key -p 2222 appuser@localhost
```

### Issue: "Container exited immediately"

**Problem**: Container exits after starting

**Solution**:
```bash
# Check container logs
docker logs <container-id>

# Check container status
docker ps -a

# Check Dockerfile CMD is long-running
# Container needs a foreground process to stay alive

# Test container manually
docker run -it aps-profile:latest bash
```

### Issue: "Volume mount failed"

**Problem**: Cannot mount volumes

**Solution**:
```bash
# Check host path exists
ls -la /tmp/aps-shared

# Check path permissions
chmod 755 /tmp/aps-shared

# Check Docker file sharing (macOS/Windows)
# Docker Desktop Settings > Resources > File Sharing

# Use absolute paths in volumes
# Correct: "/home/user/path:/container/path"
# Wrong: "~/path:/container/path"
```

### Issue: "Out of memory"

**Problem**: Container killed due to OOM (Out of Memory)

**Solution**:
```bash
# Check container logs for OOM
docker logs <container-id>

# Increase memory limit
# Edit profile.yaml:
#   container.resources.memory_mb: 2048

# Check system memory
free -h  # Linux
# Or use system monitor
```

## Cleanup

### Remove Container

```bash
# Stop and remove container
docker rm -f <container-id>

# Or via APS session
aps session delete <session-id>
```

### Remove Image

```bash
# Remove specific image
docker rmi aps-{profileID}:latest

# Remove all APS images
docker images | grep "aps-" | awk '{print $3}' | xargs docker rmi -f
```

### Remove Profile

```bash
# Delete profile
aps profile delete container-profile

# Or manually
rm -rf ~/.aps/profiles/container-profile
```

### Remove Docker Resources

```bash
# Remove all stopped containers
docker container prune -f

# Remove all unused images
docker image prune -a -f

# Remove all unused volumes
docker volume prune -f

# Remove all unused networks
docker network prune -f
```

## Advanced Configuration

### Custom Base Images

Use your own base images:

```yaml
container:
  image: "your-registry.com/custom-image:latest"
```

### Multi-Stage Builds

Optimize image size with multi-stage builds:

```yaml
build_steps:
  - type: "shell"
    run: |
      apt-get update && apt-get install -y \
        build-essential \
        && make build \
        && make install
      apt-get purge -y build-essential
      apt-get autoremove -y
```

### Custom SSH Configuration

Modify SSH server in container:

```yaml
build_steps:
  - type: "shell"
    run: |
      echo 'MaxAuthTries 3' >> /etc/ssh/sshd_config
      echo 'LoginGraceTime 20' >> /etc/ssh/sshd_config
      echo 'ClientAliveInterval 300' >> /etc/ssh/sshd_config
```

### Health Checks

Add health checks to containers:

```yaml
container:
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

### Compose Integration

Use Docker Compose for multi-container setups:

```yaml
services:
  app:
    build: .
    volumes:
      - ./data:/workspace
    ports:
      - "8080:80"
```

```bash
aps profile new compose-profile
# Copy docker-compose.yml to profile directory
# Run: aps run compose-profile -- docker-compose up
```

## Testing

### Docker Test Environment

APS provides a Docker-based testing environment for testing container isolation and user workflows without affecting your local system.

**Purpose**: Simulate a fresh Linux machine for testing APS installation, setup, and workflows.

**Benefits**:
- **Isolation**: Clean environment for every test run
- **Realism**: Simulates actual user machine setup
- **Reproducibility**: Consistent test conditions across runs
- **Linux Testing**: Tests on target Linux platform regardless of host OS
- **No Local Impact**: Keeps local development environment clean

### Test Environment Setup

The test environment uses:

- **Base Image**: Ubuntu 22.04 (matches default container isolation base)
- **Non-root User**: `testuser` for realistic environment
- **Essential Tools**: curl, wget, git, vim, openssh-client, etc.
- **XDG Directories**: Proper configuration structure
- **No Pre-installed Dependencies**: Simulates fresh install

### Running Container Tests

```bash
# Quick start - verify Docker test environment
make docker-quick-start

# Build test image
make docker-build-test

# Run full container isolation test suite
make docker-test-e2e-user

# Interactive testing
make docker-test-shell
# Inside container:
aps profile new container-test --isolation-level container
aps run container-test -- echo "Testing container isolation"
```

### Test Volumes

The test environment uses three persistent volumes:

1. **test-home**: User home directory (`/home/testuser`)
2. **test-config**: APS configuration directory (`/home/testuser/.config/aps`)
3. **test-fixtures**: Read-only test fixtures

### Container Isolation Tests

**Basic Isolation Test**:
```bash
# Create container profile
aps profile new iso-test --isolation-level container

# Verify isolation
aps run iso-test -- whoami  # Should be 'appuser' or container user
aps run iso-test -- cat /etc/os-release  # Should show Ubuntu 22.04
```

**Volume Mount Test**:
```bash
# Create profile with volumes
aps profile new vol-test --isolation-level container
# Edit: ~/.local/share/aps/profiles/vol-test/profile.yaml
# Add: container.volumes: ["/tmp/test:/workspace"]

# Run with access to workspace
aps run vol-test -- ls -la /workspace
```

**Network Isolation Test**:
```bash
# Create container profile
aps profile new net-test --isolation-level container

# Test network (bridge mode - default)
aps run net-test -- ping -c 2 google.com

# Test host networking (if enabled)
# Edit: container.network: "host"
aps run net-test -- curl http://localhost:8080
```

### Testing Documentation

- [Docker Testing Strategy](../../testing/docker-testing-strategy.md) - Comprehensive Docker testing guide
- [Docker Testing for Users](../../../agent/docker-testing.md) - User-friendly testing workflows
- [Makefile Docker Targets](../../../../../) - `make docker-*` commands

### CI/CD Testing

Container isolation is tested automatically in CI/CD:

```bash
# Run Docker user journey tests
gh workflow run docker-user-journey.yml

# Or locally:
make docker-test-e2e-user
```

The CI/CD workflow builds APS, builds Docker test image, and runs comprehensive container isolation tests.

## Performance

### Container Startup Time

- **Cold Start**: 2-5 seconds (pull image, create container)
- **Warm Start**: 100-500ms (create container from cached image)

### Resource Overhead

- **Memory**: ~50-100MB base overhead (OS + SSH + tmux)
- **CPU**: Minimal overhead when idle
- **Disk**: 1-5GB for container images

### Optimization Tips

1. **Use Smaller Base Images**: `alpine` vs `ubuntu`
2. **Layer Caching**: Order Dockerfile commands by frequency of change
3. **Remove Build Artifacts**: Clean up after build steps
4. **Use Volume Caching**: Cache node_modules, etc. in volumes
5. **Limit Resources**: Set appropriate CPU/memory limits

## Related Documentation

### APS Documentation
- [container-implementation.md](container-implementation.md) - Detailed implementation requirements and architecture
- [../../implementation/summaries/container-isolation-summary.md](../../implementation/summaries/container-isolation-summary.md) - Implementation summary with technical details
- [../../architecture/design/container-design-summary.md](../../architecture/design/container-design-summary.md) - Design overview and decisions
- [../../architecture/design/container-isolation-interface.md](../../architecture/design/container-isolation-interface.md) - Interface specifications
- [../../architecture/design/container-session-registry.md](../../architecture/design/container-session-registry.md) - Session registry design
- [../../testing/container-test-strategy.md](../../testing/container-test-strategy.md) - Testing approach and strategy
- [../../security/security-audit.md](../../security/security-audit.md) - Security considerations for containers

### External References
- [Docker Documentation](https://docs.docker.com/)
- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Container Security](https://docs.docker.com/engine/security/)
- [Docker Networking](https://docs.docker.com/network/)

## Support

For issues or questions:
- Check troubleshooting section above
- Review logs in `~/.aps/logs/`
- Check Docker logs: `docker logs <container-id>`
- Open an issue on GitHub
