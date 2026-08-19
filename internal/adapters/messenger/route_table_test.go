package messenger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/msgroute"
)

func saveRoutedWhatsAppService(t *testing.T, routing *msgroute.Config, options map[string]string) {
	t.Helper()
	if options == nil {
		options = map[string]string{}
	}
	options["provider"] = "twilio"
	options["from"] = "whatsapp:+15550100002"
	service := &core.ServiceConfig{
		ID:      "support-line",
		Type:    "message",
		Adapter: "whatsapp",
		Profile: "assistant",
		Options: options,
		Routing: routing,
	}
	require.NoError(t, core.SaveService(service))
}

func acmeRouting() *msgroute.Config {
	return &msgroute.Config{
		Contacts: &msgroute.ContactsConfig{Entries: []msgroute.Contact{
			{ID: "jane", Name: "Jane Doe", Org: "acme", Keys: []string{"whatsapp:+15551234567"}},
		}},
		Routes: []msgroute.Route{
			{Match: "org:acme", Profile: "acme", Action: "inbox"},
			{Match: "+1555*", Action: "sales"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}
}

func whatsappMessage(sender string) *coremessenger.NormalizedMessage {
	return &coremessenger.NormalizedMessage{
		ID:       "msg_route_table",
		Platform: "whatsapp",
		Sender:   coremessenger.Sender{ID: sender},
		Channel:  coremessenger.Channel{ID: "+15550100002", Type: coremessenger.ChannelTypeDirect},
		Text:     "hello",
		PlatformMetadata: map[string]any{
			"messenger_name": "support-line",
		},
	}
}

func emptyBaseResolver() *mockResolver {
	return &mockResolver{
		links:   map[string]*coremessenger.ProfileMessengerLink{},
		actions: map[string]string{},
	}
}

func TestServiceRouteResolver_RouteTable_SenderDispatch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	tests := []struct {
		name        string
		sender      string
		wantMapping string
		wantProfile string
		wantTerm    bool
		wantContact string
	}{
		{name: "twilio prefixed contact -> org route", sender: "whatsapp:+15551234567", wantMapping: "acme=inbox", wantProfile: "acme", wantContact: "jane"},
		{name: "cloud wa_id same contact", sender: "15551234567", wantMapping: "acme=inbox", wantProfile: "acme", wantContact: "jane"},
		{name: "glob route inherits service profile", sender: "whatsapp:+15559990000", wantMapping: "assistant=sales", wantProfile: "assistant"},
		{name: "unknown sender -> terminal triage", sender: "whatsapp:+16005550000", wantMapping: "triage=triage", wantProfile: "triage", wantTerm: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := whatsappMessage(tt.sender)
			link, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", msg)
			require.NoError(t, err)
			require.NotNil(t, link)
			assert.Equal(t, tt.wantMapping, mapping)
			assert.Equal(t, tt.wantProfile, link.ProfileID)
			assert.Equal(t, "support-line", link.MessengerName)
			assert.True(t, link.Enabled)

			routing, ok := msg.PlatformMetadata["routing"].(map[string]any)
			require.True(t, ok, "routing decision must be stamped on platform_metadata")
			assert.Equal(t, tt.wantTerm, routing["terminal"])
			assert.Equal(t, tt.wantProfile, routing["profile"])
			if tt.wantContact != "" {
				contact, ok := routing["contact"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, tt.wantContact, contact["id"])
			} else {
				_, hasContact := routing["contact"]
				assert.False(t, hasContact)
			}
		})
	}
}

func TestServiceRouteResolver_RouteTable_ChannelLinkMappingWins(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	base := emptyBaseResolver()
	base.links["support-line:+15550100002"] = &coremessenger.ProfileMessengerLink{ProfileID: "linked", MessengerName: "support-line", Enabled: true}
	base.actions["support-line:+15550100002"] = "linked=explicit"

	resolver := &serviceRouteResolver{base: base}
	msg := whatsappMessage("whatsapp:+15551234567")
	link, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", msg)
	require.NoError(t, err)
	assert.Equal(t, "linked=explicit", mapping)
	assert.Equal(t, "linked", link.ProfileID)
	_, stamped := msg.PlatformMetadata["routing"]
	assert.False(t, stamped, "explicit link mappings bypass the route table")
}

func TestServiceRouteResolver_RouteTable_WinsOverDefaultAction(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), map[string]string{"default_action": "reply"})
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	_, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", whatsappMessage("whatsapp:+16005550000"))
	require.NoError(t, err)
	assert.Equal(t, "triage=triage", mapping, "routing block takes precedence over default_action")
}

func TestServiceRouteResolver_RouteTable_LoadErrorFailsClosed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, &msgroute.Config{File: "missing-routes.yaml"}, map[string]string{"default_action": "reply"})
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	link, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", whatsappMessage("whatsapp:+15551234567"))
	require.Error(t, err)
	assert.Nil(t, link)
	assert.Empty(t, mapping)
	assert.False(t, coremessenger.IsUnknownChannel(err), "a broken route table is a routing failure, not an unknown channel")
	assert.Contains(t, err.Error(), "route table")
}

