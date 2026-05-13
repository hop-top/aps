package chat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/session"
	"hop.top/kit/go/ai/llm"
)

type fakeCompleter struct {
	req llm.Request
}

func (f *fakeCompleter) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	f.req = req
	return llm.Response{Role: string(RoleAssistant), Content: "hello back"}, nil
}

func TestServicePersistsChatSessionAndTurns(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	registry := session.NewForTesting()
	client := &fakeCompleter{}
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	service, err := NewServiceWithProfile(&core.Profile{
		ID:          "agent",
		DisplayName: "Agent",
		Persona:     core.Persona{Tone: "precise"},
	}, client, ServiceOptions{
		Registry: registry,
		Store:    NewStore(dataDir),
		Now:      func() time.Time { return now },
		NewID:    func() string { return "chat-1" },
		Model:    "gpt-4o",
	})
	requireNoError(t, err)

	transcript, err := service.Open(context.Background())
	requireNoError(t, err)
	if transcript.SessionID != "chat-1" {
		t.Fatalf("SessionID = %q", transcript.SessionID)
	}
	info, err := registry.Get("chat-1")
	requireNoError(t, err)
	if info.Type != session.SessionTypeChat {
		t.Fatalf("session type = %q, want chat", info.Type)
	}

	reply, err := service.Send(context.Background(), "chat-1", "hello")
	requireNoError(t, err)
	if reply.Content != "hello back" {
		t.Fatalf("reply = %#v", reply)
	}
	if client.req.Model != "gpt-4o" {
		t.Fatalf("request model = %q", client.req.Model)
	}
	if len(client.req.Messages) != 2 {
		t.Fatalf("messages = %#v", client.req.Messages)
	}
	if client.req.Messages[0].Role != "system" || client.req.Messages[0].Content == "" {
		t.Fatalf("system message = %#v", client.req.Messages[0])
	}
	if client.req.Messages[1].Role != "user" || client.req.Messages[1].Content != "hello" {
		t.Fatalf("user message = %#v", client.req.Messages[1])
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "sessions", "chat", "chat-1.json"))
	requireNoError(t, err)
	var saved Transcript
	requireNoError(t, json.Unmarshal(data, &saved))
	if len(saved.Turns) != 2 {
		t.Fatalf("saved turns = %#v", saved.Turns)
	}
}
