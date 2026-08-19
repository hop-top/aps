package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core/msgroute"
)

func routedSMSService(routing *msgroute.Config, options map[string]string) *ServiceConfig {
	if options == nil {
		options = map[string]string{}
	}
	options["provider"] = "twilio"
	options["from"] = "+15550100002"
	options["receive"] = "webhook"
	options["reply"] = "text"
	return &ServiceConfig{
		ID:      "support-line",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_ACCOUNT_SID": "secret:sid",
			"TWILIO_AUTH_TOKEN":  "secret:token",
		},
		Options: options,
		Routing: routing,
	}
}

func TestValidateServiceConfig_RoutingReplacesDefaultAction(t *testing.T) {
	result := ValidateServiceConfig(routedSMSService(&msgroute.Config{
		Routes: []msgroute.Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}, nil))
	assert.True(t, result.Valid, "issues: %v", result.Issues)
	assert.NotContains(t, result.Issues, "message service requires option default_action to dispatch inbound messages")
}

func TestValidateServiceConfig_RoutingAndDefaultActionIsAmbiguous(t *testing.T) {
	result := ValidateServiceConfig(routedSMSService(&msgroute.Config{
		Routes: []msgroute.Route{
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}, map[string]string{"default_action": "reply"}))
	assert.False(t, result.Valid)
	assert.Contains(t, result.Issues, "message service declares both routing and option default_action; remove default_action (routing wins at runtime)")
}

func TestValidateServiceConfig_RoutingWithoutTerminalIsRejected(t *testing.T) {
	result := ValidateServiceConfig(routedSMSService(&msgroute.Config{
		Routes: []msgroute.Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
		},
	}, nil))
	assert.False(t, result.Valid)
	assert.Contains(t, result.Issues, "routing: "+msgroute.ErrMissingTerminalRoute.Error())
}

func TestValidateServiceConfig_RoutingReportsEveryProblem(t *testing.T) {
	result := ValidateServiceConfig(routedSMSService(&msgroute.Config{
		Routes: []msgroute.Route{
			{Profile: "acme", Action: "inbox"},
			{Match: "+15551234567", Profile: "acme"},
		},
	}, nil))
	assert.False(t, result.Valid)
	assert.Contains(t, result.Issues, "routing: route 0: match is required")
	assert.Contains(t, result.Issues, "routing: route 1: action is required")
}

func TestLoadServiceRouteTable_ResolvesFileRelativeToServicesDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	servicesDir, err := GetServicesDir()
	require.NoError(t, err)
	tablePath := filepath.Join(servicesDir, "routes", "support-line.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(tablePath), 0o750))
	require.NoError(t, os.WriteFile(tablePath, []byte(`
contacts:
  entries:
    - id: jane
      org: acme
      keys: ["whatsapp:+15551234567"]
routes:
  - match: org:acme
    action: inbox
  - match: unknown
    profile: triage
    action: triage
`), 0o600))

	service := routedSMSService(&msgroute.Config{File: "routes/support-line.yaml"}, nil)
	table, err := LoadServiceRouteTable(service)
	require.NoError(t, err)
	assert.Equal(t, tablePath, table.Source())
	assert.Equal(t, "sms", table.Platform())

	routes := table.Routes()
	require.Len(t, routes, 2)
	assert.Equal(t, "assistant", routes[0].Profile, "service profile is the default route profile")

	decision := table.Resolve("+1 (555) 123-4567")
	assert.Equal(t, "assistant=inbox", decision.Mapping())
	require.NotNil(t, decision.Contact)
	assert.Equal(t, "jane", decision.Contact.ID)
}

func TestLoadServiceRouteTable_NoRouting(t *testing.T) {
	table, err := LoadServiceRouteTable(routedSMSService(nil, map[string]string{"default_action": "reply"}))
	require.Error(t, err)
	assert.Nil(t, table)
	assert.True(t, IsServiceNotRouted(err))
}

func TestSaveLoadService_RoutingRoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := routedSMSService(&msgroute.Config{
		Contacts: &msgroute.ContactsConfig{Path: "contacts.yaml"},
		Routes: []msgroute.Route{
			{Match: "org:acme", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}, nil)
	require.NoError(t, SaveService(service))

	loaded, err := LoadService(service.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Routing)
	assert.Equal(t, service.Routing, loaded.Routing)
}

func TestDescribeServiceRuntime_RoutedMessageService(t *testing.T) {
	got := DescribeServiceRuntime(routedSMSService(&msgroute.Config{
		Routes: []msgroute.Route{{Match: "unknown", Profile: "triage", Action: "triage"}},
	}, nil))
	assert.Equal(t, "sender route table -> profile action", got.Executes)
	assert.Equal(t, "sender route table", got.Metadata.Routing)

	plain := DescribeServiceRuntime(routedSMSService(nil, map[string]string{"default_action": "reply"}))
	assert.Equal(t, "normalized message execution handoff", plain.Executes)
	assert.Equal(t, "default_action", plain.Metadata.Routing)
}
