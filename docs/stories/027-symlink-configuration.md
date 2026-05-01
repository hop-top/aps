---
status: paper
---

# Symlink Configuration

**ID**: 027
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want APS to manage my tool configurations by linking them so that I have a central place for all my dotfiles and agent tools.

## Acceptance Scenarios

1. **Given** an installed capability `my-vim-config` in `~/.aps/capabilities/vim`, **When** I run `aps capability link --target ~/.vimrc`, **Then** a symlink is created at `~/.vimrc` pointing to the APS managed file.
2. **Given** existing config at `~/.config/gh/config.yml`, **When** I run `aps capability adopt ~/.config/gh/config.yml --name gh-config`, **Then** the file is moved to `~/.aps/capabilities/gh-config` and a symlink is left in its place.

## Tests

### E2E
- planned: `tests/e2e/capability_link_test.go::TestCapability_LinkCreatesSymlink`
- planned: `tests/e2e/capability_link_test.go::TestCapability_AdoptMovesAndSymlinks`

### Unit
- `tests/unit/core/capability/capability_test.go` — `TestCapabilityLifecycle`
