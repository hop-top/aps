package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var apsBinary string

func TestMain(m *testing.M) {
	// 1. Compile the binary
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile aps binary: %v\n", err)
		os.Exit(1)
	}

	// 2. Run tests
	code := m.Run()

	// 3. Cleanup
	os.Remove(apsBinary)

	os.Exit(code)
}

func compileBinary() error {
	// Determine binary name based on OS
	binName := "aps-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	tmpDir := os.TempDir()
	apsBinary = filepath.Join(tmpDir, binName)

	// Build from project root (../../)
	// Assuming tests/e2e is 2 levels deep
	rootDir, err := filepath.Abs("../../")
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Compiling aps binary to %s...\n", apsBinary)
	return cmd.Run()
}
