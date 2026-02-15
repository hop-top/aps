package messenger

import "fmt"

// ErrorCode identifies messenger error categories.
type ErrorCode string

const (
	ErrCodeLinkNotFound      ErrorCode = "link_not_found"
	ErrCodeLinkAlreadyExists ErrorCode = "link_already_exists"
	ErrCodeMappingConflict   ErrorCode = "mapping_conflict"
	ErrCodeUnknownChannel    ErrorCode = "unknown_channel"
	ErrCodeActionNotFound    ErrorCode = "action_not_found"
	ErrCodeActionFailed      ErrorCode = "action_failed"
	ErrCodeIsolationViolation ErrorCode = "isolation_violation"
	ErrCodeRoutingFailed     ErrorCode = "routing_failed"
	ErrCodeNormalizeFailed   ErrorCode = "normalize_failed"
	ErrCodeLogWriteFailed    ErrorCode = "log_write_failed"
	ErrCodeMissingSecret     ErrorCode = "missing_secret"
	ErrCodeInvalidMapping    ErrorCode = "invalid_mapping"
)

// MessengerError is the structured error type for messenger operations.
type MessengerError struct {
	Name    string
	Message string
	Code    ErrorCode
	Cause   error
}

func (e *MessengerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func (e *MessengerError) Unwrap() error {
	return e.Cause
}

func ErrLinkNotFound(messenger, profile string) error {
	return &MessengerError{
		Name:    messenger,
		Message: fmt.Sprintf("no link found for profile '%s'", profile),
		Code:    ErrCodeLinkNotFound,
	}
}

func ErrLinkAlreadyExists(messenger, profile string) error {
	return &MessengerError{
		Name:    messenger,
		Message: fmt.Sprintf("already linked to profile '%s'", profile),
		Code:    ErrCodeLinkAlreadyExists,
	}
}

func ErrMappingConflict(channelID, existingProfile, existingAction string) error {
	return &MessengerError{
		Name:    channelID,
		Message: fmt.Sprintf("channel already mapped to %s=%s", existingProfile, existingAction),
		Code:    ErrCodeMappingConflict,
	}
}

func ErrUnknownChannel(messengerName, channelID string) error {
	return &MessengerError{
		Name:    messengerName,
		Message: fmt.Sprintf("no mapping for channel '%s'", channelID),
		Code:    ErrCodeUnknownChannel,
	}
}

func ErrActionNotFound(profile, action string) error {
	return &MessengerError{
		Name:    profile,
		Message: fmt.Sprintf("action '%s' not found", action),
		Code:    ErrCodeActionNotFound,
	}
}

func ErrActionFailed(profile, action string, cause error) error {
	return &MessengerError{
		Name:    profile,
		Message: fmt.Sprintf("action '%s' failed", action),
		Code:    ErrCodeActionFailed,
		Cause:   cause,
	}
}

func ErrIsolationViolation(channelID, expectedProfile, attemptedProfile string) error {
	return &MessengerError{
		Name:    channelID,
		Message: fmt.Sprintf("isolation violation: mapped to '%s', attempted '%s'", expectedProfile, attemptedProfile),
		Code:    ErrCodeIsolationViolation,
	}
}

func ErrRoutingFailed(msgID string, cause error) error {
	return &MessengerError{
		Name:    msgID,
		Message: "routing failed",
		Code:    ErrCodeRoutingFailed,
		Cause:   cause,
	}
}

func ErrNormalizeFailed(platform string, cause error) error {
	return &MessengerError{
		Name:    platform,
		Message: "normalization failed",
		Code:    ErrCodeNormalizeFailed,
		Cause:   cause,
	}
}

func ErrInvalidMapping(mapping string, cause error) error {
	return &MessengerError{
		Name:    mapping,
		Message: "invalid mapping format",
		Code:    ErrCodeInvalidMapping,
		Cause:   cause,
	}
}

func ErrMissingSecret(name string) error {
	return &MessengerError{
		Name:    name,
		Message: fmt.Sprintf("required secret '%s' not set", name),
		Code:    ErrCodeMissingSecret,
	}
}

// Error type checkers

func IsMappingConflict(err error) bool {
	if e, ok := err.(*MessengerError); ok {
		return e.Code == ErrCodeMappingConflict
	}
	return false
}

func IsLinkNotFound(err error) bool {
	if e, ok := err.(*MessengerError); ok {
		return e.Code == ErrCodeLinkNotFound
	}
	return false
}

func IsUnknownChannel(err error) bool {
	if e, ok := err.(*MessengerError); ok {
		return e.Code == ErrCodeUnknownChannel
	}
	return false
}

func IsIsolationViolation(err error) bool {
	if e, ok := err.(*MessengerError); ok {
		return e.Code == ErrCodeIsolationViolation
	}
	return false
}

func IsActionNotFound(err error) bool {
	if e, ok := err.(*MessengerError); ok {
		return e.Code == ErrCodeActionNotFound
	}
	return false
}
