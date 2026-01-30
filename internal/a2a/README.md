# A2A Protocol Integration

This package provides integration with the official Agent2Agent (A2A) Protocol for APS.

## Overview

A2A Protocol is an open standard developed by Google and donated to the Linux Foundation. It enables seamless communication and collaboration between AI agents.

- **Official Spec**: https://a2a-protocol.org/latest/specification/
- **Go SDK**: https://github.com/a2aproject/a2a-go

## Components

- **agentcard.go**: Generate A2A Agent Cards from APS profiles
- **server.go**: A2A Server using `a2asrv` to expose profiles as agents
- **client.go**: A2A Client using `a2aclient` for profile-to-profile communication
- **storage.go**: A2A task storage backend
- **transport/**: Transport adapters (IPC, HTTP, gRPC) for isolation tiers

## Architecture

```
APS Profile
    │
    ▼
Agent Card Generator → Agent Card
    │
    ▼
A2A Server (a2asrv) ← A2A Client (a2aclient)
    │                              │
    ▼                              ▼
Transport Adapters → IPC/HTTP/gRPC
```

## Usage

See quickstart.md for complete usage examples.

## Testing

Run tests with: `go test ./internal/a2a/...`
