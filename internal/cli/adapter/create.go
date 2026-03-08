package adapter

import (
	"encoding/json"
	"fmt"
	"os"

	coreadapter "oss-aps-cli/internal/core/adapter"

	"github.com/spf13/cobra"
)

var defaultManager = coreadapter.NewManager()

func newCreateCmd() *cobra.Command {
	var deviceType string
	var strategy string
	var profileID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"new"},
		Short:   "Create a new device",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], deviceType, strategy, profileID, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&deviceType, "type", "", "Device type (messenger, protocol, desktop, mobile)")
	cmd.MarkFlagRequired("type")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Loading strategy (subprocess, script, builtin)")
	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Create as profile-scoped device")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runCreate(name, deviceType, strategy, profileID string, jsonOut bool) error {
	dt := coreadapter.AdapterType(deviceType)

	if !coreadapter.IsAdapterTypeValid(dt) {
		return renderTypeError(deviceType)
	}

	if !coreadapter.IsAdapterTypeImplemented(dt) {
		return renderTypeNotImplementedError(dt)
	}

	scope := coreadapter.ScopeGlobal
	if profileID != "" {
		scope = coreadapter.ScopeProfile
	}

	ls := coreadapter.LoadingStrategy(strategy)
	if ls == "" {
		ls = coreadapter.DefaultStrategyForType(dt)
	}

	dev, err := defaultManager.CreateDevice(name, dt, ls, scope, profileID)
	if err != nil {
		return err
	}

	if jsonOut {
		return renderCreateJSON(dev)
	}

	return renderCreateSuccess(dev, profileID)
}

func renderTypeError(deviceType string) error {
	fmt.Fprintf(os.Stderr, "Error: device type '%s' is not valid\n\n", deviceType)
	fmt.Println("  Available types:")
	for _, meta := range coreadapter.AdapterTypes {
		fmt.Printf("    %-12s %s\n", meta.Type, dimStyle.Render(meta.Description))
	}
	fmt.Println()
	return fmt.Errorf("invalid device type")
}

func renderTypeNotImplementedError(dt coreadapter.AdapterType) error {
	fmt.Fprintf(os.Stderr, "Error: device type '%s' is not yet available\n\n", dt)
	fmt.Println("  Available types:")
	for _, meta := range coreadapter.AdapterTypes {
		if meta.Implemented {
			fmt.Printf("    %-12s %s\n", meta.Type, dimStyle.Render(meta.Description))
		}
	}
	fmt.Println()
	fmt.Println("  Planned types (coming soon):")
	for _, meta := range coreadapter.AdapterTypes {
		if !meta.Implemented {
			fmt.Printf("    %-12s %s\n", meta.Type, dimStyle.Render(meta.Description))
		}
	}
	fmt.Println()
	fmt.Println("  To request: https://github.com/aps/hops/issues")
	return fmt.Errorf("device type not implemented")
}

func renderCreateSuccess(dev *coreadapter.Adapter, profileID string) error {
	fmt.Printf("Creating device '%s' (type: %s)\n\n", dev.Name, dev.Type)
	fmt.Printf("  Directory: %s\n", dimStyle.Render(dev.Path))
	fmt.Printf("  Manifest:  %s\n\n", dimStyle.Render(dev.ManifestPath))
	fmt.Println("  Next steps:")
	fmt.Printf("    1. Edit manifest:  $EDITOR %s\n", dev.ManifestPath)
	if dev.Type == coreadapter.AdapterTypeMessenger {
		tokenEnv := fmt.Sprintf("%s_TOKEN", toEnvName(dev.Name))
		fmt.Printf("    2. Set secrets:    aps secrets set %s \"...\"\n", tokenEnv)
	}
	fmt.Printf("    3. Start device:   aps device start %s\n", dev.Name)
	fmt.Println()
	fmt.Printf("Created %s\n", dev.Name)
	return nil
}

func renderCreateJSON(dev *coreadapter.Adapter) error {
	data := map[string]interface{}{
		"name":       dev.Name,
		"type":       dev.Type,
		"scope":      dev.Scope,
		"strategy":   dev.Strategy,
		"path":       dev.Path,
		"created_at": dev.CreatedAt,
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func toEnvName(name string) string {
	result := make([]byte, 0, len(name))
	for i, c := range name {
		if c == '-' || c == '_' {
			result = append(result, '_')
		} else if c >= 'a' && c <= 'z' {
			result = append(result, byte(c-'a'+'A'))
		} else {
			result = append(result, byte(c))
		}
		if i > 0 && name[i-1] >= 'a' && name[i-1] <= 'z' && c >= 'A' && c <= 'Z' {
			result = append(result[:len(result)-1], '_')
			result = append(result, byte(c))
		}
	}
	return string(result)
}
