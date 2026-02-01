package acp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
)

// ContentBlockHandler handles content block operations
type ContentBlockHandler struct{}

// NewContentBlockHandler creates a new content block handler
func NewContentBlockHandler() *ContentBlockHandler {
	return &ContentBlockHandler{}
}

// ValidateContentBlock validates a content block
func (cbh *ContentBlockHandler) ValidateContentBlock(cb ContentBlock) error {
	if cb.Type == "" {
		return fmt.Errorf("content block type is required")
	}

	switch cb.Type {
	case "text":
		if cb.Text == "" {
			return fmt.Errorf("text content block must have text field")
		}
	case "image":
		if cb.Data == "" || cb.MimeType == "" {
			return fmt.Errorf("image content block must have data and mimeType")
		}
		if !isValidMimeType(cb.MimeType) {
			return fmt.Errorf("invalid mime type for image: %s", cb.MimeType)
		}
		if !isValidBase64(cb.Data) {
			return fmt.Errorf("image data is not valid base64")
		}
	case "audio":
		if cb.Data == "" || cb.MimeType == "" {
			return fmt.Errorf("audio content block must have data and mimeType")
		}
		if !isValidMimeType(cb.MimeType) {
			return fmt.Errorf("invalid mime type for audio: %s", cb.MimeType)
		}
	case "resource":
		if cb.URI == "" {
			return fmt.Errorf("resource content block must have URI")
		}
	default:
		return fmt.Errorf("unknown content block type: %s", cb.Type)
	}

	return nil
}

// MarshalContentBlock serializes a content block to JSON
func (cbh *ContentBlockHandler) MarshalContentBlock(cb ContentBlock) ([]byte, error) {
	if err := cbh.ValidateContentBlock(cb); err != nil {
		return nil, err
	}

	return json.Marshal(cb)
}

// UnmarshalContentBlock deserializes a content block from JSON
func (cbh *ContentBlockHandler) UnmarshalContentBlock(data []byte) (*ContentBlock, error) {
	var cb ContentBlock
	if err := json.Unmarshal(data, &cb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
	}

	if err := cbh.ValidateContentBlock(cb); err != nil {
		return nil, err
	}

	return &cb, nil
}

// CreateTextBlock creates a text content block
func (cbh *ContentBlockHandler) CreateTextBlock(text string) *ContentBlock {
	return &ContentBlock{
		Type: "text",
		Text: text,
	}
}

// CreateImageBlock creates an image content block
func (cbh *ContentBlockHandler) CreateImageBlock(data string, mimeType string) *ContentBlock {
	return &ContentBlock{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
	}
}

// CreateAudioBlock creates an audio content block
func (cbh *ContentBlockHandler) CreateAudioBlock(data string, mimeType string) *ContentBlock {
	return &ContentBlock{
		Type:     "audio",
		Data:     data,
		MimeType: mimeType,
	}
}

// CreateResourceBlock creates a resource content block
func (cbh *ContentBlockHandler) CreateResourceBlock(uri string) *ContentBlock {
	return &ContentBlock{
		Type: "resource",
		URI:  uri,
	}
}

// isValidMimeType checks if a mime type is valid
func isValidMimeType(mimeType string) bool {
	// Use Go's mime package to validate
	_, _, err := mime.ParseMediaType(mimeType)
	return err == nil
}

// isValidBase64 checks if a string is valid base64
func isValidBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// ExecutionPlanHandler handles execution plan operations
type ExecutionPlanHandler struct{}

// NewExecutionPlanHandler creates a new execution plan handler
func NewExecutionPlanHandler() *ExecutionPlanHandler {
	return &ExecutionPlanHandler{}
}

// CreateExecutionPlan creates a new execution plan
func (eph *ExecutionPlanHandler) CreateExecutionPlan() *ExecutionPlan {
	return &ExecutionPlan{
		Steps:  make([]PlanStep, 0),
		Status: "pending",
	}
}

// AddStep adds a step to the execution plan
func (eph *ExecutionPlanHandler) AddStep(plan *ExecutionPlan, content string, priority string, status string) {
	if priority == "" {
		priority = "medium"
	}
	if status == "" {
		status = "pending"
	}

	plan.Steps = append(plan.Steps, PlanStep{
		Content:  content,
		Priority: priority,
		Status:   status,
	})
}

// UpdateStepStatus updates the status of a step
func (eph *ExecutionPlanHandler) UpdateStepStatus(plan *ExecutionPlan, stepIndex int, status string) error {
	if stepIndex < 0 || stepIndex >= len(plan.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	plan.Steps[stepIndex].Status = status

	// Update overall status
	allCompleted := true
	anyRunning := false
	for _, step := range plan.Steps {
		if step.Status == "pending" {
			allCompleted = false
		}
		if step.Status == "in_progress" {
			anyRunning = true
		}
	}

	if anyRunning {
		plan.Status = "in_progress"
	} else if allCompleted {
		plan.Status = "completed"
	}

	return nil
}

// ValidateExecutionPlan validates an execution plan
func (eph *ExecutionPlanHandler) ValidateExecutionPlan(plan *ExecutionPlan) error {
	if plan == nil {
		return fmt.Errorf("execution plan cannot be nil")
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("execution plan must have at least one step")
	}

	validStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"completed":   true,
	}

	validPriorities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
	}

	for i, step := range plan.Steps {
		if step.Content == "" {
			return fmt.Errorf("step %d must have content", i)
		}

		if !validPriorities[step.Priority] {
			return fmt.Errorf("step %d has invalid priority: %s", i, step.Priority)
		}

		if !validStatuses[step.Status] {
			return fmt.Errorf("step %d has invalid status: %s", i, step.Status)
		}
	}

	return nil
}

// NotificationHandler handles sending notifications to clients
type NotificationHandler struct{}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// CreateContentChunkNotification creates a content chunk notification
func (nh *NotificationHandler) CreateContentChunkNotification(sessionID string, block *ContentBlock) JSONRPCNotification {
	return JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params: map[string]interface{}{
			"sessionId": sessionID,
			"type":      "ContentChunk",
			"content":   block,
		},
	}
}

// CreateToolCallNotification creates a tool call notification
func (nh *NotificationHandler) CreateToolCallNotification(sessionID string, toolName string, arguments map[string]interface{}) JSONRPCNotification {
	return JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params: map[string]interface{}{
			"sessionId": sessionID,
			"type":      "ToolCallUpdate",
			"tool":      toolName,
			"arguments": arguments,
		},
	}
}

// CreateModeUpdateNotification creates a mode update notification
func (nh *NotificationHandler) CreateModeUpdateNotification(sessionID string, mode SessionMode) JSONRPCNotification {
	return JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params: map[string]interface{}{
			"sessionId": sessionID,
			"type":      "CurrentModeUpdate",
			"mode":      mode,
		},
	}
}

// CreateStatusNotification creates a status notification
func (nh *NotificationHandler) CreateStatusNotification(sessionID string, status string, details map[string]interface{}) JSONRPCNotification {
	return JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params: map[string]interface{}{
			"sessionId": sessionID,
			"type":      "StatusUpdate",
			"status":    status,
			"details":   details,
		},
	}
}
