package session

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"oss-aps-cli/internal/core/session"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			profileFilter, _ := cmd.Flags().GetString("profile")
			statusFilter, _ := cmd.Flags().GetString("status")
			tierFilter, _ := cmd.Flags().GetString("tier")

			registry := session.GetRegistry()
			sessions := registry.List()

			if len(sessions) == 0 {
				fmt.Println("No active sessions")
				return nil
			}

			sessions = filterSessions(sessions, profileFilter, statusFilter, tierFilter)

			if len(sessions) == 0 {
				fmt.Println("No sessions match the specified filters")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tPROFILE\tPID\tSTATUS\tTIER\tCREATED\tLAST SEEN")
			for _, s := range sessions {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
					s.ID,
					s.ProfileID,
					s.PID,
					s.Status,
					s.Tier,
					s.CreatedAt.Format("2006-01-02 15:04:05"),
					s.LastSeenAt.Format("15:04:05"))
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().String("profile", "", "Filter sessions by profile ID")
	cmd.Flags().String("status", "", "Filter sessions by status (active, inactive, errored)")
	cmd.Flags().String("tier", "", "Filter sessions by tier (basic, standard, premium)")

	return cmd
}

func filterSessions(sessions []*session.SessionInfo, profileFilter, statusFilter, tierFilter string) []*session.SessionInfo {
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
		filtered = append(filtered, s)
	}

	return filtered
}
