package session

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/session"
	"hop.top/kit/go/console/output"
)

// T-0456 — the package-level `tableHeader` lipgloss style was removed
// when `aps session inspect` migrated to listing.RenderList. Header
// styling now flows from the active kit/cli theme via the styled
// table renderer (TTY-only); non-TTY writers stay on plain tabwriter.

// sessionSummaryRow is the table/json/yaml row shape for `aps session
// list`. Higher-priority columns survive narrow terminals.
type sessionSummaryRow struct {
	ID          string `table:"ID,priority=10"         json:"id"            yaml:"id"`
	Profile     string `table:"PROFILE,priority=9"     json:"profile_id"    yaml:"profile_id"`
	Status      string `table:"STATUS,priority=8"      json:"status"        yaml:"status"`
	Workspace   string `table:"WORKSPACE,priority=7"   json:"workspace_id"  yaml:"workspace_id"`
	Type        string `table:"TYPE,priority=6"        json:"type"          yaml:"type"`
	Tier        string `table:"TIER,priority=5"        json:"tier"          yaml:"tier"`
	CreatedAt   string `table:"CREATED,priority=4"     json:"created_at"    yaml:"created_at"`
	LastSeenAt  string `table:"LAST SEEN,priority=3"   json:"last_seen_at"  yaml:"last_seen_at"`
}

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			profileFilter, _ := cmd.Flags().GetString("profile")
			statusFilter, _ := cmd.Flags().GetString("status")
			tierFilter, _ := cmd.Flags().GetString("tier")
			workspaceFilter, _ := cmd.Flags().GetString("workspace")
			typeFilter, _ := cmd.Flags().GetString("type")

			if err := validateTypeFilter(typeFilter); err != nil {
				return err
			}

			registry := session.GetRegistry()
			sessions := registry.List()

			pred := listing.All(
				listing.MatchString(func(s *session.SessionInfo) string { return s.ProfileID }, profileFilter),
				listing.MatchString(func(s *session.SessionInfo) string { return string(s.Status) }, statusFilter),
				listing.MatchString(func(s *session.SessionInfo) string { return string(s.Tier) }, tierFilter),
				listing.MatchString(func(s *session.SessionInfo) string { return s.WorkspaceID }, workspaceFilter),
				typePredicate(typeFilter),
			)
			filtered := listing.Filter(sessions, pred)

			rows := make([]sessionSummaryRow, 0, len(filtered))
			for _, s := range filtered {
				rows = append(rows, sessionToSummaryRow(s))
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "" {
				format = output.Table
			}
			return listing.RenderList(os.Stdout, format, rows)
		},
	}

	// --profile and --workspace are persistent globals (T-0376); reads via
	// cmd.Flags().GetString resolve through the inherited PersistentFlags.
	cmd.Flags().String("status", "", "Filter sessions by status (active, inactive, errored)")
	cmd.Flags().String("tier", "", "Filter sessions by tier (basic, standard, premium)")
	cmd.Flags().String("type", "", "Filter sessions by type (standard, voice); default = all")

	return cmd
}

// sessionToSummaryRow projects a SessionInfo into the row shape
// rendered by `aps session list`. Times are formatted in local time
// for table mode; json/yaml emit the same string so structured
// consumers see consistent values across formats.
func sessionToSummaryRow(s *session.SessionInfo) sessionSummaryRow {
	t := string(s.Type)
	if t == "" {
		// Empty SessionType is the legacy "standard" zero value;
		// surface it explicitly so users see something useful in
		// the TYPE column.
		t = "standard"
	}
	return sessionSummaryRow{
		ID:         s.ID,
		Profile:    s.ProfileID,
		Status:     string(s.Status),
		Workspace:  s.WorkspaceID,
		Type:       t,
		Tier:       string(s.Tier),
		CreatedAt:  s.CreatedAt.Format("2006-01-02 15:04:05"),
		LastSeenAt: s.LastSeenAt.Format("2006-01-02 15:04:05"),
	}
}

// typePredicate wraps the existing matchesTypeFilter helper so the
// type filter (which has special "" → "standard" semantics for legacy
// registry entries) can be composed alongside generic listing.MatchString
// predicates.
func typePredicate(typeFilter string) listing.Predicate[*session.SessionInfo] {
	if typeFilter == "" {
		return nil
	}
	return func(s *session.SessionInfo) bool {
		return matchesTypeFilter(s, typeFilter)
	}
}

// validateTypeFilter accepts "", "standard", or "voice".
func validateTypeFilter(typeFilter string) error {
	switch typeFilter {
	case "", "standard", "voice":
		return nil
	default:
		return fmt.Errorf("invalid --type %q (expected: standard, voice)", typeFilter)
	}
}

// matchesTypeFilter returns true when the session's Type matches the
// CLI filter. Empty filter matches all. "standard" matches the
// zero-value SessionType (legacy entries written before the field
// existed) so unfiltered listings stay backward-compatible.
func matchesTypeFilter(s *session.SessionInfo, typeFilter string) bool {
	switch typeFilter {
	case "":
		return true
	case "standard":
		return s.Type == session.SessionTypeStandard
	case "voice":
		return s.Type == session.SessionTypeVoice
	default:
		return false
	}
}

// filterSessions retains the legacy signature so list_test.go keeps
// working. New callers should compose listing.Predicate directly.
func filterSessions(sessions []*session.SessionInfo, profileFilter, statusFilter, tierFilter, workspaceFilter, typeFilter string) []*session.SessionInfo {
	pred := listing.All(
		listing.MatchString(func(s *session.SessionInfo) string { return s.ProfileID }, profileFilter),
		listing.MatchString(func(s *session.SessionInfo) string { return string(s.Status) }, statusFilter),
		listing.MatchString(func(s *session.SessionInfo) string { return string(s.Tier) }, tierFilter),
		listing.MatchString(func(s *session.SessionInfo) string { return s.WorkspaceID }, workspaceFilter),
		typePredicate(typeFilter),
	)
	return listing.Filter(sessions, pred)
}
