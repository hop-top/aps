package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
	"hop.top/aps/internal/logging"
)

func newStatusCmd() *cobra.Command {
	opts := statusOptions{baseURL: "http://127.0.0.1:8080"}

	cmd := &cobra.Command{
		Use:   "status <service-id>",
		Short: "Show operator status for a service",
		Long: `Show operator-facing status for a service: id, type, adapter,
profile, lifecycle hint, reachable webhook URL (from --base-url),
runtime receives/replies, delivery health summary, last inbound
and outbound event metadata, the retry policy, and any
configuration validation issues.

The --base-url flag overrides the default http://127.0.0.1:8080
public base used to render reachable URLs. Read-only: no service
or upstream state is touched. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			renderStatus(cmd, service, opts.baseURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.baseURL, "base-url", opts.baseURL, "Public base URL used to report reachable webhook URLs")
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func newTestCmd() *cobra.Command {
	opts := testOptions{
		baseURL: "http://127.0.0.1:8080",
		timeout: 5 * time.Second,
	}

	cmd := &cobra.Command{
		Use:   "test <service-id>",
		Short: "Validate service configuration and optionally probe its webhook",
		Long: `Validate the persisted service configuration via
core.ValidateServiceConfig — surfacing any issues or warnings —
and report the reachable webhook URL derived from --base-url. The
command exits non-zero if the config is invalid.

With --probe, additionally POST a synthetic adapter-shaped
payload (signed for telegram/slack/sms/whatsapp where signing
secrets are configured) at the webhook URL and print the response
status and body. The synthetic inbound impersonates the first
configured allowlist entry (allowed_numbers, allowed_chats,
allowed_channels, allowed_guilds) and the service's own channel
identity, so it passes the same allowlist checks real traffic
must pass; probe_sender/probe_channel report what was sent. A 403
therefore means the allowlist rejected the probe identity, not
that the endpoint is down. --timeout bounds the probe round-trip;
the default is 5s.

Read-only on aps state; --probe makes a live outbound HTTP call
each invocation so the kit-level idempotency tag is Conditional.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			validation := core.ValidateServiceConfig(service)
			renderValidation(cmd, validation)
			webhookURL, urlErr := core.ServiceWebhookURL(service, opts.baseURL)
			if urlErr == nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "webhook_url: %s\n", webhookURL)
			}
			if !validation.Valid {
				return fmt.Errorf("service config is invalid")
			}
			if urlErr != nil {
				return urlErr
			}
			if !opts.probe {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "probe: skipped")
				return nil
			}
			return probeServiceWebhook(cmd, service, webhookURL, opts.timeout)
		},
	}
	cmd.Flags().StringVar(&opts.baseURL, "base-url", opts.baseURL, "Public base URL to probe")
	cmd.Flags().BoolVar(&opts.probe, "probe", false, "POST a synthetic message payload to the webhook URL")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", opts.timeout, "Webhook probe timeout")
	// Validates a service config; with --probe POSTs an outbound HTTP
	// request, but the local effect on aps state is read-only. Classify
	// as Read + Conditional (idempotent re-runs without --probe; with
	// --probe each call hits the upstream webhook).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	return cmd
}

