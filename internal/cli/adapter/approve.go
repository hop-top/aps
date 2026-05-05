package adapter

import (
	"encoding/json"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"

	"github.com/spf13/cobra"
)

func newApproveCmd() *cobra.Command {
	var (
		profileID  string
		approveAll bool
		jsonOutput bool
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:   "approve <device-id>",
		Short: "Approve a pending mobile device",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID := ""
			if len(args) > 0 {
				deviceID = args[0]
			}
			if deviceID == "" && !approveAll {
				return fmt.Errorf("provide a device ID or use --all")
			}
			return runApprove(deviceID, profileID, approveAll, jsonOutput, quiet)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&approveAll, "all", false, "Approve all pending devices")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Exit code only")
	clinote.AddFlag(cmd) // T-1291

	return cmd
}

func runApprove(deviceID, profileID string, approveAll, jsonOut, quiet bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if approveAll {
		pending, err := registry.ListPending(profileID)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			if !quiet {
				fmt.Println(dimStyle.Render("  No pending devices."))
			}
			return nil
		}

		approved := 0
		for _, d := range pending {
			if err := registry.ApproveAdapter(d.AdapterID); err != nil {
				fmt.Fprintf(os.Stderr, "  Failed to approve %s: %v\n", d.AdapterID, err)
				continue
			}
			approved++
		}

		if jsonOut {
			out, _ := json.MarshalIndent(map[string]any{
				"approved": approved,
				"profile":  profileID,
			}, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		if !quiet {
			fmt.Printf("  %s Approved %d devices.\n", successStyle.Render("✓"), approved)
		}
		return nil
	}

	if err := registry.ApproveAdapter(deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id": deviceID,
			"status":    "approved",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Device '%s' approved.\n", successStyle.Render("✓"), deviceID)
	}
	return nil
}
