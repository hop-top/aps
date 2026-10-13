package service

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/core"
)

var umaTwilioArgs = []string{
	"--type", "uma", "--uma-adapter", "twilio.sms", "--profile", "assistant",
	"--from", "+15558675310", "--webhook-url", "https://hooks.example.com/twilio/sms",
	"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
	"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
	"--allowed-number", "+15551230001", "--default-action", "triage",
}

var umaMetaArgs = []string{
	"--type", "uma", "--uma-adapter", "meta.whatsapp", "--profile", "assistant",
	"--phone-number-id", "104857600000001",
	"--env", "WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN",
	"--env", "WHATSAPP_APP_SECRET=secret:WHATSAPP_APP_SECRET",
	"--env", "WHATSAPP_VERIFY_TOKEN=secret:WHATSAPP_VERIFY_TOKEN",
	"--allowed-number", "+15551230001", "--default-action", "triage",
}

func umaWarnEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("HOP_QUIET_HINTS", "")
	globals.SetViper(nil)
	t.Cleanup(func() { globals.SetViper(nil) })
}

// setGlobal wires a viper with key set, as the root command's persistent
// flags would.
func setGlobal(t *testing.T, key string, value any) {
	t.Helper()
	v := viper.New()
	v.Set(key, value)
	globals.SetViper(v)
}

// runAddSplit runs `service add` with stdout and stderr kept apart.
func runAddSplit(t *testing.T, id string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newTestServiceCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"add", id}, args...))
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

func with(base []string, extra ...string) []string {
	return append(append([]string{}, base...), extra...)
}

func warningLines(stderr string) []string {
	var lines []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "warning:") {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestUmaFlagWarning_EachIgnoredFlagWarns(t *testing.T) {
	cases := []struct {
		name string
		base []string
		args []string
		flag string
	}{
		{"template-name", umaTwilioArgs, []string{"--template-name", "hello"}, "--template-name"},
		{"language-code", umaTwilioArgs, []string{"--language-code", "en"}, "--language-code"},
		{"template-required", umaTwilioArgs, []string{"--template-required"}, "--template-required"},
		{"verify-token", umaMetaArgs, []string{"--verify-token", "not-a-real-token"}, "--verify-token"},
		{"verify-token-env", umaMetaArgs, []string{"--verify-token-env", "FAKE_VERIFY"}, "--verify-token-env"},
		{"signing-secret-env", umaMetaArgs, []string{"--signing-secret-env", "FAKE_SECRET"}, "--signing-secret-env"},
		{"signing-secret-env twilio", umaTwilioArgs, []string{"--signing-secret-env", "FAKE_SECRET"}, "--signing-secret-env"},
		{"webhook-secret-token", umaTwilioArgs, []string{"--webhook-secret-token", "not-a-real-token"}, "--webhook-secret-token"},
		{"webhook-secret-token-env", umaTwilioArgs, []string{"--webhook-secret-token-env", "FAKE"}, "--webhook-secret-token-env"},
		{"auth-token-env", umaTwilioArgs, []string{"--auth-token-env", "FAKE_AUTH"}, "--auth-token-env"},
		{"signature-secret-env", umaTwilioArgs, []string{"--signature-secret-env", "FAKE_SIG"}, "--signature-secret-env"},
		{"bot-user-id", umaTwilioArgs, []string{"--bot-user-id", "U0123"}, "--bot-user-id"},
		{"require-bot-mention", umaTwilioArgs, []string{"--require-bot-mention"}, "--require-bot-mention"},
		{"dedup-ttl", umaTwilioArgs, []string{"--dedup-ttl", "5m"}, "--dedup-ttl"},
		{"site", umaTwilioArgs, []string{"--site", "https://jira.example.com"}, "--site"},
		{"project", umaTwilioArgs, []string{"--project", "OPS"}, "--project"},
		{"jql", umaTwilioArgs, []string{"--jql", "assignee = me"}, "--jql"},
		{"linear-workspace", umaTwilioArgs, []string{"--linear-workspace", "acme"}, "--linear-workspace"},
		{"team", umaTwilioArgs, []string{"--team", "ENG"}, "--team"},
		{"group", umaTwilioArgs, []string{"--group", "acme/ops"}, "--group"},
		{"events", umaTwilioArgs, []string{"--events", "issue.created"}, "--events"},
		{"provider", umaTwilioArgs, []string{"--provider", "twilio"}, "--provider"},
		{"receive polling", umaTwilioArgs, []string{"--receive", "polling"}, "--receive"},
		{"phone-number-id on twilio", umaTwilioArgs, []string{"--phone-number-id", "104857600000001"}, "--phone-number-id"},
		{"from on meta", umaMetaArgs, []string{"--from", "+15558675310"}, "--from"},
		{"webhook-url on meta", umaMetaArgs, []string{"--webhook-url", "https://hooks.example.com/wa"}, "--webhook-url"},
		{"raw option", umaTwilioArgs, []string{"--option", "template_name=hello"}, "--option template_name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			umaWarnEnv(t)
			stdout, stderr, err := runAddSplit(t, "svc-uma", with(tc.base, tc.args...)...)
			require.NoError(t, err, stdout+stderr)
			lines := warningLines(stderr)
			require.Len(t, lines, 1, "one warning per ignored flag: %q", stderr)
			assert.Contains(t, lines[0], tc.flag+" has no effect on a uma service")
			assert.NotContains(t, stdout, "has no effect", "warnings go to stderr")
			exists, existsErr := core.ServiceExists("svc-uma")
			require.NoError(t, existsErr)
			assert.True(t, exists, "warn, not reject: the service is still added")
		})
	}
}

