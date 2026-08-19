package core

import (
	"errors"
	"fmt"
	"strings"

	"hop.top/aps/internal/core/msgroute"
)

// ErrServiceNotRouted is returned when a service declares no routing block.
var ErrServiceNotRouted = errors.New("service declares no routing block")

// IsServiceNotRouted reports whether err means the service has no route table.
func IsServiceNotRouted(err error) bool {
	return errors.Is(err, ErrServiceNotRouted)
}

// HasRouteTable reports whether the message service dispatches through a
// sender route table instead of option default_action.
func HasRouteTable(service *ServiceConfig) bool {
	return service != nil && service.Routing != nil
}

// LoadServiceRouteTable compiles the service's routing block. Relative
// file/contact paths resolve against the services directory; the service
// adapter drives sender-key normalization; the service profile is the
// default route profile.
func LoadServiceRouteTable(service *ServiceConfig) (*msgroute.Table, error) {
	if service == nil {
		return nil, fmt.Errorf("service is required")
	}
	if service.Routing == nil {
		return nil, fmt.Errorf("service %s: %w", service.ID, ErrServiceNotRouted)
	}
	baseDir, err := GetServicesDir()
	if err != nil {
		return nil, fmt.Errorf("resolve services directory: %w", err)
	}
	table, err := msgroute.Load(service.Routing, msgroute.Options{
		Platform:       strings.TrimSpace(strings.ToLower(service.Adapter)),
		DefaultProfile: service.Profile,
		BaseDir:        baseDir,
	})
	if err != nil {
		return nil, fmt.Errorf("service %s route table: %w", service.ID, err)
	}
	return table, nil
}

// validateServiceRouting appends route table problems to the validation
// result. The LoadServiceRouteTable wrapper is peeled and each joined problem
// becomes its own issue line so the report reads as route-local problems.
func validateServiceRouting(service *ServiceConfig, result *ServiceValidationResult) {
	_, err := LoadServiceRouteTable(service)
	if err == nil {
		return
	}
	if inner := errors.Unwrap(err); inner != nil {
		err = inner
	}
	for _, problem := range flattenErrors(err) {
		result.Issues = append(result.Issues, "routing: "+problem)
	}
}

// flattenErrors unwraps errors.Join trees into leaf messages.
func flattenErrors(err error) []string {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var out []string
		for _, e := range joined.Unwrap() {
			out = append(out, flattenErrors(e)...)
		}
		return out
	}
	return []string{err.Error()}
}
