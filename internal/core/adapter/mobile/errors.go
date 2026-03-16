package mobile

import "fmt"

type MobileErrorCode string

const (
	ErrCodePairingExpired      MobileErrorCode = "pairing_expired"
	ErrCodePairingInvalid      MobileErrorCode = "pairing_invalid"
	ErrCodeAdapterNotFound      MobileErrorCode = "mobile_adapter_not_found"
	ErrCodeAdapterRevoked       MobileErrorCode = "adapter_revoked"
	ErrCodeAdapterPending       MobileErrorCode = "adapter_pending_approval"
	ErrCodeMaxAdaptersReached   MobileErrorCode = "max_adapters_reached"
	ErrCodeTokenExpired        MobileErrorCode = "token_expired"
	ErrCodeTokenInvalid        MobileErrorCode = "token_invalid"
	ErrCodePortInUse           MobileErrorCode = "port_in_use"
	ErrCodeMobileNotEnabled    MobileErrorCode = "mobile_not_enabled"
	ErrCodeCapabilityInvalid   MobileErrorCode = "capability_invalid"
	ErrCodeRegistryCorrupt     MobileErrorCode = "registry_corrupt"
)

type MobileError struct {
	Message string
	Code    MobileErrorCode
	Cause   error
}

func (e *MobileError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *MobileError) Unwrap() error {
	return e.Cause
}

func ErrPairingExpired() error {
	return &MobileError{
		Message: "pairing code has expired",
		Code:    ErrCodePairingExpired,
	}
}

func ErrPairingInvalid() error {
	return &MobileError{
		Message: "invalid pairing code",
		Code:    ErrCodePairingInvalid,
	}
}

func ErrMobileAdapterNotFound(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("mobile adapter '%s' not found", deviceID),
		Code:    ErrCodeAdapterNotFound,
	}
}

func ErrAdapterRevoked(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("adapter '%s' has been revoked", deviceID),
		Code:    ErrCodeAdapterRevoked,
	}
}

func ErrAdapterPending(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("adapter '%s' is pending approval", deviceID),
		Code:    ErrCodeAdapterPending,
	}
}

func ErrMaxAdaptersReached(profileID string, max int) error {
	return &MobileError{
		Message: fmt.Sprintf("maximum adapters reached for profile '%s' (%d/%d)", profileID, max, max),
		Code:    ErrCodeMaxAdaptersReached,
	}
}

func ErrTokenExpired() error {
	return &MobileError{
		Message: "adapter token has expired",
		Code:    ErrCodeTokenExpired,
	}
}

func ErrTokenInvalid(cause error) error {
	return &MobileError{
		Message: "invalid adapter token",
		Code:    ErrCodeTokenInvalid,
		Cause:   cause,
	}
}

func ErrPortInUse(port int, cause error) error {
	return &MobileError{
		Message: fmt.Sprintf("cannot start adapter server on port %d", port),
		Code:    ErrCodePortInUse,
		Cause:   cause,
	}
}

func ErrMobileNotEnabled(profileID string) error {
	return &MobileError{
		Message: fmt.Sprintf("mobile adapter linking not enabled for profile '%s'", profileID),
		Code:    ErrCodeMobileNotEnabled,
	}
}

func ErrCapabilityInvalid(cap string) error {
	return &MobileError{
		Message: fmt.Sprintf("invalid adapter capability '%s'", cap),
		Code:    ErrCodeCapabilityInvalid,
	}
}

func IsMobileError(err error, code MobileErrorCode) bool {
	if e, ok := err.(*MobileError); ok {
		return e.Code == code
	}
	return false
}
