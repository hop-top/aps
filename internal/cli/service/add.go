package service

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

func newAddCmd() *cobra.Command {
	opts := addOptions{}

	cmd := &cobra.Command{
		Use:   "add <service-id> --type <type-or-adapter-alias> --profile <profile-id>",
		Short: "Add a profile-facing service",
		Long: `Add a profile-facing service.

The --type flag accepts canonical service types and adapter aliases. Adapter
aliases are resolved through kit aliasing before APS persists the service.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.typeInput, "type", "", "Service type or adapter alias")
	cmd.Flags().StringVar(&opts.adapter, "adapter", "", "Concrete adapter when --type is canonical")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "Profile that owns the service")
	cmd.Flags().StringArrayVar(&opts.env, "env", nil, "Environment binding KEY=VALUE, repeatable")
	cmd.Flags().StringArrayVar(&opts.labels, "label", nil, "Metadata label KEY=VALUE, repeatable")
	cmd.Flags().StringVar(&opts.description, "description", "", "Human-readable description")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate without writing")

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
		typeInput, _ := cmd.Flags().GetString("type")
		adapterInput, _ := cmd.Flags().GetString("adapter")
		if strings.TrimSpace(typeInput) == "" {
			return
		}
		resolved, err := core.ResolveServiceType(typeInput, adapterInput)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nResolution error: %v\n", err)
			return
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(),
			"\nResolved:\n  input_type: %s\n  type: %s\n  adapter: %s\n",
			resolved.InputType, resolved.Type, resolved.Adapter)
	})

	return cmd
}

type addOptions struct {
	typeInput   string
	adapter     string
	profile     string
	env         []string
	labels      []string
	description string
	dryRun      bool
}

func runAdd(cmd *cobra.Command, id string, opts addOptions) error {
	resolved, err := core.ResolveServiceType(opts.typeInput, opts.adapter)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.profile) == "" {
		return fmt.Errorf("--profile is required")
	}

	env, err := parseKeyValues(opts.env, "--env")
	if err != nil {
		return err
	}
	labels, err := parseKeyValues(opts.labels, "--label")
	if err != nil {
		return err
	}

	service := &core.ServiceConfig{
		ID:          id,
		Type:        resolved.Type,
		Adapter:     resolved.Adapter,
		Profile:     opts.profile,
		Description: opts.description,
		Env:         env,
		Labels:      labels,
	}

	printResolved(cmd, resolved)
	if opts.dryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "dry_run: true")
		return nil
	}

	if err := core.SaveService(service); err != nil {
		return err
	}
	path, err := core.GetServicePath(id)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "saved: %s\n", path)
	return nil
}

func printResolved(cmd *cobra.Command, resolved core.ResolvedServiceType) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"input_type: %s\ntype: %s\nadapter: %s\n",
		resolved.InputType, resolved.Type, resolved.Adapter)
	if resolved.Aliased {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "resolved_by: kit alias")
	}
}

func parseKeyValues(values []string, flagName string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%s must be KEY=VALUE, got %q", flagName, value)
		}
		out[strings.TrimSpace(key)] = val
	}
	return out, nil
}