func TestUmaFlagWarning_AdviceNamesWhatToUse(t *testing.T) {
	umaWarnEnv(t)
	_, stderr, err := runAddSplit(t, "wa-uma", with(umaMetaArgs,
		"--verify-token-env", "FAKE_VERIFY", "--signing-secret-env", "FAKE_SECRET", "--template-name", "hello")...)
	require.NoError(t, err, stderr)
	assert.Contains(t, stderr, "--env WHATSAPP_VERIFY_TOKEN=secret:")
	assert.Contains(t, stderr, "--env WHATSAPP_APP_SECRET=secret:")
	assert.Contains(t, stderr, "templates are not sent")
	assert.Len(t, warningLines(stderr), 3)
}

func TestUmaFlagWarning_NeverEchoesAValue(t *testing.T) {
	umaWarnEnv(t)
	_, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs,
		"--verify-token", "hunter2-verify", "--webhook-secret-token", "hunter2-hook",
		"--option", "auth_token=hunter2-auth")...)
	require.NoError(t, err, stderr)
	assert.Len(t, warningLines(stderr), 3)
	assert.NotContains(t, stderr, "hunter2")
}

func TestUmaFlagWarning_TwoPairedFlagsWarnOnceEach(t *testing.T) {
	umaWarnEnv(t)
	_, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs,
		"--auth-scheme", "bearer", "--auth-token-env", "FAKE_AUTH")...)
	require.NoError(t, err, stderr)
	lines := warningLines(stderr)
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0]+lines[1], "--auth-scheme has no effect")
	assert.Contains(t, lines[0]+lines[1], "--auth-token-env has no effect")
}

func TestUmaFlagWarning_FlagPositionDoesNotMatter(t *testing.T) {
	for name, args := range map[string][]string{
		"flags after the id":  append([]string{"add", "svc-uma"}, with(umaTwilioArgs, "--template-name", "hello")...),
		"flags before the id": append(append([]string{"add", "--template-name", "hello"}, umaTwilioArgs...), "svc-uma"),
		"ignored flag first":  append([]string{"add", "svc-uma", "--template-name", "hello"}, umaTwilioArgs...),
	} {
		t.Run(name, func(t *testing.T) {
			umaWarnEnv(t)
			cmd := newTestServiceCmd()
			var out, errOut bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)
			cmd.SetArgs(args)
			require.NoError(t, cmd.Execute(), out.String()+errOut.String())
			assert.Len(t, warningLines(errOut.String()), 1)
		})
	}
}

