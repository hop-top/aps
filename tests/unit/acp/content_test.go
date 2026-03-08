package acp

import (
	"encoding/base64"
	"testing"

	"hop.top/aps/internal/acp"
)

// TestContentBlockValidation tests content block validation
func TestContentBlockValidation(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	testCases := []struct {
		name    string
		block   acp.ContentBlock
		valid   bool
	}{
		{
			name: "valid text block",
			block: acp.ContentBlock{
				Type: "text",
				Text: "Hello world",
			},
			valid: true,
		},
		{
			name: "text block without text",
			block: acp.ContentBlock{
				Type: "text",
				Text: "",
			},
			valid: false,
		},
		{
			name: "block without type",
			block: acp.ContentBlock{
				Text: "Hello",
			},
			valid: false,
		},
		{
			name: "resource block with URI",
			block: acp.ContentBlock{
				Type: "resource",
				URI:  "https://example.com/file",
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := handler.ValidateContentBlock(tc.block)
			if tc.valid && err != nil {
				t.Errorf("expected valid block, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected invalid block, got no error")
			}
		})
	}
}

// TestCreateTextBlock tests creating a text block
func TestCreateTextBlock(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	block := handler.CreateTextBlock("test content")

	if block.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", block.Type)
	}

	if block.Text != "test content" {
		t.Errorf("expected text 'test content', got '%s'", block.Text)
	}
}

// TestCreateImageBlock tests creating an image block
func TestCreateImageBlock(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	// Create valid base64 image data
	imageData := base64.StdEncoding.EncodeToString([]byte("fake image data"))

	block := handler.CreateImageBlock(imageData, "image/png")

	if block.Type != "image" {
		t.Errorf("expected type 'image', got '%s'", block.Type)
	}

	if block.MimeType != "image/png" {
		t.Errorf("expected mime type 'image/png', got '%s'", block.MimeType)
	}
}

// TestCreateResourceBlock tests creating a resource block
func TestCreateResourceBlock(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	block := handler.CreateResourceBlock("https://example.com/resource")

	if block.Type != "resource" {
		t.Errorf("expected type 'resource', got '%s'", block.Type)
	}

	if block.URI != "https://example.com/resource" {
		t.Errorf("expected URI 'https://example.com/resource', got '%s'", block.URI)
	}
}

// TestMarshalContentBlock tests marshaling content blocks
func TestMarshalContentBlock(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	block := handler.CreateTextBlock("test")
	data, err := handler.MarshalContentBlock(*block)

	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if len(data) == 0 {
		t.Error("marshaled data should not be empty")
	}
}

// TestUnmarshalContentBlock tests unmarshaling content blocks
func TestUnmarshalContentBlock(t *testing.T) {
	handler := acp.NewContentBlockHandler()

	// Create and marshal a block
	original := handler.CreateTextBlock("test content")
	data, _ := handler.MarshalContentBlock(*original)

	// Unmarshal it
	restored, err := handler.UnmarshalContentBlock(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if restored.Type != original.Type {
		t.Errorf("type mismatch: expected '%s', got '%s'", original.Type, restored.Type)
	}

	if restored.Text != original.Text {
		t.Errorf("text mismatch: expected '%s', got '%s'", original.Text, restored.Text)
	}
}

// TestExecutionPlanCreate tests creating an execution plan
func TestExecutionPlanCreate(t *testing.T) {
	handler := acp.NewExecutionPlanHandler()

	plan := handler.CreateExecutionPlan()

	if plan.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", plan.Status)
	}

	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}
}

// TestExecutionPlanAddStep tests adding steps to a plan
func TestExecutionPlanAddStep(t *testing.T) {
	handler := acp.NewExecutionPlanHandler()

	plan := handler.CreateExecutionPlan()
	handler.AddStep(plan, "Step 1", "high", "pending")
	handler.AddStep(plan, "Step 2", "medium", "pending")

	if len(plan.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].Content != "Step 1" {
		t.Errorf("expected content 'Step 1', got '%s'", plan.Steps[0].Content)
	}

	if plan.Steps[0].Priority != "high" {
		t.Errorf("expected priority 'high', got '%s'", plan.Steps[0].Priority)
	}
}

// TestExecutionPlanUpdateStepStatus tests updating step status
func TestExecutionPlanUpdateStepStatus(t *testing.T) {
	handler := acp.NewExecutionPlanHandler()

	plan := handler.CreateExecutionPlan()
	handler.AddStep(plan, "Step 1", "high", "pending")

	if err := handler.UpdateStepStatus(plan, 0, "in_progress"); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	if plan.Steps[0].Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got '%s'", plan.Steps[0].Status)
	}

	if plan.Status != "in_progress" {
		t.Errorf("expected plan status 'in_progress', got '%s'", plan.Status)
	}
}

// TestExecutionPlanValidation tests plan validation
func TestExecutionPlanValidation(t *testing.T) {
	handler := acp.NewExecutionPlanHandler()

	plan := handler.CreateExecutionPlan()

	// Should fail - no steps
	if err := handler.ValidateExecutionPlan(plan); err == nil {
		t.Error("should fail validation with no steps")
	}

	// Add a step and try again
	handler.AddStep(plan, "Step 1", "high", "pending")

	if err := handler.ValidateExecutionPlan(plan); err != nil {
		t.Errorf("should pass validation: %v", err)
	}
}

// TestNotificationHandler tests notification creation
func TestNotificationHandler(t *testing.T) {
	handler := acp.NewNotificationHandler()

	// Test content chunk notification
	block := acp.ContentBlock{
		Type: "text",
		Text: "Hello",
	}
	notif := handler.CreateContentChunkNotification("sess_123", &block)

	if notif.Method != "session/update" {
		t.Errorf("expected method 'session/update', got '%s'", notif.Method)
	}

	if notif.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", notif.JSONRPC)
	}

	// Verify params have sessionId
	params := notif.Params.(map[string]interface{})
	if params["sessionId"] != "sess_123" {
		t.Errorf("expected sessionId 'sess_123'")
	}
}

// TestMCPBridge tests MCP server registration
func TestMCPBridge(t *testing.T) {
	bridge := acp.NewMCPBridge()

	// Register a server
	config := map[string]string{"type": "stdio", "command": "my-tool"}
	if err := bridge.RegisterMCPServer("my-tool", config); err != nil {
		t.Fatalf("failed to register server: %v", err)
	}

	// List servers
	servers := bridge.ListMCPServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}

	// Get server config
	retrieved, err := bridge.GetMCPServerConfig("my-tool")
	if err != nil {
		t.Fatalf("failed to get server config: %v", err)
	}

	if retrieved == nil {
		t.Error("server config should not be nil")
	}
}

// TestMCPBridgeNonExistent tests accessing non-existent server
func TestMCPBridgeNonExistent(t *testing.T) {
	bridge := acp.NewMCPBridge()

	_, err := bridge.GetMCPServerConfig("nonexistent")
	if err == nil {
		t.Error("should return error for non-existent server")
	}
}
