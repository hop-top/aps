# Custom Tools

This guide explains how to add custom tools to your APS profiles.

## Overview

APS supports two types of tools:

1. **Registry Tools** - Pre-configured tools in APS registry (Claude Code, Google Gemini, OpenAI Codex, etc.)
2. **Profile Scripts** - Custom scripts in your profile directories

## Registry Tools

### Available Tools

APS comes with a registry of common AI and development tools:

| Tool Name | Description | Install Command |
|-----------|-------------|----------------|
| `claude` | Anthropic's AI coding assistant | `npm install -g @anthropic-ai/claude-code@latest` |
| `gemini` | Google Gemini CLI | `npm install -g @google/gemini-cli` |
| `codex` | OpenAI Codex CLI | `npm install -g @openai/codex-cli` |
| `python3` | Python 3 interpreter | System package |
| `node` | Node.js JavaScript runtime | System package |
| `git` | Git version control | System package |

### Installing Registry Tools

Tools from the registry can be automatically installed:

```yaml
# ~/.agents/profiles/my-profile/profile.yaml
id: my-profile
display_name: My Profile
```

APS will auto-install tools when you:
1. Run `aps tools install <tool-name>`
2. Execute an action that requires the tool

### Specifying Tool Versions

You can pin tools to specific versions:

```bash
# Install specific version
aps tools install claude@1.2.0
```

Or in your profile:
```yaml
tools:
  claude:
    type: registry
    version: "1.2.0"
    auto_install: true
```

## Profile Scripts

Profile scripts are custom tools defined in your profile's `tools/` directory.

### Creating Profile Scripts

1. Create `tools/` directory in your profile:
```bash
mkdir -p ~/.agents/profiles/my-profile/tools
```

2. Add script files with appropriate extensions:
- `*.sh` - Shell scripts
- `*.py` - Python scripts  
- `*.js` - Node.js scripts

### Example: Shell Script Tool

```bash
# ~/.agents/profiles/my-profile/tools/my-tool.sh
#!/bin/bash
set -euo pipefail

echo "Running custom tool..."
# Your tool logic here
echo "Tool completed successfully!"
```

Make it executable:
```bash
chmod +x ~/.agents/profiles/my-profile/tools/my-tool.sh
```

### Example: Python Script Tool

```python
# ~/.agents/profiles/my-profile/tools/my-python-tool.py
#!/usr/bin/env python3

print("Python tool running...")
# Your tool logic here
print("Tool completed!")
```

### Example: Node.js Script Tool

```javascript
// ~/.agents/profiles/my-profile/tools/my-js-tool.js
#!/usr/bin/env node

console.log("Node.js tool running...");
// Your tool logic here
console.log("Tool completed!");
```

### Executing Profile Scripts

```bash
# Execute a profile script
aps tools run my-profile my-tool.sh

# Execute with arguments
aps tools run my-profile my-tool.sh --arg1 --arg2
```

## Auto-Installation

### Registry Tools

Registry tools support auto-installation:

1. Check if tool is installed
2. Verify version matches specification
3. Auto-install if missing or wrong version
4. Fail gracefully if installation fails

Example in profile:
```yaml
tools:
  claude:
    type: registry
    auto_install: true
    version: "latest"
```

### Profile Scripts

Profile scripts are discovered automatically:
- APS scans `tools/` directory in profile
- Scripts must be executable (`chmod +x`)
- Scripts have access to profile environment variables

## Container Build Steps

For profiles using container isolation (`level: container`), tools can be installed during image build:

```yaml
# ~/.agents/profiles/container-profile/profile.yaml
id: container-profile
display_name: Container Profile

isolation:
  level: container
  container:
    image: ubuntu:22.04
    packages:
      - nodejs
      - python3
      - git
    build_steps:
      - type: shell
        run: npm install -g @anthropic-ai/claude-code@latest
      - type: shell
        run: pip install -q google-generativeai
```

