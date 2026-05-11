package messenger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	coremessenger "hop.top/aps/internal/core/messenger"
)

type TelegramProviderConfig struct {
	BotToken  string
	BaseURL   string
	Transport TelegramTransport
	Now       func() time.Time
}

type TelegramProvider struct {
	botToken  string
	transport TelegramTransport
	now       func() time.Time
}

var _ coremessenger.MessageProvider = (*TelegramProvider)(nil)
var _ coremessenger.ProviderDelivery = (*TelegramProvider)(nil)

func NewTelegramProvider(config TelegramProviderConfig) *TelegramProvider {
	transport := config.Transport
	if transport == nil {
		transport = &TelegramHTTPTransport{
			BaseURL:    config.BaseURL,
			HTTPClient: http.DefaultClient,
		}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &TelegramProvider{
		botToken:  strings.TrimSpace(config.BotToken),
		transport: transport,
		now:       now,
	}
}

func (p *TelegramProvider) Metadata() coremessenger.ProviderRuntimeMetadata {
	return coremessenger.ProviderRuntimeMetadata{
		Provider:            string(coremessenger.PlatformTelegram),
		DisplayName:         "Telegram",
		IngressModes:        []coremessenger.IngressMode{coremessenger.IngressModeWebhook},
		DeliveryModes:       []coremessenger.DeliveryMode{coremessenger.DeliveryModeText},
		SupportsThreads:     true,
		SupportsAttachments: true,
	}
}

func (p *TelegramProvider) NormalizeIngress(ctx context.Context, ingress coremessenger.NativeIngress) (*coremessenger.NormalizedMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var update telegramUpdate
	if err := json.Unmarshal(ingress.Body, &update); err != nil {
		return nil, fmt.Errorf("invalid telegram update JSON: %w", err)
	}

	msg, source, err := telegramMessageFromUpdate(update)
	if err != nil {
		return nil, err
	}
	if msg.Chat == nil {
		return nil, fmt.Errorf("telegram update %d has no chat", update.UpdateID)
	}

	sender := telegramSender(source, msg)
	if source == "callback_query" && update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		sender = telegramSenderFromUser(update.CallbackQuery.From)
	}
	if sender.ID == "" {
		return nil, fmt.Errorf("telegram update %d has no sender", update.UpdateID)
	}
	timestamp := ingress.ReceivedAt
	if msg.Date > 0 {
		timestamp = time.Unix(msg.Date, 0).UTC()
	}
	if timestamp.IsZero() {
		timestamp = p.now().UTC()
	}

	callbackData := ""
	if update.CallbackQuery != nil {
		callbackData = update.CallbackQuery.Data
	}
	text := firstNonEmpty(strings.TrimSpace(msg.Text), strings.TrimSpace(msg.Caption), strings.TrimSpace(callbackData))
	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	messageID := strconv.FormatInt(msg.MessageID, 10)
	metadata := map[string]any{
		"service_id":           ingress.ServiceID,
		"messenger_name":       ingress.ServiceID,
		"telegram_update_type": source,
		"telegram_message_id":  messageID,
		"telegram_chat_id":     chatID,
	}
	if update.UpdateID != 0 {
		metadata["telegram_update_id"] = strconv.FormatInt(update.UpdateID, 10)
	}
	if msg.MessageThreadID != 0 {
		metadata["telegram_message_thread_id"] = strconv.FormatInt(msg.MessageThreadID, 10)
	}
	if update.CallbackQuery != nil {
		metadata["telegram_callback_query_id"] = update.CallbackQuery.ID
	}

	normalized := &coremessenger.NormalizedMessage{
		ID:        telegramNormalizedID(update.UpdateID, msg.Chat.ID, msg.MessageID),
		Timestamp: timestamp,
		Platform:  string(coremessenger.PlatformTelegram),
		Sender:    sender,
		Channel: coremessenger.Channel{
			ID:         chatID,
			Name:       firstNonEmpty(msg.Chat.Title, msg.Chat.Username),
			Type:       telegramChannelType(msg.Chat.Type),
			PlatformID: chatID,
		},
		Text:             text,
		Attachments:      telegramAttachments(msg),
		PlatformMetadata: metadata,
	}
	if msg.MessageThreadID != 0 {
		normalized.Thread = &coremessenger.Thread{ID: strconv.FormatInt(msg.MessageThreadID, 10), Type: coremessenger.ThreadTypeTopic}
	} else if msg.ReplyToMessage != nil && msg.ReplyToMessage.MessageID != 0 {
		normalized.Thread = &coremessenger.Thread{ID: strconv.FormatInt(msg.ReplyToMessage.MessageID, 10), Type: coremessenger.ThreadTypeReply}
	}
	return normalized, nil
}

