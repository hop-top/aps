package trust

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/core"
)

func TestVerifier_NilConfig(t *testing.T) {
	v := NewVerifier(nil)
	if err := v.Verify(context.Background(), "", "", nil); err != nil {
		t.Fatalf("expected nil config to pass: %v", err)
	}
}

func TestVerifier_NoRequireIdentity_NoDID(t *testing.T) {
	cfg := &core.TrustConfig{
		RequireIdentity: false,
	}
	v := NewVerifier(cfg)

	if err := v.Verify(context.Background(), "", "", nil); err != nil {
		t.Fatalf("expected pass when identity not required and no DID: %v", err)
	}
}

func TestVerifier_RequireIdentity_NoDID(t *testing.T) {
	cfg := &core.TrustConfig{
		RequireIdentity: true,
	}
	v := NewVerifier(cfg)

	err := v.Verify(context.Background(), "", "", nil)
	if err == nil {
		t.Fatal("expected error when identity required but no DID")
	}
}

func TestVerifier_RequireIdentity_WithDID(t *testing.T) {
	pub, _, _ := identity.GenerateKeyPair()
	did := encodeDIDKeyForTest(pub)

	cfg := &core.TrustConfig{
		RequireIdentity: true,
	}
	v := NewVerifier(cfg)

	if err := v.Verify(context.Background(), did, "", nil); err != nil {
		t.Fatalf("expected pass with valid DID: %v", err)
	}
}

func TestVerifier_AllowedIssuers_Accepted(t *testing.T) {
	pub, _, _ := identity.GenerateKeyPair()
	did := encodeDIDKeyForTest(pub)

	cfg := &core.TrustConfig{
		RequireIdentity: true,
		AllowedIssuers:  []string{did},
	}
	v := NewVerifier(cfg)

	if err := v.Verify(context.Background(), did, "", nil); err != nil {
		t.Fatalf("expected allowed issuer to pass: %v", err)
	}
}

func TestVerifier_AllowedIssuers_Rejected(t *testing.T) {
	pub, _, _ := identity.GenerateKeyPair()
	did := encodeDIDKeyForTest(pub)

	cfg := &core.TrustConfig{
		RequireIdentity: true,
		AllowedIssuers:  []string{"did:key:zSomeOtherAgent"},
	}
	v := NewVerifier(cfg)

	err := v.Verify(context.Background(), did, "", nil)
	if err == nil {
		t.Fatal("expected rejection for unknown issuer")
	}
}

func TestVerifier_ValidSignature(t *testing.T) {
	pub, priv, _ := identity.GenerateKeyPair()
	did := encodeDIDKeyForTest(pub)

	message := []byte("test payload")
	sig := ed25519.Sign(priv, message)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	cfg := &core.TrustConfig{
		RequireIdentity: true,
	}
	v := NewVerifier(cfg)

	if err := v.Verify(context.Background(), did, sigB64, message); err != nil {
		t.Fatalf("expected valid signature to pass: %v", err)
	}
}

func TestVerifier_InvalidSignature(t *testing.T) {
	pub, priv, _ := identity.GenerateKeyPair()
	did := encodeDIDKeyForTest(pub)

	message := []byte("original")
	sig := ed25519.Sign(priv, message)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	cfg := &core.TrustConfig{
		RequireIdentity: true,
	}
	v := NewVerifier(cfg)

	tampered := []byte("tampered")
	err := v.Verify(context.Background(), did, sigB64, tampered)
	if err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

// encodeDIDKeyForTest creates a did:key from a public key (test helper).
func encodeDIDKeyForTest(pub ed25519.PublicKey) string {
	mcKey := append([]byte{0xed, 0x01}, pub...)
	encoded := "z" + base58EncodeForTest(mcKey)
	return "did:key:" + encoded
}

func base58EncodeForTest(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	zeros := 0
	for _, b := range input {
		if b != 0 {
			break
		}
		zeros++
	}

	size := len(input)*138/100 + 1
	buf := make([]byte, size)
	high := size - 1

	for _, b := range input {
		carry := int(b)
		j := size - 1
		for ; j > high || carry != 0; j-- {
			carry += 256 * int(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}
		high = j
	}

	start := 0
	for start < size && buf[start] == 0 {
		start++
	}

	result := make([]byte, zeros+size-start)
	for i := 0; i < zeros; i++ {
		result[i] = alphabet[0]
	}
	for i := start; i < size; i++ {
		result[zeros+i-start] = alphabet[buf[i]]
	}

	return string(result)
}
