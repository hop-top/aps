package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/styles"
)

// actionSummaryRow is the table row shape for `aps action list`.
//
// Type surfaces the resolved runtime (sh|py|js|unknown) derived from
// the entrypoint extension; --type filters on it.
type actionSummaryRow struct {
	ID    string `table:"ID,priority=10"    json:"id"    yaml:"id"`
	Title string `table:"TITLE,priority=8"  json:"title" yaml:"title"`
	Type  string `table:"TYPE,priority=6"   json:"type"  yaml:"type"`
}

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

		typeFilter, _ := cmd.Flags().GetString("type")
		format := root.Viper.GetString("format")

		rows := make([]actionSummaryRow, len(actions))
		for i, a := range actions {
			rows[i] = actionSummaryRow{ID: a.ID, Title: a.Title, Type: a.Type}
		}

		pred := listing.All(
			listing.MatchString(func(r actionSummaryRow) string { return r.Type }, typeFilter),
		)
		rows = listing.Filter(rows, pred)

		// Non-table formats: emit raw rows and skip the header / dim
		// placeholders so JSON/YAML consumers see clean data.
		if format != "" && format != output.Table {
			return listing.RenderList(os.Stdout, format, rows)
		}

		if len(rows) == 0 {
			fmt.Println(styles.Dim.Render("No actions found."))
			return nil
		}

		fmt.Printf("%s\n\n", styles.Title.Render(
			fmt.Sprintf("Actions (%s)", profileID)))
		// Fill blank titles with the dimmed placeholder before render.
		for i := range rows {
			if rows[i].Title == "" {
				rows[i].Title = styles.Dim.Render("(no description)")
			}
		}
		if err := listing.RenderList(os.Stdout, output.Table, rows); err != nil {
			return err
		}
		fmt.Printf("\n%s\n", styles.Dim.Render(
			fmt.Sprintf("%d actions", len(rows))))
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

	actionListCmd.Flags().String("type", "",
		"Filter by runtime type (sh, py, js)")

	actionRunCmd.Flags().String("payload-file", "", "File to send to action stdin")
	actionRunCmd.Flags().Bool("payload-stdin", false, "Read stdin and forward to action") // effectively default if interactive, but explicit flag requested
	actionRunCmd.Flags().BoolP("dry-run", "n", false, "Print action details without running")
}
