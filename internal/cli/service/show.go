package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <service-id>",
		Short: "Show a persisted service",
		Long: `Show the full configuration for a persisted service: id, type,
backing adapter, owning profile, optional description, and the
adapter-specific option map sorted by key. Message services also
print the inbound auth the validator will enforce (auth: scheme,
header, token/secret env names, timestamp and replay headers, or
"auth: none"). Runtime metadata from core.DescribeServiceRuntime
(receives, executes, replies, maturity) is also printed.

Read-only: loads the service record from the service store and
prints it. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "id: %s\n", service.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "type: %s\n", service.Type)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "adapter: %s\n", service.Adapter)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", service.Profile)
			if service.Description != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "description: %s\n", service.Description)
			}
			if len(service.Options) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "options:")
				keys := make([]string, 0, len(service.Options))
				for key := range service.Options {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					value := service.Options[key]
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", key, value)
				}
			}
			printRouting(cmd, service)
			printAuth(cmd, service)
			runtime := core.DescribeServiceRuntime(service)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "receives: %s\n", runtime.Receives)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "executes: %s\n", runtime.Executes)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "replies: %s\n", runtime.Replies)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "maturity: %s\n", runtime.Maturity)
			return nil
		},
	}
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

// printAuth renders the inbound auth the messenger validator will enforce
// for a message service, so operators can confirm generic webhook auth
// (auth_scheme/auth_token_env/signature_secret_env) and provider hooks
// without reading the validator. Secret values are never printed.
func printAuth(cmd *cobra.Command, service *core.ServiceConfig) {
	if service == nil || service.Type != core.ServiceTypeMessage {
		return
	}
	out := cmd.OutOrStdout()
	summary := msgtypes.NewServiceValidator().DescribeAuth(msgtypes.ServiceValidationConfig{
		ID:      service.ID,
		Adapter: service.Adapter,
		Env:     service.Env,
		Options: service.Options,
	})
	req := summary.Requirements
	if req.Scheme == msgtypes.AuthSchemeNone && !summary.ProviderValidated {
		_, _ = fmt.Fprintln(out, "auth: none")
		return
	}
	_, _ = fmt.Fprintln(out, "auth:")
	scheme := string(req.Scheme)
	if summary.ProviderValidated && req.Scheme == msgtypes.AuthSchemeNone {
		scheme = summary.Provider + "-signature"
	}
	_, _ = fmt.Fprintf(out, "  scheme: %s\n", scheme)
	if summary.Provider != "" {
		_, _ = fmt.Fprintf(out, "  provider: %s\n", summary.Provider)
	}
	if req.Header != "" {
		_, _ = fmt.Fprintf(out, "  header: %s\n", req.Header)
	}
	if req.Token != "" {
		_, _ = fmt.Fprintln(out, "  token: (literal in options)")
	}
	if req.TokenEnv != "" {
		_, _ = fmt.Fprintf(out, "  token_env: %s\n", req.TokenEnv)
	}
	if req.SignatureSecret != "" {
		_, _ = fmt.Fprintln(out, "  signature_secret: (literal in options)")
	}
	if req.SignatureSecretEnv != "" {
		_, _ = fmt.Fprintf(out, "  signature_secret_env: %s\n", req.SignatureSecretEnv)
	}
	if req.TimestampHeader != "" {
		_, _ = fmt.Fprintf(out, "  timestamp_header: %s (tolerance %s)\n", req.TimestampHeader, req.TimestampTolerance)
	}
	if req.ReplayIDHeader != "" {
		_, _ = fmt.Fprintf(out, "  replay_id_header: %s\n", req.ReplayIDHeader)
	}
}

// printRouting renders the compiled sender route table, or the reason it
// failed to compile, so operators can eyeball dispatch without reading YAML.
func printRouting(cmd *cobra.Command, service *core.ServiceConfig) {
	if !core.HasRouteTable(service) {
		return
	}
	out := cmd.OutOrStdout()
	table, err := core.LoadServiceRouteTable(service)
	if err != nil {
		_, _ = fmt.Fprintf(out, "routing_error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintln(out, "routing:")
	_, _ = fmt.Fprintf(out, "  file: %s\n", table.Source())
	if service.Routing.Contacts != nil {
		source := strings.TrimSpace(service.Routing.Contacts.Path)
		if source == "" {
			source = "inline"
		}
		_, _ = fmt.Fprintf(out, "  contacts: %s (%d entries)\n", source, len(table.Contacts()))
	} else if contacts := table.Contacts(); len(contacts) > 0 {
		_, _ = fmt.Fprintf(out, "  contacts: %s (%d entries)\n", table.Source(), len(contacts))
	}
	_, _ = fmt.Fprintln(out, "  routes:")
	routes := table.Routes()
	for i, route := range routes {
		suffix := ""
		if i == len(routes)-1 {
			suffix = " (terminal)"
		}
		_, _ = fmt.Fprintf(out, "    - %s -> %s=%s%s\n", route.Match, route.Profile, route.Action, suffix)
	}
}

func newRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes <service-id>",
		Short: "Show reachable routes for a persisted service",
		Long: `Show the HTTP routes that aps serve (or aps service start)
would mount for the named service. Routes are derived by
core.DescribeServiceRuntime from the service's adapter type and
configuration; "routes: none" prints when the adapter exposes no
endpoints.

Read-only: no service state mutation. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			runtime := core.DescribeServiceRuntime(service)
			if len(runtime.Routes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "routes: none")
				return nil
			}
			for _, route := range runtime.Routes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", route)
			}
			return nil
		},
	}
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}
