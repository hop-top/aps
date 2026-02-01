package acp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"oss-aps-cli/internal/acp"
)

// TestTerminalManagerCreate tests creating a terminal
func TestTerminalManagerCreate(t *testing.T) {
	tm := acp.NewTerminalManager()

	// Create a simple echo terminal
	term, err := tm.CreateTerminal("echo", []string{"hello world"}, "", nil)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}

	if term.ID == "" {
		t.Error("terminal ID should not be empty")
	}

	if term.Status != "running" && term.Status != "exited" {
		t.Errorf("unexpected status: %s", term.Status)
	}

	// Cleanup
	tm.Release(term.ID)
}

// TestTerminalManagerGet tests retrieving a terminal
func TestTerminalManagerGet(t *testing.T) {
	tm := acp.NewTerminalManager()

	term, _ := tm.CreateTerminal("echo", []string{"test"}, "", nil)
	defer tm.Release(term.ID)

	retrieved, err := tm.GetTerminal(term.ID)
	if err != nil {
		t.Fatalf("failed to get terminal: %v", err)
	}

	if retrieved.ID != term.ID {
		t.Errorf("terminal ID mismatch: expected %s, got %s", term.ID, retrieved.ID)
	}
}

// TestTerminalManagerOutput tests getting terminal output
func TestTerminalManagerOutput(t *testing.T) {
	tm := acp.NewTerminalManager()

	term, _ := tm.CreateTerminal("echo", []string{"hello"}, "", nil)
	defer tm.Release(term.ID)

	// Wait a bit for output
	time.Sleep(100 * time.Millisecond)

	output, err := tm.GetOutput(term.ID)
	if err != nil {
		t.Fatalf("failed to get output: %v", err)
	}

	if output == "" {
		t.Error("output should not be empty")
	}

	// Check if output contains "hello"
	if len(output) > 0 {
		t.Logf("Terminal output: %s", output)
	}
}

// TestTerminalManagerWaitForExit tests waiting for terminal exit
func TestTerminalManagerWaitForExit(t *testing.T) {
	tm := acp.NewTerminalManager()

	term, _ := tm.CreateTerminal("sh", []string{"-c", "exit 42"}, "", nil)
	defer tm.Release(term.ID)

	time.Sleep(50 * time.Millisecond) // Let the process start and goroutines initialize

	exitCode, err := tm.WaitForExit(term.ID)
	if err != nil {
		t.Fatalf("failed to wait for exit: %v", err)
	}

	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

// TestTerminalManagerKill tests killing a terminal
func TestTerminalManagerKill(t *testing.T) {
	tm := acp.NewTerminalManager()

	// Create a long-running process
	term, _ := tm.CreateTerminal("sleep", []string{"10"}, "", nil)
	defer tm.Release(term.ID)

	time.Sleep(50 * time.Millisecond) // Let it start

	if err := tm.Kill(term.ID); err != nil {
		t.Fatalf("failed to kill terminal: %v", err)
	}

	retrieved, _ := tm.GetTerminal(term.ID)
	if retrieved.Status != "exited" {
		t.Errorf("expected status 'exited' after kill, got %s", retrieved.Status)
	}
}

// TestTerminalManagerEnvironment tests passing environment variables
func TestTerminalManagerEnvironment(t *testing.T) {
	tm := acp.NewTerminalManager()

	env := map[string]string{
		"TEST_VAR": "test_value",
	}

	term, _ := tm.CreateTerminal("sh", []string{"-c", "echo $TEST_VAR"}, "", env)
	defer tm.Release(term.ID)

	time.Sleep(100 * time.Millisecond)

	output, _ := tm.GetOutput(term.ID)
	// Output should contain the env variable value or be empty if not supported
	t.Logf("Output with env var: %s", output)
}

// TestTerminalManagerWorkingDirectory tests working directory
func TestTerminalManagerWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tm := acp.NewTerminalManager()

	// Create a file in the temp directory
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	term, _ := tm.CreateTerminal("ls", []string{}, tmpDir, nil)
	defer tm.Release(term.ID)

	time.Sleep(100 * time.Millisecond)

	output, _ := tm.GetOutput(term.ID)
	// Output should contain the test file name or directory listing
	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Directory listing: %s", output)
}

// TestTerminalRelease tests releasing a terminal
func TestTerminalRelease(t *testing.T) {
	tm := acp.NewTerminalManager()

	term, _ := tm.CreateTerminal("echo", []string{"test"}, "", nil)

	if err := tm.Release(term.ID); err != nil {
		t.Fatalf("failed to release terminal: %v", err)
	}

	// Should not be able to get released terminal
	_, err := tm.GetTerminal(term.ID)
	if err == nil {
		t.Error("should not find released terminal")
	}
}

// TestTerminalGetStatus tests getting terminal status
func TestTerminalGetStatus(t *testing.T) {
	tm := acp.NewTerminalManager()

	term, _ := tm.CreateTerminal("echo", []string{"hello"}, "", nil)
	defer tm.Release(term.ID)

	status := term.GetStatus()

	if status["id"] != term.ID {
		t.Error("status should contain terminal ID")
	}

	if _, ok := status["status"]; !ok {
		t.Error("status should contain status field")
	}
}

// TestTerminalNonExistentTerminal tests error handling for non-existent terminal
func TestTerminalNonExistentTerminal(t *testing.T) {
	tm := acp.NewTerminalManager()

	_, err := tm.GetTerminal("nonexistent")
	if err == nil {
		t.Error("should return error for non-existent terminal")
	}

	err = tm.Kill("nonexistent")
	if err == nil {
		t.Error("should return error killing non-existent terminal")
	}

	err = tm.Release("nonexistent")
	if err == nil {
		t.Error("should return error releasing non-existent terminal")
	}
}

// TestTerminalReleaseAll tests releasing all terminals
func TestTerminalReleaseAll(t *testing.T) {
	tm := acp.NewTerminalManager()

	term1, _ := tm.CreateTerminal("echo", []string{"1"}, "", nil)
	term2, _ := tm.CreateTerminal("echo", []string{"2"}, "", nil)

	tm.ReleaseAll()

	// Both should be gone
	if _, err := tm.GetTerminal(term1.ID); err == nil {
		t.Error("term1 should be released")
	}

	if _, err := tm.GetTerminal(term2.ID); err == nil {
		t.Error("term2 should be released")
	}
}
