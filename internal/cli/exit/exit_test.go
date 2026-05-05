package exit_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"testing"

	"hop.top/aps/internal/cli/exit"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/runtime/domain"
)

func TestCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"plain", errors.New("boom"), 1},
		{"domain.ErrNotFound", domain.ErrNotFound, 3},
		{"wrapped not found",
			fmt.Errorf("profile %q: %w", "foo", domain.ErrNotFound), 3},
		{"fs.ErrNotExist",
			fmt.Errorf("read: %w", fs.ErrNotExist), 3},
		{"domain.ErrConflict", domain.ErrConflict, 4},
		{"wrapped conflict",
			fmt.Errorf("profile already exists: %w", domain.ErrConflict), 4},
		{"unauthorized", exit.ErrUnauthorized, 5},
		{"wrapped unauthorized",
			fmt.Errorf("auth: %w", exit.ErrUnauthorized), 5},
		{"child exit code",
			&exec.ExitError{ProcessState: exitProcessState(t, 42)}, 42},
		{"wrapped child exit code",
			fmt.Errorf("running command: %w", &exec.ExitError{ProcessState: exitProcessState(t, 37)}), 37},
		{"structured output error",
			&output.Error{Code: output.CodeGeneric, Message: "child failed", ExitCode: 42}, 42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := exit.Code(tc.err)
			if got != tc.want {
				t.Fatalf("Code(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func exitProcessState(t *testing.T, code int) *os.ProcessState {
	t.Helper()

	cmd := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error for code %d, got %v", code, err)
	}
	return exitErr.ProcessState
}
