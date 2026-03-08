package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List built-in and user bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type bundleRow struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

func runList(jsonOut bool) error {
	builtins, err := corebundle.LoadBuiltins()
	if err != nil {
		return fmt.Errorf("failed to load built-in bundles: %w", err)
	}

	userBundles, err := corebundle.LoadUserOverrides()
	if err != nil {
		return fmt.Errorf("failed to load user bundles: %w", err)
	}

	// Build set of built-in names for override detection.
	builtinNames := make(map[string]bool, len(builtins))
	for _, b := range builtins {
		builtinNames[b.Name] = true
	}

	// Build set of user bundle names.
	userNames := make(map[string]bool, len(userBundles))
	for _, b := range userBundles {
		userNames[b.Name] = true
	}

	var rows []bundleRow

	// Built-ins that are NOT overridden.
	for _, b := range builtins {
		if userNames[b.Name] {
			continue // will show as user (overrides built-in)
		}
		rows = append(rows, bundleRow{
			Name:        b.Name,
			Source:      "built-in",
			Description: b.Description,
		})
	}

	// User bundles.
	for _, b := range userBundles {
		source := "user"
		if builtinNames[b.Name] {
			source = "user (overrides built-in)"
		}
		rows = append(rows, bundleRow{
			Name:        b.Name,
			Source:      source,
			Description: b.Description,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	if jsonOut {
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if len(rows) == 0 {
		fmt.Println(dimStyle.Render("No bundles found."))
		return nil
	}

	fmt.Printf("%s\n\n", headerStyle.Render("Bundles"))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tableHeader.Render("NAME")+"\t"+
		tableHeader.Render("SOURCE")+"\t"+
		tableHeader.Render("DESCRIPTION"))

	for _, r := range rows {
		src := renderSource(r.Source)
		desc := r.Description
		if desc == "" {
			desc = dimStyle.Render("--")
		}
		fmt.Fprintf(w, "%-20s\t%s\t%s\n", r.Name, src, desc)
	}
	w.Flush()

	builtinCount := 0
	userCount := 0
	for _, r := range rows {
		if r.Source == "built-in" {
			builtinCount++
		} else {
			userCount++
		}
	}
	fmt.Printf("\n%s\n", dimStyle.Render(fmt.Sprintf(
		"%d bundles (%d built-in, %d user)", len(rows), builtinCount, userCount)))

	return nil
}

func renderSource(source string) string {
	switch source {
	case "built-in":
		return builtinStyle.Render("built-in")
	case "user":
		return userStyle.Render("user")
	default:
		// "user (overrides built-in)"
		return userStyle.Render(source)
	}
}
