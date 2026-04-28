package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/styles"
)

var profileTrustCmd = &cobra.Command{
	Use:   "trust <profile-id>",
	Short: "Show trust scores and history for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		domain, _ := cmd.Flags().GetString("domain")
		showHistory, _ := cmd.Flags().GetBool("history")
		jsonOut, _ := cmd.Flags().GetBool("json")

		profile, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}

		if jsonOut {
			return renderTrustJSON(profile, domain, showHistory)
		}

		return renderTrustTable(profile, domain, showHistory)
	},
}

func renderTrustJSON(p *core.Profile, domain string, history bool) error {
	out := struct {
		Roles   []string                `json:"roles"`
		Scores  map[string]float64      `json:"scores"`
		History []core.TrustEntry       `json:"history,omitempty"`
	}{
		Roles:  p.Roles,
		Scores: make(map[string]float64),
	}

	if p.TrustLedger != nil && p.TrustLedger.Scores != nil {
		out.Scores = p.TrustLedger.Scores
	}

	if domain != "" {
		// filter scores to requested domain
		filtered := make(map[string]float64)
		if v, ok := out.Scores[domain]; ok {
			filtered[domain] = v
		}
		out.Scores = filtered
	}

	if history {
		out.History = p.TrustHistory(domain)
	}

	return json.NewEncoder(os.Stdout).Encode(out)
}

func renderTrustTable(p *core.Profile, domain string, history bool) error {
	fmt.Printf("%s\n\n", styles.Title.Render(
		fmt.Sprintf("Trust — %s", p.ID)))

	// Roles
	if len(p.Roles) > 0 {
		fmt.Printf("Roles: %s\n\n",
			styles.Accent.Render(joinStrings(p.Roles, ", ")))
	} else {
		fmt.Printf("Roles: %s\n\n", styles.Dim.Render("(none)"))
	}

	// Scores
	fmt.Println(styles.Bold.Render("Scores"))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, profileTableHeader.Render("DOMAIN\tSCORE"))

	domains := core.TrustDomains
	if domain != "" {
		domains = []string{domain}
	}

	for _, d := range domains {
		score := p.TrustScore(d)
		fmt.Fprintf(w, "%s\t%.2f\n", d, score)
	}
	w.Flush()

	// History
	if history {
		entries := p.TrustHistory(domain)
		fmt.Printf("\n%s (%d entries)\n",
			styles.Bold.Render("History"), len(entries))

		if len(entries) > 0 {
			hw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(hw, profileTableHeader.Render(
				"TIMESTAMP\tDOMAIN\tDELTA\tTASK\tDIFFICULTY"))
			for _, e := range entries {
				ts := e.Timestamp.Format("2006-01-02 15:04")
				fmt.Fprintf(hw, "%s\t%s\t%+.2f\t%s\t%s\n",
					ts, e.Domain, e.Delta, e.TaskRef, e.Difficulty)
			}
			hw.Flush()
		}
	}

	return nil
}

func init() {
	profileCmd.AddCommand(profileTrustCmd)
	profileTrustCmd.Flags().String("domain", "", "Filter by trust domain")
	profileTrustCmd.Flags().Bool("history", false, "Show trust history entries")
	profileTrustCmd.Flags().Bool("json", false, "Output as JSON")
}
