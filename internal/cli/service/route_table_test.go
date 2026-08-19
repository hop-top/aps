package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func writeRouteTableFixture(t *testing.T, dir string) (routesPath, contactsPath string) {
	t.Helper()
	routesPath = filepath.Join(dir, "acme-routes.yaml")
	contactsPath = filepath.Join(dir, "acme-contacts.yaml")
	require.NoError(t, os.WriteFile(routesPath, []byte(`
routes:
  - match: org:acme
    profile: acme
    action: inbox
  - match: "+1555*"
    action: sales
  - match: unknown
    profile: triage
    action: triage
`), 0o600))
	require.NoError(t, os.WriteFile(contactsPath, []byte(`
contacts:
  - id: jane
    org: acme
    keys: ["whatsapp:+15551234567"]
`), 0o600))
	return routesPath, contactsPath
}

func TestAddCmd_RouteTablePersistsRoutingBlock(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	routesPath, contactsPath := writeRouteTableFixture(t, t.TempDir())

	add := newTestServiceCmd()
	var addOut bytes.Buffer
	add.SetOut(&addOut)
	add.SetErr(&addOut)
	add.SetArgs([]string{
		"add", "support-line",
		"--type", "whatsapp",
		"--profile", "assistant",
		"--provider", "twilio",
		"--from", "whatsapp:+15550100002",
		"--env", "TWILIO_ACCOUNT_SID=secret:sid",
		"--env", "TWILIO_AUTH_TOKEN=secret:token",
		"--route-table", routesPath,
		"--contacts", contactsPath,
	})
	require.NoError(t, add.Execute())
	assert.Contains(t, addOut.String(), "config_valid: true", addOut.String())
	assert.NotContains(t, addOut.String(), "requires option default_action")

	service, err := core.LoadService("support-line")
	require.NoError(t, err)
	require.NotNil(t, service.Routing)
	assert.Equal(t, routesPath, service.Routing.File)
	require.NotNil(t, service.Routing.Contacts)
	assert.Equal(t, contactsPath, service.Routing.Contacts.Path)
	_, hasDefault := service.Options["default_action"]
	assert.False(t, hasDefault)

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "support-line"})
	require.NoError(t, show.Execute())
	out := showOut.String()
	assert.Contains(t, out, "routing:")
	assert.Contains(t, out, "  file: "+routesPath)
	assert.Contains(t, out, "  contacts: "+contactsPath+" (1 entries)")
	assert.Contains(t, out, "  routes:")
	assert.Contains(t, out, "    - org:acme -> acme=inbox")
	assert.Contains(t, out, "    - +1555* -> assistant=sales")
	assert.Contains(t, out, "    - unknown -> triage=triage (terminal)")
	assert.Contains(t, out, "executes: sender route table -> profile action")
}

func TestAddCmd_RouteTableWithDefaultActionReportsIssue(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	routesPath, _ := writeRouteTableFixture(t, t.TempDir())

	add := newTestServiceCmd()
	var out bytes.Buffer
	add.SetOut(&out)
	add.SetErr(&out)
	add.SetArgs([]string{
		"add", "support-line",
		"--type", "sms",
		"--profile", "assistant",
		"--provider", "twilio",
		"--from", "+15550100002",
		"--route-table", routesPath,
		"--default-action", "reply",
		"--dry-run",
	})
	require.NoError(t, add.Execute())
	assert.Contains(t, out.String(), "config_valid: false")
	assert.Contains(t, out.String(), "config_issue: message service declares both routing and option default_action")
	assert.Contains(t, out.String(), "config_issue: routing: route 0: match \"org:acme\" requires a contacts source")
}

func TestAddCmd_ContactsWithoutRouteTableIsRejected(t *testing.T) {
	_, contactsPath := writeRouteTableFixture(t, t.TempDir())
	add := newTestServiceCmd()
	add.SetArgs([]string{
		"add", "support-line",
		"--type", "sms",
		"--profile", "assistant",
		"--contacts", contactsPath,
		"--dry-run",
	})
	err := add.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--contacts requires --route-table")
}

func TestServiceShow_InlineRoutingSummary(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "inline-line",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Options: map[string]string{"provider": "twilio", "from": "+15550100002"},
		Routing: routingInline(),
	}))

	show := newTestServiceCmd()
	var out bytes.Buffer
	show.SetOut(&out)
	show.SetErr(&out)
	show.SetArgs([]string{"show", "inline-line"})
	require.NoError(t, show.Execute())
	assert.Contains(t, out.String(), "  file: inline")
	assert.Contains(t, out.String(), "    - +15551234567 -> acme=inbox")
	assert.Contains(t, out.String(), "    - unknown -> assistant=triage (terminal)")
}

func TestServiceShow_BrokenRoutingReportsError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "broken-line",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Routing: routingWithoutTerminal(),
	}))

	show := newTestServiceCmd()
	var out bytes.Buffer
	show.SetOut(&out)
	show.SetErr(&out)
	show.SetArgs([]string{"show", "broken-line"})
	require.NoError(t, show.Execute())
	assert.Contains(t, out.String(), "routing_error: ")
	assert.Contains(t, out.String(), "terminal")
}
