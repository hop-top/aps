package adapter

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	coreadapter "hop.top/aps/internal/core/adapter"
	"hop.top/aps/internal/events"

	"github.com/spf13/cobra"
)

// newLinkDeleteCmd creates the `aps adapter link delete` subcommand.
// T-0398 renamed from `unlink` (verb-with-un-) to `delete` (CRUD verb
// under the `link` noun parent). `rm`/`remove` kept as conventional
// shorthands.
func newLinkDeleteCmd() *cobra.Command {
	var profileID string
	var jsonOutput bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "delete <device>",
		Aliases: []string{"remove", "rm"},
		Short:   "Unlink a device from a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlink(args[0], profileID, jsonOutput, dryRun, clinote.FromCmd(cmd))
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to unlink (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be unlinked without unlinking")
	clinote.AddFlag(cmd) // T-1291 (long-form only; -n taken by --dry-run)

	return cmd
}

func runUnlink(deviceName, profileID string, jsonOut, dryRun bool, note string) error {
	dev, err := coreadapter.LoadAdapter(deviceName)
	if err != nil {
		return err
	}

	if dryRun {
		return renderUnlinkDryRun(dev, profileID)
	}

	if !dev.IsLinkedToProfile(profileID) {
		return fmt.Errorf("device '%s' is not linked to profile '%s'", deviceName, profileID)
	}

	err = defaultManager.UnlinkAdapter(deviceName, profileID)
	if err != nil {
		return err
	}

	publishEvent(string(events.TopicAdapterUnlinked), "", events.AdapterUnlinkedPayload{
		ProfileID:   profileID,
		AdapterType: string(dev.Type),
		AdapterID:   deviceName,
		Note:        note,
	})

	if jsonOut {
		return renderUnlinkJSON(dev, profileID)
	}

	fmt.Printf("Unlinked %s from %s\n", deviceName, profileID)
	return nil
}

func renderUnlinkDryRun(dev *coreadapter.Adapter, profileID string) error {
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

func renderUnlinkJSON(dev *coreadapter.Adapter, profileID string) error {
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
