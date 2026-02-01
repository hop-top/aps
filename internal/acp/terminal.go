package acp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Terminal represents a running terminal/command process
type Terminal struct {
	ID             string
	Command        string
	Args           []string
	WorkingDir     string
	Environment    []string
	Cmd            *exec.Cmd
	Stdin          io.WriteCloser
	Stdout         *bufio.Reader
	Stderr         *bufio.Reader
	ExitCode       int
	Status         string // "created", "running", "exited"
	StartTime      time.Time
	EndTime        time.Time
	OutputBuffer   []string
	mu             sync.RWMutex
	outputMutex    sync.Mutex
}

// TerminalManager manages running terminals
type TerminalManager struct {
	terminals map[string]*Terminal
	mu        sync.RWMutex
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	return &TerminalManager{
		terminals: make(map[string]*Terminal),
	}
}

// CreateTerminal creates and starts a new terminal
func (tm *TerminalManager) CreateTerminal(command string, args []string, workingDir string, env map[string]string) (*Terminal, error) {
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	termID := generateTerminalID()

	// Build environment
	environment := os.Environ()
	if len(env) > 0 {
		for key, value := range env {
			// Remove existing key if present
			for i, e := range environment {
				if len(e) > len(key) && e[:len(key)+1] == key+"=" {
					environment = append(environment[:i], environment[i+1:]...)
					break
				}
			}
			environment = append(environment, key+"="+value)
		}
	}

	// Create command
	cmd := exec.Command(command, args...)
	cmd.Env = environment

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Connect stdin, stdout, stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Create terminal struct
	term := &Terminal{
		ID:          termID,
		Command:     command,
		Args:        args,
		WorkingDir:  workingDir,
		Environment: environment,
		Cmd:         cmd,
		Stdin:       stdin,
		Stdout:      bufio.NewReader(stdout),
		Stderr:      bufio.NewReader(stderr),
		Status:      "created",
		StartTime:   time.Now(),
		OutputBuffer: make([]string, 0),
	}

	// Store terminal
	tm.mu.Lock()
	tm.terminals[termID] = term
	tm.mu.Unlock()

	// Start the command
	if err := cmd.Start(); err != nil {
		tm.mu.Lock()
		delete(tm.terminals, termID)
		tm.mu.Unlock()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	term.Status = "running"

	// Start output reading goroutines
	go term.readOutput(term.Stdout, false)
	go term.readOutput(term.Stderr, true)

	// Wait for process to complete in background
	go func() {
		cmd.Wait()
		term.mu.Lock()
		term.Status = "exited"
		term.EndTime = time.Now()
		if cmd.ProcessState != nil {
			term.ExitCode = cmd.ProcessState.ExitCode()
		}
		term.mu.Unlock()
	}()

	return term, nil
}

// GetTerminal retrieves a terminal by ID
func (tm *TerminalManager) GetTerminal(terminalID string) (*Terminal, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	term, exists := tm.terminals[terminalID]
	if !exists {
		return nil, fmt.Errorf("terminal not found: %s", terminalID)
	}

	return term, nil
}

// GetOutput returns current output from the terminal
func (tm *TerminalManager) GetOutput(terminalID string) (string, error) {
	term, err := tm.GetTerminal(terminalID)
	if err != nil {
		return "", err
	}

	term.outputMutex.Lock()
	defer term.outputMutex.Unlock()

	var output string
	for _, line := range term.OutputBuffer {
		output += line + "\n"
	}

	return output, nil
}

// WaitForExit waits for a terminal to exit and returns the exit code
func (tm *TerminalManager) WaitForExit(terminalID string) (int, error) {
	term, err := tm.GetTerminal(terminalID)
	if err != nil {
		return 0, err
	}

	// Poll for process completion (Wait() is already called in background goroutine)
	for i := 0; i < 1000; i++ {
		term.mu.RLock()
		status := term.Status
		exitCode := term.ExitCode
		term.mu.RUnlock()

		if status == "exited" {
			return exitCode, nil
		}

		time.Sleep(10 * time.Millisecond)
	}

	return 0, fmt.Errorf("timeout waiting for process to exit")
}

// Kill terminates a terminal
func (tm *TerminalManager) Kill(terminalID string) error {
	term, err := tm.GetTerminal(terminalID)
	if err != nil {
		return err
	}

	if term.Cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	if err := term.Cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	term.mu.Lock()
	term.Status = "exited"
	term.EndTime = time.Now()
	term.mu.Unlock()

	return nil
}

// Release closes and removes a terminal
func (tm *TerminalManager) Release(terminalID string) error {
	tm.mu.Lock()
	term, exists := tm.terminals[terminalID]
	tm.mu.Unlock()

	if !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}

	// Kill if still running
	if term.Status == "running" {
		term.Cmd.Process.Kill()
	}

	// Close pipes
	if term.Stdin != nil {
		term.Stdin.Close()
	}

	tm.mu.Lock()
	delete(tm.terminals, terminalID)
	tm.mu.Unlock()

	return nil
}

// ReleaseAll closes all terminals
func (tm *TerminalManager) ReleaseAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for termID := range tm.terminals {
		tm.mu.Unlock()
		tm.Release(termID)
		tm.mu.Lock()
	}
}

// readOutput reads output from a stream and buffers it
func (t *Terminal) readOutput(reader *bufio.Reader, isStderr bool) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				line = fmt.Sprintf("Error reading output: %v\n", err)
			}
		}

		if line != "" {
			t.outputMutex.Lock()
			if isStderr {
				line = "[STDERR] " + line
			}
			t.OutputBuffer = append(t.OutputBuffer, line)
			t.outputMutex.Unlock()
		}

		if err == io.EOF {
			break
		}
	}
}

// GetStatus returns the terminal status
func (t *Terminal) GetStatus() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"id":         t.ID,
		"command":    t.Command,
		"status":     t.Status,
		"exitCode":   t.ExitCode,
		"startTime":  t.StartTime,
		"endTime":    t.EndTime,
		"outputLines": len(t.OutputBuffer),
	}
}

// generateTerminalID generates a unique terminal ID
func generateTerminalID() string {
	return "term_" + generateID()
}

// TerminalRequest represents a terminal operation request
type TerminalRequest struct {
	SessionID        string            `json:"sessionId"`
	Command          string            `json:"command,omitempty"`
	Arguments        []string          `json:"arguments,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	WorkingDirectory string            `json:"workingDirectory,omitempty"`
	TerminalID       string            `json:"terminalId,omitempty"`
}

// TerminalResponse represents a terminal operation response
type TerminalResponse struct {
	TerminalID string `json:"terminalId,omitempty"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exitCode,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}
