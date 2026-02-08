package a2a

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	a2apkg "oss-aps-cli/internal/a2a"
)

func NewServerCmd() *cobra.Command {
	var profileID string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start an A2A server for a profile",
		Long: `Start an A2A server to expose a profile as an A2A agent.

The server will listen on the address configured in the profile's A2A settings
(default: 127.0.0.1:8081) and serve:
  - A2A JSON-RPC endpoint at /
  - Agent Card at /.well-known/agent-card

Example:
  aps a2a server --profile worker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			profile, err := loadProfile(profileID)
			if err != nil {
				return err
			}

			if profile.A2A == nil || !profile.A2A.Enabled {
				return fmt.Errorf("A2A is not enabled for profile %s", profileID)
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

			if err := server.Start(ctx, config); err != nil {
				return fmt.Errorf("failed to start A2A server: %w", err)
			}

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

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile ID (required)")
	cmd.MarkFlagRequired("profile")

	return cmd
}
