package identity

import (
	"strings"
	"testing"
)

func TestGenerateDID_Key(t *testing.T) {
	dir := t.TempDir()
	// Temporarily override GetProfileDir by using a profile ID that maps there
	// Since GenerateDID calls core.GetProfileDir, we test the DID format instead.

	pub, _, _ := GenerateKeyPair()
	did := encodeDIDKey(pub)

	if !strings.HasPrefix(did, "did:key:z") {
		t.Errorf("expected did:key:z... prefix, got %s", did)
	}
	_ = dir
}

func TestEncodeDIDKey_Format(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	did := encodeDIDKey(pub)

	if !strings.HasPrefix(did, "did:key:z") {
		t.Errorf("expected did:key:z prefix, got: %s", did)
	}
}

func TestResolveDIDKey_RoundTrip(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	did := encodeDIDKey(pub)

	doc, err := ResolveDID(did)
	if err != nil {
		t.Fatalf("ResolveDID failed: %v", err)
	}

	if doc.ID != did {
		t.Errorf("expected doc ID %s, got %s", did, doc.ID)
	}

	if len(doc.VerificationMethod) != 1 {
		t.Fatalf("expected 1 verification method, got %d", len(doc.VerificationMethod))
	}

	vm := doc.VerificationMethod[0]
	if vm.Type != "Ed25519VerificationKey2020" {
		t.Errorf("expected Ed25519VerificationKey2020, got %s", vm.Type)
	}
}

func TestExtractPublicKeyFromDID_RoundTrip(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	did := encodeDIDKey(pub)

	extracted, err := ExtractPublicKeyFromDID(did)
	if err != nil {
		t.Fatalf("ExtractPublicKeyFromDID failed: %v", err)
	}

	if !pub.Equal(extracted) {
		t.Error("extracted public key doesn't match original")
	}
}

func TestResolveDID_Web(t *testing.T) {
	doc, err := ResolveDID("did:web:example.com:agents:test")
	if err != nil {
		t.Fatalf("ResolveDID(did:web) failed: %v", err)
	}

	if doc.ID != "did:web:example.com:agents:test" {
		t.Errorf("unexpected doc ID: %s", doc.ID)
	}
}

func TestResolveDID_Unsupported(t *testing.T) {
	_, err := ResolveDID("did:unknown:test")
	if err == nil {
		t.Fatal("expected error for unsupported DID method")
	}
}

func TestResolveDID_Empty(t *testing.T) {
	_, err := ResolveDID("")
	if err == nil {
		t.Fatal("expected error for empty DID")
	}
}

func TestExtractPublicKeyFromDID_NonKey(t *testing.T) {
	_, err := ExtractPublicKeyFromDID("did:web:example.com")
	if err == nil {
		t.Fatal("expected error for non-key DID")
	}
}

func TestBase58_RoundTrip(t *testing.T) {
	input := []byte{0xed, 0x01, 0x00, 0xff, 0x42, 0x99}
	encoded := base58Encode(input)
	decoded, err := base58Decode(encoded)
	if err != nil {
		t.Fatalf("base58Decode failed: %v", err)
	}

	if len(decoded) != len(input) {
		t.Fatalf("decoded length mismatch: expected %d, got %d", len(input), len(decoded))
	}

	for i := range input {
		if input[i] != decoded[i] {
			t.Errorf("byte %d mismatch: expected %d, got %d", i, input[i], decoded[i])
		}
	}
}
