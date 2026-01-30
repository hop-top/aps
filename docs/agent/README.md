# APS Agent Documentation

This documentation is intended for AI Agents (and their human collaborators) working on the APS codebase. It defines patterns, conventions, and context required to maintain and extend the system effectively.

## 🧠 Core Context

*   **[Codebase Structure](structure.md)**: Overview of the standard package layout (`internal/core`, `internal/cli`, etc.).
*   **[Design Patterns](patterns.md)**: Standard patterns used (e.g., Capability Manager, Registry, Adapter Pattern for Isolation).
*   **[Testing Strategy](testing.md)**: How to write and run Unit vs E2E tests.
*   **[A2A Protocol Integration](a2a-integration.md)**: Agent-to-Agent communication for inter-profile messaging.
*   **[Docker Testing](docker-testing.md)**: Using Docker for isolated Linux testing and user journey validation.

## 🚀 Release Process

**Source of truth**: Git tags → GoReleaser → binaries

```bash
# To release
git tag v1.0.0-alpha.1
git push origin v1.0.0-alpha.1
```

**Key files**:
- `.goreleaser.yaml` - Build config
- `internal/version/version.go` - Version injection via ldflags
- `VERSION.txt` - Auto-updated from tag

## 🤖 Rules & Conventions

1.  **Strict Layering**: `internal/core` MUST NOT import `internal/cli`. Logic goes in `core`, user interaction goes in `cli`.
2.  **Configuration**: Use `viper` or `yaml` for config. Prefer XDG paths.
3.  **Error Handling**: Return errors, don't panic. Wrap errors with context.
4.  **Versioning**: Use semantic versioning. Tags trigger releases.

## 🔗 Useful Links

*   **[Dev Docs](../dev/readme.md)**: Full architecture and implementation details.
*   **[Release Process](../dev/operations/release-process.md)**: Detailed release documentation.
*   **[User Docs](../user/README.md)**: User-facing behavior and expectations.