func TestServiceRouteResolver_RouteTable_ExternalFile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	servicesDir, err := core.GetServicesDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(servicesDir, "routes"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "routes", "support-line.yaml"), []byte(`
routes:
  - match: "+15551234567"
    profile: acme
    action: inbox
  - match: unknown
    profile: triage
    action: triage
`), 0o600))
	saveRoutedWhatsAppService(t, &msgroute.Config{File: "routes/support-line.yaml"}, nil)
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	_, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", whatsappMessage("whatsapp:+1 (555) 123-4567"))
	require.NoError(t, err)
	assert.Equal(t, "acme=inbox", mapping)
}

func TestServiceRouteResolver_ChannelRouteStillFallsBackToDefaultAction(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, nil, map[string]string{"default_action": "reply"})
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	link, mapping, err := resolver.ResolveRouteForMessage(context.Background(), "support-line", whatsappMessage("whatsapp:+15551234567"))
	require.NoError(t, err)
	assert.Equal(t, "assistant=reply", mapping)
	assert.Equal(t, "assistant", link.ProfileID)
}

func TestMessageRouter_Route_UsesSenderAwareResolver(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	router := NewMessageRouterWithExecutor(&serviceRouteResolver{base: emptyBaseResolver()}, NewNormalizer(), &fakeActionExecutor{})

	msg := whatsappMessage("whatsapp:+15551234567")
	result, err := router.Route(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "routed", result.Status)
	assert.Equal(t, "acme", result.ProfileID)
	assert.Equal(t, "inbox", result.ActionName)
	assert.Equal(t, "acme=inbox", result.Route)
	assert.Equal(t, "acme", msg.ProfileID, "message is stamped with the resolved profile")
}

func TestMessageRouter_HandleMessage_RouteTableDispatchesResolvedProfile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	executor := &fakeActionExecutor{}
	router := NewMessageRouterWithExecutor(&serviceRouteResolver{base: emptyBaseResolver()}, NewNormalizer(), executor)

	result, err := router.HandleMessage(context.Background(), whatsappMessage("whatsapp:+15551234567"))
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "acme", executor.input.ProfileID, "action runs under the resolved profile, not the service profile")
	assert.Equal(t, "inbox", executor.input.ActionID)

	result, err = router.HandleMessage(context.Background(), whatsappMessage("whatsapp:+16005550000"))
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "triage", executor.input.ProfileID, "unknown sender lands on the terminal profile")
	assert.Equal(t, "triage", executor.input.ActionID)
}

func TestMessageRouter_ResolveMessageRoute_RouteTable(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	router := NewMessageRouterWithExecutor(&serviceRouteResolver{base: emptyBaseResolver()}, NewNormalizer(), &fakeActionExecutor{})

	route, err := router.ResolveMessageRoute(context.Background(), whatsappMessage("15551234567"))
	require.NoError(t, err)
	assert.Equal(t, coremessenger.ExecutionRoute{ProfileID: "acme", ActionName: "inbox", Mapping: "acme=inbox"}, route)
}

func postSignedSMSWebhookFrom(t *testing.T, handler *Handler, from, message string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("AccountSid", "AC123")
	form.Set("From", from)
	form.Set("To", "+15550100002")
	form.Set("Body", message)
	const webhookURL = "https://hooks.example.test/services/sms-alerts/webhook"
	req := httptest.NewRequest(http.MethodPost, webhookURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(coremessenger.TwilioSignatureHeader, coremessenger.TwilioSignature("twilio-token", webhookURL, form))
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "sms-alerts", "sms")
	return rec
}

func TestHandler_ServiceWebhookRouteTableDispatchesBySender(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env:     map[string]string{"TWILIO_AUTH_TOKEN": "twilio-token"},
		Options: map[string]string{"provider": "twilio", "from": "+15550100002"},
		Routing: &msgroute.Config{
			Contacts: &msgroute.ContactsConfig{Entries: []msgroute.Contact{
				{ID: "jane", Org: "acme", Keys: []string{"+1 (555) 010-0001"}},
			}},
			Routes: []msgroute.Route{
				{Match: "org:acme", Profile: "acme", Action: "inbox"},
				{Match: "unknown", Profile: "triage", Action: "triage"},
			},
		},
	}))

	executor := &fakeActionExecutor{output: "ok"}
	handler := newServiceTestHandler(executor)

	rec := postSignedSMSWebhookFrom(t, handler, "+15550100001", "hi from jane")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "acme", executor.input.ProfileID, "known contact runs under its org profile")
	assert.Equal(t, "inbox", executor.input.ActionID)
	assert.Contains(t, string(executor.input.Payload), `"sender_key":"+15550100001"`, "routing decision travels with the action payload")
	assert.Contains(t, string(executor.input.Payload), `"terminal":false`)

	rec = postSignedSMSWebhookFrom(t, handler, "+16005550000", "who dis")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "triage", executor.input.ProfileID, "unknown sender lands on the terminal profile")
	assert.Equal(t, "triage", executor.input.ActionID)
	assert.Contains(t, string(executor.input.Payload), `"terminal":true`)
}

func TestServiceRouteResolver_ChannelOnlyResolutionRejectsRouteTableServices(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveRoutedWhatsAppService(t, acmeRouting(), nil)
	resolver := &serviceRouteResolver{base: emptyBaseResolver()}

	link, mapping, err := resolver.ResolveChannelRoute("support-line", "+15550100002")
	require.Error(t, err)
	assert.Nil(t, link)
	assert.Empty(t, mapping)
	assert.False(t, coremessenger.IsUnknownChannel(err))
	assert.Contains(t, err.Error(), "needs the message")
}
