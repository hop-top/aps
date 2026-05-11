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
	cmd.Flags().StringVar(&opts.site, "site", "", "Ticket adapter site or base URL")
	cmd.Flags().StringVar(&opts.project, "project", "", "Ticket adapter project key, ID, or path")
	cmd.Flags().StringVar(&opts.jql, "jql", "", "Jira issue query")
	cmd.Flags().StringVar(&opts.workspace, "workspace", "", "Linear workspace key or ID")
	cmd.Flags().StringVar(&opts.team, "team", "", "Linear team key or ID")
	cmd.Flags().StringVar(&opts.group, "group", "", "GitLab group path or ID")
	cmd.Flags().StringVar(&opts.events, "events", "", "Comma-separated ticket events to receive")
	cmd.Flags().StringVar(&opts.receive, "receive", "", "Message receive mode: polling or webhook")
	cmd.Flags().StringVar(&opts.provider, "provider", "", "Message provider, such as twilio or whatsapp-cloud")
	cmd.Flags().StringVar(&opts.from, "from", "", "Sender phone number or provider identity")
	cmd.Flags().StringVar(&opts.phoneNumberID, "phone-number-id", "", "WhatsApp phone number ID")
	cmd.Flags().StringArrayVar(&opts.allowedChannels, "allowed-channel", nil, "Allowed message channel ID, repeatable")
	cmd.Flags().StringArrayVar(&opts.allowedChats, "allowed-chat", nil, "Allowed Telegram chat ID, repeatable")
	cmd.Flags().StringArrayVar(&opts.allowedNumbers, "allowed-number", nil, "Allowed phone number, repeatable")
	cmd.Flags().StringVar(&opts.defaultAction, "default-action", "", "Default profile action for routed messages or tickets")
	cmd.Flags().StringVar(&opts.reply, "reply", "", "Reply behavior: text, comment, status, auto, or none")
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
	typeInput       string
	adapter         string
	profile         string
	env             []string
	labels          []string
	description     string
	site            string
	project         string
	jql             string
	workspace       string
	team            string
	group           string
	events          string
	receive         string
	provider        string
	from            string
	phoneNumberID   string
	allowedChannels []string
	allowedChats    []string
	allowedNumbers  []string
	defaultAction   string
	reply           string
	dryRun          bool
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
		Options:     serviceOptions(opts),
	}

	printResolved(cmd, resolved)
	validation := core.ValidateServiceConfig(service)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_valid: %t\n", validation.Valid)
	for _, issue := range validation.Issues {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_issue: %s\n", issue)
	}
	for _, warning := range validation.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_warning: %s\n", warning)
	}
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

func serviceOptions(opts addOptions) map[string]string {
	options := map[string]string{}
	addOption(options, "site", opts.site)
	addOption(options, "project", opts.project)
	addOption(options, "jql", opts.jql)
	addOption(options, "workspace", opts.workspace)
	addOption(options, "team", opts.team)
	addOption(options, "group", opts.group)
	addOption(options, "events", opts.events)
	addOption(options, "receive", opts.receive)
	addOption(options, "provider", opts.provider)
	addOption(options, "from", opts.from)
	addOption(options, "phone_number_id", opts.phoneNumberID)
	addOption(options, "allowed_channels", joinValues(opts.allowedChannels))
	addOption(options, "allowed_chats", joinValues(opts.allowedChats))
	addOption(options, "allowed_numbers", joinValues(opts.allowedNumbers))
	addOption(options, "default_action", opts.defaultAction)
	addOption(options, "reply", opts.reply)
	if len(options) == 0 {
		return nil
	}
	return options
}

func joinValues(values []string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return strings.Join(clean, ",")
}

func addOption(options map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		options[key] = value
	}
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
