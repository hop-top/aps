package session

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/styles"
	"hop.top/kit/go/console/output"
)

// sessionPropertyRow is the "PROPERTY/VALUE" row shape rendered by
// `aps session inspect`. T-0456 — moved off hand-rolled tabwriter so
// styled tables activate on a TTY.
type sessionPropertyRow struct {
	Property string `table:"PROPERTY,priority=10" json:"property" yaml:"property"`
	Value    string `table:"VALUE,priority=9"     json:"value"    yaml:"value"`
}

// sessionEnvRow is the "ENVIRONMENT/VALUE" row shape used for the
// optional environment block in `aps session inspect`. Same shape
// as sessionPropertyRow but with header text that matches the
// existing CLI surface.
type sessionEnvRow struct {
	Key   string `table:"ENVIRONMENT,priority=10" json:"key"   yaml:"key"`
	Value string `table:"VALUE,priority=9"        json:"value" yaml:"value"`
}

func NewInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <session-id>",
		Short: "Inspect a session's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			registry := session.GetRegistry()
			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			if jsonOutput {
				return outputJSON(sess, pretty)
			}

			return outputTable(sess)
		},
	}

	cmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

func outputTable(sess *session.SessionInfo) error {
	fmt.Printf("%s\n\n", styles.Title.Render("Session: "+sess.ID))

	rows := []sessionPropertyRow{
		{Property: "ID", Value: sess.ID},
		{Property: "Profile ID", Value: sess.ProfileID},
		{Property: "Profile Dir", Value: sess.ProfileDir},
		{Property: "Command", Value: sess.Command},
		{Property: "PID", Value: fmt.Sprintf("%d", sess.PID)},
		{Property: "Status", Value: styles.SessionStatusBadge(string(sess.Status))},
		{Property: "Tier", Value: styles.TierBadge(string(sess.Tier))},
		{Property: "Created At", Value: sess.CreatedAt.Format("2006-01-02 15:04:05")},
		{Property: "Last Seen At", Value: sess.LastSeenAt.Format("2006-01-02 15:04:05")},
	}
	if sess.TmuxSocket != "" {
		rows = append(rows, sessionPropertyRow{Property: "Tmux Socket", Value: sess.TmuxSocket})
	}
	if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
		return err
	}

	if len(sess.Environment) > 0 {
		fmt.Println()
		envRows := make([]sessionEnvRow, 0, len(sess.Environment))
		for k, v := range sess.Environment {
			envRows = append(envRows, sessionEnvRow{Key: k, Value: v})
		}
		if err := listing.RenderList(os.Stdout, output.Table, envRows); err != nil {
			return err
		}
	}

	return nil
}

func outputJSON(sess *session.SessionInfo, pretty bool) error {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(sess, "", "  ")
	} else {
		data, err = json.Marshal(sess)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
