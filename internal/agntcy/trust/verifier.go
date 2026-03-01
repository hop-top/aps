package trust

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"hop.top/aps/internal/agntcy/identity"
	"hop.top/aps/internal/core"
)

// Verifier performs inbound trust verification on A2A requests.
type Verifier struct {
	cfg *core.TrustConfig
}

// NewVerifier creates a new trust verifier.
func NewVerifier(cfg *core.TrustConfig) *Verifier {
	return &Verifier{cfg: cfg}
}

// VerifyHTTP verifies trust from HTTP request headers.
// Extracts sender DID from X-Agent-DID and signature from X-Agent-Signature.
func (v *Verifier) VerifyHTTP(ctx context.Context, r *http.Request, body []byte) error {
	if v.cfg == nil {
		return nil
	}

	senderDID := r.Header.Get("X-Agent-DID")
	signatureB64 := r.Header.Get("X-Agent-Signature")

	return v.verify(senderDID, signatureB64, body)
}

// Verify verifies trust from extracted header values.
func (v *Verifier) Verify(ctx context.Context, senderDID, signatureB64 string, body []byte) error {
	if v.cfg == nil {
		return nil
	}

	return v.verify(senderDID, signatureB64, body)
}

func (v *Verifier) verify(senderDID, signatureB64 string, body []byte) error {
	// If identity not required and no DID provided, pass through
	if !v.cfg.RequireIdentity && senderDID == "" {
		return nil
	}

	// If identity required but no DID provided, reject
	if v.cfg.RequireIdentity && senderDID == "" {
		return fmt.Errorf("trust verification failed: X-Agent-DID header required")
	}

	// Check allowed issuers
	if len(v.cfg.AllowedIssuers) > 0 {
		allowed := false
		for _, issuer := range v.cfg.AllowedIssuers {
			if issuer == senderDID {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("trust verification failed: DID %s not in allowed issuers", senderDID)
		}
	}

	// Verify signature if provided
	if signatureB64 != "" {
		sig, err := base64.StdEncoding.DecodeString(signatureB64)
		if err != nil {
			return fmt.Errorf("trust verification failed: invalid signature encoding: %w", err)
		}

		pubKey, err := identity.ExtractPublicKeyFromDID(senderDID)
		if err != nil {
			return fmt.Errorf("trust verification failed: cannot extract public key from DID: %w", err)
		}

		if !identity.VerifySignature(pubKey, body, sig) {
			return fmt.Errorf("trust verification failed: signature invalid for DID %s", senderDID)
		}
	} else if v.cfg.RequireIdentity {
		// DID present but no signature — still allow if we only require identity presence
		// A stricter mode could require signatures; for now, DID presence suffices
	}

	return nil
}
