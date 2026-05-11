package messenger

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xrr "hop.top/xrr"
	xhttp "hop.top/xrr/adapters/http"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

func TestFirstClassMessageProviders_FixtureReceiveAuthAllowListAndExecutionUseXRR(t *testing.T) {
	tests := []struct {
		name          string
		service       *core.ServiceConfig
		adapter       string
		fixture       string
		contentType   string
		headers       func([]byte) http.Header
		wantMessageID string
		wantRoute     string
		xrrDelivery   func(t *testing.T) []func(*Handler)
	}{
		{
			name:        "telegram",
			adapter:     "telegram",
			fixture:     "telegram_message.json",
			contentType: "application/json",
			service: messageService("telegram-support", "telegram", map[string]string{
				"default_action":       "assistant=handle_telegram",
				"allowed_chats":        "-1001234567890",
				"webhook_secret_token": "telegram-secret",
				"reply":                "text",
			}, map[string]string{"TELEGRAM_BOT_TOKEN": "secret:TELEGRAM_BOT_TOKEN"}),
			headers: func(_ []byte) http.Header {
				return http.Header{msgtypes.TelegramSecretTokenHeader: {"telegram-secret"}}
			},
			wantMessageID: "telegram:update:555",
			wantRoute:     "assistant=handle_telegram",
			xrrDelivery: func(t *testing.T) []func(*Handler) {
				t.Setenv("TELEGRAM_BOT_TOKEN", "bot-token")
				return []func(*Handler){WithTelegramTransport(&TelegramHTTPTransport{
					BaseURL:    "https://telegram.test",
					HTTPClient: newXRRHTTPDoer(t, "telegram-delivery", xrrTelegramSendMessageResponder(t), xrr.ModeRecord),
				})}
			},
		},
		{
			name:        "slack",
			adapter:     "slack",
			fixture:     "slack_event_callback.json",
			contentType: "application/json",
			service: messageService("slack-support", "slack", map[string]string{
				"default_action":      "assistant=handle_slack",
				"allowed_channels":    "C012CHAN",
				"require_bot_mention": "true",
				"bot_user_id":         "U012BOT",
				"dedup_ttl":           "24h",
				"reply":               "text",
			}, map[string]string{
				"SLACK_BOT_TOKEN":      "secret:SLACK_BOT_TOKEN",
				"SLACK_SIGNING_SECRET": "test-secret",
			}),
			headers: func(body []byte) http.Header {
				ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
				return http.Header{
					"X-Slack-Request-Timestamp": {ts},
					"X-Slack-Signature":         {"v0=" + signSlack("test-secret", ts, body)},
				}
			},
			wantMessageID: "msg-slack-1",
			wantRoute:     "assistant=handle_slack",
			xrrDelivery: func(t *testing.T) []func(*Handler) {
				t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
				return []func(*Handler){WithSlackTransport(&SlackHTTPTransport{
					BaseURL:    "https://slack.test/api",
					HTTPClient: newXRRHTTPDoer(t, "slack-delivery", xrrSlackPostMessageResponder(t, http.StatusOK, `{"ok":true,"channel":"C012CHAN","ts":"1710000000.000099"}`), xrr.ModeRecord),
				})}
			},
		},
		{
			name:        "discord",
			adapter:     "discord",
			fixture:     "discord_message_create.json",
			contentType: "application/json",
			service: messageService("discord-support", "discord", map[string]string{
				"default_action":   "assistant=handle_discord",
				"allowed_channels": "1200000000000000002",
				"allowed_guilds":   "1300000000000000003",
			}, nil),
			wantRoute: "",
		},
		{
			name:        "sms",
			adapter:     "sms",
			fixture:     "sms_twilio.form",
			contentType: "application/x-www-form-urlencoded",
			service: messageService("sms-alerts", "sms", map[string]string{
				"default_action":  "assistant=handle_sms",
				"provider":        "twilio",
				"from":            "+15550100002",
				"allowed_numbers": "+15550100001",
				"reply":           "text",
				"webhook_url":     "https://hooks.example.test/services/sms-alerts/webhook",
			}, map[string]string{"TWILIO_AUTH_TOKEN": "twilio-token"}),
			headers: func(body []byte) http.Header {
				form, err := url.ParseQuery(string(body))
				require.NoError(t, err)
				return http.Header{msgtypes.TwilioSignatureHeader: {msgtypes.TwilioSignature("twilio-token", "https://hooks.example.test/services/sms-alerts/webhook", form)}}
			},
			wantMessageID: "SM123",
			wantRoute:     "",
		},
		{
			name:        "whatsapp",
			adapter:     "whatsapp",
			fixture:     "whatsapp_cloud_message.json",
			contentType: "application/json",
			service: messageService("wa-support", "whatsapp", map[string]string{
				"default_action":  "assistant=handle_whatsapp",
				"provider":        "whatsapp-cloud",
				"phone_number_id": "123456789012345",
				"allowed_numbers": "+15551230001,15551230001",
				"reply":           "text",
			}, map[string]string{"WHATSAPP_ACCESS_TOKEN": "secret:WHATSAPP_ACCESS_TOKEN"}),
			wantMessageID: "wamid.HBgLMTU1NTEyMzAwMDE=",
			wantRoute:     "assistant=handle_whatsapp",
			xrrDelivery: func(t *testing.T) []func(*Handler) {
				t.Setenv("WHATSAPP_ACCESS_TOKEN", "wa-token")
				transport := msgtypes.NewWhatsAppCloudTransport("", "https://whatsapp.test", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return newXRRHTTPDoer(t, "whatsapp-delivery", xrrWhatsAppCloudResponder(t, http.StatusOK, `{"messages":[{"id":"wamid.delivery.1"}]}`), xrr.ModeRecord).Do(req)
					}),
				})
				return []func(*Handler){WithWhatsAppTransport(transport)}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			require.NoError(t, core.SaveService(tt.service))
			body := providerFixture(t, tt.fixture)
			executor := &fakeActionExecutor{output: "reply from action"}
			handler := newServiceTestHandler(executor, appendHandlerOptions(t, tt.xrrDelivery)...)
			headers := providerHeaders(tt.contentType, body, tt.headers)

			resp := serveWebhookThroughXRR(t, handler, tt.service.ID, tt.adapter, tt.contentType, headers, body)

			assert.Equal(t, http.StatusOK, resp.Status)
			if tt.wantMessageID != "" {
				assert.Contains(t, resp.Body, tt.wantMessageID)
			}
			if tt.wantRoute != "" {
				assert.Contains(t, resp.Body, tt.wantRoute)
			}
			assert.Equal(t, "assistant", executor.input.ProfileID)
			assert.Equal(t, strings.TrimPrefix(tt.service.Options["default_action"], "assistant="), executor.input.ActionID)
		})
	}
}

