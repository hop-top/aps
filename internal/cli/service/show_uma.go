package service

import (
	"fmt"
	"io"

	"hop.top/aps/internal/core"
)

// printUmaWebhookAuth renders the webhook auth uma enforces for a uma service
// whose adapter takes generic webhooks (relay.messages, http.mail): aps's
// generic auth options as mapped onto uma's webhookauth, not aps's own
// validator, which never runs on a uma route. It reports whether it printed.
// Secret values are never printed; the binding name is.
func printUmaWebhookAuth(out io.Writer, service *core.ServiceConfig) bool {
	spec, ok := core.UmaAdapterSpecForService(service)
	if !ok || !spec.GenericWebhook {
		return false
	}
	auth, issues := core.UmaWebhookAuthFor(service.Options)
	if auth.Scheme == "" {
		_, _ = fmt.Fprintln(out, "auth: none configured (uma refuses every request)")
		return true
	}
	_, _ = fmt.Fprintln(out, "auth:")
	_, _ = fmt.Fprintf(out, "  scheme: %s (uma webhook auth; aps %s layout)\n", auth.Scheme, core.ApsAuthScheme(service.Options))
	_, _ = fmt.Fprintf(out, "  header: %s\n", auth.Header)
	if auth.SecretBinding != "" {
		_, _ = fmt.Fprintf(out, "  secret_binding: %s\n", auth.SecretBinding)
	}
	if auth.Payload != "" {
		_, _ = fmt.Fprintf(out, "  payload: %s\n", auth.Payload)
	}
	if auth.MinSecretBytes > 0 {
		_, _ = fmt.Fprintf(out, "  min_secret_bytes: %d\n", auth.MinSecretBytes)
	}
	if auth.TimestampHeader != "" {
		_, _ = fmt.Fprintf(out, "  timestamp_header: %s (tolerance %s, %s)\n",
			auth.TimestampHeader, auth.TimestampTolerance, signedLabel(auth.TimestampUnsigned))
	}
	if auth.ReplayHeader != "" {
		_, _ = fmt.Fprintf(out, "  replay_id_header: %s (%s)\n", auth.ReplayHeader, signedLabel(auth.ReplayUnsigned))
	}
	for _, issue := range issues {
		_, _ = fmt.Fprintf(out, "  issue: %s\n", issue)
	}
	return true
}

func signedLabel(unsigned bool) string {
	if unsigned {
		return "unsigned: accepted"
	}
	return "signed"
}