func (p *TelegramProvider) DeliverMessage(ctx context.Context, delivery coremessenger.DeliveryRequest) (*coremessenger.DeliveryReceipt, error) {
	if strings.TrimSpace(p.botToken) == "" {
		return nil, fmt.Errorf("telegram bot token is required for delivery")
	}
	req := TelegramSendMessageRequest{
		ChatID: parseTelegramChatID(delivery.ChannelID),
		Text:   delivery.Text,
	}
	if threadID := firstNonEmpty(metadataString(delivery.Metadata, "message_thread_id"), metadataString(delivery.Metadata, "telegram_message_thread_id")); threadID != "" {
		req.MessageThreadID = parseOptionalInt64(threadID)
	}
	if replyID := firstNonEmpty(metadataString(delivery.Metadata, "reply_to_message_id"), metadataString(delivery.Metadata, "telegram_message_id")); replyID != "" {
		if id := parseOptionalInt64(replyID); id != 0 {
			req.ReplyParameters = &TelegramReplyParameters{MessageID: id}
		}
	}

	resp, err := p.transport.SendMessage(ctx, p.botToken, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("telegram sendMessage returned nil response")
	}
	if !resp.OK {
		description := strings.TrimSpace(resp.Description)
		if description == "" {
			description = "telegram sendMessage failed"
		}
		return nil, errors.New(description)
	}

	deliveredAt := p.now().UTC()
	providerData := map[string]any{
		"chat_id": delivery.ChannelID,
	}
	deliveryID := ""
	if resp.Result != nil {
		deliveryID = strconv.FormatInt(resp.Result.MessageID, 10)
		providerData["message_id"] = deliveryID
	}
	return &coremessenger.DeliveryReceipt{
		Provider:     string(coremessenger.PlatformTelegram),
		DeliveryID:   deliveryID,
		Status:       "success",
		DeliveredAt:  deliveredAt,
		ProviderData: providerData,
	}, nil
}

type TelegramTransport interface {
	SendMessage(ctx context.Context, botToken string, req TelegramSendMessageRequest) (*TelegramAPIResponse, error)
}

type TelegramHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type TelegramHTTPTransport struct {
	BaseURL    string
	HTTPClient TelegramHTTPDoer
}

func (t *TelegramHTTPTransport) SendMessage(ctx context.Context, botToken string, payload TelegramSendMessageRequest) (*TelegramAPIResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(t.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/bot"+botToken+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := t.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var apiResp TelegramAPIResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("telegram sendMessage response decode failed: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if apiResp.Description != "" {
			return &apiResp, nil
		}
		return nil, fmt.Errorf("telegram sendMessage returned HTTP %d", resp.StatusCode)
	}
	return &apiResp, nil
}

type TelegramSendMessageRequest struct {
	ChatID          any                      `json:"chat_id"`
	Text            string                   `json:"text"`
	MessageThreadID int64                    `json:"message_thread_id,omitempty"`
	ReplyParameters *TelegramReplyParameters `json:"reply_parameters,omitempty"`
}

type TelegramReplyParameters struct {
	MessageID int64 `json:"message_id"`
}

type TelegramAPIResponse struct {
	OK          bool                    `json:"ok"`
	Description string                  `json:"description,omitempty"`
	Result      *TelegramMessageSummary `json:"result,omitempty"`
}

type TelegramMessageSummary struct {
	MessageID int64 `json:"message_id"`
}

type telegramUpdate struct {
	UpdateID          int64                  `json:"update_id"`
	Message           *telegramMessage       `json:"message,omitempty"`
	EditedMessage     *telegramMessage       `json:"edited_message,omitempty"`
	ChannelPost       *telegramMessage       `json:"channel_post,omitempty"`
	EditedChannelPost *telegramMessage       `json:"edited_channel_post,omitempty"`
	CallbackQuery     *telegramCallbackQuery `json:"callback_query,omitempty"`
}

type telegramCallbackQuery struct {
	ID      string           `json:"id"`
	From    *telegramUser    `json:"from,omitempty"`
	Message *telegramMessage `json:"message,omitempty"`
	Data    string           `json:"data,omitempty"`
}

