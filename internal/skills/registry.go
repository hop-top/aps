package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Registry manages skill discovery and loading
type Registry struct {
	paths  *SkillPaths
	skills map[string]*Skill // key: skill name, value: skill (highest priority)
}

// NewRegistry creates a new skill registry
func NewRegistry(profileID string, userPaths []string, includeDetectedPaths bool) *Registry {
	paths := NewSkillPaths(profileID)
	paths.UserPaths = userPaths

	if includeDetectedPaths {
		paths.DetectedPaths = paths.DetectIDEPaths()
	}

	return &Registry{
		paths:  paths,
		skills: make(map[string]*Skill),
	}
}

// Discover scans all configured paths and loads skill metadata
func (r *Registry) Discover() error {
	r.skills = make(map[string]*Skill) // Reset

	// Scan all paths in priority order
	allPaths := r.paths.AllPaths()

	for _, searchPath := range allPaths {
		// Check if path exists
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		// Scan for skill directories
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue // Skip paths we can't read
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillName := entry.Name()
			skillPath := filepath.Join(searchPath, skillName)

			// Check if SKILL.md exists
			skillMdPath := filepath.Join(skillPath, "SKILL.md")
			if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
				continue // Not a valid skill
			}

			// If we've already found this skill (from higher priority path), skip
			if _, exists := r.skills[skillName]; exists {
				continue
			}

			// Parse the skill
			skill, err := ParseSkill(skillPath)
			if err != nil {
				// Log error but continue discovering other skills
				fmt.Fprintf(os.Stderr, "Warning: Failed to parse skill %s: %v\n", skillName, err)
				continue
			}

			skill.SourcePath = searchPath
			r.skills[skillName] = skill
		}
	}

	return nil
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (*Skill, bool) {
	skill, found := r.skills[name]
	return skill, found
}

// List returns all discovered skills
func (r *Registry) List() []*Skill {
	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}

	// Sort by name for consistent ordering
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}

// ListBySource returns skills grouped by their source path
func (r *Registry) ListBySource() map[string][]*Skill {
	bySource := make(map[string][]*Skill)

	for _, skill := range r.skills {
		source := r.getSourceLabel(skill.SourcePath)
		bySource[source] = append(bySource[source], skill)
	}

	// Sort each group
	for _, skills := range bySource {
		sort.Slice(skills, func(i, j int) bool {
			return skills[i].Name < skills[j].Name
		})
	}

	return bySource
}

// getSourceLabel returns a human-readable label for a source path
func (r *Registry) getSourceLabel(sourcePath string) string {
	switch {
	case sourcePath == r.paths.ProfilePath:
		return "Profile"
	case sourcePath == r.paths.GlobalPath:
		return "Global"
	case contains(r.paths.UserPaths, sourcePath):
		return "User"
	case contains(r.paths.DetectedPaths, sourcePath):
		// Try to extract IDE name
		if strings.Contains(sourcePath, ".claude") {
			return "Claude Code"
		} else if strings.Contains(sourcePath, ".cursor") {
			return "Cursor"
		} else if strings.Contains(sourcePath, "Code") && strings.Contains(sourcePath, "vscode") {
			return "VS Code"
		} else if strings.Contains(sourcePath, "Zed") {
			return "Zed"
		} else if strings.Contains(sourcePath, ".windsurf") {
			return "Windsurf"
		}
		return "Detected"
	default:
		return "Other"
	}
}

// ToPromptXML generates Agent Skills XML for LLM context
func (r *Registry) ToPromptXML() string {
	return r.ToPromptXMLFiltered(nil)
}

// ToPromptXMLFiltered generates XML with filtering
func (r *Registry) ToPromptXMLFiltered(filter *SkillFilter) string {
	skills := r.List()
	if len(skills) == 0 {
		return ""
	}

	// Apply filter
	var filtered []*Skill
	for _, skill := range skills {
		if filter == nil || filter.Matches(skill) {
			filtered = append(filtered, skill)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	for _, skill := range filtered {
		sb.WriteString("  <skill>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", xmlEscape(skill.Name)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", xmlEscape(skill.Description)))
		sb.WriteString(fmt.Sprintf("    <location>%s</location>\n", xmlEscape(filepath.Join(skill.BasePath, "SKILL.md"))))
		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}

// ToPromptXMLWithMetadata generates verbose XML including metadata
func (r *Registry) ToPromptXMLWithMetadata() string {
	return r.ToPromptXMLWithMetadataFiltered(nil)
}

// ToPromptXMLWithMetadataFiltered generates verbose XML with filtering
func (r *Registry) ToPromptXMLWithMetadataFiltered(filter *SkillFilter) string {
	skills := r.List()
	if len(skills) == 0 {
		return ""
	}

	// Apply filter
	var filtered []*Skill
	for _, skill := range skills {
		if filter == nil || filter.Matches(skill) {
			filtered = append(filtered, skill)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	for _, skill := range filtered {
		sb.WriteString("  <skill>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", xmlEscape(skill.Name)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", xmlEscape(skill.Description)))
		sb.WriteString(fmt.Sprintf("    <location>%s</location>\n", xmlEscape(filepath.Join(skill.BasePath, "SKILL.md"))))

		if skill.License != "" {
			sb.WriteString(fmt.Sprintf("    <license>%s</license>\n", xmlEscape(skill.License)))
		}

		if skill.Compatibility != "" {
			sb.WriteString(fmt.Sprintf("    <compatibility>%s</compatibility>\n", xmlEscape(skill.Compatibility)))
		}

		if len(skill.Metadata) > 0 {
			sb.WriteString("    <metadata>\n")
			for k, v := range skill.Metadata {
				sb.WriteString(fmt.Sprintf("      <%s>%s</%s>\n", xmlEscape(k), xmlEscape(v), xmlEscape(k)))
			}
			sb.WriteString("    </metadata>\n")
		}

		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}

// ToPromptXMLForPlatform generates XML filtered by platform (OS)
func (r *Registry) ToPromptXMLForPlatform(platform string) string {
	filter := &SkillFilter{
		Platform:       platform,
		CompatibleOnly: true,
	}
	return r.ToPromptXMLFiltered(filter)
}

// ToPromptXMLForProtocol generates XML filtered by protocol
func (r *Registry) ToPromptXMLForProtocol(protocol string) string {
	filter := &SkillFilter{
		Protocol: protocol,
	}
	return r.ToPromptXMLFiltered(filter)
}

// ToPromptXMLForIsolation generates XML filtered by isolation level
func (r *Registry) ToPromptXMLForIsolation(isolationLevel string) string {
	filter := &SkillFilter{
		IsolationLevel: isolationLevel,
	}
	return r.ToPromptXMLFiltered(filter)
}

// Count returns the number of discovered skills
func (r *Registry) Count() int {
	return len(r.skills)
}

// GetPaths returns the paths configuration
func (r *Registry) GetPaths() *SkillPaths {
	return r.paths
}

// xmlEscape escapes special XML characters
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