func TestFirstClassMessageProviders_AuthFailureAndAllowListRejectionUseXRR(t *testing.T) {
	tests := []struct {
		name        string
		service     *core.ServiceConfig
		adapter     string
		fixture     string
		contentType string
		authHeaders http.Header
		badHeaders  http.Header
	}{
		{
			name:        "telegram",
			adapter:     "telegram",
			fixture:     "telegram_message.json",
			contentType: "application/json",
			service: messageService("telegram-support", "telegram", map[string]string{
				"default_action":       "assistant=handle_telegram",
				"allowed_chats":        "-1001234567890",
				"webhook_secret_token": "telegram-secret",
			}, map[string]string{"TELEGRAM_BOT_TOKEN": "bot-token"}),
			authHeaders: http.Header{msgtypes.TelegramSecretTokenHeader: {"telegram-secret"}},
			badHeaders:  http.Header{msgtypes.TelegramSecretTokenHeader: {"wrong-secret"}},
		},
		{
			name:        "slack",
			adapter:     "slack",
			fixture:     "slack_event_callback.json",
			contentType: "application/json",
			service: messageService("slack-support", "slack", map[string]string{
				"default_action":   "assistant=handle_slack",
				"allowed_channels": "C012CHAN",
			}, map[string]string{
				"SLACK_BOT_TOKEN":      "xoxb-test",
				"SLACK_SIGNING_SECRET": "test-secret",
			}),
			badHeaders: http.Header{
				"X-Slack-Request-Timestamp": {strconv.FormatInt(time.Now().UTC().Unix(), 10)},
				"X-Slack-Signature":         {"v0=bad"},
			},
		},
		{
			name:        "discord",
			adapter:     "discord",
			fixture:     "discord_message_create.json",
			contentType: "application/json",
			service: messageService("discord-support", "discord", map[string]string{
				"default_action":   "assistant=handle_discord",
				"allowed_channels": "1200000000000000002",
				"allowed_guilds":   "1300000000000000003",
				"auth_scheme":      "bearer",
				"auth_token":       "discord-token",
			}, nil),
			authHeaders: http.Header{"Authorization": {"Bearer discord-token"}},
			badHeaders:  http.Header{"Authorization": {"Bearer wrong-token"}},
		},
		{
			name:        "sms",
			adapter:     "sms",
			fixture:     "sms_twilio.form",
			contentType: "application/x-www-form-urlencoded",
			service: messageService("sms-alerts", "sms", map[string]string{
				"default_action":  "assistant=handle_sms",
				"provider":        "twilio",
				"from":            "+15550100002",
				"allowed_numbers": "+15550100001",
				"webhook_url":     "https://hooks.example.test/services/sms-alerts/webhook",
			}, map[string]string{"TWILIO_AUTH_TOKEN": "twilio-token"}),
			badHeaders: http.Header{msgtypes.TwilioSignatureHeader: {"bad-signature"}},
		},
		{
			name:        "whatsapp",
			adapter:     "whatsapp",
			fixture:     "whatsapp_cloud_message.json",
			contentType: "application/json",
			service: messageService("wa-support", "whatsapp", map[string]string{
				"default_action":  "assistant=handle_whatsapp",
				"provider":        "whatsapp-cloud",
				"phone_number_id": "123456789012345",
				"allowed_numbers": "+15551230001,15551230001",
				"auth_scheme":     "bearer",
				"auth_token":      "wa-token",
			}, nil),
			authHeaders: http.Header{"Authorization": {"Bearer wa-token"}},
			badHeaders:  http.Header{"Authorization": {"Bearer wrong-token"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/auth_failure", func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			require.NoError(t, core.SaveService(tt.service))
			body := providerFixture(t, tt.fixture)
			executor := &fakeActionExecutor{}
			handler := newServiceTestHandler(executor)
			headers := providerHeaders(tt.contentType, body, func([]byte) http.Header { return tt.badHeaders })

			resp := serveWebhookThroughXRR(t, handler, tt.service.ID, tt.adapter, tt.contentType, headers, body)

			assert.Equal(t, http.StatusUnauthorized, resp.Status)
			assert.Empty(t, executor.input.ProfileID)
		})

		t.Run(tt.name+"/allow_list_rejection", func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			blocked := cloneService(tt.service)
			blocked.Options = cloneStringMap(blocked.Options)
			blocked.Options["allowed_channels"] = "blocked-channel"
			blocked.Options["allowed_guilds"] = "blocked-guild"
			blocked.Options["allowed_chats"] = "-1009999999999"
			blocked.Options["allowed_numbers"] = "+15559999999"
			require.NoError(t, core.SaveService(blocked))
			body := providerFixture(t, tt.fixture)
			executor := &fakeActionExecutor{}
			handler := newServiceTestHandler(executor)
			headers := providerHeaders(tt.contentType, body, func(body []byte) http.Header {
				if tt.adapter == "slack" {
					ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
					return http.Header{
						"X-Slack-Request-Timestamp": {ts},
						"X-Slack-Signature":         {"v0=" + signSlack("test-secret", ts, body)},
					}
				}
				if tt.adapter == "sms" {
					form, err := url.ParseQuery(string(body))
					require.NoError(t, err)
					return http.Header{msgtypes.TwilioSignatureHeader: {msgtypes.TwilioSignature("twilio-token", "https://hooks.example.test/services/sms-alerts/webhook", form)}}
				}
				return tt.authHeaders
			})

			resp := serveWebhookThroughXRR(t, handler, tt.service.ID, tt.adapter, tt.contentType, headers, body)

			assert.Equal(t, http.StatusForbidden, resp.Status)
			assert.Empty(t, executor.input.ProfileID)
		})
	}
}

