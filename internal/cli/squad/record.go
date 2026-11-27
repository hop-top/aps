package squad

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/globals"
	coresquad "hop.top/aps/internal/core/squad"
	"hop.top/kit/go/console/output"
)

// squadRecord is the structured (json/yaml) view of one squad, shared by
// show and by the mutation commands that echo the resulting record. Keys
// are always present, so consumers can decode without presence checks;
// members is a list of profile IDs, [] when empty, never null. Only scope
// is optional, and is omitted when the squad has none.
type squadRecord struct {
	ID                string               `json:"id"                  yaml:"id"`
	Name              string               `json:"name"                yaml:"name"`
	Type              string               `json:"type"                yaml:"type"`
	Domain            string               `json:"domain"              yaml:"domain"`
	Description       string               `json:"description"         yaml:"description"`
	Members           []string             `json:"members"             yaml:"members"`
	GoldenPathDefined bool                 `json:"golden_path_defined" yaml:"golden_path_defined"`
	Scope             *coresquad.ScopeRule `json:"scope,omitempty"     yaml:"scope,omitempty"`
	CreatedAt         time.Time            `json:"created_at"          yaml:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"          yaml:"updated_at"`
}

func newSquadRecord(s *coresquad.Squad) squadRecord {
	members := make([]string, len(s.Members))
	copy(members, s.Members)
	return squadRecord{
		ID:                s.ID,
		Name:              s.Name,
		Type:              string(s.Type),
		Domain:            s.Domain,
		Description:       s.Description,
		Members:           members,
		GoldenPathDefined: s.GoldenPathDefined,
		Scope:             s.Scope,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

// deleteResult is the structured outcome of `squad delete`.
type deleteResult struct {
	ID      string `json:"id"      yaml:"id"`
	Deleted bool   `json:"deleted" yaml:"deleted"`
}

// formatOf resolves the output format: the --format flag when given on
// the command line, else the global (config/env backed) value. Empty means
// the default human/table rendering.
func formatOf(cmd *cobra.Command) string {
	f, _ := cmd.Flags().GetString("format")
	if f == "" {
		f = globals.Format()
	}
	return strings.ToLower(f)
}

// structured reports whether format asks for a machine-readable document
// on stdout. Everything else gets the human text.
func structured(format string) bool {
	switch strings.ToLower(format) {
	case output.JSON, output.YAML:
		return true
	}
	return false
}

// renderStructured writes v as one json/yaml document. It is the single
// path every squad command uses for structured output, through the same
// kit/output renderer as the other show commands.
func renderStructured(w io.Writer, format string, v any) error {
	if err := output.Render(w, strings.ToLower(format), v); err != nil {
		return fmt.Errorf("render squad: %w", err)
	}
	return nil
}
