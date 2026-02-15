package mobile

import "fmt"

type MobileErrorCode string

const (
	ErrCodePairingExpired      MobileErrorCode = "pairing_expired"
	ErrCodePairingInvalid      MobileErrorCode = "pairing_invalid"
	ErrCodeDeviceNotFound      MobileErrorCode = "mobile_device_not_found"
	ErrCodeDeviceRevoked       MobileErrorCode = "device_revoked"
	ErrCodeDevicePending       MobileErrorCode = "device_pending_approval"
	ErrCodeMaxDevicesReached   MobileErrorCode = "max_devices_reached"
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

func ErrMobileDeviceNotFound(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("mobile device '%s' not found", deviceID),
		Code:    ErrCodeDeviceNotFound,
	}
}

func ErrDeviceRevoked(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("device '%s' has been revoked", deviceID),
		Code:    ErrCodeDeviceRevoked,
	}
}

func ErrDevicePending(deviceID string) error {
	return &MobileError{
		Message: fmt.Sprintf("device '%s' is pending approval", deviceID),
		Code:    ErrCodeDevicePending,
	}
}

func ErrMaxDevicesReached(profileID string, max int) error {
	return &MobileError{
		Message: fmt.Sprintf("maximum devices reached for profile '%s' (%d/%d)", profileID, max, max),
		Code:    ErrCodeMaxDevicesReached,
	}
}

func ErrTokenExpired() error {
	return &MobileError{
		Message: "device token has expired",
		Code:    ErrCodeTokenExpired,
	}
}

func ErrTokenInvalid(cause error) error {
	return &MobileError{
		Message: "invalid device token",
		Code:    ErrCodeTokenInvalid,
		Cause:   cause,
	}
}

func ErrPortInUse(port int, cause error) error {
	return &MobileError{
		Message: fmt.Sprintf("cannot start device server on port %d", port),
		Code:    ErrCodePortInUse,
		Cause:   cause,
	}
}

func ErrMobileNotEnabled(profileID string) error {
	return &MobileError{
		Message: fmt.Sprintf("mobile device linking not enabled for profile '%s'", profileID),
		Code:    ErrCodeMobileNotEnabled,
	}
}

func ErrCapabilityInvalid(cap string) error {
	return &MobileError{
		Message: fmt.Sprintf("invalid device capability '%s'", cap),
		Code:    ErrCodeCapabilityInvalid,
	}
}

func IsMobileError(err error, code MobileErrorCode) bool {
	if e, ok := err.(*MobileError); ok {
		return e.Code == code
	}
	return false
}