func TestFirstClassProviderDelivery_UsesXRRHTTPMocks(t *testing.T) {
	tests := []struct {
		name       string
		deliver    func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error)
		assertions func(t *testing.T, receipt *msgtypes.DeliveryReceipt)
	}{
		{
			name: "telegram",
			deliver: func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error) {
				provider := NewTelegramProvider(TelegramProviderConfig{
					BotToken: "bot-token",
					Transport: &TelegramHTTPTransport{
						BaseURL:    "https://telegram.test",
						HTTPClient: newXRRHTTPDoerWithDir(t, cassetteDir, xrrTelegramSendMessageResponder(t), mode),
					},
					Now: fixedProviderNow,
				})
				return provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
					Provider:  "telegram",
					ServiceID: "telegram-support",
					ChannelID: "-1001234567890",
					Text:      "ack",
					Metadata: map[string]any{
						"reply_to_message_id": "77",
						"message_thread_id":   "12",
					},
				})
			},
			assertions: func(t *testing.T, receipt *msgtypes.DeliveryReceipt) {
				assert.Equal(t, "9001", receipt.DeliveryID)
				assert.Equal(t, "success", receipt.Status)
			},
		},
		{
			name: "slack",
			deliver: func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error) {
				provider := NewSlackProvider(SlackProviderConfig{
					BotToken: "xoxb-test",
					Transport: &SlackHTTPTransport{
						BaseURL:    "https://slack.test/api",
						HTTPClient: newXRRHTTPDoerWithDir(t, cassetteDir, xrrSlackPostMessageResponder(t, http.StatusOK, `{"ok":true,"channel":"C012CHAN","ts":"1710000000.000099"}`), mode),
					},
					Now: fixedProviderNow,
				})
				return provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
					Provider:  "slack",
					ServiceID: "slack-support",
					ChannelID: "C012CHAN",
					ThreadID:  "1710000000.000001",
					Text:      "ack",
				})
			},
			assertions: func(t *testing.T, receipt *msgtypes.DeliveryReceipt) {
				assert.Equal(t, "1710000000.000099", receipt.DeliveryID)
				assert.Equal(t, "success", receipt.Status)
			},
		},
		{
			name: "discord",
			deliver: func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error) {
				provider := NewDiscordProvider(DiscordProviderConfig{
					BotToken:   "test-token",
					APIBaseURL: "https://discord.test/api/v10",
					Client:     newXRRHTTPDoerWithDir(t, cassetteDir, xrrDiscordMessageResponder(t, http.StatusOK, `{"id":"2200000000000000001","channel_id":"1200000000000000002"}`), mode),
				})
				return provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
					Provider:  "discord",
					ServiceID: "discord-support",
					ChannelID: "1200000000000000002",
					ThreadID:  "1100000000000000001",
					Text:      "ack",
				})
			},
			assertions: func(t *testing.T, receipt *msgtypes.DeliveryReceipt) {
				assert.Equal(t, "2200000000000000001", receipt.DeliveryID)
				assert.Equal(t, "success", receipt.Status)
			},
		},
		{
			name: "sms",
			deliver: func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error) {
				transport := msgtypes.NewTwilioSMSTransport("AC123", "token", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return newXRRHTTPDoerWithDir(t, cassetteDir, xrrTwilioMessageResponder(t, http.StatusCreated, `{"sid":"SMDELIVERED","status":"queued"}`), mode).Do(req)
					}),
				})
				transport.BaseURL = "https://twilio.test"
				provider := msgtypes.NewSMSProvider(msgtypes.SMSProviderConfig{
					ServiceID:  "sms-alerts",
					Provider:   "twilio",
					AccountSID: "AC123",
					AuthToken:  "token",
					From:       "+15550100002",
				}, transport)
				provider.Now = fixedProviderNow
				return provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
					Provider:  "twilio",
					ServiceID: "sms-alerts",
					ChannelID: "+15550100001",
					Text:      "ack",
				})
			},
			assertions: func(t *testing.T, receipt *msgtypes.DeliveryReceipt) {
				assert.Equal(t, "SMDELIVERED", receipt.DeliveryID)
				assert.Equal(t, "queued", receipt.Status)
			},
		},
		{
			name: "whatsapp",
			deliver: func(t *testing.T, mode xrr.Mode, cassetteDir string) (*msgtypes.DeliveryReceipt, error) {
				transport := msgtypes.NewWhatsAppCloudTransport("", "https://whatsapp.test", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return newXRRHTTPDoerWithDir(t, cassetteDir, xrrWhatsAppCloudResponder(t, http.StatusOK, `{"messages":[{"id":"wamid.delivery.1"}]}`), mode).Do(req)
					}),
				})
				provider := msgtypes.NewWhatsAppProvider(msgtypes.WhatsAppProviderConfig{
					ServiceID:     "wa-support",
					Provider:      "whatsapp-cloud",
					AccessToken:   "wa-token",
					PhoneNumberID: "123456789012345",
				}, transport)
				provider.Now = fixedProviderNow
				return provider.DeliverMessage(context.Background(), msgtypes.DeliveryRequest{
					Provider:  "whatsapp",
					ServiceID: "wa-support",
					ChannelID: "15551230001",
					Text:      "ack",
				})
			},
			assertions: func(t *testing.T, receipt *msgtypes.DeliveryReceipt) {
				assert.Equal(t, "wamid.delivery.1", receipt.DeliveryID)
				assert.Equal(t, "sent", receipt.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			recorded, err := tt.deliver(t, xrr.ModeRecord, dir)
			require.NoError(t, err)
			tt.assertions(t, recorded)

			replayed, err := tt.deliver(t, xrr.ModeReplay, dir)
			require.NoError(t, err)
			tt.assertions(t, replayed)
		})
	}
}