type telegramMessage struct {
	MessageID       int64            `json:"message_id"`
	MessageThreadID int64            `json:"message_thread_id,omitempty"`
	From            *telegramUser    `json:"from,omitempty"`
	SenderChat      *telegramChat    `json:"sender_chat,omitempty"`
	Chat            *telegramChat    `json:"chat,omitempty"`
	Date            int64            `json:"date,omitempty"`
	Text            string           `json:"text,omitempty"`
	Caption         string           `json:"caption,omitempty"`
	ReplyToMessage  *telegramMessage `json:"reply_to_message,omitempty"`
	Document        *telegramFile    `json:"document,omitempty"`
	Audio           *telegramFile    `json:"audio,omitempty"`
	Voice           *telegramFile    `json:"voice,omitempty"`
	Video           *telegramFile    `json:"video,omitempty"`
	Photo           []telegramPhoto  `json:"photo,omitempty"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}

type telegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type telegramFile struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type telegramPhoto struct {
	FileID   string `json:"file_id"`
	FileSize int64  `json:"file_size,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

func telegramMessageFromUpdate(update telegramUpdate) (*telegramMessage, string, error) {
	switch {
	case update.Message != nil:
		return update.Message, "message", nil
	case update.EditedMessage != nil:
		return update.EditedMessage, "edited_message", nil
	case update.ChannelPost != nil:
		return update.ChannelPost, "channel_post", nil
	case update.EditedChannelPost != nil:
		return update.EditedChannelPost, "edited_channel_post", nil
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		return update.CallbackQuery.Message, "callback_query", nil
	default:
		return nil, "", fmt.Errorf("telegram update has no supported message payload")
	}
}

func telegramSender(source string, msg *telegramMessage) coremessenger.Sender {
	if source == "channel_post" || source == "edited_channel_post" {
		return telegramSenderFromChat(msg.Chat)
	}
	if msg.From != nil {
		return telegramSenderFromUser(msg.From)
	}
	if msg.SenderChat != nil {
		return telegramSenderFromChat(msg.SenderChat)
	}
	return telegramSenderFromChat(msg.Chat)
}

func telegramSenderFromUser(user *telegramUser) coremessenger.Sender {
	if user == nil || user.ID == 0 {
		return coremessenger.Sender{}
	}
	id := strconv.FormatInt(user.ID, 10)
	return coremessenger.Sender{
		ID:             id,
		Name:           buildTelegramName(user.FirstName, user.LastName),
		PlatformHandle: user.Username,
		PlatformID:     id,
	}
}

func telegramSenderFromChat(chat *telegramChat) coremessenger.Sender {
	if chat == nil || chat.ID == 0 {
		return coremessenger.Sender{}
	}
	id := strconv.FormatInt(chat.ID, 10)
	return coremessenger.Sender{
		ID:             id,
		Name:           firstNonEmpty(chat.Title, buildTelegramName(chat.FirstName, chat.LastName), chat.Username),
		PlatformHandle: chat.Username,
		PlatformID:     id,
	}
}

func telegramChannelType(chatType string) string {
	switch chatType {
	case "group", "supergroup":
		return coremessenger.ChannelTypeGroup
	case "channel":
		return coremessenger.ChannelTypeBroadcast
	default:
		return coremessenger.ChannelTypeDirect
	}
}

func telegramNormalizedID(updateID, chatID, messageID int64) string {
	if updateID != 0 {
		return "telegram:update:" + strconv.FormatInt(updateID, 10)
	}
	return "telegram:message:" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(messageID, 10)
}

func telegramAttachments(msg *telegramMessage) []coremessenger.Attachment {
	var out []coremessenger.Attachment
	addFile := func(kind string, file *telegramFile) {
		if file == nil || file.FileID == "" {
			return
		}
		out = append(out, coremessenger.Attachment{
			Type:      kind,
			URL:       "telegram:file:" + file.FileID,
			MimeType:  file.MimeType,
			SizeBytes: file.FileSize,
		})
	}
	addFile("file", msg.Document)
	addFile("audio", msg.Audio)
	addFile("audio", msg.Voice)
	addFile("video", msg.Video)
	if len(msg.Photo) > 0 {
		best := msg.Photo[len(msg.Photo)-1]
		if best.FileID != "" {
			out = append(out, coremessenger.Attachment{
				Type:      "image",
				URL:       "telegram:file:" + best.FileID,
				SizeBytes: best.FileSize,
			})
		}
	}
	return out
}

func parseTelegramChatID(value string) any {
	if id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
		return id
	}
	return strings.TrimSpace(value)
}

func parseOptionalInt64(value string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	switch value := metadata[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
	}
	return ""
}
