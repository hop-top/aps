# Custom Profile Tools

**ID**: 034
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to define custom tools in my profiles so that I can extend the CLI with profile-specific scripts and executables.

## Acceptance Scenarios

1. **Given** a profile with custom tools defined, **When** I look up a tool by name, **Then** the registry returns the correct tool definition.
2. **Given** a tool that requires installation, **When** I run `EnsureTool`, **Then** the tool is installed if missing.
3. **Given** a profile with scripts, **When** I list profile tools, **Then** all profile-scoped scripts are returned.

## Tests

### E2E
- planned: `tests/e2e/profile_tools_test.go::TestProfileTools_LookupReturnsDefinition`
- planned: `tests/e2e/profile_tools_test.go::TestProfileTools_EnsureToolInstallsMissing`
- planned: `tests/e2e/profile_tools_test.go::TestProfileTools_ListShowsProfileScripts`

### Unit
- `tests/unit/core/tools_test.go` — `TestToolRegistry_Lookup`, `TestToolRegistry_List`, `TestTool_IsInstalled`, `TestTool_EnsureTool`, `TestTool_ProfileScripts`, `TestTool_ExecuteProfileTool`
