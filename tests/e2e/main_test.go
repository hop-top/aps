package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	apsBinary string
	// binDir is a per-run temp dir holding the compiled binary, so
	// concurrent runs of this package never share a path.
	binDir string
)

func TestMain(m *testing.M) {
	// 1. Compile the binary
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile aps binary: %v\n", err)
		os.Exit(1)
	}

	// 2. Run tests
	code := m.Run()

	// 3. Cleanup
	_ = os.RemoveAll(binDir)

	os.Exit(code)
}

func compileBinary() error {
	// Determine binary name based on OS
	binName := "aps-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	var (
		err     error
		rootDir string
	)
	binDir, err = os.MkdirTemp("", "aps-e2e-*")
	if err != nil {
		return err
	}
	apsBinary = filepath.Join(binDir, binName)

	// Build from project root (../../)
	// Assuming tests/e2e is 2 levels deep
	rootDir, err = filepath.Abs("../../")
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Compiling aps binary to %s...\n", apsBinary)
	return cmd.Run()
}
