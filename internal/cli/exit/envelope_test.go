package exit_test

import (
	"fmt"
	"io/fs"
	"testing"

	"hop.top/aps/internal/cli/exit"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/runtime/domain"
)

// asCLIError mirrors the conversion interface kit's RunE middleware
// looks for (kit/go/console/cli/error_render.go).
type asCLIError interface{ AsCLIError() *output.Error }

// TestEnvelopeClassifiesDomainErrors pins that a domain error carries
// its own envelope, so kit's middleware preserves the class instead of
// flattening it.
//
// Without this, kit's toCLIError falls through to its default arm and
// wraps the error as GENERIC/ExitCode=1 BEFORE cmd/aps/main.go's
// exit.Code runs — so `aps profile create <dup>` exited 1 despite the
// error being domain.ErrConflict, and a missing profile exited 1
// instead of the not-found class.
func TestEnvelopeClassifiesDomainErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
		wantExit int
	}{
		{
			"conflict",
			fmt.Errorf("creating profile: %w", domain.ErrConflict),
			output.CodeConflict, 4,
		},
		{
			"not found via domain sentinel",
			fmt.Errorf("profile %q: %w", "ghost", domain.ErrNotFound),
			output.CodeNotFound, 3,
		},
		{
			"not found via fs sentinel",
			fmt.Errorf("failed to read profile ghost: %w", fs.ErrNotExist),
			output.CodeNotFound, 3,
		},
		{
			"unauthorized",
			fmt.Errorf("auth: %w", exit.ErrUnauthorized),
			output.CodeUnauthorized, 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enveloped := exit.Envelope(tc.err)

			ce, ok := enveloped.(asCLIError)
			if !ok {
				t.Fatalf("%T does not implement AsCLIError; kit will flatten it to GENERIC/1", enveloped)
			}
			got := ce.AsCLIError()
			if got == nil {
				t.Fatal("AsCLIError returned nil")
			}
			if got.Code != tc.wantCode || got.ExitCode != tc.wantExit {
				t.Fatalf("envelope = %s/%d, want %s/%d",
					got.Code, got.ExitCode, tc.wantCode, tc.wantExit)
			}
			// The original message must survive for correlation.
			if got.Message == "" {
				t.Error("envelope dropped the diagnostic message")
			}
			// exit.Code must agree with the envelope it produced.
			if code := exit.Code(enveloped); code != tc.wantExit {
				t.Errorf("exit.Code = %d, want %d (must agree with envelope)", code, tc.wantExit)
			}
		})
	}
}

// TestEnvelopeLeavesUnclassifiedErrorsAlone pins that a plain error is
// returned unchanged: inventing a class for an unknown failure would
// tell agents a retry is safe when nothing established that.
func TestEnvelopeLeavesUnclassifiedErrorsAlone(t *testing.T) {
	plain := fmt.Errorf("something broke")
	if got := exit.Envelope(plain); got != plain {
		t.Fatalf("Envelope rewrapped an unclassified error: %#v", got)
	}
	if got := exit.Envelope(nil); got != nil {
		t.Fatalf("Envelope(nil) = %v, want nil", got)
	}
}

// TestEnvelopePreservesExistingEnvelopes pins that an error already
// carrying an envelope is passed through untouched, so a command that
// crafted a precise diagnostic keeps it.
func TestEnvelopePreservesExistingEnvelopes(t *testing.T) {
	orig := &output.Error{
		Code:         output.CodeUsage,
		Message:      "bad flag",
		SuggestedFix: "see --help",
		ExitCode:     2,
	}
	got := exit.Envelope(orig)
	ce, ok := got.(asCLIError)
	if !ok {
		t.Fatalf("%T lost its envelope", got)
	}
	if e := ce.AsCLIError(); e.Code != output.CodeUsage || e.ExitCode != 2 {
		t.Fatalf("envelope = %s/%d, want USAGE/2", e.Code, e.ExitCode)
	}
}
