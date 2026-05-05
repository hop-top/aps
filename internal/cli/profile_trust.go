package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"
)

// trustScoreRow is the table/json/yaml row shape for the per-domain
// score block of `aps profile trust`. T-0456 — moved off hand-rolled
// tabwriter so the kit-themed styled renderer activates on a TTY.
type trustScoreRow struct {
	Domain string  `table:"DOMAIN,priority=10" json:"domain" yaml:"domain"`
	Score  float64 `table:"SCORE,priority=9"   json:"score"  yaml:"score"`
}

// trustHistoryRow is the table row shape for the optional --history
// block. Times render as a single string for human-friendly column
// alignment; structured (json/yaml) formats keep them in the same
// shape because we don't currently expose --format on this subcommand.
type trustHistoryRow struct {
	Timestamp  string  `table:"TIMESTAMP,priority=10"  json:"timestamp"  yaml:"timestamp"`
	Domain     string  `table:"DOMAIN,priority=9"      json:"domain"     yaml:"domain"`
	Delta      float64 `table:"DELTA,priority=8"       json:"delta"      yaml:"delta"`
	TaskRef    string  `table:"TASK,priority=7"        json:"task_ref"   yaml:"task_ref"`
	Difficulty string  `table:"DIFFICULTY,priority=6"  json:"difficulty" yaml:"difficulty"`
}

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
	domains := core.TrustDomains
	if domain != "" {
		domains = []string{domain}
	}
	scoreRows := make([]trustScoreRow, 0, len(domains))
	for _, d := range domains {
		scoreRows = append(scoreRows, trustScoreRow{
			Domain: d,
			Score:  p.TrustScore(d),
		})
	}
	if err := listing.RenderList(os.Stdout, output.Table, scoreRows); err != nil {
		return err
	}

	// History
	if history {
		entries := p.TrustHistory(domain)
		fmt.Printf("\n%s (%d entries)\n",
			styles.Bold.Render("History"), len(entries))

		if len(entries) > 0 {
			historyRows := make([]trustHistoryRow, 0, len(entries))
			for _, e := range entries {
				historyRows = append(historyRows, trustHistoryRow{
					Timestamp:  e.Timestamp.Format("2006-01-02 15:04"),
					Domain:     e.Domain,
					Delta:      e.Delta,
					TaskRef:    e.TaskRef,
					Difficulty: e.Difficulty,
				})
			}
			if err := listing.RenderList(os.Stdout, output.Table, historyRows); err != nil {
				return err
			}
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
