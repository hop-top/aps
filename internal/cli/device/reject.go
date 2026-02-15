package device

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newRejectCmd() *cobra.Command {
	var (
		profileID  string
		jsonOutput bool
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:   "reject <device-id>",
		Short: "Reject a pending mobile device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReject(args[0], profileID, jsonOutput, quiet)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Exit code only")

	return cmd
}

func runReject(deviceID, profileID string, jsonOut, quiet bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if err := registry.RejectDevice(deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id": deviceID,
			"status":    "rejected",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Device '%s' rejected.\n", successStyle.Render("✓"), deviceID)
	}
	return nil
}
