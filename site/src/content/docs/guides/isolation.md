---
title: Isolation Levels
description: Choose the right isolation level for your agent workload.
---

APS offers three isolation levels, balancing speed against security.

## Process (default)

Runs the command as a child process with isolated environment variables.
No filesystem or network sandboxing.

**Best for:** Development, trusted workloads, maximum speed.

```bash
aps profile new myagent --isolation-level process
```

## Platform

Uses the OS-native sandbox mechanism:
- **macOS:** `sandbox-exec` with a restrictive Seatbelt profile
- **Linux:** `bwrap` (bubblewrap) with namespaces

Restricts filesystem writes outside the profile home and limits network
to explicitly allowed endpoints.

**Best for:** Semi-trusted agents, local development with guardrails.

```bash
aps profile new myagent --isolation-level platform
```

## Container

Runs the agent inside a Docker container. Full filesystem and network
isolation. Requires Docker to be installed and running.

**Best for:** Production, untrusted code, strong reproducibility requirements.

```bash
aps profile new myagent --isolation-level container
```

## Comparison

| Feature              | Process | Platform | Container |
|----------------------|---------|----------|-----------|
| Startup overhead     | ~0ms    | ~50ms    | ~500ms    |
| Filesystem isolation | No      | Partial  | Full      |
| Network isolation    | No      | Partial  | Full      |
| Requires Docker      | No      | No       | Yes       |
| Cross-platform       | Yes     | macOS/Linux | Yes   |
