// Package exit maps domain errors to canonical CLI exit codes.
//
// Convention §8.1 specifies:
//
//	0 success
//	1 generic error
//	2 usage / bad flags     (cobra handles this)
//	3 not found             (resource lookup failed)
//	4 conflict / exists     (uniqueness violation)
//	5 unauthorized          (auth failure)
//	6 permission denied
//	7 timeout
//	8 cancelled
//
// kit/go/console/cli already declares matching ExitCode constants
// (kitcli.ExitNotFound, ExitConflict, ExitAuth, …). This package
// reuses those values and adds an aps-local sentinel for unauthorized
// (the runtime/domain package doesn't ship one) plus a Code(err)
// classifier that consumers (currently cmd/aps/main) call to translate
// any error returned from cobra RunE into the right exit code.
package exit

import (
	"errors"
	"io/fs"
	"os/exec"

	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/runtime/domain"
)

// ErrUnauthorized is the aps-local sentinel for auth failures. Wrap
// with fmt.Errorf("...: %w", exit.ErrUnauthorized, err) to opt-in to
// exit code 5 from a cobra RunE.
//
// runtime/domain does not declare this; if it ever does, alias here.
var ErrUnauthorized = errors.New("unauthorized")

// classifiedError pairs a domain error with the structured envelope
// its class implies. It implements AsCLIError, the interface kit's RunE
// middleware looks for, while keeping the original error reachable
// through Unwrap so errors.Is/As still match the sentinel.
type classifiedError struct {
	err      error
	envelope *output.Error
}

func (e *classifiedError) Error() string { return e.err.Error() }
func (e *classifiedError) Unwrap() error { return e.err }

// AsCLIError satisfies kit's conversion interface.
func (e *classifiedError) AsCLIError() *output.Error { return e.envelope }

// Envelope attaches the structured envelope implied by err's domain
// class, so the class survives kit's RunE middleware.
//
// Kit's toCLIError only preserves a class when the error already
// implements AsCLIError; every other error is wrapped as
// GENERIC/ExitCode=1 (kit/go/console/cli/error_render.go). That
// wrapping happens BEFORE cmd/aps/main.go calls Code(), so an
// unenveloped domain.ErrConflict reached the process exit as 1 rather
// than 4, and a missing profile as 1 rather than 3 — the exit code and
// the error text disagreed, and agents branching on $? could not
// distinguish "already exists" from any other failure.
//
// Errors that already carry an envelope pass through untouched, and an
// unclassified error is returned unchanged rather than being given an
// invented class.
func Envelope(err error) error {
	if err == nil {
		return nil
	}
	var existing interface{ AsCLIError() *output.Error }
	if errors.As(err, &existing) {
		return err
	}

	var code string
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		code = output.CodeNotFound
	case errors.Is(err, domain.ErrConflict):
		code = output.CodeConflict
	case errors.Is(err, ErrUnauthorized):
		code = output.CodeUnauthorized
	default:
		// Unclassified: leave it alone so kit renders GENERIC/1.
		return err
	}

	return &classifiedError{
		err: err,
		envelope: &output.Error{
			Code:     code,
			Message:  err.Error(),
			ExitCode: Code(err),
		},
	}
}

// Code returns the canonical exit code for err.
//
// Mapping (errors.Is unwrap-aware):
//
//	domain.ErrNotFound, fs.ErrNotExist → 3
//	domain.ErrConflict                 → 4
//	ErrUnauthorized                    → 5
//	nil                                → 0
//	anything else                      → 1
func Code(err error) int {
	if err == nil {
		return int(kitcli.ExitOK)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	var outputErr *output.Error
	if errors.As(err, &outputErr) && outputErr.ExitCode != 0 {
		return outputErr.ExitCode
	}
	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, fs.ErrNotExist):
		return int(kitcli.ExitNotFound)
	case errors.Is(err, domain.ErrConflict):
		return int(kitcli.ExitConflict)
	case errors.Is(err, ErrUnauthorized):
		return int(kitcli.ExitAuth)
	default:
		return int(kitcli.ExitError)
	}
}