func TestFirstClassProviderRuntime_RetryAndErrorHooksUseXRRDeliveryFailure(t *testing.T) {
	body := providerFixture(t, "slack_event_callback.json")
	cassetteDir := t.TempDir()
	provider := NewSlackProvider(SlackProviderConfig{
		BotToken: "xoxb-test",
		Transport: &SlackHTTPTransport{
			BaseURL:    "https://slack.test/api",
			HTTPClient: newXRRHTTPDoerWithDir(t, cassetteDir, xrrSlackPostMessageResponder(t, http.StatusTooManyRequests, `{"ok":false,"error":"rate_limited"}`), xrr.ModeRecord),
		},
		Now: fixedProviderNow,
	})
	router := &runtimeRouter{route: msgtypes.ExecutionRoute{ProfileID: "assistant", ActionName: "handle_slack"}}
	executor := &runtimeExecutor{reply: "ack"}
	var reported *msgtypes.RuntimeError
	var retry msgtypes.RetryDecision
	var attempts []msgtypes.DeliveryAttempt
	runtime, err := msgtypes.NewRuntime(provider, router, executor, msgtypes.RuntimeOptions{
		ServiceID: "slack-support",
		Now:       fixedProviderNow,
		Sleep:     func(context.Context, time.Duration) error { return nil },
		RetryPolicy: msgtypes.RetryPolicy{
			MaxAttempts: 1,
			BaseDelay:   time.Second,
			MaxDelay:    time.Second,
			Stages:      []msgtypes.RuntimeStage{msgtypes.RuntimeStageDeliver},
		},
		Hooks: msgtypes.RuntimeHooks{
			OnError: func(_ context.Context, err *msgtypes.RuntimeError) error {
				reported = err
				return nil
			},
			OnRetry: func(_ context.Context, decision msgtypes.RetryDecision) error {
				retry = decision
				return nil
			},
			OnDeliveryAttempt: func(_ context.Context, attempt msgtypes.DeliveryAttempt) error {
				attempts = append(attempts, attempt)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = runtime.HandleIngress(context.Background(), msgtypes.NativeIngress{
		ServiceID: "slack-support",
		Provider:  "slack",
		Mode:      msgtypes.IngressModeWebhook,
		Body:      body,
	})

	require.Error(t, err)
	require.NotNil(t, reported)
	assert.Equal(t, msgtypes.RuntimeStageDeliver, reported.Stage)
	assert.Equal(t, "slack-support", reported.Service)
	assert.Equal(t, "slack", reported.Provider)
	assert.False(t, reported.Decision.Retry)
	assert.False(t, retry.Retry)
	require.Len(t, attempts, 1)
	assert.Equal(t, "dead_letter", attempts[0].Status)
	assert.Contains(t, attempts[0].RedactedError, "rate_limited")
}

func TestWhatsAppSupport_CurrentFixtureNormalizeAndDenormalize(t *testing.T) {
	var raw map[string]any
	require.NoError(t, json.Unmarshal(providerFixture(t, "whatsapp_cloud_message.json"), &raw))
	normalizer := NewNormalizer()

	msg, err := normalizer.Normalize("whatsapp", raw)
	require.NoError(t, err)
	assert.Equal(t, "whatsapp", msg.Platform)
	assert.Equal(t, "wamid.HBgLMTU1NTEyMzAwMDE=", msg.ID)
	assert.Equal(t, "15551230001", msg.Sender.ID)
	assert.Equal(t, "123456789012345", msg.Channel.ID)

	response, err := normalizer.Denormalize("whatsapp", &ActionResult{Status: "success", Output: "reply from action"})
	require.NoError(t, err)
	text := response["text"].(map[string]any)
	assert.Equal(t, "reply from action", text["body"])
}

func messageService(id, adapter string, options map[string]string, env map[string]string) *core.ServiceConfig {
	return &core.ServiceConfig{
		ID:      id,
		Type:    "message",
		Adapter: adapter,
		Profile: "assistant",
		Env:     env,
		Options: options,
	}
}

func appendHandlerOptions(t *testing.T, build func(*testing.T) []func(*Handler)) []func(*Handler) {
	if build == nil {
		return nil
	}
	return build(t)
}

func providerFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "provider_fixtures", name))
	require.NoError(t, err)
	return data
}

func providerHeaders(contentType string, body []byte, build func([]byte) http.Header) http.Header {
	headers := http.Header{"Content-Type": {contentType}}
	if build != nil {
		for key, values := range build(body) {
			headers[key] = values
		}
	}
	return headers
}

func serveWebhookThroughXRR(t *testing.T, handler *Handler, serviceID, adapter, contentType string, headers http.Header, body []byte) *xhttp.Response {
	t.Helper()
	url := "https://hooks.example.test/services/" + serviceID + "/webhook"
	xreq := &xhttp.Request{
		Method:  http.MethodPost,
		URL:     url,
		Headers: singleHeaders(headers),
		Body:    string(body),
	}
	dir := t.TempDir()
	recSess := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(dir))
	resp, err := recSess.Record(context.Background(), xhttp.NewAdapter(), xreq, func() (xrr.Response, error) {
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		req.Header = headers.Clone()
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		handler.ServeServiceWebhook(rec, req, serviceID, adapter)
		return &xhttp.Response{
			Status:  rec.Code,
			Headers: singleHeaders(rec.Header()),
			Body:    rec.Body.String(),
		}, nil
	})
	require.NoError(t, err)

	replaySess := xrr.NewSession(xrr.ModeReplay, xrr.NewFileCassette(dir))
	replayed, err := replaySess.Record(context.Background(), xhttp.NewAdapter(), xreq, func() (xrr.Response, error) {
		t.Fatal("xrr replay must not execute webhook handler")
		return nil, nil
	})
	require.NoError(t, err)
	assert.Equal(t, resp.(*xhttp.Response).Status, xrrStatus(t, replayed))
	return resp.(*xhttp.Response)
}

