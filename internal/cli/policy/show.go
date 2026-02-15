package policy

import (
	"encoding/json"
	"fmt"

	"oss-aps-cli/internal/core/multidevice"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var (
		workspaceID string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show effective workspace policy",
		Long: `Show the effective access policy for a workspace, including
the mode and device lists.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyShow(workspaceID, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runPolicyShow(workspaceID string, jsonOut bool) error {
	policy, err := multidevice.LoadPolicy(workspaceID)
	if err != nil {
		return err
	}

	mgr := multidevice.NewManager()
	links, _ := mgr.ListDeviceLinks(workspaceID)

	if jsonOut {
		out := map[string]interface{}{
			"policy":         policy,
			"linked_devices": len(links),
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Policy: %s", workspaceID)))

	fmt.Printf("  Mode:            %s\n", policyModeBadge(string(policy.Mode)))
	fmt.Printf("  Linked Devices:  %d\n", len(links))

	switch policy.Mode {
	case multidevice.PolicyAllowAll:
		fmt.Println()
		fmt.Println(dimStyle.Render(
			"  All linked devices have full access based on their roles."))

	case multidevice.PolicyAllowList:
		fmt.Println()
		if len(policy.AllowDevices) > 0 {
			fmt.Println(boldStyle.Render("  Allowed Devices:"))
			for _, dev := range policy.AllowDevices {
				accessDot := styles.PresenceDot("online")
				fmt.Printf("    %s %s\n", accessDot, dev)
			}
		} else {
			fmt.Println(warnStyle.Render(
				"  No devices in allow list. All access is blocked."))
		}

		// Show devices that are linked but not in allow list.
		if len(links) > 0 {
			blocked := 0
			for _, link := range links {
				if !contains(policy.AllowDevices, link.DeviceID) {
					blocked++
				}
			}
			if blocked > 0 {
				fmt.Printf("\n  %s %d linked devices are blocked by this policy.\n",
					warnStyle.Render("Note:"), blocked)
			}
		}

	case multidevice.PolicyDenyList:
		fmt.Println()
		if len(policy.DenyDevices) > 0 {
			fmt.Println(boldStyle.Render("  Denied Devices:"))
			for _, dev := range policy.DenyDevices {
				accessDot := styles.PresenceDot("offline")
				fmt.Printf("    %s %s\n", accessDot, dev)
			}
		} else {
			fmt.Println(dimStyle.Render(
				"  No devices in deny list. All linked devices have access."))
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("  Change policy mode:"))
	fmt.Printf(dimStyle.Render(
		"    aps policy set --workspace %s --mode <mode>")+"\n", workspaceID)

	return nil
}
