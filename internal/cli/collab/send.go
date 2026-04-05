package collab

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	collab "hop.top/aps/internal/core/collaboration"

	"github.com/spf13/cobra"
)

// NewSendCmd creates the "collab send" command.
func NewSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <recipient>",
		Short: "Send a task to an agent",
		Long: `Send a task to another agent in the workspace.

Input can be provided as:
  --input '{"key": "value"}'   JSON string
  --input @file.json           Read from file
  --input -                    Read from stdin
  --set key=value              Shorthand for simple key-value input`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipient := args[0]

			wsID, err := resolveWorkspace(cmd, nil)
			if err != nil {
				return err
			}

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			action, _ := cmd.Flags().GetString("action")
			if action == "" {
				return fmt.Errorf("--action is required")
			}

			input, err := buildInput(cmd)
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}

			timeoutStr, _ := cmd.Flags().GetString("timeout")
			timeout := 5 * time.Minute
			if timeoutStr != "" {
				d, err := time.ParseDuration(timeoutStr)
				if err != nil {
					return fmt.Errorf("invalid timeout %q: %w", timeoutStr, err)
				}
				timeout = d
			}

			store, err := getStorage()
			if err != nil {
				return err
			}

			audit := collab.NewWorkspaceAuditLog(store)
			router := collab.NewMessageRouter(store, audit)

			task := collab.TaskInfo{
				SenderID:    profile,
				RecipientID: recipient,
				Action:      action,
				Input:       input,
				Timeout:     timeout,
			}

			result, err := router.Send(cmd.Context(), wsID, task)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(result)
			}

			fmt.Printf("Task %s sent to %s\n", result.ID, recipient)
			fmt.Printf("  Action: %s\n", action)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	cmd.Flags().String("action", "", "Task action (required)")
	cmd.Flags().String("input", "", "Task input (JSON string, @file, or - for stdin)")
	cmd.Flags().StringSlice("set", nil, "Set key=value input pairs")
	cmd.Flags().String("timeout", "", "Task timeout (e.g. 5m, 1h)")
	addJSONFlag(cmd)

	return cmd
}

// buildInput constructs JSON input from --input and --set flags.
func buildInput(cmd *cobra.Command) (json.RawMessage, error) {
	inputStr, _ := cmd.Flags().GetString("input")
	sets, _ := cmd.Flags().GetStringSlice("set")

	var data json.RawMessage

	if inputStr != "" {
		raw, err := readInputSource(inputStr)
		if err != nil {
			return nil, err
		}
		if !json.Valid(raw) {
			return nil, fmt.Errorf("input is not valid JSON")
		}
		data = raw
	}

	if len(sets) > 0 {
		kv := make(map[string]string, len(sets))
		for _, s := range sets {
			parts := strings.SplitN(s, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid --set value %q (expected key=value)", s)
			}
			kv[parts[0]] = parts[1]
		}

		if data != nil {
			// Merge --set into existing input
			var existing map[string]any
			if err := json.Unmarshal(data, &existing); err != nil {
				return nil, fmt.Errorf("cannot merge --set into non-object input")
			}
			for k, v := range kv {
				existing[k] = v
			}
			merged, err := json.Marshal(existing)
			if err != nil {
				return nil, err
			}
			data = merged
		} else {
			raw, err := json.Marshal(kv)
			if err != nil {
				return nil, err
			}
			data = raw
		}
	}

	return data, nil
}

// readInputSource reads input from a JSON string, @file reference, or stdin.
func readInputSource(input string) (json.RawMessage, error) {
	if input == "-" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		return json.RawMessage(raw), nil
	}

	if strings.HasPrefix(input, "@") {
		path := strings.TrimPrefix(input, "@")
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file %q: %w", path, err)
		}
		return json.RawMessage(raw), nil
	}

	return json.RawMessage(input), nil
}