### Build Step Types

| Type | Description | Example |
|------|-------------|---------|
| `shell` | Execute shell command | `run: apt-get update` |
| `copy` | Copy files into image | `run: ./local/file /container/path` |
| `env` | Set environment variable | `run: MY_VAR=value` |
| `expose` | Expose port | `run: 8080` |
| `volume` | Define volume | `run: /data` |
| `workdir` | Set working directory | `run: /app` |
| `user` | Switch user | `run: appuser` |

### Using `content` Field

For `shell` steps, use `content` instead of `run` for inline commands:

```yaml
build_steps:
  - type: shell
    content: |
      echo "Multi-line shell command"
      apt-get update
      apt-get install -y curl
```

## Best Practices

1. **Shebang Lines**
   - Always include shebang: `#!/bin/bash`, `#!/usr/bin/env python3`, etc.
   - Make scripts executable: `chmod +x`

2. **Error Handling**
   - Use `set -euo pipefail` in bash scripts
   - Check return codes in all commands
   - Handle errors gracefully

3. **Environment Variables**
   - Profile scripts inherit profile secrets: `APS_PROFILE_SECRETS`
   - Access profile directory: `APS_PROFILE_DIR`
   - Use profile-specific tools directory

4. **Security**
   - Don't hardcode secrets in scripts
   - Use `APS_PROFILE_SECRETS` environment variable
   - Validate user input
   - Use least privilege necessary

5. **Performance**
   - Cache dependencies where possible
   - Use lightweight base images for containers
   - Minimize build step complexity

## Examples

### Complete Profile with Tools

```yaml
id: ai-tools-profile
display_name: AI Tools Profile
isolation:
  level: process
  strict: false
  fallback: true

tools:
  claude:
    type: registry
    version: "1.2.0"
    auto_install: true

  my-custom-tool:
    type: script
    path: tools/my-custom-tool.sh
    auto_install: false
```

### Container Profile with Build Steps

```yaml
id: dev-environment
display_name: Development Environment

isolation:
  level: container
  strict: false
  fallback: true

  container:
    image: ubuntu:22.04
    network: bridge
    volumes:
      - /Users/jadb/code:/workspace
    resources:
      memory_mb: 2048
    packages:
      - nodejs
      - python3
      - python3-pip
      - git
      - curl
    build_steps:
      - type: shell
        run: npm install -g @anthropic-ai/claude-code@latest
      - type: shell
        run: pip install -q google-generativeai
      - type: shell
        run: pip install -q openai
      - type: env
        run: NODE_ENV=production
      - type: workdir
        run: /workspace
```

### Profile Script with Environment Access

```bash
#!/bin/bash
# ~/.agents/profiles/my-profile/tools/env-tool.sh

set -euo pipefail

# Access profile environment
echo "Profile ID: $APS_PROFILE_ID"
echo "Profile Directory: $APS_PROFILE_DIR"
echo "Secrets File: $APS_PROFILE_SECRETS"

# Read secrets if needed
if [ -f "$APS_PROFILE_SECRETS" ]; then
    source "$APS_PROFILE_SECRETS"
    echo "Loaded secrets from: $APS_PROFILE_SECRETS"
fi

# Your tool logic
echo "Tool running..."
```

## Troubleshooting

### Tool Not Found

**Error**: `tool 'my-tool' not found in registry`

**Solution**:
- Check tool name spelling
- Verify tool exists in registry (`aps tools list`)
- For custom tools, ensure script is in `tools/` directory

### Auto-Install Fails

**Error**: `failed to install tool 'claude'`

**Solutions**:
- Check npm is installed: `npm --version`
- Check internet connectivity
- Try manual installation: `npm install -g @anthropic-ai/claude-code@latest`
- Check available disk space

### Script Permission Denied

**Error**: `permission denied: ./tools/my-tool.sh`

