package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"
	coreadapter "hop.top/aps/internal/core/adapter"
)

var defaultManager = coreadapter.NewManager()

func newCreateCmd() *cobra.Command {
	var deviceType string
	var strategy string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"new"},
		Short:   "Create a new device",
		Long: `Create a new aps adapter device entry under the active
profile (or globally when no profile is set). The record lives in
$APS_DATA_PATH/profiles/<profile>/adapters/<name>/ (or the global
equivalent) and registers an external transport — messenger,
protocol, mobile, desktop, sense, or actuator — that aps can talk
to. The record holds the device's type, loading strategy
(subprocess / script / builtin), and a manifest scaffold; it does
NOT start the device — pair that with aps adapter start <name>.

If --type is omitted the command launches an interactive huh prompt
listing the implemented types; --strategy defaults to the type's
canonical strategy when omitted. --json emits a structured result.
--profile is inherited from the root global; no explicit flag is
needed when the active profile is already set.

Mints a new local record. Each invocation creates a fresh entry
(not idempotent); dry-run is opted out because the prospective ID
is exactly the <name> argument.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Interactive type selection when not provided via flag
			if deviceType == "" {
				var types []string
				for _, meta := range coreadapter.AdapterTypes {
					if meta.Implemented {
						types = append(types, string(meta.Type))
					}
				}
				sort.Strings(types)

				var opts []huh.Option[string]
				for _, t := range types {
					meta := coreadapter.AdapterTypes[coreadapter.AdapterType(t)]
					opts = append(opts,
						huh.NewOption(
							fmt.Sprintf("%s — %s", t, meta.Description),
							t))
				}
				if err := huh.NewSelect[string]().
					Title("Device type").
					Options(opts...).
					Value(&deviceType).
					Run(); err != nil {
					return err
				}
			}
			// T-0648 — read --profile from kit-managed global (root.Viper)
			// instead of redeclaring locally.
			return runCreate(args[0], deviceType, strategy, globals.Profile(), jsonOutput)
		},
	}

	cmd.Flags().StringVar(&deviceType, "type", "", "Device type (messenger, protocol, mobile, desktop, sense, actuator)")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Loading strategy (subprocess, script, builtin)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — create is atomic-by-design; the prospective ID is the
	// name argument the user already supplied.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "create is atomic-by-design; the prospective adapter ID is the name argument the user supplied, and preview would only restate it."); err != nil {
		panic(err)
	}
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
	if !coreadapter.IsLoadingStrategyValid(ls) {
		return fmt.Errorf("invalid loading strategy %q (valid: subprocess, script, builtin)", strategy)
	}

	dev, err := defaultManager.CreateAdapter(name, dt, ls, scope, profileID)
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