type xrrHTTPDoer struct {
	t         *testing.T
	session   xrr.Session
	responder func(*xhttp.Request) (*xhttp.Response, error)
}

func newXRRHTTPDoer(t *testing.T, name string, responder func(*xhttp.Request) (*xhttp.Response, error), mode xrr.Mode) *xrrHTTPDoer {
	t.Helper()
	return newXRRHTTPDoerWithDir(t, filepath.Join(t.TempDir(), name), responder, mode)
}

func newXRRHTTPDoerWithDir(t *testing.T, dir string, responder func(*xhttp.Request) (*xhttp.Response, error), mode xrr.Mode) *xrrHTTPDoer {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	if mode == xrr.ModeReplay {
		responder = nil
	}
	return &xrrHTTPDoer{
		t:         t,
		session:   xrr.NewSession(mode, xrr.NewFileCassette(dir)),
		responder: responder,
	}
}

func (d *xrrHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	xreq := &xhttp.Request{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: singleHeaders(req.Header),
		Body:    string(body),
	}
	resp, err := d.session.Record(req.Context(), xhttp.NewAdapter(), xreq, func() (xrr.Response, error) {
		if d.responder == nil {
			d.t.Fatal("xrr replay attempted live provider HTTP")
			return nil, nil
		}
		return d.responder(xreq)
	})
	if err != nil {
		return nil, err
	}
	return httpResponseFromXRR(d.t, resp), nil
}

