package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if len(pub) != 32 {
		t.Errorf("expected 32-byte public key, got %d", len(pub))
	}

	if len(priv) != 64 {
		t.Errorf("expected 64-byte private key, got %d", len(priv))
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Save keys to temp dir
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")
	if err := SaveKeyPair(keyPath, pub, priv); err != nil {
		t.Fatalf("SaveKeyPair failed: %v", err)
	}

	message := []byte("hello agent world")

	sig, err := SignMessage(keyPath, message)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	if !VerifySignature(pub, message, sig) {
		t.Fatal("expected signature to verify")
	}
}

func TestVerifySignature_Tampered(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")
	SaveKeyPair(keyPath, pub, priv)

	message := []byte("original message")
	sig, _ := SignMessage(keyPath, message)

	tampered := []byte("tampered message")
	if VerifySignature(pub, tampered, sig) {
		t.Fatal("expected tampered signature to fail verification")
	}
}

func TestSaveLoadKeyPair(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	if err := SaveKeyPair(keyPath, pub, priv); err != nil {
		t.Fatalf("SaveKeyPair failed: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("private key file missing: %v", err)
	}
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Fatalf("public key file missing: %v", err)
	}

	// Load and verify
	loadedPriv, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}

	loadedPub, err := LoadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPublicKey failed: %v", err)
	}

	if !pub.Equal(loadedPub) {
		t.Error("loaded public key doesn't match original")
	}

	if !priv.Equal(loadedPriv) {
		t.Error("loaded private key doesn't match original")
	}
}
