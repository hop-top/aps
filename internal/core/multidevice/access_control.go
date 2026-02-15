package multidevice

import (
	"fmt"
	"time"
)

// AccessDeniedError provides detailed information about why access was denied.
type AccessDeniedError struct {
	DeviceID    string
	WorkspaceID string
	Action      string
	Resource    string
	Reason      string        // human-readable reason
	Step        string        // which evaluation step denied
	Suggestion  string        // actionable suggestion for user
	RetryAfter  time.Duration // for rate limit denials
}

func (e *AccessDeniedError) Error() string {
	msg := fmt.Sprintf("access denied: device=%s workspace=%s action=%s", e.DeviceID, e.WorkspaceID, e.Action)
	if e.Resource != "" {
		msg += fmt.Sprintf(" resource=%s", e.Resource)
	}
	msg += fmt.Sprintf(" reason=%s (step: %s)", e.Reason, e.Step)
	if e.Suggestion != "" {
		msg += fmt.Sprintf(" suggestion=%s", e.Suggestion)
	}
	return msg
}

// AccessController evaluates device permissions using a multi-step chain.
type AccessController struct {
	linker      *Linker
	rateLimiter *RateLimiter
}

// NewAccessController creates a new AccessController.
func NewAccessController(linker *Linker) *AccessController {
	return &AccessController{
		linker:      linker,
		rateLimiter: NewRateLimiter(),
	}
}

// EvaluatePermission checks if a device can perform an action on a resource.
// Returns nil if allowed, or an *AccessDeniedError with details.
//
// Evaluation chain:
//  1. Link existence check
//  2. Link status check (must not be unlinked)
//  3. Policy check (workspace-level allow/deny list)
//  4. Role-based permission check
//  5. Explicit allow/deny action lists
//  6. Access schedule check
//  7. Rate limit check
func (ac *AccessController) EvaluatePermission(workspaceID, deviceID, action, resource string) error {
	// Step 1: Check that the device is linked to the workspace.
	link, err := ac.linker.GetLink(workspaceID, deviceID)
	if err != nil {
		return &AccessDeniedError{
			DeviceID:    deviceID,
			WorkspaceID: workspaceID,
			Action:      action,
			Resource:    resource,
			Reason:      "device is not linked to this workspace",
			Step:        "link_existence",
			Suggestion:  fmt.Sprintf("link the device first: aps workspace device attach %s", deviceID),
		}
	}

	// Step 2: Check that the link is not in an unlinked state.
	if link.Status == PresenceUnlinked {
		return &AccessDeniedError{
			DeviceID:    deviceID,
			WorkspaceID: workspaceID,
			Action:      action,
			Resource:    resource,
			Reason:      "device has been unlinked from this workspace",
			Step:        "link_status",
			Suggestion:  "re-link the device to restore access",
		}
	}

	// Step 3: Check workspace policy (allow/deny lists).
	policy, err := LoadPolicy(workspaceID)
	if err == nil && policy != nil {
		if !policy.IsDeviceAllowed(deviceID) {
			return &AccessDeniedError{
				DeviceID:    deviceID,
				WorkspaceID: workspaceID,
				Action:      action,
				Resource:    resource,
				Reason:      "device is not allowed by workspace policy",
				Step:        "policy",
				Suggestion:  "contact the workspace owner to update the access policy",
			}
		}
	}

	// Step 4: Check role-based permissions.
	perms := link.Permissions
	if err := ac.checkRolePermission(perms, action); err != nil {
		return &AccessDeniedError{
			DeviceID:    deviceID,
			WorkspaceID: workspaceID,
			Action:      action,
			Resource:    resource,
			Reason:      err.Error(),
			Step:        "role_permission",
			Suggestion:  fmt.Sprintf("current role is '%s'; request a role upgrade from the workspace owner", perms.Role),
		}
	}

	// Step 5: Check explicit allow/deny action lists.
	if len(perms.DeniedActions) > 0 {
		for _, denied := range perms.DeniedActions {
			if denied == action {
				return &AccessDeniedError{
					DeviceID:    deviceID,
					WorkspaceID: workspaceID,
					Action:      action,
					Resource:    resource,
					Reason:      "action is explicitly denied",
					Step:        "denied_actions",
					Suggestion:  "contact the workspace owner to remove this action from the deny list",
				}
			}
		}
	}

	if len(perms.AllowedActions) > 0 {
		found := false
		for _, allowed := range perms.AllowedActions {
			if allowed == action {
				found = true
				break
			}
		}
		if !found {
			return &AccessDeniedError{
				DeviceID:    deviceID,
				WorkspaceID: workspaceID,
				Action:      action,
				Resource:    resource,
				Reason:      "action is not in the allowed actions list",
				Step:        "allowed_actions",
				Suggestion:  "contact the workspace owner to add this action to the allow list",
			}
		}
	}

	// Step 6: Check access schedule.
	if perms.AccessSchedule != nil {
		if !ac.isWithinSchedule(perms.AccessSchedule) {
			return &AccessDeniedError{
				DeviceID:    deviceID,
				WorkspaceID: workspaceID,
				Action:      action,
				Resource:    resource,
				Reason:      fmt.Sprintf("outside access schedule (%s-%s)", perms.AccessSchedule.StartTime, perms.AccessSchedule.EndTime),
				Step:        "access_schedule",
				Suggestion:  fmt.Sprintf("try again during allowed hours: %s-%s", perms.AccessSchedule.StartTime, perms.AccessSchedule.EndTime),
			}
		}
	}

	// Step 7: Check rate limits.
	if perms.RateLimitPerMin > 0 {
		allowed, _, retryAfter := ac.rateLimiter.Consume(deviceID, workspaceID, perms.RateLimitPerMin)
		if !allowed {
			return &AccessDeniedError{
				DeviceID:    deviceID,
				WorkspaceID: workspaceID,
				Action:      action,
				Resource:    resource,
				Reason:      "rate limit exceeded",
				Step:        "rate_limit",
				Suggestion:  fmt.Sprintf("try again in %s", retryAfter.Round(time.Second)),
				RetryAfter:  retryAfter,
			}
		}
	}

	return nil
}

