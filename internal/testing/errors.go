package testing

import "errors"

var (
	// ErrTimeoutWaitingForCondition is returned when a WaitFor condition is not met within the timeout
	ErrTimeoutWaitingForCondition = errors.New("timeout waiting for condition")

	// ErrProfileNotFound is returned when a profile is not found
	ErrProfileNotFound = errors.New("profile not found")

	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrTransportClosed is returned when trying to use a closed transport
	ErrTransportClosed = errors.New("transport closed")

	// ErrInvalidTaskID is returned when a task ID is invalid
	ErrInvalidTaskID = errors.New("invalid task ID")

	// ErrInvalidMessage is returned when a message is invalid
	ErrInvalidMessage = errors.New("invalid message")

	// ErrMockNotConfigured is returned when a mock is not properly configured
	ErrMockNotConfigured = errors.New("mock not configured")
)
