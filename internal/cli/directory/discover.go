package directory

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/agntcy/discovery"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
)

// resolveInstance is overridable for tests; defaults to core.Resolve.
var resolveInstance = core.Resolve

// instanceFlagValue reads the --instance global from the root command's
// persistent flags. kit/cli's Globals create the flag on root and bind
// it to a private viper, so reading the flag value directly is the
// portable way to get it from a subpackage.
func instanceFlagValue(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	root := cmd.Root()
	if root == nil {
		return ""
	}
	if f := root.PersistentFlags().Lookup("instance"); f != nil {
		return f.Value.String()
	}
	return ""
}

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
  aps dir discover --capability a2a --endpoint https://dir.example.com
  aps --instance prod directory discover --capability a2a`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — gate AGNTCY Directory lookups on --offline.
			if globals.IsOffline() {
				return fmt.Errorf("directory discover: %w", globals.ErrOffline)
			}

			// T-0412 — resolve --instance to a backend bundle. An explicit
			// --endpoint on this command always wins; --instance only
			// supplies the default when --endpoint is empty.
			resolved := endpoint
			if resolved == "" {
				if name := instanceFlagValue(cmd); name != "" {
					inst, err := resolveInstance(name)
					if err != nil {
						return fmt.Errorf("resolve instance: %w", err)
					}
					resolved = inst.DirectoryEndpoint
				}
			}

			cfg := &core.DirectoryConfig{
				Endpoint: resolved,
			}

			ctx := cmd.Context()
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: resolved})

			client, err := discovery.NewClient(cfg)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: resolved, OK: &okFalse})
				return fmt.Errorf("failed to create directory client: %w", err)
			}
			defer client.Close()
			okConnect := true
			r.Emit(ctx, progress.Event{Phase: phaseConnect, Item: resolved, OK: &okConnect})

			r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: capability})
			results, err := client.Discover(ctx, capability)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: capability, OK: &okFalse})
				return fmt.Errorf("failed to discover agents: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: phaseFetch, Item: capability, OK: &okTrue})

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

	// T-0648 — kit/cli signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}
