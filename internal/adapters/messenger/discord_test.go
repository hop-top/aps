package messenger

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	msgtypes "hop.top/aps/internal/core/messenger"
)

func TestDiscordProviderNormalizeIngress_MessageCreate(t *testing.T) {
	provider := NewDiscordProvider(DiscordProviderConfig{})

	msg, err := provider.NormalizeIngress(context.Background(), msgtypes.NativeIngress{
		ServiceID: "discord-support",
		Provider:  "discord",
		Mode:      msgtypes.IngressModeStream,
		Body: []byte(`{
			"op": 0,
			"t": "MESSAGE_CREATE",
			"d": {
				"id": "1100000000000000001",
				"guild_id": "1300000000000000003",
				"channel_id": "1200000000000000002",
				"content": "hello from gateway",
				"author": {"id": "1400000000000000004", "username": "alice"}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("NormalizeIngress: %v", err)
	}
	if msg.Platform != "discord" || msg.WorkspaceID != "1300000000000000003" || msg.Channel.ID != "1200000000000000002" {
		t.Fatalf("normalized message = %#v", msg)
	}
	if msg.PlatformMetadata["service_id"] != "discord-support" {
		t.Fatalf("service metadata = %#v", msg.PlatformMetadata)
	}
}

func TestDiscordProviderDeliverMessage_ChannelReplyWithAttachment(t *testing.T) {
	transport := &captureDiscordTransport{
		statusCode: http.StatusOK,
		body:       `{"id":"2200000000000000001","channel_id":"1200000000000000002"}`,
	}
	provider := NewDiscordProvider(DiscordProviderConfig{
		BotToken:   "test-token",
		APIBaseURL: "https://discord.test/api/v10",
		Client:     transport,
	})

	receipt, err := provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
		Provider:  "discord",
		ServiceID: "discord-support",
		ChannelID: "1200000000000000002",
		ThreadID:  "1100000000000000001",
		Text:      "ack",
		Attachments: []msgtypes.Attachment{{
			Type:     "image",
			URL:      "https://cdn.discordapp.com/file.png",
			MimeType: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("DeliverMessage: %v", err)
	}
	if receipt.DeliveryID != "2200000000000000001" || receipt.Status != "success" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if transport.method != http.MethodPost {
		t.Fatalf("method = %s, want POST", transport.method)
	}
	if transport.url != "https://discord.test/api/v10/channels/1200000000000000002/messages" {
		t.Fatalf("url = %s", transport.url)
	}
	if got := transport.authorization; got != "Bot test-token" {
		t.Fatalf("authorization = %q", got)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(transport.requestBody), &payload); err != nil {
		t.Fatalf("payload JSON: %v", err)
	}
	if payload["content"] != "ack" {
		t.Fatalf("content = %v", payload["content"])
	}
	ref := payload["message_reference"].(map[string]any)
	if ref["message_id"] != "1100000000000000001" {
		t.Fatalf("message_reference = %#v", ref)
	}
	embeds := payload["embeds"].([]any)
	image := embeds[0].(map[string]any)["image"].(map[string]any)
	if image["url"] != "https://cdn.discordapp.com/file.png" {
		t.Fatalf("image embed = %#v", image)
	}
}

func TestDiscordProviderDeliverMessage_ThreadChannelTarget(t *testing.T) {
	transport := &captureDiscordTransport{statusCode: http.StatusOK, body: `{"id":"m2","channel_id":"thread-1"}`}
	provider := NewDiscordProvider(DiscordProviderConfig{
		BotToken:   "Bot already-prefixed",
		APIBaseURL: "https://discord.test/api/v10",
		Client:     transport,
	})

	_, err := provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
		Provider:  "discord",
		ServiceID: "discord-support",
		ChannelID: "channel-1",
		ThreadID:  "thread-1",
		Text:      "thread ack",
		Metadata: map[string]any{
			"thread_type": msgtypes.ThreadTypeTopic,
		},
	})
	if err != nil {
		t.Fatalf("DeliverMessage: %v", err)
	}
	if transport.url != "https://discord.test/api/v10/channels/thread-1/messages" {
		t.Fatalf("url = %s", transport.url)
	}
	if strings.Contains(transport.requestBody, "message_reference") {
		t.Fatalf("thread channel delivery should not set message_reference: %s", transport.requestBody)
	}
	if transport.authorization != "Bot already-prefixed" {
		t.Fatalf("authorization = %q", transport.authorization)
	}
}

type captureDiscordTransport struct {
	method        string
	url           string
	authorization string
	requestBody   string
	statusCode    int
	body          string
}

func (t *captureDiscordTransport) Do(req *http.Request) (*http.Response, error) {
	t.method = req.Method
	t.url = req.URL.String()
	t.authorization = req.Header.Get("Authorization")
	body, _ := io.ReadAll(req.Body)
	t.requestBody = string(body)
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}
