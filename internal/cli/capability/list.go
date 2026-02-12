package capability

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/capability"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var profileFilter string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all capabilities (builtin + external)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(profileFilter, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&profileFilter, "profile", "p", "",
		"Filter to capabilities on a specific profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type capRow struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Type     string   `json:"type"`
	Links    int      `json:"links"`
	Profiles []string `json:"profiles"`
}

func runList(profileFilter string, jsonOut bool) error {
	rows := buildCapRows(profileFilter)

	if len(rows) == 0 {
		fmt.Println(dimStyle.Render("No capabilities found."))
		fmt.Println(dimStyle.Render(
			"  Install: aps cap install <source> --name <name>"))
		return nil
	}

	if jsonOut {
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(headerStyle.Render("Capabilities"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("NAME")+"\t"+
		tableHeader.Render("KIND")+"\t"+
		tableHeader.Render("TYPE")+"\t"+
		tableHeader.Render("LINKS")+"\t"+
		tableHeader.Render("PROFILES"))

	for _, r := range rows {
		kind := styles.KindBadge(r.Kind)
		typ := styles.TypeBadge(r.Type)
		links := "--"
		if r.Links > 0 {
			links = fmt.Sprintf("%d", r.Links)
		}
		profs := "--"
		if len(r.Profiles) > 0 {
			profs = joinMax(r.Profiles, 3)
		}
		fmt.Fprintf(w, "%-18s\t%s\t%s\t%s\t%s\n",
			r.Name, kind, typ, links, profs)
	}
	w.Flush()

	builtins := 0
	externals := 0
	for _, r := range rows {
		if r.Kind == "builtin" {
			builtins++
		} else {
			externals++
		}
	}
	fmt.Printf("\n%s\n", dimStyle.Render(fmt.Sprintf(
		"%d capabilities (%d builtin, %d external)",
		len(rows), builtins, externals)))

	return nil
}

func buildCapRows(profileFilter string) []capRow {
	var rows []capRow
	seen := make(map[string]bool)

	// Builtins
	for _, b := range capability.ListBuiltins() {
		profs, _ := core.ProfilesUsingCapability(b.Name)
		if profileFilter != "" && !contains(profs, profileFilter) {
			continue
		}
		rows = append(rows, capRow{
			Name:     b.Name,
			Kind:     "builtin",
			Type:     "--",
			Profiles: profs,
		})
		seen[b.Name] = true
	}

	// Externals
	caps, _ := capability.List()
	for _, c := range caps {
		if seen[c.Name] {
			continue
		}
		profs, _ := core.ProfilesUsingCapability(c.Name)
		if profileFilter != "" && !contains(profs, profileFilter) {
			continue
		}
		rows = append(rows, capRow{
			Name:     c.Name,
			Kind:     "external",
			Type:     string(c.Type),
			Links:    len(c.Links),
			Profiles: profs,
		})
		seen[c.Name] = true
	}

	return rows
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func joinMax(ss []string, max int) string {
	if len(ss) <= max {
		return join(ss)
	}
	return join(ss[:max]) + fmt.Sprintf(" (+%d)", len(ss)-max)
}

func join(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
