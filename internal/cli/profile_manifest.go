package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/manifest"
)

// isAgentManifestPath reports whether the import argument points at an
// agent role manifest (markdown with YAML frontmatter) rather than an
// aps profile bundle. Dispatch is by extension: .md → manifest.
func isAgentManifestPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".md")
}

var slugScrub = regexp.MustCompile(`[^a-z0-9]+`)

// slugifyManifestName lowercases a manifest name and collapses any
// non-alphanumeric runs into single hyphens, producing a profile id.
func slugifyManifestName(name string) string {
	s := slugScrub.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// manifestToProfile maps a parsed agent manifest to a profile id and a
// core.Profile record. Precedence:
//   - id: idOverride > manifest slug > slugified manifest name
//   - display name: title > name
//
// The returned profile carries identity fields only — capabilities are
// linked separately after partitioning, and secrets/isolation/paths are
// never populated from a manifest (the normal create path provides
// defaults).
func manifestToProfile(m *manifest.AgentManifest, idOverride string) (string, core.Profile) {
	id := idOverride
	if id == "" {
		id = m.Slug
	}
	if id == "" {
		id = slugifyManifestName(m.Name)
	}

	displayName := m.Title
	if displayName == "" {
		displayName = m.Name
	}

	return id, core.Profile{
		DisplayName: displayName,
		Description: m.Description,
		ReportsTo:   m.ReportsTo,
	}
}

// partitionManifestSkills splits manifest skill shortnames into those
// resolvable in the capability registry (linkable) and the rest
// (skipped). Order is preserved within each partition.
func partitionManifestSkills(skills []string, exists func(string) bool) (linkable, skipped []string) {
	for _, s := range skills {
		if exists(s) {
			linkable = append(linkable, s)
		} else {
			skipped = append(skipped, s)
		}
	}
	return linkable, skipped
}

// profileExists reports whether a profile directory with a profile.yaml
// is present for id. The bool is only meaningful when err is nil; an
// unreadable data dir returns an error so callers can stay silent
// rather than warn about a profile that may well exist.
func profileExists(id string) (bool, error) {
	path, err := core.GetProfilePath(id)
	if err != nil {
		return false, fmt.Errorf("resolving profile path: %w", err)
	}
	switch _, err := os.Stat(path); {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, fmt.Errorf("checking profile %q: %w", id, err)
	}
}

// checkReportsTo verifies that reportsTo names a profile on disk.
// A dangling reference fails the import so a typo'd or out-of-order
// manifest is caught at the boundary rather than persisted as silent
// data rot; force downgrades the failure to a stderr warning for the
// legitimate case of importing a hierarchy before its supervisors.
//
// A failed existence check (unreadable data dir) is reported as-is
// rather than treated as "missing" — the profile may well exist.
func checkReportsTo(reportsTo string, force bool, errOut io.Writer) error {
	if reportsTo == "" {
		return nil
	}
	exists, err := profileExists(reportsTo)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if force {
		//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
		fmt.Fprintf(errOut,
			"Warning: reportsTo %q does not match an existing profile; kept as-is (--force)\n",
			reportsTo)
		return nil
	}
	return fmt.Errorf(
		"reportsTo %q does not match an existing profile\n\nHint: import the supervising profile first, or pass --force to import anyway",
		reportsTo)
}

// runManifestImport imports an agent role manifest file as a profile.
// dryRun previews the resulting profile.yaml, intended capability
// links, and skipped shortnames without writing anything. Missing
// capabilities never fail the import — they are warned to errOut and
// skipped. A dangling reportsTo does fail it, unless force downgrades
// the failure to a warning.
func runManifestImport(ctx context.Context, path, idOverride string, dryRun, force bool, out, errOut io.Writer) error {
	m, err := manifest.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	id, profile := manifestToProfile(m, idOverride)
	if id == "" {
		return fmt.Errorf("could not derive a profile id from manifest %q; pass --id", path)
	}

	linkable, skipped := partitionManifestSkills(m.Skills, capability.Exists)

	// Guarded before any write (and before the dry-run preview, so the
	// preview reports the same verdict the real import would).
	if err := checkReportsTo(profile.ReportsTo, force, errOut); err != nil {
		return err
	}

	if dryRun {
		profile.ID = id
		data, err := yaml.Marshal(&profile)
		if err != nil {
			return fmt.Errorf("marshaling profile preview: %w", err)
		}
		//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
		fmt.Fprintf(out, "# profile.yaml (dry-run — nothing written)\n%s", data)
		if len(linkable) > 0 {
			//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
			fmt.Fprintf(out, "# would link capabilities: %s\n", strings.Join(linkable, ", "))
		}
		if len(skipped) > 0 {
			//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
			fmt.Fprintf(out, "# would skip (not in capability registry): %s\n", strings.Join(skipped, ", "))
		}
		if m.Body != "" {
			//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
			fmt.Fprintf(out, "# would write manifest body to notes.md (%d bytes)\n", len(m.Body))
		}
		return nil
	}

	if err := core.CreateProfileWithContext(ctx, id, profile); err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}

	// Manifest body → notes.md, the profile's free-form markdown
	// side-car (persona plumbing composes prompts from structured
	// fields; notes.md is the body's home).
	if m.Body != "" {
		dir, err := core.GetProfileDir(id)
		if err != nil {
			return fmt.Errorf("resolving profile dir: %w", err)
		}
		// #nosec G306 -- matches the 0644 core.CreateProfile uses for the
		// notes.md it seeds; an imported profile must not differ from a
		// created one. Not secret material (that is secrets.env, 0600).
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(m.Body+"\n"), 0o644); err != nil {
			return fmt.Errorf("writing notes.md: %w", err)
		}
	}

	for _, capName := range linkable {
		if err := core.AddCapabilityToProfileWithContext(ctx, id, capName); err != nil {
			return fmt.Errorf("linking capability %q: %w", capName, err)
		}
	}
	for _, s := range skipped {
		//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
		fmt.Fprintf(errOut, "Warning: skill %q not found in capability registry; skipped\n", s)
	}

	//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
	fmt.Fprintf(out, "Profile '%s' imported from manifest %s.\n", id, path)
	if len(linkable) > 0 {
		//nolint:errcheck // CLI progress output; a failed write to stdout/stderr is not actionable
		fmt.Fprintf(out, "Linked capabilities: %s\n", strings.Join(linkable, ", "))
	}
	return nil
}
