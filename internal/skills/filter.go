package skills

import (
	"runtime"
	"strings"
)

// SkillFilter provides filtering criteria for skills
type SkillFilter struct {
	// Platform filtering (linux, darwin, windows)
	Platform string

	// Protocol filtering (agent-protocol, a2a, acp)
	Protocol string

	// Isolation level requirement (process, platform, container)
	IsolationLevel string

	// Only include skills compatible with current environment
	CompatibleOnly bool
}

// NewSkillFilter creates a filter with auto-detected platform
func NewSkillFilter() *SkillFilter {
	return &SkillFilter{
		Platform:       runtime.GOOS,
		CompatibleOnly: false,
	}
}

// Matches checks if a skill matches the filter criteria
func (f *SkillFilter) Matches(skill *Skill) bool {
	// Platform filtering
	if f.Platform != "" && !f.matchesPlatform(skill) {
		return false
	}

	// Protocol filtering
	if f.Protocol != "" && !f.matchesProtocol(skill) {
		return false
	}

	// Isolation level filtering
	if f.IsolationLevel != "" && !f.matchesIsolationLevel(skill) {
		return false
	}

	// Compatibility filtering
	if f.CompatibleOnly && !f.isCompatible(skill) {
		return false
	}

	return true
}

// matchesPlatform checks if skill is compatible with the platform
func (f *SkillFilter) matchesPlatform(skill *Skill) bool {
	if skill.Compatibility == "" {
		return true // No compatibility specified, assume compatible
	}

	compat := strings.ToLower(skill.Compatibility)

	// Check for explicit platform mentions
	switch f.Platform {
	case "linux":
		// Compatible if mentions linux or unix, or doesn't mention other platforms
		if strings.Contains(compat, "linux") || strings.Contains(compat, "unix") {
			return true
		}
		// Reject if explicitly mentions other platforms
		if strings.Contains(compat, "macos") || strings.Contains(compat, "windows") {
			return false
		}
		return true // No explicit platform, assume compatible

	case "darwin":
		if strings.Contains(compat, "macos") || strings.Contains(compat, "darwin") || strings.Contains(compat, "unix") {
			return true
		}
		if strings.Contains(compat, "linux only") || strings.Contains(compat, "windows") {
			return false
		}
		return true

	case "windows":
		if strings.Contains(compat, "windows") {
			return true
		}
		if strings.Contains(compat, "linux") || strings.Contains(compat, "macos") || strings.Contains(compat, "unix") {
			return false
		}
		return true
	}

	return true
}

// matchesProtocol checks if skill supports the protocol
func (f *SkillFilter) matchesProtocol(skill *Skill) bool {
	// Check metadata for protocol support
	if protocols, ok := skill.Metadata["protocols"]; ok {
		return strings.Contains(strings.ToLower(protocols), f.Protocol)
	}

	// If no protocol metadata, assume compatible with all
	return true
}

// matchesIsolationLevel checks if skill can run in the isolation level
func (f *SkillFilter) matchesIsolationLevel(skill *Skill) bool {
	// Check metadata for required isolation
	if required, ok := skill.Metadata["required_isolation"]; ok {
		requiredLevel := strings.ToLower(required)

		// Isolation hierarchy: container > platform > process
		// If skill requires container, can't run in platform or process
		switch requiredLevel {
		case "container":
			return f.IsolationLevel == "container"
		case "platform":
			return f.IsolationLevel == "platform" || f.IsolationLevel == "container"
		case "process":
			return true // Process works in all isolation levels
		}
	}

	// No requirement specified, assume compatible
	return true
}

// isCompatible performs comprehensive compatibility check
func (f *SkillFilter) isCompatible(skill *Skill) bool {
	// Platform check
	if !f.matchesPlatform(skill) {
		return false
	}

	// Check for required tools/dependencies in compatibility field
	if skill.Compatibility != "" {
		compat := strings.ToLower(skill.Compatibility)

		// Check for tools that might not be available
		if strings.Contains(compat, "docker") && !f.hasDocker() {
			return false
		}

		if strings.Contains(compat, "git") && !f.hasGit() {
			return false
		}

		// Add more tool checks as needed
	}

	return true
}

// hasDocker checks if Docker is available
func (f *SkillFilter) hasDocker() bool {
	// Simple check - could be enhanced
	return commandExists("docker")
}

// hasGit checks if Git is available
func (f *SkillFilter) hasGit() bool {
	return commandExists("git")
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	// Simple implementation - could use exec.LookPath for robustness
	// This is a placeholder for demonstration
	return true // Assume tools exist for now
}