func newStartCmd() *cobra.Command {
	opts := startOptions{
		addr:    "127.0.0.1:8080",
		baseURL: "http://127.0.0.1:8080",
	}

	cmd := &cobra.Command{
		Use:   "start <service-id>",
		Short: "Start APS HTTP serving for a service webhook",
		Long: `Start APS HTTP serving for a service webhook.

APS serves message services through the shared protocol server. This command
validates the selected service, reports its reachable webhook URL, starts the
same mounted service routes as aps serve, and runs in the foreground until
interrupted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := core.LoadService(args[0])
			if err != nil {
				return err
			}
			validation := core.ValidateServiceConfig(service)
			renderValidation(cmd, validation)
			if !validation.Valid {
				return fmt.Errorf("service config is invalid")
			}
			webhookURL, err := core.ServiceWebhookURL(service, opts.baseURL)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "lifecycle: starting\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "listen_addr: %s\n", opts.addr)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "webhook_url: %s\n", webhookURL)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "stop: interrupt this process")
			return serveServiceHTTP(cmd.Context(), opts.addr)
		},
	}
	cmd.Flags().StringVar(&opts.addr, "addr", opts.addr, "Address to listen on")
	cmd.Flags().StringVar(&opts.baseURL, "base-url", opts.baseURL, "Public base URL used to report reachable webhook URLs")
	// Long-running daemon — interactive band per kit's 6-tier ladder.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectInteractive)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	return cmd
}

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <service-id>",
		Short: "Show how to stop a foreground service server",
		Long: `Print guidance for stopping the foreground HTTP server that
hosts the named service. The aps service lifecycle is external:
the server runs only while aps service start (or aps serve) is
in the foreground, so stop prints the canonical "interrupt the
process" instruction rather than sending any signal itself.

Read-only: loads the service record to confirm the id exists,
then prints the lifecycle hint. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := core.LoadService(args[0]); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "lifecycle: external")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "stop: interrupt the aps service start or aps serve process that owns this service")
			return nil
		},
	}
	// `stop` here only prints guidance — no state mutation.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

type statusOptions struct {
	baseURL string
}

type testOptions struct {
	baseURL string
	probe   bool
	timeout time.Duration
}

type startOptions struct {
	addr    string
	baseURL string
}

func renderStatus(cmd *cobra.Command, service *core.ServiceConfig, baseURL string) {
	runtime := core.DescribeServiceRuntime(service)
	validation := core.ValidateServiceConfig(service)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "id: %s\n", service.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "type: %s\n", service.Type)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "adapter: %s\n", service.Adapter)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", service.Profile)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "lifecycle: configured")
	if url, err := core.ServiceWebhookURL(service, baseURL); err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "webhook_url: %s\n", url)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "webhook_url: none")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "receives: %s\n", runtime.Receives)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "replies: %s\n", runtime.Replies)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "delivery_health: %s\n", deliveryHealth(service))
	renderDelivery(cmd, service)
	renderEvent(cmd, "last_inbound", service.LastInbound)
	renderEvent(cmd, "last_outbound", service.LastOutbound)
	renderValidation(cmd, validation)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "start: aps service start "+service.ID)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "stop: interrupt the foreground server")
}

func renderValidation(cmd *cobra.Command, validation core.ServiceValidationResult) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_valid: %t\n", validation.Valid)
	for _, issue := range validation.Issues {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_issue: %s\n", issue)
	}
	for _, warning := range validation.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "config_warning: %s\n", warning)
	}
}

func renderDelivery(cmd *cobra.Command, service *core.ServiceConfig) {
	if service == nil || service.Delivery == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "delivery_status: unknown")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "retry_policy: max_attempts=3 base_delay=1s max_delay=30s")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "delivery_attempts: none")
		return
	}
	delivery := service.Delivery
	status := strings.TrimSpace(delivery.Status)
	if status == "" {
		status = deliveryHealth(service)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "delivery_status: %s\n", status)
	if delivery.LastError != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "delivery_last_error: %s\n", logging.Apply(delivery.LastError))
	}
	if delivery.RetryPolicy != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "retry_policy: max_attempts=%d base_delay=%s max_delay=%s\n",
			delivery.RetryPolicy.MaxAttempts,
			delivery.RetryPolicy.BaseDelay,
			delivery.RetryPolicy.MaxDelay,
		)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "retry_policy: max_attempts=3 base_delay=1s max_delay=30s")
	}
	if len(delivery.Attempts) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "delivery_attempts: none")
		return
	}
	for _, attempt := range delivery.Attempts {
		renderDeliveryAttempt(cmd, attempt)
	}
}

