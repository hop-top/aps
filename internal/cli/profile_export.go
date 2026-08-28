package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/core"
)

// agentcoFrontmatter is the frontmatter shape written by
// --manifest-format agentco. reportsTo is emitted whenever the profile
// carries one so a reporting hierarchy survives an export/import
// round-trip; graph integrity is validated separately by `aps org
// check`. Only identity-level fields appear — never secrets, isolation,
// gitconfig, knowledge references, or machine paths.
type agentcoFrontmatter struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title,omitempty"`
	Slug        string   `yaml:"slug,omitempty"`
	Description string   `yaml:"description,omitempty"`
	ReportsTo   string   `yaml:"reportsTo,omitempty"`
	Skills      []string `yaml:"skills,omitempty"`
}

// renderAgentcoManifest renders a profile as an agent role manifest
// (AGENTS.md): YAML frontmatter + markdown body. body is the profile's
// notes.md content (the same file manifest import writes to).
func renderAgentcoManifest(p *core.Profile, body string) (string, error) {
	fm := agentcoFrontmatter{
		Name:        p.DisplayName,
		Slug:        p.ID,
		Description: p.Description,
		ReportsTo:   p.ReportsTo,
		Skills:      p.Capabilities,
	}
	if fm.Name == "" {
		fm.Name = p.ID
	}

	data, err := yaml.Marshal(&fm)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(data)
	b.WriteString("---\n")
	if body = strings.TrimSpace(body); body != "" {
		b.WriteString("\n")
		b.WriteString(body)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// exportFormat declares one profile export format: the
// --manifest-format value (empty for the flag-omitted default), the
// doc-facing description of the rendering and of what it emits, and
// the renderer itself. runProfileExport dispatches through this table
// and derives the unknown-format error's supported list from it, so a
// format cannot be declared without a renderer nor rendered without a
// declaration.
type exportFormat struct {
	// Name is the --manifest-format value; "" means the flag was
	// omitted (native yaml default).
	Name string
	// Summary is the one-line rendering description shown in the
	// generated docs enumeration.
	Summary string
	// Emits describes the export's data scope — what the output
	// contains and, where it matters, what it never contains.
	Emits string
	// render writes the export for an already-loaded profile.
	render func(profile *core.Profile, id string, out io.Writer) error
}

// profileExportFormats is the single source of truth for `aps profile
// export` formats, in docs order: the flag-omitted default first, then
// named formats sorted by name. The enumeration in
// docs/cli/reference.md is generated from this table via
// internal/tools/exportmd (`make docs-gen`).
var profileExportFormats = []exportFormat{
	{
		Name:    "",
		Summary: "Native profile record (default when the flag is omitted)",
		Emits:   "The full `profile.yaml` record as stored on disk; secret values live in `secrets.env` and are never part of the record.",
		render:  renderNativeExport,
	},
	{
		Name:    "agentco",
		Summary: "Agent role manifest (`AGENTS.md`: YAML frontmatter + markdown body)",
		Emits:   "Identity only — name, slug, description, reportsTo, skills, and the `notes.md` body; never secrets, isolation, gitconfig, knowledge references, or machine paths.",
		render:  renderAgentcoExport,
	},
}

// ExportFormatDoc is the doc-facing projection of one export format
// registry row, consumed by internal/tools/exportmd when regenerating
// the enumeration in docs/cli/reference.md.
type ExportFormatDoc struct {
	// Flag is the --manifest-format value; empty means the flag is
	// omitted (default format).
	Flag    string
	Summary string
	Emits   string
}

// ExportFormatDocs returns the declared profile export formats in
// registry (docs) order.
func ExportFormatDocs() []ExportFormatDoc {
	docs := make([]ExportFormatDoc, 0, len(profileExportFormats))
	for _, f := range profileExportFormats {
		docs = append(docs, ExportFormatDoc{Flag: f.Name, Summary: f.Summary, Emits: f.Emits})
	}
	return docs
}

// lookupExportFormat resolves a --manifest-format value against the
// registry; ok is false for undeclared formats.
func lookupExportFormat(name string) (exportFormat, bool) {
	for _, f := range profileExportFormats {
		if f.Name == name {
			return f, true
		}
	}
	return exportFormat{}, false
}

// supportedExportFormats joins the named (non-default) registry
// formats for the unknown-format error, in registry order.
func supportedExportFormats() string {
	names := make([]string, 0, len(profileExportFormats))
	for _, f := range profileExportFormats {
		if f.Name != "" {
			names = append(names, f.Name)
		}
	}
	return strings.Join(names, ", ")
}

// renderNativeExport dumps the native profile.yaml record.
func renderNativeExport(profile *core.Profile, _ string, out io.Writer) error {
	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("writing profile: %w", err)
	}
	return nil
}

// renderAgentcoExport renders an agent role manifest, pulling the body
// from the profile's notes.md when present.
func renderAgentcoExport(profile *core.Profile, id string, out io.Writer) error {
	body := ""
	if dir, err := core.GetProfileDir(id); err == nil {
		// #nosec G304 -- notes.md is resolved under the profile's own
		// directory, not from caller-supplied input.
		if notes, err := os.ReadFile(filepath.Join(dir, "notes.md")); err == nil {
			body = string(notes)
		}
	}
	doc, err := renderAgentcoManifest(profile, body)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(out, doc); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// runProfileExport writes a profile export to out. The format is
// resolved against profileExportFormats: "" (default) dumps the native
// profile.yaml record; named formats render their registry row.
// Undeclared formats error with the registry-derived supported list.
func runProfileExport(id, format string, out io.Writer) error {
	profile, err := core.LoadProfile(id)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}

	f, ok := lookupExportFormat(format)
	if !ok {
		return fmt.Errorf("unknown export format %q (supported: %s, or omit for native yaml)",
			format, supportedExportFormats())
	}
	return f.render(profile, id, out)
}

// profileExportCmd implements `aps profile export <id>`.
var profileExportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export a profile record",
	Long: `Export a profile to stdout (or --out <path>). The default
output is the native profile.yaml record. --manifest-format agentco
renders an agent role manifest (AGENTS.md — YAML frontmatter + markdown
body): name from the display name, slug from the profile id,
skills from the linked capability shortnames, and the body from
notes.md (the same file manifest import writes).

The agentco format exports identity only — secrets, isolation
config, gitconfig, knowledge references, and machine-specific
paths are never included.

Read-only: loads profile state; writes only to stdout or the
--out destination. Idempotent.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (retErr error) {
		id := args[0]
		format, _ := cmd.Flags().GetString("manifest-format")
		outPath, _ := cmd.Flags().GetString("out")

		var out io.Writer = os.Stdout
		if outPath != "" {
			// #nosec G304 -- outPath is the operator's --out argument;
			// writing there is the flag's purpose.
			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			out = f
			// Close is checked, not deferred-and-dropped: a write-back
			// error on close means the exported manifest is truncated.
			defer func() {
				if cerr := f.Close(); cerr != nil && retErr == nil {
					retErr = fmt.Errorf("closing output file: %w", cerr)
				}
			}()
		}
		if err := runProfileExport(id, format, out); err != nil {
			return err
		}
		if outPath != "" {
			fmt.Printf("Profile '%s' exported to %s.\n", id, outPath)
		}
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileExportCmd)
	// Named manifest-format (not format) so it does not shadow kit's
	// global --format output-mode flag (table|json|yaml); the signature
	// validator rejects leaf-level redefinition of globals at startup.
	// Same qualified <noun>-format pattern as profile create's
	// --avatar-format.
	profileExportCmd.Flags().String("manifest-format", "", "Manifest format: agentco (agent role manifest); omit for native yaml")
	profileExportCmd.Flags().String("out", "", "Write to a file instead of stdout")

	kitcli.SetSideEffect(profileExportCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(profileExportCmd, kitcli.IdempotencyYes)
}
