package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
)

// NewDiscoverCmd creates the directory discover command.
func NewDiscoverCmd() *cobra.Command {
	var (
		capability string
		endpoint   string
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover agents by capability in the AGNTCY Directory",
		Long: `Search the AGNTCY Directory for agents matching a capability query.

Example:
  aps directory discover --capability "invoice-processing"
  aps dir discover --capability a2a --endpoint https://dir.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate AGNTCY Directory lookups on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory discover: %w", globals.ErrOffline)
			}

			cfg := &core.DirectoryConfig{
				Endpoint: endpoint,
			}

			client, err := discovery.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()

			results, err := client.Discover(cmd.Context(), capability)
			if err != nil {
				return fmt.Errorf("failed to discover agents: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No agents found matching the query.")
				return nil
			}

			fmt.Printf("Found %d agent(s):\n\n", len(results))
			for _, r := range results {
				fmt.Printf("  Name: %s\n", r.Name)
				if r.DID != "" {
					fmt.Printf("  DID:  %s\n", r.DID)
				}
				fmt.Printf("  URL:  %s\n", r.Endpoint)
				if len(r.Capabilities) > 0 {
					fmt.Printf("  Caps: %v\n", r.Capabilities)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&capability, "capability", "", "Capability to search for (required)")
	cmd.MarkFlagRequired("capability")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Directory endpoint URL (default: https://dir.agntcy.org)")

	return cmd
}
