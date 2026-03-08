package adapter

import (
	"encoding/json"
	"fmt"
	"time"

	coreadapter "hop.top/aps/internal/core/adapter"
	"hop.top/aps/internal/styles"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:     "status <name>",
		Aliases: []string{"show"},
		Short:   "Show device status",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0], jsonOutput, verbose)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")

	return cmd
}

func runStatus(name string, jsonOut, verbose bool) error {
	dev, err := coreadapter.LoadAdapter(name)
	if err != nil {
		return err
	}

	runtime, _ := defaultManager.GetRuntime(name)

	if jsonOut {
		return renderStatusJSON(dev, runtime)
	}

	return renderStatusDetail(dev, runtime, verbose)
}

func renderStatusJSON(dev *coreadapter.Adapter, runtime *coreadapter.AdapterRuntime) error {
	data := map[string]interface{}{
		"name":       dev.Name,
		"type":       dev.Type,
		"scope":      dev.Scope,
		"state":      runtime.State,
		"health":     runtime.Health,
		"strategy":   dev.Strategy,
		"created_at": dev.CreatedAt,
	}
	if dev.ProfileID != "" {
		data["profile_id"] = dev.ProfileID
	}
	if runtime.PID > 0 {
		data["pid"] = runtime.PID
	}
	if runtime.StartedAt != nil {
		data["started_at"] = runtime.StartedAt
		data["uptime"] = time.Since(*runtime.StartedAt).String()
	}
	if runtime.LastError != "" {
		data["last_error"] = runtime.LastError
	}
	if len(dev.LinkedTo) > 0 {
		data["linked_profiles"] = dev.LinkedTo
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func renderStatusDetail(dev *coreadapter.Adapter, runtime *coreadapter.AdapterRuntime, verbose bool) error {
	fmt.Printf("%s\n\n", headerStyle.Render(fmt.Sprintf("Device: %s", dev.Name)))

	fmt.Printf("  Type:       %s\n", styles.DeviceTypeBadge(string(dev.Type)))
	fmt.Printf("  Scope:      %s", styles.ScopeBadge(string(dev.Scope)))
	if dev.Scope == coreadapter.ScopeGlobal {
		fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("(%s)", dev.Path)))
	} else if dev.ProfileID != "" {
		fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("(profile: %s)", dev.ProfileID)))
	} else {
		fmt.Println()
	}

	fmt.Printf("  State:      %s\n", styles.DeviceStateBadge(string(runtime.State)))

	if runtime.State == coreadapter.StateRunning {
		health := runtime.Health
		if health == "" {
			health = "unknown"
		}
		healthStr := styles.HealthBadge(string(health))
		if runtime.LastCheck != nil {
			ago := formatTimeAgo(*runtime.LastCheck)
			healthStr += dimStyle.Render(fmt.Sprintf(" (last check %s)", ago))
		}
		fmt.Printf("  Health:     %s\n", healthStr)

		if runtime.StartedAt != nil {
			uptime := formatUptime(time.Since(*runtime.StartedAt))
			fmt.Printf("  Uptime:     %s\n", uptime)
		}

		if runtime.PID > 0 {
			fmt.Printf("  PID:        %d\n", runtime.PID)
		}
	}

	fmt.Printf("  Strategy:   %s\n", styles.StrategyBadge(string(dev.Strategy)))

	if !dev.CreatedAt.IsZero() {
		fmt.Printf("  Created:    %s\n", dev.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	if len(dev.LinkedTo) > 0 {
		fmt.Println()
		fmt.Printf("  %s\n", boldStyle.Render("Profiles:"))
		for _, p := range dev.LinkedTo {
			fmt.Printf("    %s\n", p)
		}
	}

	if runtime.State == coreadapter.StateFailed {
		fmt.Println()
		fmt.Printf("  %s\n", errorStyle.Render("Error:"))
		if runtime.LastError != "" {
			fmt.Printf("    %s\n", runtime.LastError)
		}
		if runtime.Restarts > 0 {
			fmt.Printf("  Restarts:   %d/5", runtime.Restarts)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("  Hint: Check the device logs:")
		fmt.Printf("    aps device logs %s\n", dev.Name)
	}

	fmt.Println()
	fmt.Printf("  Logs: aps device logs %s\n", dev.Name)

	return nil
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}
