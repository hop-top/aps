package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"oss-aps-cli/internal/core/multidevice"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list <workspace-id>",
		Aliases: []string{"ls"},
		Short:   "List workspace policies",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyList(args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runPolicyList(workspaceID string, jsonOut bool) error {
	policy, err := multidevice.LoadPolicy(workspaceID)
	if err != nil {
		return err
	}

	if jsonOut {
		data, err := json.MarshalIndent(policy, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Policies: %s", workspaceID)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w,
		tableHeader.Render("SETTING")+"\t"+
			tableHeader.Render("VALUE"))

	fmt.Fprintf(w, "Mode\t%s\n", policyModeBadge(string(policy.Mode)))

	if len(policy.AllowDevices) > 0 {
		for i, dev := range policy.AllowDevices {
			if i == 0 {
				fmt.Fprintf(w, "Allowed Devices\t%s\n", dev)
			} else {
				fmt.Fprintf(w, "\t%s\n", dev)
			}
		}
	}

	if len(policy.DenyDevices) > 0 {
		for i, dev := range policy.DenyDevices {
			if i == 0 {
				fmt.Fprintf(w, "Denied Devices\t%s\n", dev)
			} else {
				fmt.Fprintf(w, "\t%s\n", dev)
			}
		}
	}

	w.Flush()

	return nil
}

func policyModeBadge(mode string) string {
	switch mode {
	case "allow-all":
		return successStyle.Render("allow-all")
	case "allow-list":
		return warnStyle.Render("allow-list")
	case "deny-list":
		return warnStyle.Render("deny-list")
	default:
		return dimStyle.Render(mode)
	}
}
