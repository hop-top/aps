package session

import (
	"fmt"
	"os"
	"text/tabwriter"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/styles"
)

var (
	tableHeader = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)
)

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

			if len(sessions) == 0 {
				fmt.Println(styles.Dim.Render("No active sessions"))
				return nil
			}

			sessions = filterSessions(sessions, profileFilter, statusFilter, tierFilter, workspaceFilter, typeFilter)

			if len(sessions) == 0 {
				fmt.Println(styles.Dim.Render("No sessions match the specified filters"))
				return nil
			}

			fmt.Printf("%s\n\n", styles.Title.Render("Sessions"))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, tableHeader.Render("ID")+"\t"+
				tableHeader.Render("PROFILE")+"\t"+
				tableHeader.Render("WORKSPACE")+"\t"+
				tableHeader.Render("PID")+"\t"+
				tableHeader.Render("STATUS")+"\t"+
				tableHeader.Render("TIER")+"\t"+
				tableHeader.Render("CREATED")+"\t"+
				tableHeader.Render("LAST SEEN"))

			for _, s := range sessions {
				wsID := s.WorkspaceID
				if wsID == "" {
					wsID = styles.Dim.Render("--")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
					s.ID,
					s.ProfileID,
					wsID,
					s.PID,
					styles.SessionStatusBadge(string(s.Status)),
					styles.TierBadge(string(s.Tier)),
					styles.Dim.Render(s.CreatedAt.Format("2006-01-02 15:04:05")),
					styles.Dim.Render(s.LastSeenAt.Format("15:04:05")))
			}
			w.Flush()

			summary := fmt.Sprintf("%d sessions", len(sessions))
			fmt.Printf("\n%s\n", styles.Dim.Render(summary))

			return nil
		},
	}

	// --profile and --workspace are persistent globals (T-0376); reads via
	// cmd.Flags().GetString resolve through the inherited PersistentFlags.
	cmd.Flags().String("status", "", "Filter sessions by status (active, inactive, errored)")
	cmd.Flags().String("tier", "", "Filter sessions by tier (basic, standard, premium)")
	cmd.Flags().String("type", "", "Filter sessions by type (standard, voice); default = all")

	return cmd
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

func filterSessions(sessions []*session.SessionInfo, profileFilter, statusFilter, tierFilter, workspaceFilter, typeFilter string) []*session.SessionInfo {
	var filtered []*session.SessionInfo

	for _, s := range sessions {
		if profileFilter != "" && s.ProfileID != profileFilter {
			continue
		}
		if statusFilter != "" && string(s.Status) != statusFilter {
			continue
		}
		if tierFilter != "" && string(s.Tier) != tierFilter {
			continue
		}
		if workspaceFilter != "" && s.WorkspaceID != workspaceFilter {
			continue
		}
		if !matchesTypeFilter(s, typeFilter) {
			continue
		}
		filtered = append(filtered, s)
	}

	return filtered
}
