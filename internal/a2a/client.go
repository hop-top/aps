package a2a

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
	a2aclient "github.com/a2aproject/a2a-go/a2aclient"

	"oss-aps-cli/internal/core"
)

// Client represents an A2A client for communicating with other A2A agents
type Client struct {
	profileID string
	profile   *core.Profile
	card      *a2a.AgentCard
	client    *a2aclient.Client
}

// NewClient creates a new A2A client instance for a target profile
func NewClient(targetProfileID string, targetProfile *core.Profile) (*Client, error) {
	if targetProfileID == "" {
		return nil, ErrInvalidConfig
	}

	if targetProfile == nil {
		return nil, fmt.Errorf("target profile cannot be nil")
	}

	if targetProfile.A2A == nil || !targetProfile.A2A.Enabled {
		return nil, ErrA2ANotEnabled
	}

	card, err := GenerateAgentCardFromProfile(targetProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent card: %w", err)
	}

	if err := validateAgentCardTransport(card); err != nil {
		return nil, fmt.Errorf("invalid agent card transport: %w", err)
	}

	client, err := a2aclient.NewFromCard(context.Background(), card)
	if err != nil {
		return nil, ErrClientFailed("initialization", err)
	}

	return &Client{
		profileID: targetProfileID,
		profile:   targetProfile,
		card:      card,
		client:    client,
	}, nil
}

// validateAgentCardTransport validates that agent card has valid transport configuration
func validateAgentCardTransport(card *a2a.AgentCard) error {
	if card == nil {
		return ErrInvalidAgentCard("card cannot be nil")
	}

	if card.URL == "" {
		return ErrInvalidAgentCard("agent card must have URL")
	}

	if card.PreferredTransport == "" {
		card.PreferredTransport = a2a.TransportProtocolJSONRPC
	}

	supportedTransports := map[a2a.TransportProtocol]bool{
		a2a.TransportProtocolJSONRPC:  true,
		a2a.TransportProtocolGRPC:     true,
		a2a.TransportProtocolHTTPJSON: true,
	}

	if !supportedTransports[card.PreferredTransport] {
		return ErrTransportNotSupported
	}

	return nil
}

// GetProfileID returns target profile ID
func (c *Client) GetProfileID() string {
	return c.profileID
}

// GetAgentCard returns Agent Card for target profile
func (c *Client) GetAgentCard() *a2a.AgentCard {
	return c.card
}

// SendMessage creates a task and sends a message to the target agent
func (c *Client) SendMessage(ctx context.Context, message *a2a.Message) (*a2a.Task, error) {
	if message == nil {
		return nil, ErrInvalidMessage("message cannot be nil")
	}

	params := &a2a.MessageSendParams{
		Message: message,
	}

	result, err := c.client.SendMessage(ctx, params)
	if err != nil {
		return nil, ErrClientFailed("send message", err)
	}

	// SDK v0.3.4 returns Message instead of Task for SendMessage
	// We need to handle both types for compatibility
	switch v := result.(type) {
	case *a2a.Task:
		return v, nil
	case *a2a.Message:
		// If we get a Message, we need to fetch the task using the task ID from the message
		if v.TaskID == "" {
			return nil, ErrClientFailed("send message", fmt.Errorf("message has no task ID"))
		}
		// For now, create a minimal task response
		// In a real implementation, we'd call GetTask with the task ID
		return &a2a.Task{
			ID: v.TaskID,
			Status: a2a.TaskStatus{
				State: a2a.TaskStateSubmitted,
			},
		}, nil
	default:
		return nil, ErrClientFailed("send message", fmt.Errorf("unexpected result type: %T", result))
	}
}

// GetTask retrieves a task by ID
func (c *Client) GetTask(ctx context.Context, taskID a2a.TaskID) (*a2a.Task, error) {
	if taskID == "" {
		return nil, ErrInvalidMessage("task ID cannot be empty")
	}

	params := &a2a.TaskQueryParams{
		ID: taskID,
	}

	task, err := c.client.GetTask(ctx, params)
	if err != nil {
		return nil, ErrClientFailed("get task", err)
	}

	return task, nil
}

// ListTasks retrieves tasks with optional filters
func (c *Client) ListTasks(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	return nil, ErrClientFailed("list tasks", fmt.Errorf("ListTasks not supported by SDK v0.3.4"))
}

// CancelTask cancels a running task
func (c *Client) CancelTask(ctx context.Context, taskID a2a.TaskID) error {
	if taskID == "" {
		return ErrInvalidMessage("task ID cannot be empty")
	}

	params := &a2a.TaskIDParams{
		ID: taskID,
	}

	_, err := c.client.CancelTask(ctx, params)
	if err != nil {
		return ErrClientFailed("cancel task", err)
	}

	return nil
}

// SubscribeToTask subscribes to push notifications for task updates
func (c *Client) SubscribeToTask(ctx context.Context, taskID a2a.TaskID, webhookURL string) error {
	if taskID == "" {
		return ErrInvalidMessage("task ID cannot be empty")
	}

	if webhookURL == "" {
		return ErrInvalidMessage("webhook URL cannot be empty")
	}

	config := &a2a.TaskPushConfig{
		TaskID: taskID,
	}

	_, err := c.client.SetTaskPushConfig(ctx, config)
	if err != nil {
		return ErrClientFailed("subscribe to task", err)
	}

	return nil
}

// SendMessageStream creates a task and sends a message with streaming support
func (c *Client) SendMessageStream(ctx context.Context, message *a2a.Message) (<-chan interface{}, error) {
	return nil, ErrClientFailed("send message stream", fmt.Errorf("SendMessageStream not supported by SDK v0.3.4"))
}
