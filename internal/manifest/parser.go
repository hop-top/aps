// Package manifest parses agent role manifest files (AGENTS.md): YAML
// frontmatter followed by a markdown body. It is a leaf boundary package —
// consumed by CLI commands only, never imported by profile schema packages.
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentManifest is the intermediate representation of a parsed agent
// manifest. All fields except Name are optional in the source document.
type AgentManifest struct {
	Name        string
	Title       string
	Slug        string
	Description string
	ReportsTo   string
	Skills      []string
	Body        string // markdown below frontmatter, trimmed
}

var (
	// ErrMissingFrontmatter indicates the document lacks a YAML
	// frontmatter block delimited by --- lines.
	ErrMissingFrontmatter = errors.New("missing YAML frontmatter")
	// ErrMissingName indicates the frontmatter omits the required name field.
	ErrMissingName = errors.New("manifest name is required")
)

// frontmatter mirrors the YAML shape. Unknown fields are ignored;
// reportsTo may be null (decodes to empty string via pointer).
type frontmatter struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Slug        string   `yaml:"slug"`
	Description string   `yaml:"description"`
	ReportsTo   *string  `yaml:"reportsTo"`
	Skills      []string `yaml:"skills"`
}

// crlf is normalized away before parsing so a manifest authored on
// Windows (or checked out with CRLF endings) yields the same body as
// the LF original. Without this the carriage returns survive into
// notes.md and back out through export, so a round-trip through a CRLF
// checkout would not be byte-identical.
var crlf = []byte("\r\n")

// Parse parses manifest content into an AgentManifest.
func Parse(content []byte) (*AgentManifest, error) {
	content = bytes.ReplaceAll(content, crlf, []byte("\n"))

	fmBytes, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var fm frontmatter
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if strings.TrimSpace(fm.Name) == "" {
		return nil, ErrMissingName
	}

	m := &AgentManifest{
		Name:        fm.Name,
		Title:       fm.Title,
		Slug:        fm.Slug,
		Description: fm.Description,
		Skills:      fm.Skills,
		Body:        body,
	}
	if fm.ReportsTo != nil {
		m.ReportsTo = *fm.ReportsTo
	}
	return m, nil
}

// ParseFile reads and parses a manifest file from disk.
func ParseFile(path string) (*AgentManifest, error) {
	// #nosec G304 -- path is the manifest argument the operator passes to
	// aps profile import; reading it is the command's purpose.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	return Parse(content)
}

// splitFrontmatter extracts the YAML frontmatter block and the trimmed
// markdown body. The document must open with a --- line and contain a
// closing --- line.
func splitFrontmatter(content []byte) (fm []byte, body string, err error) {
	lines := bytes.Split(content, []byte("\n"))

	if len(lines) < 3 || !bytes.Equal(bytes.TrimSpace(lines[0]), []byte("---")) {
		return nil, "", ErrMissingFrontmatter
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte("---")) {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return nil, "", ErrMissingFrontmatter
	}

	fm = bytes.Join(lines[1:endIdx], []byte("\n"))
	body = strings.TrimSpace(string(bytes.Join(lines[endIdx+1:], []byte("\n"))))
	return fm, body, nil
}
