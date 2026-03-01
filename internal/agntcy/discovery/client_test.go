package discovery

import (
	"context"
	"testing"

	"hop.top/aps/internal/core"
)

func TestNewClient(t *testing.T) {
	cfg := &core.DirectoryConfig{
		Endpoint: "https://dir.example.com",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.endpoint != "https://dir.example.com" {
		t.Errorf("expected endpoint 'https://dir.example.com', got %s", client.endpoint)
	}
}

func TestNewClient_Default(t *testing.T) {
	cfg := &core.DirectoryConfig{}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.endpoint != "https://dir.agntcy.org" {
		t.Errorf("expected default endpoint, got %s", client.endpoint)
	}
}

func TestNewClient_NilConfig(t *testing.T) {
	_, err := NewClient(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestClient_Register(t *testing.T) {
	cfg := &core.DirectoryConfig{}
	client, _ := NewClient(cfg)

	profile := &core.Profile{
		ID:          "test-agent",
		DisplayName: "Test Agent",
	}

	record, err := client.Register(context.Background(), profile)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if record["name"] != "Test Agent" {
		t.Errorf("expected name in record, got %v", record["name"])
	}
}

func TestClient_Discover_EmptyCapability(t *testing.T) {
	cfg := &core.DirectoryConfig{}
	client, _ := NewClient(cfg)

	_, err := client.Discover(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty capability")
	}
}

func TestClient_Deregister_EmptyID(t *testing.T) {
	cfg := &core.DirectoryConfig{}
	client, _ := NewClient(cfg)

	err := client.Deregister(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty profile ID")
	}
}
