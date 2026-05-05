package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/kit/go/console/progress"
)

func NewFetchCardCmd() *cobra.Command {
	var (
		url    string
		format string
	)

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch an Agent Card from a URL",
		Long: `Fetch an A2A Agent Card from a remote URL (typically /.well-known/agent-card).

Example:
  aps a2a card fetch --url http://localhost:8081/.well-known/agent-card`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — fetching a remote agent card requires network.
			if globals.IsOffline() {
				return fmt.Errorf("a2a card fetch: %w", globals.ErrOffline)
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// T-0463 — structured progress per §6.5.
			r := progress.FromContext(ctx)
			r.Emit(ctx, progress.Event{Phase: "fetch", Item: url})

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: "fetch", Item: url, OK: &okFalse})
				return fmt.Errorf("failed to fetch agent card: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				okFalse := false
				r.Emit(ctx, progress.Event{Phase: "fetch", Item: url, OK: &okFalse})
				return fmt.Errorf("failed to fetch agent card: status %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			r.Emit(ctx, progress.Event{Phase: "parse", Item: url, Bytes: int64(len(body))})

			var card a2a.AgentCard
			if err := json.Unmarshal(body, &card); err != nil {
				return fmt.Errorf("failed to parse agent card: %w", err)
			}
			okTrue := true
			r.Emit(ctx, progress.Event{Phase: "parse", Item: url, OK: &okTrue})

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(&card)
			default:
				fmt.Printf("Agent Card fetched from: %s\n", url)
				fmt.Printf("URL: %s\n", card.URL)
				fmt.Printf("Transport: %s\n", card.PreferredTransport)
				if card.Description != "" {
					fmt.Printf("Description: %s\n", card.Description)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "", "Agent Card URL (required)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	cmd.MarkFlagRequired("url")

	return cmd
}