func httpResponseFromXRR(t *testing.T, resp xrr.Response) *http.Response {
	t.Helper()
	status := xrrStatus(t, resp)
	body := xrrBody(t, resp)
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func xrrStatus(t *testing.T, resp xrr.Response) int {
	t.Helper()
	switch typed := resp.(type) {
	case *xhttp.Response:
		return typed.Status
	case *xrr.RawResponse:
		return intFromAny(t, typed.Payload["status"])
	default:
		t.Fatalf("unexpected xrr response %T", resp)
		return 0
	}
}

func xrrBody(t *testing.T, resp xrr.Response) string {
	t.Helper()
	switch typed := resp.(type) {
	case *xhttp.Response:
		return typed.Body
	case *xrr.RawResponse:
		value, _ := typed.Payload["body"].(string)
		return value
	default:
		t.Fatalf("unexpected xrr response %T", resp)
		return ""
	}
}

func intFromAny(t *testing.T, value any) int {
	t.Helper()
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		t.Fatalf("unexpected numeric type %T", value)
		return 0
	}
}

func xrrTelegramSendMessageResponder(t *testing.T) func(*xhttp.Request) (*xhttp.Response, error) {
	return func(req *xhttp.Request) (*xhttp.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://telegram.test/botbot-token/sendMessage", req.URL)
		assert.Contains(t, req.Body, `"text":"`)
		assert.Contains(t, req.Body, `"chat_id":-1001234567890`)
		return &xhttp.Response{Status: http.StatusOK, Body: `{"ok":true,"result":{"message_id":9001}}`}, nil
	}
}