func TestUmaFlagWarning_ConsumedFlagsDoNotWarn(t *testing.T) {
	cases := map[string][]string{
		"twilio.sms baseline": with(umaTwilioArgs),
		"twilio.sms consumed": with(umaTwilioArgs,
			"--description", "d", "--label", "k=v", "--reply", "text", "--history-turns", "5",
			"--receive", "webhook", "--allowed-sender", "a@example.com", "--allowed-channel", "c",
			"--option", "messaging_service_sid=MG0123", "--option", "execution=chat"),
		"twilio.whatsapp": {
			"--type", "uma", "--uma-adapter", "twilio.whatsapp", "--profile", "assistant",
			"--from", "+15558675310", "--webhook-url", "https://hooks.example.com/twilio/wa",
			"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
			"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
			"--allowed-number", "whatsapp:+15551230001", "--default-action", "triage",
		},
		"meta.whatsapp baseline": with(umaMetaArgs),
		"meta.whatsapp consumed": with(umaMetaArgs, "--reply", "none", "--history-turns", "3", "--receive", "webhook"),
		"explicit defaults":      with(umaTwilioArgs, "--template-required=false", "--require-bot-mention=false", "--receive", "webhook"),
		"type message adapter uma": {
			"--type", "message", "--adapter", "uma", "--uma-adapter", "twilio.sms", "--profile", "assistant",
			"--from", "+15558675310", "--webhook-url", "https://hooks.example.com/twilio/sms",
			"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
			"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
			"--allowed-number", "+15551230001", "--default-action", "triage",
		},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			umaWarnEnv(t)
			stdout, stderr, err := runAddSplit(t, "svc-uma", args...)
			require.NoError(t, err, stdout+stderr)
			assert.NotContains(t, stderr, "has no effect")
			assert.Empty(t, warningLines(stderr))
		})
	}
}

func TestUmaFlagWarning_FlagsOnATypeMessageUmaServiceWarnToo(t *testing.T) {
	umaWarnEnv(t)
	_, stderr, err := runAddSplit(t, "svc-uma",
		"--type", "message", "--adapter", "uma", "--uma-adapter", "twilio.sms", "--profile", "assistant",
		"--from", "+15558675310", "--webhook-url", "https://hooks.example.com/twilio/sms",
		"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
		"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
		"--default-action", "triage", "--template-name", "hello")
	require.NoError(t, err, stderr)
	assert.Len(t, warningLines(stderr), 1)
}

func TestUmaFlagWarning_NativeServicesAreUnchanged(t *testing.T) {
	native := map[string][]string{
		"sms twilio": with(nativeTwilioSMSArgs,
			"--template-name", "hello", "--language-code", "en", "--template-required",
			"--signing-secret-env", "FAKE", "--verify-token-env", "FAKE", "--bot-user-id", "U1"),
		"whatsapp cloud": {
			"--type", "whatsapp", "--profile", "assistant", "--provider", "whatsapp-cloud",
			"--phone-number-id", "104857600000001", "--allowed-number", "+15551230001",
			"--default-action", "triage", "--reply", "text",
			"--template-name", "hello", "--language-code", "en", "--template-required",
			"--verify-token-env", "FAKE_VERIFY", "--signing-secret-env", "FAKE_SECRET",
			"--env", "WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN",
		},
		"slack": {
			"--type", "slack", "--profile", "assistant", "--default-action", "triage",
			"--bot-user-id", "U1", "--require-bot-mention", "--dedup-ttl", "5m",
			"--env", "SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN",
			"--env", "SLACK_SIGNING_SECRET=secret:SLACK_SIGNING_SECRET",
		},
	}
	for name, args := range native {
		t.Run(name, func(t *testing.T) {
			umaWarnEnv(t)
			stdout, stderr, err := runAddSplit(t, "svc-native", args...)
			require.NoError(t, err, stdout+stderr)
			assert.Empty(t, stderr, "a native service gets no warning and no stderr output")
			assert.NotContains(t, stdout, "has no effect")
		})
	}
}