**Solution**:
- Make script executable: `chmod +x ~/.agents/profiles/my-profile/tools/my-tool.sh`
- Verify shebang line is correct

### Container Build Fails

**Error**: `Dockerfile generation failed`

**Solutions**:
- Check base image spelling: `ubuntu:22.04` vs `ubuntu 22.04`
- Verify build step syntax
- Check for missing quotes in commands
- Validate Dockerfile manually: `aps tools dockerfile <profile-id>`

### Docker Build Failures

**Error**: Container build fails during package installation

**Solutions**:
- Verify package manager matches base image:
  - `ubuntu`/`debian` → `apt-get`
  - `alpine` → `apk`
  - `fedora`/`centos` → `yum`
  - `arch` → `pacman`
- Check package names are correct
- Use non-interactive flags: `apt-get install -y`

## CLI Commands

### List Available Tools

```bash
aps tools list
```

### Install a Tool

```bash
# From registry
aps tools install claude

# Specific version
aps tools install claude@1.2.0
```

### Verify Tool Installation

```bash
# Check if tool is installed
aps tools check claude

# Check version
aps tools version claude
```

### Generate Dockerfile

```bash
# Generate Dockerfile for container profile
aps tools dockerfile <profile-id>

# Output saved to: ~/.agents/profiles/<profile-id>/Dockerfile
```

### Run Custom Tool

```bash
aps tools run <profile-id> <tool-name> [args...]
```

## Integration with Actions

Tools can be called from actions:

```yaml
# ~/.agents/profiles/my-profile/actions/use-tool.yaml
id: use-tool
title: Use Custom Tool
type: sh
path: tools/my-custom-tool.sh
accepts_stdin: true
```

Execute:
```bash
aps action my-profile run use-tool <payload>
```

## Advanced Usage

### Tool Chains

Combine multiple tools in workflows:

```bash
# Run multiple tools in sequence
aps tools run my-profile tool1.sh && \
aps tools run my-profile tool2.sh
```

### Conditional Tool Installation

```bash
# Only install if tool not available
if ! aps tools check my-tool; then
    aps tools install my-tool
fi
```

### Tool Environment Customization

Set tool-specific environment in profile:

```yaml
# profile.yaml
id: my-profile

# Tools will inherit these via APS environment variables
```

## Security Considerations

1. **Script Execution**
   - Scripts run with profile's permissions
   - Don't use `sudo` in profile scripts
   - Validate all file paths

2. **Secrets Management**
   - Store secrets in `secrets.env`, not in scripts
   - Never commit secrets to version control
   - Rotate credentials regularly

3. **Container Security**
   - Use official base images
   - Update images regularly
   - Scan images for vulnerabilities
   - Minimize attack surface (fewer packages)

4. **Network Access**
   - Containers can access network by default
   - Use `network: none` for isolated builds
   - Restrict with firewall rules

## API Reference

### Tool Registry Functions

- `GetTool(name string) (Tool, error)` - Get tool from registry
- `ListTools() []Tool` - List all available tools
- `IsToolInstalled(tool Tool) bool` - Check if tool is installed
- `EnsureTool(name string, version string) error` - Install tool if needed
- `DiscoverProfileScripts(profileID string) ([]Tool, error)` - Find profile scripts
- `ExecuteProfileTool(profileID string, toolName string, args []string) error` - Execute tool

### Dockerfile Builder Functions

- `Generate(profile *Profile) (string, error)` - Generate Dockerfile content
- `WriteDockerfile(profile *Profile, outputDir string) (string, error)` - Write Dockerfile to disk
- `BuildImageOptions(profile *Profile) (map[string]interface{}, error)` - Get container options

## See Also

- [Isolation Architecture](../../specs/001-build-cli-core/isolation-architecture.md)
- [Container Implementation](../../docs/CONTAINER_IMPLEMENTATION.md)
- [Profile Configuration](../../README.md#profile-configuration)
