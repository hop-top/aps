package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/infracloudio/msbotbuilder-go/schema"

	coremessenger "hop.top/aps/internal/core/messenger"
)

type fakeTeamsTransport struct {
	gotURL      url.URL
	gotActivity schema.Activity
	err         error
	calls       int
}

func (f *fakeTeamsTransport) PostActivity(_ context.Context, target url.URL, activity schema.Activity) error {
	f.calls++
	f.gotURL = target
	f.gotActivity = activity
	return f.err
}

func newTestTeamsProvider(transport TeamsTransport) *TeamsProvider {
	return NewTeamsProvider(TeamsProviderConfig{
		AppID:       "bot-app-id",
		AppPassword: "bot-secret",
		Transport:   transport,
		Now:         func() time.Time { return time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC) },
	})
}

func TestTeamsProvider_Metadata(t *testing.T) {
	meta := newTestTeamsProvider(&fakeTeamsTransport{}).Metadata()
	if meta.Provider != "teams" {
		t.Errorf("provider = %q, want teams", meta.Provider)
	}
	if len(meta.IngressModes) != 1 || meta.IngressModes[0] != coremessenger.IngressModeWebhook {
		t.Errorf("ingress modes = %v, want webhook", meta.IngressModes)
	}
}

func TestTeamsProvider_NormalizeIngress(t *testing.T) {
	provider := newTestTeamsProvider(&fakeTeamsTransport{})

	body := []byte(`{
		"type": "message",
		"id": "1485983408511",
		"serviceUrl": "https://smba.trafficmanager.net/amer/",
		"from": {"id": "29:1abcdef", "name": "Megan Bowen"},
		"conversation": {"conversationType": "personal", "id": "a:1conversation"},
		"recipient": {"id": "28:bot-app-id"},
		"text": "Hello bot"
	}`)

	msg, err := provider.NormalizeIngress(context.Background(), coremessenger.NativeIngress{
		ServiceID: "teams-support",
		Provider:  "teams",
		Mode:      coremessenger.IngressModeWebhook,
		Body:      body,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Platform != "teams" {
		t.Errorf("platform = %q", msg.Platform)
	}
	if msg.PlatformMetadata["service_id"] != "teams-support" {
		t.Errorf("service_id = %v, want stamped", msg.PlatformMetadata["service_id"])
	}
	if msg.PlatformMetadata["messenger_name"] != "teams-support" {
		t.Errorf("messenger_name = %v, want stamped", msg.PlatformMetadata["messenger_name"])
	}
}

func TestTeamsProvider_NormalizeIngress_InvalidJSON(t *testing.T) {
	provider := newTestTeamsProvider(&fakeTeamsTransport{})
	_, err := provider.NormalizeIngress(context.Background(), coremessenger.NativeIngress{
		ServiceID: "teams-support",
		Provider:  "teams",
		Mode:      coremessenger.IngressModeWebhook,
		Body:      []byte("not-json"),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON body")
	}
}

func TestTeamsProvider_DeliverMessage_Reply(t *testing.T) {
	transport := &fakeTeamsTransport{}
	provider := newTestTeamsProvider(transport)

	receipt, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "teams",
		ServiceID: "teams-support",
		ChannelID: "a:1conversation",
		Text:      "reply text",
		Metadata: map[string]any{
			"teams_service_url":     "https://smba.trafficmanager.net/amer/",
			"teams_conversation_id": "a:1conversation",
			"teams_activity_id":     "1485983408511",
			"teams_recipient_id":    "28:bot-app-id",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantURL := "https://smba.trafficmanager.net/amer/v3/conversations/a:1conversation/activities/1485983408511"
	if transport.gotURL.String() != wantURL {
		t.Errorf("url = %q, want %q", transport.gotURL.String(), wantURL)
	}
	if transport.gotActivity.Type != schema.Message {
		t.Errorf("activity type = %q, want message", transport.gotActivity.Type)
	}
	if transport.gotActivity.Text != "reply text" {
		t.Errorf("activity text = %q", transport.gotActivity.Text)
	}
	if transport.gotActivity.Conversation.ID != "a:1conversation" {
		t.Errorf("conversation = %q", transport.gotActivity.Conversation.ID)
	}
	if transport.gotActivity.ReplyToID != "1485983408511" {
		t.Errorf("replyToId = %q", transport.gotActivity.ReplyToID)
	}
	if transport.gotActivity.From.ID != "28:bot-app-id" {
		t.Errorf("from = %q, want bot id", transport.gotActivity.From.ID)
	}
	if receipt.Status != "success" {
		t.Errorf("receipt status = %q", receipt.Status)
	}
	if receipt.Provider != "teams" {
		t.Errorf("receipt provider = %q", receipt.Provider)
	}
}

func TestTeamsProvider_DeliverMessage_NoReplyActivity(t *testing.T) {
	transport := &fakeTeamsTransport{}
	provider := newTestTeamsProvider(transport)

	_, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "teams",
		ServiceID: "teams-support",
		ChannelID: "19:group-chat",
		Text:      "hello",
		Metadata: map[string]any{
			"teams_service_url": "https://smba.trafficmanager.net/amer",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantURL := "https://smba.trafficmanager.net/amer/v3/conversations/19:group-chat/activities"
	if transport.gotURL.String() != wantURL {
		t.Errorf("url = %q, want %q (sendToConversation)", transport.gotURL.String(), wantURL)
	}
	if transport.gotActivity.ReplyToID != "" {
		t.Errorf("replyToId = %q, want empty", transport.gotActivity.ReplyToID)
	}
}

func TestTeamsProvider_DeliverMessage_MissingServiceURL(t *testing.T) {
	provider := newTestTeamsProvider(&fakeTeamsTransport{})
	_, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "teams",
		ServiceID: "teams-support",
		ChannelID: "a:1conversation",
		Text:      "hi",
	})
	if err == nil {
		t.Fatal("expected error without teams_service_url metadata")
	}
}

func TestTeamsProvider_DeliverMessage_MissingCredentials(t *testing.T) {
	provider := NewTeamsProvider(TeamsProviderConfig{Transport: &fakeTeamsTransport{}})
	_, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "teams",
		ServiceID: "teams-support",
		ChannelID: "a:1conversation",
		Text:      "hi",
		Metadata:  map[string]any{"teams_service_url": "https://smba.trafficmanager.net/amer"},
	})
	if err == nil {
		t.Fatal("expected missing credential error")
	}
}

func TestTeamsTokenURL(t *testing.T) {
	if got := teamsTokenURL("", ""); got != "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token" {
		t.Errorf("default token URL = %q", got)
	}
	if got := teamsTokenURL("tenant-guid", ""); got != "https://login.microsoftonline.com/tenant-guid/oauth2/v2.0/token" {
		t.Errorf("tenant token URL = %q", got)
	}
	if got := teamsTokenURL("tenant-guid", "https://example.com/token"); got != "https://example.com/token" {
		t.Errorf("override token URL = %q", got)
	}
}

func TestTeamsSDKTransport_PostActivity(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		if got := r.PostForm.Get("client_id"); got != "bot-app-id" {
			t.Errorf("client_id = %q", got)
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()

	var gotAuth string
	var gotPath string
	var gotBody schema.Activity
	replyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer replyServer.Close()

	transport, err := NewTeamsSDKTransport("bot-app-id", "bot-secret", tokenServer.URL)
	if err != nil {
		t.Fatalf("transport init: %v", err)
	}

	target, _ := url.Parse(replyServer.URL + "/v3/conversations/a:1/activities/42")
	err = transport.PostActivity(context.Background(), *target, schema.Activity{
		Type: schema.Message,
		Text: "hello",
	})
	if err != nil {
		t.Fatalf("post activity: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("authorization = %q", gotAuth)
	}
	if gotPath != "/v3/conversations/a:1/activities/42" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody.Text != "hello" {
		t.Errorf("posted text = %q", gotBody.Text)
	}
}
