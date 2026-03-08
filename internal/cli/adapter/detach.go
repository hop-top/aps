package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"oss-aps-cli/internal/core/multidevice"

	"github.com/spf13/cobra"
)

func newDetachCmd() *cobra.Command {
	var (
		workspaceID string
		force       bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "detach <device-id>",
		Short: "Detach a device from a workspace",
		Long: `Detach a device from a workspace, removing all access.

This is a destructive operation. The device will lose access to the
workspace and any pending offline queue entries will be discarded.
Use --force to skip confirmation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetach(args[0], workspaceID, force, jsonOutput)
		},
		ValidArgsFunction: completeDeviceNames,
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace to detach the device from (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runDetach(deviceID, workspaceID string, force, jsonOut bool) error {
	mgr := multidevice.NewManager()

	// Check the link exists and show status before confirming.
	link, err := mgr.GetDeviceLink(workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("device '%s' is not attached to workspace '%s'",
			deviceID, workspaceID)
	}

	if !force {
		fmt.Printf("  Device:    %s\n", boldStyle.Render(deviceID))
		fmt.Printf("  Workspace: %s\n", boldStyle.Render(workspaceID))
		fmt.Printf("  Role:      %s\n", string(link.Permissions.Role))
		fmt.Printf("  Status:    %s\n", string(link.Status))
		fmt.Println()
		fmt.Println("  Detaching will:")
		fmt.Println("    - Remove device access to this workspace")
		fmt.Println("    - Discard any pending offline queue entries")
		fmt.Println("    - The device must be re-attached to regain access")
		fmt.Printf("\n  Detach this device? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	if err := mgr.DetachDevice(workspaceID, deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id":    deviceID,
			"workspace_id": workspaceID,
			"status":       "detached",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("%s Detached %s from %s\n",
		successStyle.Render("OK"),
		boldStyle.Render(deviceID),
		boldStyle.Render(workspaceID))

	return nil
}
