package adapter

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/core/multidevice"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

func newSetPermissionsCmd() *cobra.Command {
	var (
		workspaceID string
		role        string
		canWrite    bool
		canExecute  bool
		canManage   bool
		rateLimit   int
		show        bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "set <device-id>",
		Short: "Set device permissions in a workspace",
		Long: `Set or view permissions for a device in a workspace.

Use --role for quick role-based configuration (covers 90% of cases).
Use fine-grained flags to override individual permissions.

Roles:
  owner        read, write, execute, manage, sync
  collaborator read, write, execute, sync
  viewer       read, sync

Use --show to display current permissions without making changes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetPermissions(
				args[0], workspaceID, role,
				canWrite, canExecute, canManage, rateLimit,
				show, jsonOutput, cmd,
			)
		},
		ValidArgsFunction: completeDeviceNames,
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"Workspace ID (required)")
	cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&role, "role", "",
		"Set role: owner, collaborator, viewer")
	cmd.Flags().BoolVar(&canWrite, "can-write", false,
		"Override: allow write access")
	cmd.Flags().BoolVar(&canExecute, "can-execute", false,
		"Override: allow execute access")
	cmd.Flags().BoolVar(&canManage, "can-manage", false,
		"Override: allow manage access")
	cmd.Flags().IntVar(&rateLimit, "rate-limit", 0,
		"Rate limit (requests per minute, 0 = unlimited)")
	cmd.Flags().BoolVar(&show, "show", false,
		"Show current permissions without changes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runSetPermissions(
	deviceID, workspaceID, role string,
	canWrite, canExecute, canManage bool,
	rateLimit int,
	show, jsonOut bool,
	cmd *cobra.Command,
) error {
	mgr := multidevice.NewManager()

	link, err := mgr.GetDeviceLink(workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("device '%s' is not attached to workspace '%s'",
			deviceID, workspaceID)
	}

	if show {
		return renderPermissions(link, jsonOut)
	}

	// Apply role if specified.
	if role != "" {
		deviceRole := multidevice.DeviceRole(role)
		if !multidevice.IsValidRole(deviceRole) {
			return fmt.Errorf(
				"invalid role '%s': must be owner, collaborator, or viewer",
				role)
		}
		if err := mgr.SetRole(workspaceID, deviceID, deviceRole); err != nil {
			return err
		}

		// Reload to get updated permissions.
		link, err = mgr.GetDeviceLink(workspaceID, deviceID)
		if err != nil {
			return err
		}
	}

	// Apply fine-grained overrides.
	perms := link.Permissions
	changed := false

	if cmd.Flags().Changed("can-write") {
		perms.CanWrite = canWrite
		changed = true
	}
	if cmd.Flags().Changed("can-execute") {
		perms.CanExecute = canExecute
		changed = true
	}
	if cmd.Flags().Changed("can-manage") {
		perms.CanManage = canManage
		changed = true
	}
	if cmd.Flags().Changed("rate-limit") {
		perms.RateLimitPerMin = rateLimit
		changed = true
	}

	if changed {
		if err := mgr.UpdatePermissions(workspaceID, deviceID, perms); err != nil {
			return err
		}
		// Reload.
		link, err = mgr.GetDeviceLink(workspaceID, deviceID)
		if err != nil {
			return err
		}
	}

	if role == "" && !changed {
		return fmt.Errorf("specify --role or permission flags to change. " +
			"Use --show to view current permissions")
	}

	if jsonOut {
		return renderPermissions(link, true)
	}

	fmt.Printf("%s Updated permissions for %s in %s\n",
		successStyle.Render("OK"),
		boldStyle.Render(deviceID),
		boldStyle.Render(workspaceID))
	fmt.Println()
	return renderPermissionsTable(link)
}

func renderPermissions(
	link *multidevice.WorkspaceDeviceLink, jsonOut bool,
) error {
	if jsonOut {
		data, err := json.MarshalIndent(link.Permissions, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%s\n\n",
		headerStyle.Render(fmt.Sprintf("Permissions: %s in %s",
			link.DeviceID, link.WorkspaceID)))
	return renderPermissionsTable(link)
}

func renderPermissionsTable(link *multidevice.WorkspaceDeviceLink) error {
	p := link.Permissions
	fmt.Printf("  Role:        %s\n", styles.RoleBadge(string(p.Role)))
	fmt.Printf("  Read:        %s\n", permBool(p.CanRead))
	fmt.Printf("  Write:       %s\n", permBool(p.CanWrite))
	fmt.Printf("  Execute:     %s\n", permBool(p.CanExecute))
	fmt.Printf("  Manage:      %s\n", permBool(p.CanManage))
	fmt.Printf("  Sync:        %s\n", permBool(p.CanSync))

	if p.RateLimitPerMin > 0 {
		fmt.Printf("  Rate Limit:  %d/min\n", p.RateLimitPerMin)
	} else {
		fmt.Printf("  Rate Limit:  %s\n", dimStyle.Render("unlimited"))
	}

	if len(p.AllowedActions) > 0 {
		fmt.Printf("  Allowed:     %s\n", join(p.AllowedActions))
	}
	if len(p.DeniedActions) > 0 {
		fmt.Printf("  Denied:      %s\n", join(p.DeniedActions))
	}

	return nil
}

func permBool(v bool) string {
	if v {
		return successStyle.Render("yes")
	}
	return dimStyle.Render("no")
}
