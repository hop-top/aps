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
