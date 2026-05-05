package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/prompt"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/adapter/mobile"
)

func newRevokeCmd() *cobra.Command {
	var (
		profileID  string
		force      bool
		dryRun     bool
		revokeAll  bool
		jsonOutput bool
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:   "revoke [device-id]",
		Short: "Revoke a paired mobile device",
		Long: `Revoke a mobile device's access token, disconnecting it immediately.

The device must re-pair via a new QR code to reconnect.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID := ""
			if len(args) > 0 {
				deviceID = args[0]
			}
			if deviceID == "" && !revokeAll {
				return fmt.Errorf("provide a device ID or use --all")
			}
			return runRevoke(deviceID, profileID, force, dryRun, revokeAll, jsonOutput, quiet)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would happen")
	cmd.Flags().BoolVar(&revokeAll, "all", false, "Revoke all devices")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Exit code only")
	clinote.AddFlag(cmd) // T-1291 (long-form only; -n taken by --dry-run)

	return cmd
}

func runRevoke(deviceID, profileID string, force, dryRun, revokeAll, jsonOut, quiet bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if revokeAll {
		return revokeAllDevices(registry, profileID, force, dryRun, jsonOut, quiet)
	}

	device, err := registry.GetAdapter(deviceID)
	if err != nil {
		return err
	}

	if device.ProfileID != profileID {
		return fmt.Errorf("device '%s' is not linked to profile '%s'", deviceID, profileID)
	}

	if dryRun {
		fmt.Printf("Dry run: would revoke device '%s'\n\n", deviceID)
		fmt.Printf("  Device:  %s (%s)\n", device.AdapterName, device.AdapterOS)
		fmt.Printf("  Status:  %s\n", device.Status)
		fmt.Printf("  Profile: %s\n\n", profileID)
		fmt.Println("  No changes made. Remove --dry-run to revoke.")
		return nil
	}

	if !force {
		fmt.Printf("  Device: %s\n", device.AdapterName)
		fmt.Printf("  Status: %s\n\n", device.Status)

		confirmed, err := prompt.Confirm("Revoke this device?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	if err := registry.RevokeAdapter(deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id": deviceID,
			"status":    "revoked",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Device '%s' revoked.\n", successStyle.Render("✓"), deviceID)
	}
	return nil
}

func revokeAllDevices(registry *mobile.Registry, profileID string, force, dryRun, jsonOut, quiet bool) error {
	devices, err := registry.ListAdapters(profileID)
	if err != nil {
		return err
	}

	var active []*mobile.MobileAdapter
	for _, d := range devices {
		if d.Status == mobile.PairingStateActive || d.Status == mobile.PairingStatePending {
			active = append(active, d)
		}
	}

	if len(active) == 0 {
		fmt.Println(dimStyle.Render("  No active devices to revoke."))
		return nil
	}

	if dryRun {
		fmt.Printf("Dry run: would revoke %d devices for profile '%s'\n", len(active), profileID)
		for _, d := range active {
			fmt.Printf("  - %s (%s, %s)\n", d.AdapterID, d.AdapterName, d.Status)
		}
		fmt.Println("\n  No changes made.")
		return nil
	}

	if !force {
		fmt.Printf("  Revoke ALL %d devices for profile '%s'?\n\n", len(active), profileID)
		for _, d := range active {
			fmt.Printf("    %s  %s (%s)\n", dimStyle.Render(string(d.AdapterID)), d.AdapterName, d.Status)
		}

		confirmed, err := prompt.Confirm(
			fmt.Sprintf("Revoke all %d devices for profile '%s'?", len(active), profileID))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	revoked := 0
	for _, d := range active {
		if err := registry.RevokeAdapter(d.AdapterID); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to revoke %s: %v\n", d.AdapterID, err)
			continue
		}
		revoked++
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"revoked": revoked,
			"profile": profileID,
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Revoked %d devices.\n", successStyle.Render("✓"), revoked)
	}
	return nil
}

func getRegistry() (*mobile.Registry, error) {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return nil, err
	}
	return mobile.NewRegistry(filepath.Join(dataDir, "devices"))
}
