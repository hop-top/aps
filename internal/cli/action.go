package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/styles"
)

var actionTableHeader = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDim)

var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "Manage and execute profile actions",
}

var actionListCmd = &cobra.Command{
	Use:   "list [profile]",
	Short: "List available actions for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID := args[0]
		actions, err := core.LoadActions(profileID)
		if err != nil {
			return fmt.Errorf("loading actions: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(actions)
		}

		if len(actions) == 0 {
			fmt.Println(styles.Dim.Render("No actions found."))
			return nil
		}

		fmt.Printf("%s\n\n", styles.Title.Render(
			fmt.Sprintf("Actions (%s)", profileID)))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, actionTableHeader.Render("ID")+"\t"+
			actionTableHeader.Render("TITLE"))
		for _, a := range actions {
			title := a.Title
			if title == "" {
				title = styles.Dim.Render("(no description)")
			}
			fmt.Fprintf(w, "%s\t%s\n", a.ID, title)
		}
		w.Flush()

		fmt.Printf("\n%s\n", styles.Dim.Render(
			fmt.Sprintf("%d actions", len(actions))))
		return nil
	},
}

var actionShowCmd = &cobra.Command{
	Use:   "show [profile] [action]",
	Short: "Show details of a specific action",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID := args[0]
		actionID := args[1]

		action, err := core.GetAction(profileID, actionID)
		if err != nil {
			return fmt.Errorf("getting action: %w", err)
		}

		fmt.Printf("ID: %s\n", action.ID)
		fmt.Printf("Title: %s\n", action.Title)
		fmt.Printf("Type: %s\n", action.Type)
		fmt.Printf("Path: %s\n", action.Path)
		fmt.Printf("Accepts Stdin: %v\n", action.AcceptsStdin)
		return nil
	},
}

var actionRunCmd = &cobra.Command{
	Use:   "run [profile] [action]",
	Short: "Run an action",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID := args[0]
		actionID := args[1]

		payloadFile, _ := cmd.Flags().GetString("payload-file")
		payloadStdin, _ := cmd.Flags().GetBool("payload-stdin")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load action to check details
		action, err := core.GetAction(profileID, actionID)
		if err != nil {
			return fmt.Errorf("getting action: %w", err)
		}

		if dryRun {
			fmt.Printf("Would run action '%s' from profile '%s'\n", actionID, profileID)
			fmt.Printf("Path: %s\n", action.Path)
			return nil
		}

		var payload []byte
		if payloadFile != "" {
			data, err := os.ReadFile(payloadFile)
			if err != nil {
				return fmt.Errorf("reading payload file: %w", err)
			}
			payload = data
		} else if payloadStdin {
			// Read from stdin?
			// But wait, Cobra might have consumed stdin? No.
			// Actually, `exec.Command` inherits stdin by default if we set it.
			// core.RunAction handles stdin if payload is empty.
			// But if user explicitly requests --payload-stdin, maybe we should read it all first?
			// Or just let it stream. core.RunAction logic: if payload slice is empty -> sets cmd.Stdin = os.Stdin.
			// So default behavior covers streaming stdin.
			// The flag --payload-stdin might imply we read it into a buffer first?
			// Spec 12.9 says: "--payload-stdin (read stdin and forward to action)".
			// If we just rely on default inheritance, it streams.
			// If we read it all, we buffer.
			// Let's stick to default streaming for simplicity and efficiency unless buffering is required.
			// However, if we mix flags and args...
			// Let's assume default is fine.
		}

		if err := core.RunAction(profileID, actionID, payload); err != nil {
			return fmt.Errorf("running action: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
	actionCmd.AddCommand(actionListCmd)
	actionCmd.AddCommand(actionShowCmd)
	actionCmd.AddCommand(actionRunCmd)

	actionListCmd.Flags().Bool("json", false, "Output as JSON")
	actionRunCmd.Flags().String("payload-file", "", "File to send to action stdin")
	actionRunCmd.Flags().Bool("payload-stdin", false, "Read stdin and forward to action") // effectively default if interactive, but explicit flag requested
	actionRunCmd.Flags().Bool("dry-run", false, "Print action details without running")
}
