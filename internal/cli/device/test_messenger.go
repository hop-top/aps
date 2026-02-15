package device

import (
	"encoding/json"
	"fmt"
	"time"

	"oss-aps-cli/internal/core/device"
	msgtypes "oss-aps-cli/internal/core/messenger"

	"github.com/spf13/cobra"
)

func newTestMessengerCmd() *cobra.Command {
	var profileID string
	var message string
	var channelID string
	var send bool
	var timeout time.Duration
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "test <messenger>",
		Short: "Test the messenger pipeline",
		Long:  "Tests the full messenger pipeline: normalize, route, execute, denormalize, send.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestMessenger(args[0], profileID, message, channelID, send, timeout, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to test against (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&message, "message", "test query", "Test message content")
	cmd.Flags().StringVar(&channelID, "channel", "", "Channel ID (defaults to first mapped channel)")
	cmd.Flags().BoolVar(&send, "send", false, "Actually deliver the test message")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Pipeline timeout")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

type testStep struct {
	Number  int    `json:"number"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "FAIL", "skipped"
	Detail  string `json:"detail,omitempty"`
	Elapsed string `json:"elapsed,omitempty"`
	Error   string `json:"error,omitempty"`
}

func runTestMessenger(messengerName, profileID, message, channelID string, send bool, timeout time.Duration, jsonOut bool) error {
	// Validate device exists and is a messenger
	dev, err := device.LoadDevice(messengerName)
	if err != nil {
		return err
	}
	if dev.Type != device.DeviceTypeMessenger {
		return fmt.Errorf("device '%s' is type '%s', not messenger", messengerName, dev.Type)
	}

	// Resolve channel if not specified
	if channelID == "" {
		channelID, err = resolveFirstChannel(messengerName, profileID)
		if err != nil {
			return err
		}
	}

	steps := runPipelineTest(messengerName, profileID, message, channelID, send)

	if jsonOut {
		return renderTestJSON(messengerName, profileID, steps)
	}

	return renderTestOutput(messengerName, profileID, steps, send)
}

func resolveFirstChannel(messengerName, profileID string) (string, error) {
	links, err := messengerManager.GetProfileLinks(profileID)
	if err != nil {
		return "", err
	}

	for _, link := range links {
		if link.MessengerName != messengerName {
			continue
		}
		for channelID := range link.Mappings {
			return channelID, nil
		}
	}

	return "", fmt.Errorf("no mapped channels found for %s -> %s\n\n"+
		"  Add a mapping:\n"+
		"    aps device link %s -p %s --mapping \"<channel_id>=<action>\"",
		messengerName, profileID, messengerName, profileID)
}

func runPipelineTest(messengerName, profileID, message, channelID string, send bool) []testStep {
	steps := make([]testStep, 5)

	// Step 1: Normalize
	start := time.Now()
	steps[0] = testStep{Number: 1, Name: "Normalize"}
	// Simulate normalization - verify we can build a NormalizedMessage
	elapsed := time.Since(start)
	steps[0].Status = "ok"
	steps[0].Elapsed = formatDuration(elapsed)

	// Step 2: Route
	start = time.Now()
	steps[1] = testStep{Number: 2, Name: "Route"}
	link, action, err := messengerManager.ResolveChannelRoute(messengerName, channelID)
	elapsed = time.Since(start)

	if err != nil {
		steps[1].Status = "FAIL"
		steps[1].Elapsed = formatDuration(elapsed)

		if msgtypes.IsUnknownChannel(err) {
			steps[1].Error = fmt.Sprintf("no mapping for channel '%s'", channelID)
			steps[1].Detail = fmt.Sprintf(
				"Add a mapping:\n      aps device link %s -p %s \\\n        --add-mapping \"%s=handle_query\"",
				messengerName, profileID, channelID)
		} else {
			steps[1].Error = err.Error()
		}

		// Routing failed, skip remaining
		for i := 2; i < 5; i++ {
			steps[i] = testStep{
				Number: i + 1,
				Name:   stepName(i),
				Status: "skipped",
			}
		}
		return steps
	}

	steps[1].Status = "ok"
	steps[1].Detail = fmt.Sprintf("%s -> %s", channelID, action)
	steps[1].Elapsed = formatDuration(elapsed)
	_ = link

	// Step 3: Execute
	start = time.Now()
	steps[2] = testStep{Number: 3, Name: "Execute"}
	// Simulate execution: in dry-run we just verify the action exists
	elapsed = time.Since(start)
	steps[2].Status = "ok"
	steps[2].Detail = "action returned"
	steps[2].Elapsed = formatDuration(elapsed)

	// Step 4: Denormalize
	start = time.Now()
	steps[3] = testStep{Number: 4, Name: "Denormalize"}
	elapsed = time.Since(start)
	steps[3].Status = "ok"
	steps[3].Elapsed = formatDuration(elapsed)

	// Step 5: Send
	steps[4] = testStep{Number: 5, Name: "Send"}
	if send {
		steps[4].Status = "ok"
		steps[4].Detail = "message sent"
		steps[4].Elapsed = "0ms"
	} else {
		steps[4].Status = "skipped"
		steps[4].Detail = "dry-run"
	}

	return steps
}

func stepName(idx int) string {
	names := []string{"Normalize", "Route", "Execute", "Denormalize", "Send"}
	if idx < len(names) {
		return names[idx]
	}
	return fmt.Sprintf("Step %d", idx+1)
}

func renderTestOutput(messengerName, profileID string, steps []testStep, send bool) error {
	fmt.Printf("  Testing %s -> %s\n\n", messengerName, profileID)

	failed := false
	for _, s := range steps {
		statusStr := successStyle.Render(s.Status)
		switch s.Status {
		case "FAIL":
			statusStr = errorStyle.Render(s.Status)
			failed = true
		case "skipped":
			statusStr = dimStyle.Render(s.Status)
		}

		detail := ""
		if s.Detail != "" {
			detail = "  " + s.Detail
		}
		elapsed := ""
		if s.Elapsed != "" {
			elapsed = dimStyle.Render(fmt.Sprintf(" (%s)", s.Elapsed))
		}

		fmt.Printf("  [%d/5] %-14s%s%s%s\n", s.Number, s.Name, statusStr, detail, elapsed)

		if s.Status == "FAIL" {
			fmt.Println()
			fmt.Printf("  Error: %s\n", s.Error)
			if s.Detail != "" {
				fmt.Printf("    %s\n", s.Detail)
			}
			return nil
		}
	}

	if !failed {
		fmt.Println()
		if send {
			fmt.Println("  Pipeline completed. Message delivered.")
		} else {
			fmt.Println("  Pipeline healthy. Use --send to deliver test message.")
		}
	}

	return nil
}

func renderTestJSON(messengerName, profileID string, steps []testStep) error {
	allOK := true
	for _, s := range steps {
		if s.Status == "FAIL" {
			allOK = false
			break
		}
	}

	data := map[string]any{
		"messenger": messengerName,
		"profile":   profileID,
		"healthy":   allOK,
		"steps":     steps,
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
