package session

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/core/session"
)

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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Property\tValue")
	fmt.Fprintln(w, "--------\t-----")

	fmt.Fprintf(w, "ID:\t%s\n", sess.ID)
	fmt.Fprintf(w, "Profile ID:\t%s\n", sess.ProfileID)
	fmt.Fprintf(w, "Profile Dir:\t%s\n", sess.ProfileDir)
	fmt.Fprintf(w, "Command:\t%s\n", sess.Command)
	fmt.Fprintf(w, "PID:\t%d\n", sess.PID)
	fmt.Fprintf(w, "Status:\t%s\n", sess.Status)
	fmt.Fprintf(w, "Tier:\t%s\n", sess.Tier)
	fmt.Fprintf(w, "Created At:\t%s\n", sess.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Last Seen At:\t%s\n", sess.LastSeenAt.Format("2006-01-02 15:04:05"))

	if sess.TmuxSocket != "" {
		fmt.Fprintf(w, "Tmux Socket:\t%s\n", sess.TmuxSocket)
	}

	if len(sess.Environment) > 0 {
		fmt.Fprintln(w, "\nEnvironment:")
		for k, v := range sess.Environment {
			fmt.Fprintf(w, "  %s:\t%s\n", k, v)
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