func renderEvent(cmd *cobra.Command, label string, event *core.ServiceEventMeta) {
	if event == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: none\n", label)
		return
	}
	parts := []string{}
	if !event.At.IsZero() {
		parts = append(parts, event.At.UTC().Format(time.RFC3339))
	}
	addEventPart := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, key+"="+value)
		}
	}
	addEventPart("message_id", event.MessageID)
	addEventPart("platform", event.Platform)
	addEventPart("channel_id", event.ChannelID)
	addEventPart("sender_id", event.SenderID)
	addEventPart("status", event.Status)
	if event.Detail != "" {
		addEventPart("detail", logging.Apply(event.Detail))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", label, strings.Join(parts, " "))
}

func renderDeliveryAttempt(cmd *cobra.Command, attempt core.ServiceDeliveryAttempt) {
	parts := []string{}
	if !attempt.At.IsZero() {
		parts = append(parts, attempt.At.UTC().Format(time.RFC3339))
	}
	add := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, key+"="+value)
		}
	}
	add("provider", attempt.Provider)
	add("message_id", attempt.MessageID)
	add("channel_id", attempt.ChannelID)
	if attempt.Attempt > 0 {
		add("attempt", strconv.Itoa(attempt.Attempt))
	}
	if attempt.MaxAttempts > 0 {
		add("max_attempts", strconv.Itoa(attempt.MaxAttempts))
	}
	add("status", attempt.Status)
	add("delivery_id", attempt.DeliveryID)
	if attempt.Retriable {
		add("retriable", "true")
	}
	add("delay", attempt.Delay)
	if attempt.RedactedError != "" {
		add("error", logging.Apply(attempt.RedactedError))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "delivery_attempt: %s\n", strings.Join(parts, " "))
}

func deliveryHealth(service *core.ServiceConfig) string {
	if service.Delivery != nil && service.Delivery.Health != "" {
		return service.Delivery.Health
	}
	if service.LastInbound != nil || service.LastOutbound != nil {
		return "receiving"
	}
	return "unknown"
}

func probeServiceWebhook(cmd *cobra.Command, service *core.ServiceConfig, webhookURL string, timeout time.Duration) error {
	payload, identity, err := core.SyntheticMessageWebhookPayload(service.Adapter, service.Options)
	if err != nil {
		return err
	}
	renderProbeIdentity(cmd, identity)
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if service.Adapter == "telegram" {
		if token := telegramWebhookSecret(service); token != "" {
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", token)
		}
	}
	if service.Adapter == "slack" {
		signSlackProbe(req, service, payload)
	}
	signProviderProbe(req, service, payload)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook probe failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "probe_status: %d\n", resp.StatusCode)
	if len(body) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "probe_response: %s\n", strings.TrimSpace(string(body)))
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("webhook probe returned HTTP 403: synthetic inbound rejected; check the service allowed_* options against probe_sender/probe_channel above")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook probe returned HTTP %d", resp.StatusCode)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "probe_verified: webhook reachable; synthetic inbound accepted through request auth and allowlist checks")
	return nil
}

// renderProbeIdentity states which identities the synthetic inbound carries
// so operators can tell an allowlisted probe from a placeholder one.
func renderProbeIdentity(cmd *cobra.Command, identity core.SyntheticProbeIdentity) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "probe_sender: %s (%s)\n", identity.Sender, identity.SenderSource)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "probe_channel: %s (%s)\n", identity.Channel, identity.ChannelSource)
	if identity.Workspace != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "probe_workspace: %s (%s)\n", identity.Workspace, identity.WorkspaceSource)
	}
}

func telegramWebhookSecret(service *core.ServiceConfig) string {
	if service == nil || service.Options == nil {
		return ""
	}
	if token := strings.TrimSpace(service.Options["webhook_secret_token"]); token != "" {
		return token
	}
	if envName := strings.TrimSpace(service.Options["webhook_secret_token_env"]); envName != "" {
		return os.Getenv(envName)
	}
	return ""
}

