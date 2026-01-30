package a2a

import "fmt"

// A2A errors
var (
	// ErrA2ANotEnabled is returned when A2A is not enabled for a profile
	ErrA2ANotEnabled = fmt.Errorf("a2a: not enabled for this profile")

	// ErrInvalidConfig is returned when A2A configuration is invalid
	ErrInvalidConfig = fmt.Errorf("a2a: invalid configuration")

	// ErrTaskNotFound is returned when a task cannot be found
	ErrTaskNotFound = fmt.Errorf("a2a: task not found")

	// ErrAgentCardNotFound is returned when an Agent Card cannot be found
	ErrAgentCardNotFound = fmt.Errorf("a2a: agent card not found")

	// ErrTransportNotSupported is returned when a transport type is not supported
	ErrTransportNotSupported = fmt.Errorf("a2a: transport not supported")

	// ErrTransportFailed is returned when a transport operation fails
	ErrTransportFailed = fmt.Errorf("a2a: transport operation failed")
)

// ErrInvalidAgentCard creates an error for invalid Agent Card data
func ErrInvalidAgentCard(msg string) error {
	return fmt.Errorf("a2a: invalid agent card: %s", msg)
}

// ErrInvalidTaskStatus creates an error for invalid task status
func ErrInvalidTaskStatus(status string) error {
	return fmt.Errorf("a2a: invalid task status: %s", status)
}

// ErrInvalidMessage creates an error for invalid message data
func ErrInvalidMessage(msg string) error {
	return fmt.Errorf("a2a: invalid message: %s", msg)
}

// ErrStorageFailed creates an error for storage operation failures
func ErrStorageFailed(op string, err error) error {
	return fmt.Errorf("a2a: storage %s failed: %w", op, err)
}

// ErrServerFailed creates an error for server operation failures
func ErrServerFailed(op string, err error) error {
	return fmt.Errorf("a2a: server %s failed: %w", op, err)
}

// ErrClientFailed creates an error for client operation failures
func ErrClientFailed(op string, err error) error {
	return fmt.Errorf("a2a: client %s failed: %w", op, err)
}
