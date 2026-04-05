package logging

import (
	"bytes"
	"errors"
	"testing"

	"charm.land/log/v2"
)

// TestNewLogger creates and verifies a new logger
func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.logger == nil {
		t.Fatal("expected non-nil internal logger")
	}
}

// TestLoggerError tests error logging with context
func TestLoggerError(t *testing.T) {
	// Create a logger with a buffer to capture output
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	testErr := errors.New("test error")
	logger.Error("test message", testErr, "field1", "value1")

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty log output")
	}
	if !bytes.Contains([]byte(output), []byte("test message")) {
		t.Errorf("expected 'test message' in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("test error")) {
		t.Errorf("expected 'test error' in output, got: %s", output)
	}
}

// TestLoggerErrorWithCode tests error logging with error code
func TestLoggerErrorWithCode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	testErr := errors.New("profile not found")
	logger.ErrorWithCode("Resource missing", "PROFILE_NOT_FOUND", testErr,
		"profile_id", "test-profile")

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Resource missing")) {
		t.Errorf("expected 'Resource missing' in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("PROFILE_NOT_FOUND")) {
		t.Errorf("expected 'PROFILE_NOT_FOUND' in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("test-profile")) {
		t.Errorf("expected 'test-profile' in output, got: %s", output)
	}
}

// TestLoggerInfo tests info logging
func TestLoggerInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	logger.Info("info message", "key", "value")

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty log output")
	}
	if !bytes.Contains([]byte(output), []byte("info message")) {
		t.Errorf("expected 'info message' in output, got: %s", output)
	}
}

// TestLoggerWarn tests warning logging
func TestLoggerWarn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	logger.Warn("warning message", "key", "value")

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty log output")
	}
	if !bytes.Contains([]byte(output), []byte("warning message")) {
		t.Errorf("expected 'warning message' in output, got: %s", output)
	}
}

// TestLoggerDebug tests debug logging
func TestLoggerDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	// Set debug level to ensure debug messages are logged
	logger.logger.SetLevel(log.DebugLevel)

	logger.Debug("debug message", "key", "value")

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty log output")
	}
	if !bytes.Contains([]byte(output), []byte("debug message")) {
		t.Errorf("expected 'debug message' in output, got: %s", output)
	}
}

// TestGlobalLogger tests global logger instance
func TestGlobalLogger(t *testing.T) {
	logger1 := GetLogger()
	logger2 := GetLogger()

	if logger1 != logger2 {
		t.Error("expected same logger instance from GetLogger")
	}

	// Create a new logger and set it
	newLogger := NewLogger()
	SetLogger(newLogger)
	defer func() {
		// Restore original logger
		SetLogger(NewLogger())
	}()

	logger3 := GetLogger()
	if logger3 != newLogger {
		t.Error("expected SetLogger to update global logger")
	}
}

// TestLoggerStructuredFields tests that structured fields are logged correctly
func TestLoggerStructuredFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &Logger{
		logger: log.New(buf),
	}

	err := errors.New("execution failed")
	logger.Error("Action failed", err,
		"action_id", "test-action",
		"profile_id", "test-profile",
		"exit_code", 1)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test-action")) {
		t.Errorf("expected 'test-action' in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("test-profile")) {
		t.Errorf("expected 'test-profile' in output, got: %s", output)
	}
}