func xrrSlackPostMessageResponder(t *testing.T, status int, body string) func(*xhttp.Request) (*xhttp.Response, error) {
	return func(req *xhttp.Request) (*xhttp.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://slack.test/api/chat.postMessage", req.URL)
		assert.Equal(t, "Bearer xoxb-test", req.Headers["Authorization"])
		assert.Contains(t, req.Body, `"channel":"C012CHAN"`)
		return &xhttp.Response{Status: status, Body: body}, nil
	}
}

func xrrDiscordMessageResponder(t *testing.T, status int, body string) func(*xhttp.Request) (*xhttp.Response, error) {
	return func(req *xhttp.Request) (*xhttp.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://discord.test/api/v10/channels/1200000000000000002/messages", req.URL)
		assert.Equal(t, "Bot test-token", req.Headers["Authorization"])
		assert.Contains(t, req.Body, `"content":"ack"`)
		return &xhttp.Response{Status: status, Body: body}, nil
	}
}

func xrrTwilioMessageResponder(t *testing.T, status int, body string) func(*xhttp.Request) (*xhttp.Response, error) {
	return func(req *xhttp.Request) (*xhttp.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://twilio.test/2010-04-01/Accounts/AC123/Messages.json", req.URL)
		assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("AC123:token")), req.Headers["Authorization"])
		assert.Contains(t, req.Body, "From=%2B15550100002")
		assert.Contains(t, req.Body, "To=%2B15550100001")
		return &xhttp.Response{Status: status, Body: body}, nil
	}
}

func xrrWhatsAppCloudResponder(t *testing.T, status int, body string) func(*xhttp.Request) (*xhttp.Response, error) {
	return func(req *xhttp.Request) (*xhttp.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://whatsapp.test/123456789012345/messages", req.URL)
		assert.Equal(t, "Bearer wa-token", req.Headers["Authorization"])
		assert.Contains(t, req.Body, `"messaging_product":"whatsapp"`)
		assert.Contains(t, req.Body, `"to":"15551230001"`)
		return &xhttp.Response{Status: status, Body: body}, nil
	}
}

func singleHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}

func fixedProviderNow() time.Time {
	return time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
}

func cloneService(service *core.ServiceConfig) *core.ServiceConfig {
	copied := *service
	copied.Env = cloneStringMap(service.Env)
	copied.Options = cloneStringMap(service.Options)
	return &copied
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type runtimeRouter struct {
	route msgtypes.ExecutionRoute
}

func (r *runtimeRouter) ResolveMessageRoute(context.Context, *msgtypes.NormalizedMessage) (msgtypes.ExecutionRoute, error) {
	return r.route, nil
}

type runtimeExecutor struct {
	reply string
}

func (e *runtimeExecutor) ExecuteMessage(context.Context, msgtypes.ExecutionHandoff) (*msgtypes.ExecutionResult, error) {
	return &msgtypes.ExecutionResult{
		Status: "completed",
		Reply:  &msgtypes.DeliveryRequest{Text: e.reply},
	}, nil
}
