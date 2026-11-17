package service

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func nativeNoticeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	ResetNativeTransportNotice()
	t.Cleanup(ResetNativeTransportNotice)
}

func runService(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

var nativeTwilioSMSArgs = []string{
	"--type", "sms", "--profile", "assistant", "--provider", "twilio",
	"--from", "+15559870002", "--allowed-number", "+15551230001",
	"--default-action", "triage", "--reply", "text",
	"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
	"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
}

func addNative(t *testing.T, id string) {
	t.Helper()
	out, err := runService(t, append([]string{"add", id}, nativeTwilioSMSArgs...)...)
	require.NoError(t, err, out)
}

func TestNativeNotice_AddNativeSMSNotesIt(t *testing.T) {
	nativeNoticeEnv(t)
	assert.False(t, NativeTransportNoted())
	addNative(t, "sms-main")
	assert.True(t, NativeTransportNoted())
}

func TestNativeNotice_AddDryRunNotesIt(t *testing.T) {
	nativeNoticeEnv(t)
	out, err := runService(t, append([]string{"add", "sms-main", "--dry-run"}, nativeTwilioSMSArgs...)...)
	require.NoError(t, err, out)
	assert.True(t, NativeTransportNoted())
}

func TestNativeNotice_AddUmaDoesNotNoteIt(t *testing.T) {
	nativeNoticeEnv(t)
	out, err := runService(t, "add", "sms-uma", "--type", "uma", "--uma-adapter", "twilio.sms",
		"--profile", "assistant", "--from", "+15559870002",
		"--webhook-url", "https://hooks.example.com/services/sms-uma/webhook",
		"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
		"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
		"--default-action", "triage")
	require.NoError(t, err, out)
	assert.False(t, NativeTransportNoted())
}

func TestNativeNotice_AddOtherServicesDoNotNoteIt(t *testing.T) {
	cases := map[string][]string{
		"slack": {
			"--type", "slack", "--default-action", "triage",
			"--env", "SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN",
			"--env", "SLACK_SIGNING_SECRET=secret:SLACK_SIGNING_SECRET",
		},
		"sms generic": {
			"--type", "sms", "--provider", "generic", "--from", "+15550100002",
			"--default-action", "triage",
		},
		"email relay": {"--type", "message", "--adapter", "email", "--default-action", "triage"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			nativeNoticeEnv(t)
			out, err := runService(t, append([]string{"add", "svc", "--profile", "assistant"}, args...)...)
			require.NoError(t, err, out)
			assert.False(t, NativeTransportNoted())
		})
	}
}

func TestNativeNotice_ShowAndStatusNoteNativeServices(t *testing.T) {
	for _, verb := range []string{"show", "status"} {
		t.Run(verb, func(t *testing.T) {
			nativeNoticeEnv(t)
			addNative(t, "sms-main")
			ResetNativeTransportNotice()
			out, err := runService(t, verb, "sms-main")
			require.NoError(t, err, out)
			assert.True(t, NativeTransportNoted())
		})
	}
}

func TestNativeNotice_ShowAndStatusSkipUmaServices(t *testing.T) {
	for _, verb := range []string{"show", "status"} {
		t.Run(verb, func(t *testing.T) {
			nativeNoticeEnv(t)
			require.NoError(t, core.SaveService(&core.ServiceConfig{
				ID: "sms-uma", Type: "message", Adapter: "uma", Profile: "assistant",
				Env: map[string]string{
					"TWILIO_ACCOUNT_SID": "secret:TWILIO_ACCOUNT_SID",
					"TWILIO_AUTH_TOKEN":  "secret:TWILIO_AUTH_TOKEN",
				},
				Options: map[string]string{
					core.OptionUmaAdapter: "twilio.sms", "from": "+15559870002",
					"webhook_url": "https://hooks.example.com/services/sms-uma/webhook", "default_action": "triage",
				},
			}))
			out, err := runService(t, verb, "sms-uma")
			require.NoError(t, err, out)
			assert.False(t, NativeTransportNoted())
		})
	}
}

func TestNativeNotice_DoesNotChangeFailures(t *testing.T) {
	nativeNoticeEnv(t)
	// A native service missing its bindings stays invalid and unsaved.
	out, err := runService(t, "add", "sms-bad", "--type", "sms", "--profile", "assistant",
		"--provider", "twilio", "--from", "+15559870002", "--default-action", "triage")
	require.Error(t, err)
	assert.Contains(t, out, "config_valid: false")
	exists, existsErr := core.ServiceExists("sms-bad")
	require.NoError(t, existsErr)
	assert.False(t, exists)

	_, err = runService(t, "show", "missing")
	require.Error(t, err)
}

func TestNativeNotice_TextIsActionable(t *testing.T) {
	assert.Contains(t, NativeTransportNotice, "--type uma")
	assert.Contains(t, NativeTransportNotice, "docs/user/uma-migration.md")
	assert.NotRegexp(t, `(?i)\b(20\d\d|remov(e|al)|sunset|end of life)\b`, NativeTransportNotice,
		"the notice must not promise a removal date")
}
