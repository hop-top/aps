package logging

import (
	"context"
	"os"

	"charm.land/log/v2"
)

// Logger wraps charmbracelet's logger with structured error logging
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stderr),
	}
}

// Error logs an error with context
func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	l.logger.Error(msg, append(fields, "error", err)...)
}

// ErrorWithCode logs an error with a specific error code
func (l *Logger) ErrorWithCode(msg string, code string, err error, fields ...interface{}) {
	l.logger.Error(msg, append(fields, "code", code, "error", err)...)
}

// Warn logs a warning with context
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.logger.Warn(msg, fields...)
}

// Info logs information
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.logger.Info(msg, fields...)
}

// Debug logs debug information
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.logger.Debug(msg, fields...)
}

// WithContext returns a logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level log.Level) {
	l.logger.SetLevel(level)
}

// Global logger instance
var globalLogger = NewLogger()

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// SetLogger sets the global logger instance
func SetLogger(logger *Logger) {
	globalLogger = logger
}
