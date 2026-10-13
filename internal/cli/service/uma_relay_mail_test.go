package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var umaRelayArgs = []string{
	"--type", "uma", "--uma-adapter", "relay.messages", "--profile", "assistant",
	"--option", "medium=sms", "--from", "+15558675310",
	"--option", "callback_url=https://gateway.example.com/uma/reply",
	"--auth-scheme", "hmac-sha256", "--signature-secret-env", "RELAY_SIGNING_SECRET",
	"--option", "require_timestamp=true", "--option", "auth_allow_unsigned=true",
	"--env", "RELAY_SIGNING_SECRET=secret:RELAY_SIGNING_SECRET",
	"--env", "RELAY_CALLBACK_TOKEN=secret:RELAY_CALLBACK_TOKEN",
	"--allowed-number", "+15551230001", "--default-action", "triage", "--reply", "text",
}

var umaMailArgs = []string{
	"--type", "uma", "--uma-adapter", "http.mail", "--profile", "assistant",
	"--option", "recipients=support@example.com",
	"--auth-scheme", "bearer", "--auth-token-env", "MAIL_RELAY_TOKEN",
	"--env", "MAIL_RELAY_TOKEN=secret:MAIL_RELAY_TOKEN",
	"--allowed-sender", "*@example.org", "--default-action", "triage",
}

// umaRelayHostEnv points the global config at one allowing gateway.example.com
// as a callback host.
func umaRelayHostEnv(t *testing.T, hosts ...string) {
	t.Helper()
	umaWarnEnv(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	body := "uma:\n  relay:\n    callback_hosts:\n"
	for _, h := range hosts {
		body += "      - \"" + h + "\"\n"
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "aps"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "aps", "config.yaml"), []byte(body), 0o600))
}

func TestUmaRelayAdd_GenericAuthFlagsAreReadNotWarned(t *testing.T) {
	umaRelayHostEnv(t, "gateway.example.com")
	stdout, stderr, err := runAddSplit(t, "relay-uma", umaRelayArgs...)
	require.NoError(t, err, stdout)
	assert.Contains(t, stdout, "config_valid: true")
	assert.Empty(t, warningLines(stderr), "the auth flags configure uma's webhook auth")
}

func TestUmaRelayAdd_CallbackHostNotAllowedIsRefused(t *testing.T) {
	umaRelayHostEnv(t)
	stdout, _, err := runAddSplit(t, "relay-uma", umaRelayArgs...)
	require.Error(t, err)
	assert.Contains(t, stdout, "config_valid: false")
	assert.Contains(t, stdout, "uma.relay.callback_hosts")
	_, statErr := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "aps", "services", "relay-uma.yaml"))
	assert.True(t, os.IsNotExist(statErr), "an invalid service is never saved")
}

func TestUmaRelayAdd_UnsignedTimestampNeedsConsent(t *testing.T) {
	umaRelayHostEnv(t, "gateway.example.com")
	args := make([]string, 0, len(umaRelayArgs))
	for i := 0; i < len(umaRelayArgs); i++ {
		if umaRelayArgs[i] == "--option" && umaRelayArgs[i+1] == "auth_allow_unsigned=true" {
			i++
			continue
		}
		args = append(args, umaRelayArgs[i])
	}
	stdout, _, err := runAddSplit(t, "relay-uma", args...)
	require.Error(t, err)
	assert.Contains(t, stdout, "auth_allow_unsigned")
}

func TestUmaMailAdd_ReplyModeWarns(t *testing.T) {
	umaWarnEnv(t)
	_, stderr, err := runAddSplit(t, "mail-uma", umaMailArgs...)
	require.NoError(t, err, "a reply mode is warned about, not refused")
	assert.Equal(t, []string{
		"warning: the default --reply text has no effect on a uma service: http.mail receives only; " +
			"an action's reply is refused and logged, never sent (set --reply none)",
	}, warningLines(stderr))

	_, stderr, err = runAddSplit(t, "mail-uma", with(umaMailArgs, "--reply", "auto", "--force")...)
	require.NoError(t, err)
	require.Len(t, warningLines(stderr), 1)
	assert.Contains(t, warningLines(stderr)[0], "warning: --reply has no effect on a uma service")

	_, stderr, err = runAddSplit(t, "mail-uma", with(umaMailArgs, "--reply", "none", "--force")...)
	require.NoError(t, err)
	assert.Empty(t, warningLines(stderr))
}

func TestUmaRelayShow_PrintsUmaWebhookAuthAndReplies(t *testing.T) {
	umaRelayHostEnv(t, "gateway.example.com")
	_, _, err := runAddSplit(t, "relay-uma", umaRelayArgs...)
	require.NoError(t, err)

	show := newTestServiceCmd()
	var out bytes.Buffer
	show.SetOut(&out)
	show.SetArgs([]string{"show", "relay-uma"})
	require.NoError(t, show.Execute())
	text := out.String()
	for _, want := range []string{
		"auth:\n  scheme: hmac (uma webhook auth; aps hmac-sha256 layout)\n",
		"  header: X-APS-Signature\n",
		"  secret_binding: RELAY_SIGNING_SECRET\n",
		"  payload: {body}\n",
		"  timestamp_header: X-APS-Timestamp (tolerance 5m0s, unsigned: accepted)\n",
		"receives: HTTP POST /services/relay-uma/webhook (uma verifies webhook auth: hmac)\n",
		"replies: uma/relay.messages text, posted to the gateway's callback\n",
	} {
		assert.Contains(t, text, want)
	}
}

func TestUmaMailShow_RepliesDisabled(t *testing.T) {
	umaWarnEnv(t)
	_, _, err := runAddSplit(t, "mail-uma", umaMailArgs...)
	require.NoError(t, err)

	for _, verb := range []string{"show", "routes"} {
		cmd := newTestServiceCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{verb, "mail-uma"})
		require.NoError(t, cmd.Execute(), verb)
		assert.Contains(t, out.String(), "replies: disabled: uma/http.mail receives only", verb)
	}
}
