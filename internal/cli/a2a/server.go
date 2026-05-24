package a2a

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/agntcy/observability"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
)

const phaseListen = "listen"

func NewServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start an A2A server for a profile",
		Long: `Start an A2A server to expose a profile as an A2A agent.

The server will listen on the address configured in the profile's A2A settings
(default: 127.0.0.1:8081) and serve:
  - A2A JSON-RPC endpoint at /
  - Agent Card at /.well-known/agent-card

Example:
  aps --profile worker a2a server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Inherit from cmd.Context() so kit/cli's progress.Reporter
			// (and any other context-bound values) flow into the daemon.
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// T-0648 — read --profile from root globals; no local shadow.
			profileID := globals.Profile()
			if profileID == "" {
				return errors.New("--profile is required")
			}

			// Load profile (without requiring A2A capability yet)
			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("failed to load profile %s: %w", profileID, err)
			}

			// Auto-enable A2A if not already configured
			if !core.ProfileHasCapability(profile, "a2a") {
				fmt.Printf("A2A not enabled for profile %s, auto-enabling...\n", profileID)
				if err := enableA2A(profile, "jsonrpc", "127.0.0.1", "8081", ""); err != nil {
					return fmt.Errorf("failed to auto-enable A2A: %w", err)
				}
				// Reload profile with new configuration
				profile, err = core.LoadProfile(profileID)
				if err != nil {
					return fmt.Errorf("failed to reload profile: %w", err)
				}
				fmt.Printf("A2A enabled with defaults: jsonrpc/127.0.0.1:8081\n\n")
			}

			// Initialize observability if enabled
			if core.ProfileHasCapability(profile, "agntcy-observability") && profile.Observability != nil {
				if err := observability.InitTracer(profile.Observability, profileID); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to init tracer: %v\n", err)
				}
				if err := observability.InitMeter(profile.Observability, profileID); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to init meter: %v\n", err)
				}
				defer observability.Shutdown(ctx)
			}

			agentsDir, err := core.GetAgentsDir()
			if err != nil {
				return fmt.Errorf("failed to get agents directory: %w", err)
			}

			config := &a2apkg.StorageConfig{
				BasePath: filepath.Join(agentsDir, "a2a", profile.ID),
			}

			server, err := a2apkg.NewServer(profile, config)
			if err != nil {
				return fmt.Errorf("failed to create A2A server: %w", err)
			}

			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: phaseListen, Item: profileID})
			if err := server.Start(ctx, config); err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: phaseListen, Item: profileID, OK: &okFalse})
				return fmt.Errorf("failed to start A2A server: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: phaseListen, Item: profileID, OK: &okTrue})

			addr := profile.A2A.ListenAddr
			if addr == "" {
				addr = "127.0.0.1:8081"
			}

			fmt.Printf("A2A server started for profile: %s\n", profile.ID)
			fmt.Printf("Listening on: %s\n", addr)
			fmt.Printf("Agent Card: http://%s/.well-known/agent-card\n", addr)
			fmt.Println("\nPress Ctrl+C to stop the server")

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nShutting down server...")
			if err := server.Stop(); err != nil {
				return fmt.Errorf("failed to stop server: %w", err)
			}

			fmt.Println("Server stopped")
			return nil
		},
	}

	// T-0648 — kit 0.4 signature annotations. Long-running listener:
	// interactive class, not naturally idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectInteractive)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)

	return cmd
}
