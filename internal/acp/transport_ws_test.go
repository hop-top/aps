package acp

import (
	"encoding/json"
	"strings"
	"testing"
)

type fakeMCPServer struct {
	tools []MCPTool
	calls []fakeMCPCall
}

type fakeMCPCall struct {
	toolName  string
	arguments json.RawMessage
}

func (s *fakeMCPServer) ListMCPTools() ([]MCPTool, error) {
	tools := make([]MCPTool, len(s.tools))
	copy(tools, s.tools)
	return tools, nil
}

func (s *fakeMCPServer) CallMCPTool(toolName string, arguments json.RawMessage) (interface{}, error) {
	s.calls = append(s.calls, fakeMCPCall{
		toolName:  toolName,
		arguments: append(json.RawMessage(nil), arguments...),
	})
	return map[string]interface{}{
		"tool": toolName,
		"ok":   true,
	}, nil
}

func TestMCPBridgeListToolsReturnsRegisteredMetadata(t *testing.T) {
	bridge := NewMCPBridge()
	server := &fakeMCPServer{
		tools: []MCPTool{
			{
				Name:        "write_file",
				Description: "Write a file",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
			{
				Name:        "read_file",
				Description: "Read a file",
			},
		},
	}

	if err := bridge.RegisterMCPServer("fs", server); err != nil {
		t.Fatalf("register MCP server: %v", err)
	}

	tools, err := bridge.ListMCPTools()
	if err != nil {
		t.Fatalf("list MCP tools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Server != "fs" || tools[0].Name != "read_file" || tools[0].Description != "Read a file" {
		t.Fatalf("unexpected first tool: %#v", tools[0])
	}
	if tools[1].Server != "fs" || tools[1].Name != "write_file" || tools[1].InputSchema == nil {
		t.Fatalf("unexpected second tool: %#v", tools[1])
	}

	available := bridge.GetAvailableTools()
	if len(available) != 2 {
		t.Fatalf("expected 2 available tools, got %d", len(available))
	}
	if available[0]["server"] != "fs" || available[0]["name"] != "read_file" {
		t.Fatalf("unexpected available tool: %#v", available[0])
	}
}

func TestMCPBridgeCallToolDispatchesToRegisteredServer(t *testing.T) {
	bridge := NewMCPBridge()
	server := &fakeMCPServer{
		tools: []MCPTool{{Name: "read_file"}},
	}

	if err := bridge.RegisterMCPServer("fs", server); err != nil {
		t.Fatalf("register MCP server: %v", err)
	}

	result, err := bridge.CallMCPTool("fs", "read_file", json.RawMessage(`{"path":"README.md"}`))
	if err != nil {
		t.Fatalf("call MCP tool: %v", err)
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if resultMap["tool"] != "read_file" || resultMap["ok"] != true {
		t.Fatalf("unexpected call result: %#v", resultMap)
	}
	if len(server.calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(server.calls))
	}
	if server.calls[0].toolName != "read_file" {
		t.Fatalf("unexpected tool call name: %s", server.calls[0].toolName)
	}
	if string(server.calls[0].arguments) != `{"path":"README.md"}` {
		t.Fatalf("unexpected arguments: %s", server.calls[0].arguments)
	}
}

func TestMCPBridgeCallToolErrorsForUnknownServerOrTool(t *testing.T) {
	bridge := NewMCPBridge()
	server := &fakeMCPServer{
		tools: []MCPTool{{Name: "read_file"}},
	}

	if err := bridge.RegisterMCPServer("fs", server); err != nil {
		t.Fatalf("register MCP server: %v", err)
	}

	if _, err := bridge.CallMCPTool("missing", "read_file", nil); err == nil || !strings.Contains(err.Error(), "MCP server not found: missing") {
		t.Fatalf("expected unknown server error, got %v", err)
	}
	if _, err := bridge.CallMCPTool("fs", "write_file", nil); err == nil || !strings.Contains(err.Error(), "MCP tool not found: fs/write_file") {
		t.Fatalf("expected unknown tool error, got %v", err)
	}
	if len(server.calls) != 0 {
		t.Fatalf("expected no calls for unknown server/tool, got %d", len(server.calls))
	}
}
