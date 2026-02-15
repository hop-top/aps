package device

import (
	"encoding/json"
	"fmt"

	"oss-aps-cli/internal/core/device"

	"github.com/spf13/cobra"
)

func newUnlinkCmd() *cobra.Command {
	var profileID string
	var jsonOutput bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "unlink <device>",
		Aliases: []string{"detach"},
		Short:   "Unlink a device from a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlink(args[0], profileID, jsonOutput, dryRun)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to unlink (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be unlinked without unlinking")

	return cmd
}

func runUnlink(deviceName, profileID string, jsonOut, dryRun bool) error {
	dev, err := device.LoadDevice(deviceName)
	if err != nil {
		return err
	}

	if dryRun {
		return renderUnlinkDryRun(dev, profileID)
	}

	if !dev.IsLinkedToProfile(profileID) {
		return fmt.Errorf("device '%s' is not linked to profile '%s'", deviceName, profileID)
	}

	err = defaultManager.UnlinkDevice(deviceName, profileID)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderUnlinkJSON(dev, profileID)
	}

	fmt.Printf("Unlinked %s from %s\n", deviceName, profileID)
	return nil
}

func renderUnlinkDryRun(dev *device.Device, profileID string) error {
	fmt.Printf("Dry run: unlinking %s from %s\n\n", dev.Name, profileID)
	fmt.Printf("  Device:     %s (%s)\n", dev.Name, dev.Type)
	fmt.Printf("  Profile:    %s\n", profileID)
	if dev.IsLinkedToProfile(profileID) {
		fmt.Printf("  Status:     will be unlinked\n")
	} else {
		fmt.Printf("  Status:     not linked\n")
	}
	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to unlink.")
	return nil
}

func renderUnlinkJSON(dev *device.Device, profileID string) error {
	data := map[string]interface{}{
		"device":  dev.Name,
		"profile": profileID,
		"linked":  false,
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
