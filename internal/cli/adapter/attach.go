package adapter

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/core/multidevice"

	"github.com/spf13/cobra"
)

func newAttachCmd() *cobra.Command {
	var (
		workspaceID string
		role        string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "attach <device-id>",
		Short: "Attach a device to a workspace",
		Long: `Attach a device to a workspace with a specified role.

Roles control what the device can do:
  owner        Full access (read, write, execute, manage, sync)
  collaborator Operational access (read, write, execute, sync)
  viewer       Read-only access (read, sync)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAttach(args[0], workspaceID, role, jsonOutput)
		},
		ValidArgsFunction: completeDeviceNames,
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace to attach the device to (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&role, "role", "viewer",
		"Device role: owner, collaborator, viewer")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runAttach(deviceID, workspaceID, role string, jsonOut bool) error {
	deviceRole := multidevice.DeviceRole(role)
	if !multidevice.IsValidRole(deviceRole) {
		return fmt.Errorf("invalid role '%s': must be owner, collaborator, or viewer",
			role)
	}

	mgr := multidevice.NewManager()
	link, err := mgr.AttachDevice(workspaceID, deviceID, deviceRole)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderAttachJSON(link)
	}

	fmt.Printf("%s Attached %s to %s (role: %s)\n",
		successStyle.Render("OK"),
		boldStyle.Render(deviceID),
		boldStyle.Render(workspaceID),
		role)
	fmt.Println()
	fmt.Println(dimStyle.Render("  Set permissions:"))
	fmt.Printf(dimStyle.Render("    aps device set-permissions %s --workspace %s --role %s")+"\n",
		deviceID, workspaceID, role)
	fmt.Println(dimStyle.Render("  View presence:"))
	fmt.Printf(dimStyle.Render("    aps device presence --workspace %s")+"\n",
		workspaceID)

	return nil
}

func renderAttachJSON(link *multidevice.WorkspaceDeviceLink) error {
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