// checkRolePermission verifies that the device's permissions allow the action.
func (ac *AccessController) checkRolePermission(perms DevicePermissions, action string) error {
	switch action {
	case "read":
		if !perms.CanRead {
			return fmt.Errorf("role '%s' does not have read permission", perms.Role)
		}
	case "write":
		if !perms.CanWrite {
			return fmt.Errorf("role '%s' does not have write permission", perms.Role)
		}
	case "execute":
		if !perms.CanExecute {
			return fmt.Errorf("role '%s' does not have execute permission", perms.Role)
		}
	case "manage":
		if !perms.CanManage {
			return fmt.Errorf("role '%s' does not have manage permission", perms.Role)
		}
	case "sync":
		if !perms.CanSync {
			return fmt.Errorf("role '%s' does not have sync permission", perms.Role)
		}
	}
	return nil
}

// isWithinSchedule checks if the current time falls within the access schedule.
func (ac *AccessController) isWithinSchedule(schedule *AccessSchedule) bool {
	now := time.Now()

	// Apply timezone if specified.
	if schedule.Timezone != "" {
		loc, err := time.LoadLocation(schedule.Timezone)
		if err == nil {
			now = now.In(loc)
		}
	}

	// Check day of week (0=Monday in our convention).
	if len(schedule.DaysOfWeek) > 0 {
		// Go's time.Weekday: Sunday=0, Monday=1, ..., Saturday=6
		// Our convention: Monday=0, Tuesday=1, ..., Sunday=6
		goDay := int(now.Weekday())
		ourDay := (goDay + 6) % 7 // Convert: Go Sunday(0)->6, Monday(1)->0, etc.

		dayAllowed := false
		for _, d := range schedule.DaysOfWeek {
			if d == ourDay {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	// Check time range.
	if schedule.StartTime != "" && schedule.EndTime != "" {
		currentTime := now.Format("15:04")
		if currentTime < schedule.StartTime || currentTime >= schedule.EndTime {
			return false
		}
	}

	return true
}

// IsAccessDenied checks if an error is an AccessDeniedError.
func IsAccessDenied(err error) bool {
	_, ok := err.(*AccessDeniedError)
	return ok
}
