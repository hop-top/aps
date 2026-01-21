package cli

import (
	"fmt"
	"os"

	"oss-aps-cli/internal/core"

	"github.com/spf13/cobra"
)

var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "Manage and execute profile actions",
}

var actionListCmd = &cobra.Command{
	Use:   "list [profile]",
	Short: "List available actions for a profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profileID := args[0]
		actions, err := core.LoadActions(profileID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading actions: %v\n", err)
			os.Exit(1)
		}

		if len(actions) == 0 {
			fmt.Println("No actions found.")
			return
		}

		for _, a := range actions {
			title := a.Title
			if title == "" {
				title = "(no description)"
			}
			fmt.Printf("%s\t%s\n", a.ID, title)
		}
	},
}

var actionShowCmd = &cobra.Command{
	Use:   "show [profile] [action]",
	Short: "Show details of a specific action",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		profileID := args[0]
		actionID := args[1]

		action, err := core.GetAction(profileID, actionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("ID: %s\n", action.ID)
		fmt.Printf("Title: %s\n", action.Title)
		fmt.Printf("Type: %s\n", action.Type)
		fmt.Printf("Path: %s\n", action.Path)
		fmt.Printf("Accepts Stdin: %v\n", action.AcceptsStdin)
	},
}

var actionRunCmd = &cobra.Command{
	Use:   "run [profile] [action]",
	Short: "Run an action",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		profileID := args[0]
		actionID := args[1]

		payloadFile, _ := cmd.Flags().GetString("payload-file")
		payloadStdin, _ := cmd.Flags().GetBool("payload-stdin")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load action to check details
		action, err := core.GetAction(profileID, actionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Printf("Would run action '%s' from profile '%s'\n", actionID, profileID)
			fmt.Printf("Path: %s\n", action.Path)
			return
		}

		var payload []byte
		if payloadFile != "" {
			data, err := os.ReadFile(payloadFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading payload file: %v\n", err)
				os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Error running action: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
	actionCmd.AddCommand(actionListCmd)
	actionCmd.AddCommand(actionShowCmd)
	actionCmd.AddCommand(actionRunCmd)

	actionRunCmd.Flags().String("payload-file", "", "File to send to action stdin")
	actionRunCmd.Flags().Bool("payload-stdin", false, "Read stdin and forward to action") // effectively default if interactive, but explicit flag requested
	actionRunCmd.Flags().Bool("dry-run", false, "Print action details without running")
}
