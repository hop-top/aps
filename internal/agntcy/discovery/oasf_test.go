package discovery

import (
	"testing"

	"hop.top/aps/internal/core"
)

func TestGenerateOASFRecord_Full(t *testing.T) {
	profile := &core.Profile{
		ID:          "test-agent",
		DisplayName: "Test Agent",
		Capabilities: []string{"a2a", "webhooks"},
		A2A: &core.A2AConfig{
			ProtocolBinding: "jsonrpc",
			PublicEndpoint:  "http://localhost:8081",
		},
		Identity: &core.IdentityConfig{
			DID: "did:key:z6MkTest",
		},
	}

	record, err := GenerateOASFRecord(profile)
	if err != nil {
		t.Fatalf("GenerateOASFRecord failed: %v", err)
	}

	if record["name"] != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %v", record["name"])
	}

	if record["id"] != "test-agent" {
		t.Errorf("expected id 'test-agent', got %v", record["id"])
	}

	if record["type"] != "agent" {
		t.Errorf("expected type 'agent', got %v", record["type"])
	}

	// Verify capabilities are included
	caps, ok := record["capabilities"].([]string)
	if !ok || len(caps) != 2 {
		t.Errorf("expected 2 capabilities, got %v", record["capabilities"])
	}

	// Verify endpoints
	endpoints, ok := record["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("expected endpoints map")
	}
	if _, ok := endpoints["a2a"]; !ok {
		t.Error("expected a2a endpoint")
	}

	// Verify identity
	id, ok := record["identity"].(map[string]interface{})
	if !ok {
		t.Fatal("expected identity map")
	}
	if id["did"] != "did:key:z6MkTest" {
		t.Errorf("expected DID 'did:key:z6MkTest', got %v", id["did"])
	}
}

func TestGenerateOASFRecord_Minimal(t *testing.T) {
	profile := &core.Profile{
		ID:          "minimal",
		DisplayName: "Minimal Agent",
	}

	record, err := GenerateOASFRecord(profile)
	if err != nil {
		t.Fatalf("GenerateOASFRecord failed: %v", err)
	}

	if record["name"] != "Minimal Agent" {
		t.Errorf("expected name 'Minimal Agent', got %v", record["name"])
	}

	// No endpoints, identity, or capabilities expected
	if _, ok := record["endpoints"]; ok {
		t.Error("expected no endpoints for minimal profile")
	}
	if _, ok := record["identity"]; ok {
		t.Error("expected no identity for minimal profile")
	}
}

func TestGenerateOASFRecord_NilProfile(t *testing.T) {
	_, err := GenerateOASFRecord(nil)
	if err == nil {
		t.Fatal("expected error for nil profile")
	}
}

func TestGenerateOASFRecord_EmptyID(t *testing.T) {
	_, err := GenerateOASFRecord(&core.Profile{DisplayName: "No ID"})
	if err == nil {
		t.Fatal("expected error for empty profile ID")
	}
}

func TestValidateOASFRecord_Valid(t *testing.T) {
	record := map[string]interface{}{
		"schema_version": "1.0",
		"type":           "agent",
		"name":           "Test",
		"id":             "test",
	}

	if err := ValidateOASFRecord(record); err != nil {
		t.Fatalf("ValidateOASFRecord failed: %v", err)
	}
}

func TestValidateOASFRecord_MissingField(t *testing.T) {
	record := map[string]interface{}{
		"schema_version": "1.0",
		"type":           "agent",
		// missing name and id
	}

	if err := ValidateOASFRecord(record); err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestValidateOASFRecord_WrongType(t *testing.T) {
	record := map[string]interface{}{
		"schema_version": "1.0",
		"type":           "service",
		"name":           "Test",
		"id":             "test",
	}

	if err := ValidateOASFRecord(record); err == nil {
		t.Fatal("expected error for wrong type")
	}
}
