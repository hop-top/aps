package policy

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/multidevice"

	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	var (
		workspaceID string
		mode        string
		addAllow    []string
		removeAllow []string
		addDeny     []string
		removeDeny  []string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set workspace access policy",
		Long: `Set the access control mode for a workspace.

Modes:
  allow-all   All linked devices have access (default)
  allow-list  Only specified devices have access
  deny-list   All devices except specified ones have access

Use --add-allow / --remove-allow to manage the allow list.
Use --add-deny / --remove-deny to manage the deny list.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicySet(
				workspaceID, mode,
				addAllow, removeAllow,
				addDeny, removeDeny,
				jsonOutput,
			)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&mode, "mode", "",
		"Policy mode: allow-all, allow-list, deny-list")
	cmd.Flags().StringSliceVar(&addAllow, "add-allow", nil,
		"Add devices to allow list")
	cmd.Flags().StringSliceVar(&removeAllow, "remove-allow", nil,
		"Remove devices from allow list")
	cmd.Flags().StringSliceVar(&addDeny, "add-deny", nil,
		"Add devices to deny list")
	cmd.Flags().StringSliceVar(&removeDeny, "remove-deny", nil,
		"Remove devices from deny list")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	return cmd
}

func runPolicySet(
	workspaceID, mode string,
	addAllow, removeAllow, addDeny, removeDeny []string,
	jsonOut bool,
) error {
	policy, err := multidevice.LoadPolicy(workspaceID)
	if err != nil {
		return err
	}

	if mode != "" {
		policyMode := multidevice.PolicyMode(mode)
		switch policyMode {
		case multidevice.PolicyAllowAll,
			multidevice.PolicyAllowList,
			multidevice.PolicyDenyList:
			policy.Mode = policyMode
		default:
			return fmt.Errorf(
				"invalid mode '%s': must be allow-all, allow-list, or deny-list",
				mode)
		}
	}

	// Update allow list.
	for _, dev := range addAllow {
		if !contains(policy.AllowDevices, dev) {
			policy.AllowDevices = append(policy.AllowDevices, dev)
		}
	}
	for _, dev := range removeAllow {
		policy.AllowDevices = removeItem(policy.AllowDevices, dev)
	}

	// Update deny list.
	for _, dev := range addDeny {
		if !contains(policy.DenyDevices, dev) {
			policy.DenyDevices = append(policy.DenyDevices, dev)
		}
	}
	for _, dev := range removeDeny {
		policy.DenyDevices = removeItem(policy.DenyDevices, dev)
	}

	// Warn about allow-list with no devices.
	if policy.Mode == multidevice.PolicyAllowList &&
		len(policy.AllowDevices) == 0 {
		fmt.Printf("%s allow-list mode with no allowed devices. "+
			"No device will have access.\n",
			warnStyle.Render("Warning:"))
		fmt.Println(dimStyle.Render("  Add devices with --add-allow <device-id>"))
		fmt.Println()
	}

	// Show affected devices.
	mgr := multidevice.NewManager()
	links, _ := mgr.ListDeviceLinks(workspaceID)
	if len(links) > 0 && mode != "" {
		fmt.Printf("  %d devices linked to this workspace.\n", len(links))
		fmt.Printf("  New policy mode: %s\n\n", policyModeBadge(mode))
	}

	if err := multidevice.SavePolicy(workspaceID, policy); err != nil {
		return err
	}

	if jsonOut {
		data, _ := json.MarshalIndent(policy, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s Policy updated for workspace '%s'\n",
		successStyle.Render("OK"), workspaceID)
	fmt.Printf("  Mode: %s\n", policyModeBadge(string(policy.Mode)))

	if len(policy.AllowDevices) > 0 {
		fmt.Printf("  Allow: %v\n", policy.AllowDevices)
	}
	if len(policy.DenyDevices) > 0 {
		fmt.Printf("  Deny:  %v\n", policy.DenyDevices)
	}

	return nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func removeItem(ss []string, s string) []string {
	var result []string
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