func TestUmaFlagWarning_NativeOutputIsByteForByteTheSame(t *testing.T) {
	umaWarnEnv(t)
	base, baseErr, err := runAddSplit(t, "sms-main", nativeTwilioSMSArgs...)
	require.NoError(t, err)

	umaWarnEnv(t)
	again, againErr, err := runAddSplit(t, "sms-main", nativeTwilioSMSArgs...)
	require.NoError(t, err)

	// The data dir differs per run, so compare with the saved path masked.
	mask := func(s string) string {
		var keep []string
		for _, line := range strings.Split(s, "\n") {
			if !strings.HasPrefix(line, "saved: ") {
				keep = append(keep, line)
			}
		}
		return strings.Join(keep, "\n")
	}
	assert.Equal(t, mask(base), mask(again))
	assert.Equal(t, baseErr, againErr)
	assert.Empty(t, againErr)
}

func TestUmaFlagWarning_MachineReadableModesAreSilent(t *testing.T) {
	for _, format := range []string{"json", "yaml", "JSON"} {
		t.Run(format, func(t *testing.T) {
			umaWarnEnv(t)
			setGlobal(t, "format", format)
			stdout, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--template-name", "hello")...)
			require.NoError(t, err, stdout+stderr)
			assert.Empty(t, stderr)
			assert.NotContains(t, stdout, "has no effect")
			exists, existsErr := core.ServiceExists("svc-uma")
			require.NoError(t, existsErr)
			assert.True(t, exists, "silencing the warning does not change what is saved")
		})
	}
}

func TestUmaFlagWarning_TableFormatStillWarns(t *testing.T) {
	umaWarnEnv(t)
	setGlobal(t, "format", "table")
	_, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--template-name", "hello")...)
	require.NoError(t, err)
	assert.Len(t, warningLines(stderr), 1)
}

func TestUmaFlagWarning_QuietIsSilent(t *testing.T) {
	umaWarnEnv(t)
	setGlobal(t, "quiet", true)
	stdout, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--template-name", "hello")...)
	require.NoError(t, err, stdout+stderr)
	assert.Empty(t, stderr)
	exists, existsErr := core.ServiceExists("svc-uma")
	require.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestUmaFlagWarning_DryRunWarnsAndSavesNothing(t *testing.T) {
	umaWarnEnv(t)
	stdout, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--dry-run", "--template-name", "hello")...)
	require.NoError(t, err, stdout+stderr)
	assert.Len(t, warningLines(stderr), 1)
	assert.Contains(t, stdout, "dry_run: true")
	exists, existsErr := core.ServiceExists("svc-uma")
	require.NoError(t, existsErr)
	assert.False(t, exists)
}

func TestUmaFlagWarning_ForceOverwriteWarnsAndKeepsExitCodeZero(t *testing.T) {
	umaWarnEnv(t)
	_, _, err := runAddSplit(t, "svc-uma", umaTwilioArgs...)
	require.NoError(t, err)
	stdout, stderr, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--force", "--language-code", "en")...)
	require.NoError(t, err, stdout+stderr)
	assert.Len(t, warningLines(stderr), 1)
	assert.Contains(t, stdout, "saved: ")
}

func TestUmaFlagWarning_InvalidServiceStillFailsAsBefore(t *testing.T) {
	umaWarnEnv(t)
	// Missing --webhook-url: invalid, refused, exactly as without the flag.
	_, stderr, err := runAddSplit(t, "svc-uma",
		"--type", "uma", "--uma-adapter", "twilio.sms", "--profile", "assistant", "--from", "+15558675310",
		"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
		"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
		"--default-action", "triage", "--template-name", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	assert.Len(t, warningLines(stderr), 1, "the warning is about the flags, valid or not")
	exists, existsErr := core.ServiceExists("svc-uma")
	require.NoError(t, existsErr)
	assert.False(t, exists)
}

func TestUmaFlagWarning_HistoryNoteIsUnchanged(t *testing.T) {
	umaWarnEnv(t)
	stdout, _, err := runAddSplit(t, "svc-uma", with(umaTwilioArgs, "--template-name", "hello")...)
	require.NoError(t, err)
	assert.Contains(t, stdout, "note: conversation history is keyed by the service id")
}
