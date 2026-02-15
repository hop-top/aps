package device

import (
	"encoding/json"
	"fmt"

	"oss-aps-cli/internal/core/device"

	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	var profileID string
	var jsonOutput bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "link <device>",
		Short: "Link a device to a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLink(args[0], profileID, jsonOutput, dryRun)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to link (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be linked without linking")

	return cmd
}

func runLink(deviceName, profileID string, jsonOut, dryRun bool) error {
	dev, err := device.LoadDevice(deviceName)
	if err != nil {
		return err
	}

	if dryRun {
		return renderLinkDryRun(dev, profileID)
	}

	if dev.IsLinkedToProfile(profileID) {
		if jsonOut {
			return renderLinkJSON(dev, profileID)
		}
		fmt.Printf("%s is already linked to %s\n", deviceName, profileID)
		return nil
	}

	err = defaultManager.LinkDevice(deviceName, profileID)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderLinkJSON(dev, profileID)
	}

	fmt.Printf("Linked %s to %s\n", deviceName, profileID)
	return nil
}

func renderLinkDryRun(dev *device.Device, profileID string) error {
	fmt.Printf("Dry run: linking %s to %s\n\n", dev.Name, profileID)
	fmt.Printf("  Device:     %s (%s)\n", dev.Name, dev.Type)
	fmt.Printf("  Profile:    %s\n", profileID)
	if dev.IsLinkedToProfile(profileID) {
		fmt.Printf("  Status:     already linked\n")
	} else {
		fmt.Printf("  Status:     will be linked\n")
	}
	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to link.")
	return nil
}

func renderLinkJSON(dev *device.Device, profileID string) error {
	data := map[string]interface{}{
		"device":   dev.Name,
		"profile":  profileID,
		"linked":   true,
		"strategy": dev.Strategy,
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
