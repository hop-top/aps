package identity

import (
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"
)

func newTestProfileWithIdentity(t *testing.T) (*core.Profile, string) {
	t.Helper()
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "identity.key")

	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if err := SaveKeyPair(keyPath, pub, priv); err != nil {
		t.Fatalf("SaveKeyPair failed: %v", err)
	}

	did := encodeDIDKey(pub)

	profile := &core.Profile{
		ID:          "test-agent",
		DisplayName: "Test Agent",
		Identity: &core.IdentityConfig{
			DID:     did,
			KeyPath: keyPath,
		},
	}

	return profile, dir
}

func TestIssueBadge(t *testing.T) {
	profile, _ := newTestProfileWithIdentity(t)

	badge, err := IssueBadge(profile, "invoice-processing")
	if err != nil {
		t.Fatalf("IssueBadge failed: %v", err)
	}

	if badge.Issuer != profile.Identity.DID {
		t.Errorf("expected issuer %s, got %s", profile.Identity.DID, badge.Issuer)
	}

	if badge.Subject.Capability != "invoice-processing" {
		t.Errorf("expected capability 'invoice-processing', got %s", badge.Subject.Capability)
	}

	if badge.Proof == nil {
		t.Fatal("expected proof to be set")
	}

	if badge.Proof.Type != "Ed25519Signature2020" {
		t.Errorf("expected proof type 'Ed25519Signature2020', got %s", badge.Proof.Type)
	}
}

func TestIssueBadge_NoIdentity(t *testing.T) {
	profile := &core.Profile{
		ID: "no-identity",
	}

	_, err := IssueBadge(profile, "test")
	if err == nil {
		t.Fatal("expected error for profile without identity")
	}
}

func TestSaveAndVerifyBadge(t *testing.T) {
	profile, dir := newTestProfileWithIdentity(t)

	badge, err := IssueBadge(profile, "data-analysis")
	if err != nil {
		t.Fatalf("IssueBadge failed: %v", err)
	}

	badgePath := filepath.Join(dir, "badge.json")
	if err := SaveBadge(badge, badgePath); err != nil {
		t.Fatalf("SaveBadge failed: %v", err)
	}

	result, err := VerifyBadge(badgePath)
	if err != nil {
		t.Fatalf("VerifyBadge failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected badge to be valid, got error: %s", result.Error)
	}

	if result.Capability != "data-analysis" {
		t.Errorf("expected capability 'data-analysis', got %s", result.Capability)
	}

	if result.Issuer != profile.Identity.DID {
		t.Errorf("expected issuer %s, got %s", profile.Identity.DID, result.Issuer)
	}
}

func TestVerifyBadge_Tampered(t *testing.T) {
	profile, dir := newTestProfileWithIdentity(t)

	badge, _ := IssueBadge(profile, "test-cap")

	// Tamper with capability after signing
	badge.Subject.Capability = "tampered-cap"

	badgePath := filepath.Join(dir, "tampered.json")
	SaveBadge(badge, badgePath)

	result, err := VerifyBadge(badgePath)
	if err != nil {
		t.Fatalf("VerifyBadge failed: %v", err)
	}

	if result.Valid {
		t.Fatal("expected tampered badge to be invalid")
	}
}

func TestVerifyBadge_NonexistentFile(t *testing.T) {
	result, err := VerifyBadge("/nonexistent/badge.json")
	if err != nil {
		t.Fatalf("VerifyBadge should return result, not error: %v", err)
	}

	if result.Valid {
		t.Fatal("expected invalid result for nonexistent file")
	}
}
