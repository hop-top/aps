package skills

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents an Agent Skill
type Skill struct {
	// Frontmatter fields
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`

	// Body content (after frontmatter)
	BodyContent string `yaml:"-"`

	// Runtime metadata
	BasePath   string `yaml:"-"` // Absolute path to skill directory
	SourcePath string `yaml:"-"` // Which search path found this skill
}

// Frontmatter validation constraints
const (
	MaxNameLength          = 64
	MaxDescriptionLength   = 1024
	MaxCompatibilityLength = 500
)

var (
	// Name must be lowercase alphanumeric + hyphens, no leading/trailing/consecutive hyphens
	namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	ErrInvalidSkillFormat   = errors.New("invalid SKILL.md format")
	ErrMissingFrontmatter   = errors.New("missing YAML frontmatter")
	ErrInvalidName          = errors.New("invalid skill name")
	ErrInvalidDescription   = errors.New("invalid description")
	ErrNameMismatch         = errors.New("skill name does not match directory name")
	ErrMissingSkillFile     = errors.New("SKILL.md not found")
)

// ParseSkill reads and parses a SKILL.md file from a directory
func ParseSkill(skillPath string) (*Skill, error) {
	skillMdPath := filepath.Join(skillPath, "SKILL.md")

	// Check if SKILL.md exists
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrMissingSkillFile, skillMdPath)
	}

	// Read file
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// Parse frontmatter
	skill, bodyContent, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill.BodyContent = bodyContent
	skill.BasePath = skillPath

	// Validate name matches directory
	dirName := filepath.Base(skillPath)
	if skill.Name != dirName {
		return nil, fmt.Errorf("%w: skill name '%s' does not match directory '%s'", ErrNameMismatch, skill.Name, dirName)
	}

	// Validate skill
	if err := skill.Validate(); err != nil {
		return nil, err
	}

	return skill, nil
}

// parseFrontmatter extracts YAML frontmatter and body content
func parseFrontmatter(content []byte) (*Skill, string, error) {
	// Look for YAML frontmatter delimiters (---)
	lines := bytes.Split(content, []byte("\n"))

	if len(lines) < 3 || !bytes.Equal(bytes.TrimSpace(lines[0]), []byte("---")) {
		return nil, "", ErrMissingFrontmatter
	}

	// Find closing ---
	var endIdx int
	found := false
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte("---")) {
			endIdx = i
			found = true
			break
		}
	}

	if !found {
		return nil, "", ErrMissingFrontmatter
	}

	// Extract frontmatter YAML
	frontmatterBytes := bytes.Join(lines[1:endIdx], []byte("\n"))

	var skill Skill
	if err := yaml.Unmarshal(frontmatterBytes, &skill); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract body content (after closing ---)
	bodyLines := lines[endIdx+1:]
	bodyContent := string(bytes.Join(bodyLines, []byte("\n")))

	return &skill, strings.TrimSpace(bodyContent), nil
}

// Validate checks if the skill meets Agent Skills specification
func (s *Skill) Validate() error {
	// Validate name
	if s.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidName)
	}
	if len(s.Name) > MaxNameLength {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidName, MaxNameLength)
	}
	if !namePattern.MatchString(s.Name) {
		return fmt.Errorf("%w: name must be lowercase alphanumeric with hyphens, no leading/trailing/consecutive hyphens", ErrInvalidName)
	}

	// Validate description
	if s.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidDescription)
	}
	if len(s.Description) > MaxDescriptionLength {
		return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidDescription, MaxDescriptionLength)
	}

	// Validate compatibility (optional)
	if s.Compatibility != "" && len(s.Compatibility) > MaxCompatibilityLength {
		return fmt.Errorf("compatibility field exceeds %d characters", MaxCompatibilityLength)
	}

	return nil
}

// HasScript checks if a script exists in the skill's scripts/ directory
func (s *Skill) HasScript(scriptName string) bool {
	scriptPath := filepath.Join(s.BasePath, "scripts", scriptName)
	info, err := os.Stat(scriptPath)
	return err == nil && !info.IsDir()
}

// GetScriptPath returns the absolute path to a script
func (s *Skill) GetScriptPath(scriptName string) string {
	return filepath.Join(s.BasePath, "scripts", scriptName)
}

// HasReference checks if a reference file exists
func (s *Skill) HasReference(refName string) bool {
	refPath := filepath.Join(s.BasePath, "references", refName)
	info, err := os.Stat(refPath)
	return err == nil && !info.IsDir()
}

// GetReferencePath returns the absolute path to a reference file
func (s *Skill) GetReferencePath(refName string) string {
	return filepath.Join(s.BasePath, "references", refName)
}

// HasAsset checks if an asset exists
func (s *Skill) HasAsset(assetName string) bool {
	assetPath := filepath.Join(s.BasePath, "assets", assetName)
	_, err := os.Stat(assetPath)
	return err == nil
}

// GetAssetPath returns the absolute path to an asset
func (s *Skill) GetAssetPath(assetName string) string {
	return filepath.Join(s.BasePath, "assets", assetName)
}

// ListScripts returns all scripts in the skill's scripts/ directory
func (s *Skill) ListScripts() ([]string, error) {
	scriptsDir := filepath.Join(s.BasePath, "scripts")
	entries, err := os.ReadDir(scriptsDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			scripts = append(scripts, entry.Name())
		}
	}
	return scripts, nil
}

// ListReferences returns all reference files
func (s *Skill) ListReferences() ([]string, error) {
	refsDir := filepath.Join(s.BasePath, "references")
	entries, err := os.ReadDir(refsDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var refs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			refs = append(refs, entry.Name())
		}
	}
	return refs, nil
}

// ListAssets returns all asset files
func (s *Skill) ListAssets() ([]string, error) {
	assetsDir := filepath.Join(s.BasePath, "assets")
	entries, err := os.ReadDir(assetsDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var assets []string
	for _, entry := range entries {
		if !entry.IsDir() {
			assets = append(assets, entry.Name())
		}
	}
	return assets, nil
}
