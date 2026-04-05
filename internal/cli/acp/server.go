package acp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/acp"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/protocol"
)

// NewServerCmd creates the `aps acp server` command
func NewServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server [profile]",
		Short: "Start an ACP server for a profile",
		Long: `Start an ACP (Agent Client Protocol) server for a profile.

The server communicates with editor clients via JSON-RPC 2.0 over stdio
or other transports (HTTP, WebSocket).

Example:
  aps acp server my-profile`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID := args[0]
			return runACPServer(profileID)
		},
	}
}

// runACPServer starts an ACP server for the specified profile
func runACPServer(profileID string) error {
	// Load profile (validate it exists)
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// Auto-enable ACP if not already configured
	if profile.ACP == nil || !profile.ACP.Enabled {
		fmt.Fprintf(os.Stderr, "ACP not enabled for profile %s, auto-enabling...\n", profileID)
		if err := enableACP(profile, "stdio", "127.0.0.1", "8088"); err != nil {
			return fmt.Errorf("failed to auto-enable ACP: %w", err)
		}
		// Reload profile with new configuration
		profile, err = core.LoadProfile(profileID)
		if err != nil {
			return fmt.Errorf("failed to reload profile: %w", err)
		}
		fmt.Fprintf(os.Stderr, "ACP enabled with defaults: stdio transport\n\n")
	}

	// Get the protocol core adapter
	coreAdapter, err := protocol.NewAPSAdapter()
	if err != nil {
		return fmt.Errorf("failed to create core adapter: %w", err)
	}

	// Create ACP server
	acpServer, err := acp.NewServer(profileID, coreAdapter)
	if err != nil {
		return fmt.Errorf("failed to create ACP server: %w", err)
	}

	// Create context for server lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived signal: %v\n", sig)
		acpServer.Stop()
		cancel()
	}()

	// Start server
	if err := acpServer.Start(ctx, nil); err != nil {
		return fmt.Errorf("failed to start ACP server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "ACP server started for profile: %s\n", profileID)
	fmt.Fprintf(os.Stderr, "Protocol version: 1\nReady to accept connections...\n")

	// Wait for context to be cancelled
	<-ctx.Done()

	return nil
}