func signSlackProbe(req *http.Request, service *core.ServiceConfig, payload []byte) {
	secret := serviceEnvSecret(service, "SLACK_SIGNING_SECRET")
	if secret == "" {
		return
	}
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + ts + ":"))
	_, _ = mac.Write(payload)
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
}

func signProviderProbe(req *http.Request, service *core.ServiceConfig, payload []byte) {
	if service == nil {
		return
	}
	switch strings.TrimSpace(strings.ToLower(service.Adapter)) {
	case "sms":
		if serviceProvider(service, "twilio") == "twilio" {
			signTwilioProbe(req, service, payload)
		}
	case "whatsapp":
		switch serviceProvider(service, "whatsapp-cloud") {
		case "twilio":
			signTwilioProbe(req, service, payload)
		case "whatsapp-cloud":
			signWhatsAppCloudProbe(req, service, payload)
		}
	}
}

func signTwilioProbe(req *http.Request, service *core.ServiceConfig, payload []byte) {
	token := serviceConfiguredSecret(
		service,
		[]string{"twilio_auth_token", "auth_token", "signature_secret"},
		[]string{"auth_token_env", "signature_secret_env"},
		"TWILIO_AUTH_TOKEN",
	)
	if token == "" {
		return
	}
	form, _ := url.ParseQuery(string(payload))
	req.Header.Set(msgtypes.TwilioSignatureHeader, msgtypes.TwilioSignature(token, req.URL.String(), form))
}

func signWhatsAppCloudProbe(req *http.Request, service *core.ServiceConfig, payload []byte) {
	secret := serviceConfiguredSecret(
		service,
		[]string{"app_secret", "signature_secret"},
		[]string{"app_secret_env", "signature_secret_env", "signing_secret_env"},
		"WHATSAPP_APP_SECRET",
	)
	if secret == "" {
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	req.Header.Set(msgtypes.WhatsAppSignatureHeader, "sha256="+hex.EncodeToString(mac.Sum(nil)))
}

func serviceProvider(service *core.ServiceConfig, fallback string) string {
	if service != nil && service.Options != nil {
		if provider := strings.TrimSpace(strings.ToLower(service.Options["provider"])); provider != "" {
			return provider
		}
	}
	return fallback
}

func serviceConfiguredSecret(service *core.ServiceConfig, optionKeys []string, optionEnvKeys []string, envKeys ...string) string {
	if service != nil && service.Options != nil {
		for _, key := range optionKeys {
			if value := resolveServiceSecretValue(service.Options[key]); value != "" {
				return value
			}
		}
		for _, key := range optionEnvKeys {
			if value := os.Getenv(strings.TrimSpace(service.Options[key])); value != "" {
				return value
			}
		}
	}
	for _, key := range envKeys {
		if value := serviceEnvSecret(service, key); value != "" {
			return value
		}
	}
	return ""
}

func serviceEnvSecret(service *core.ServiceConfig, key string) string {
	if service != nil && service.Env != nil {
		if value := resolveServiceSecretValue(service.Env[key]); value != "" {
			return value
		}
	}
	return os.Getenv(key)
}

func resolveServiceSecretValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "secret:") {
		return os.Getenv(strings.TrimSpace(strings.TrimPrefix(value, "secret:")))
	}
	return value
}

func serveServiceHTTP(ctx context.Context, addr string) error {
	mgr := adapters.DefaultManager()
	if err := mgr.InitAll(ctx); err != nil {
		return fmt.Errorf("initialising adapters: %w", err)
	}
	defer func() {
		if errs := mgr.CloseAll(); len(errs) != 0 {
			logging.GetLogger().Error("adapter close errors", nil, "errors", errs)
		}
	}()

	apsCore, err := protocol.NewAPSAdapter()
	if err != nil {
		return fmt.Errorf("creating core adapter: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}` + "\n"))
	})
	if err := mgr.RegisterRoutes(mux, apsCore); err != nil {
		return fmt.Errorf("registering adapter routes: %w", err)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
	case <-sigCh:
	case err := <-errCh:
		return err
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("service server shutdown: %w", err)
	}
	return <-errCh
}
