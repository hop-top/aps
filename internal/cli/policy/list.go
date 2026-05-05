package policy

import (
	"encoding/json"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/multidevice"
	"hop.top/kit/go/console/output"

	"github.com/spf13/cobra"
)

// policyRow is the table row shape for `aps policy list`. T-0456 —
// moved off hand-rolled tabwriter so styled tables activate on a TTY.
// Settings ("Mode", "Allowed Devices", "Denied Devices") are emitted
// as separate rows so per-device entries align in the same table.
type policyRow struct {
	Setting string `table:"SETTING,priority=10" json:"setting" yaml:"setting"`
	Value   string `table:"VALUE,priority=9"    json:"value"   yaml:"value"`
}

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

	rows := []policyRow{
		{Setting: "Mode", Value: policyModeBadge(string(policy.Mode))},
	}

	if len(policy.AllowDevices) > 0 {
		for i, dev := range policy.AllowDevices {
			setting := ""
			if i == 0 {
				setting = "Allowed Devices"
			}
			rows = append(rows, policyRow{Setting: setting, Value: dev})
		}
	}

	if len(policy.DenyDevices) > 0 {
		for i, dev := range policy.DenyDevices {
			setting := ""
			if i == 0 {
				setting = "Denied Devices"
			}
			rows = append(rows, policyRow{Setting: setting, Value: dev})
		}
	}

	return listing.RenderList(os.Stdout, output.Table, rows)
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
