package messenger

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    ErrorCode
		wantContain string
	}{
		{
			name:        "ErrLinkNotFound",
			err:         ErrLinkNotFound("telegram", "dev"),
			wantCode:    ErrCodeLinkNotFound,
			wantContain: "no link found for profile 'dev'",
		},
		{
			name:        "ErrLinkAlreadyExists",
			err:         ErrLinkAlreadyExists("telegram", "dev"),
			wantCode:    ErrCodeLinkAlreadyExists,
			wantContain: "already linked to profile 'dev'",
		},
		{
			name:        "ErrMappingConflict",
			err:         ErrMappingConflict("chan-1", "profile-a", "handle-msg"),
			wantCode:    ErrCodeMappingConflict,
			wantContain: "channel already mapped to profile-a=handle-msg",
		},
		{
			name:        "ErrUnknownChannel",
			err:         ErrUnknownChannel("telegram", "chan-unknown"),
			wantCode:    ErrCodeUnknownChannel,
			wantContain: "no mapping for channel 'chan-unknown'",
		},
		{
			name:        "ErrActionNotFound",
			err:         ErrActionNotFound("dev", "deploy"),
			wantCode:    ErrCodeActionNotFound,
			wantContain: "action 'deploy' not found",
		},
		{
			name:        "ErrActionFailed",
			err:         ErrActionFailed("dev", "deploy", fmt.Errorf("timeout")),
			wantCode:    ErrCodeActionFailed,
			wantContain: "action 'deploy' failed",
		},
		{
			name:        "ErrIsolationViolation",
			err:         ErrIsolationViolation("chan-1", "profile-a", "profile-b"),
			wantCode:    ErrCodeIsolationViolation,
			wantContain: "isolation violation: mapped to 'profile-a', attempted 'profile-b'",
		},
		{
			name:        "ErrRoutingFailed",
			err:         ErrRoutingFailed("msg-001", fmt.Errorf("no route")),
			wantCode:    ErrCodeRoutingFailed,
			wantContain: "routing failed",
		},
		{
			name:        "ErrNormalizeFailed",
			err:         ErrNormalizeFailed("slack", fmt.Errorf("bad format")),
			wantCode:    ErrCodeNormalizeFailed,
			wantContain: "normalization failed",
		},
		{
			name:        "ErrInvalidMapping",
			err:         ErrInvalidMapping("bad=format=here", fmt.Errorf("parse error")),
			wantCode:    ErrCodeInvalidMapping,
			wantContain: "invalid mapping format",
		},
		{
			name:        "ErrMissingSecret",
			err:         ErrMissingSecret("TELEGRAM_TOKEN"),
			wantCode:    ErrCodeMissingSecret,
			wantContain: "required secret 'TELEGRAM_TOKEN' not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			me, ok := tt.err.(*MessengerError)
			if !ok {
				t.Fatalf("expected *MessengerError, got %T", tt.err)
			}
			if me.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", me.Code, tt.wantCode)
			}
			errStr := tt.err.Error()
			if !containsSubstring(errStr, tt.wantContain) {
				t.Errorf("Error() = %q, want it to contain %q", errStr, tt.wantContain)
			}
		})
	}
}

func TestErrorCheckers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{
			name:    "IsMappingConflict with mapping conflict error",
			err:     ErrMappingConflict("chan-1", "p1", "act"),
			checker: IsMappingConflict,
			want:    true,
		},
		{
			name:    "IsMappingConflict with other error",
			err:     ErrLinkNotFound("tg", "dev"),
			checker: IsMappingConflict,
			want:    false,
		},
		{
			name:    "IsMappingConflict with plain error",
			err:     fmt.Errorf("some error"),
			checker: IsMappingConflict,
			want:    false,
		},
		{
			name:    "IsLinkNotFound with link not found error",
			err:     ErrLinkNotFound("tg", "dev"),
			checker: IsLinkNotFound,
			want:    true,
		},
		{
			name:    "IsLinkNotFound with other error",
			err:     ErrMappingConflict("c", "p", "a"),
			checker: IsLinkNotFound,
			want:    false,
		},
		{
			name:    "IsUnknownChannel with unknown channel error",
			err:     ErrUnknownChannel("tg", "chan-1"),
			checker: IsUnknownChannel,
			want:    true,
		},
		{
			name:    "IsUnknownChannel with other error",
			err:     ErrLinkNotFound("tg", "dev"),
			checker: IsUnknownChannel,
			want:    false,
		},
		{
			name:    "IsIsolationViolation with isolation error",
			err:     ErrIsolationViolation("c", "p1", "p2"),
			checker: IsIsolationViolation,
			want:    true,
		},
		{
			name:    "IsIsolationViolation with other error",
			err:     ErrLinkNotFound("tg", "dev"),
			checker: IsIsolationViolation,
			want:    false,
		},
		{
			name:    "IsActionNotFound with action not found error",
			err:     ErrActionNotFound("dev", "deploy"),
			checker: IsActionNotFound,
			want:    true,
		},
		{
			name:    "IsActionNotFound with other error",
			err:     ErrLinkNotFound("tg", "dev"),
			checker: IsActionNotFound,
			want:    false,
		},
		{
			name:    "IsLinkNotFound with nil error",
			err:     nil,
			checker: IsLinkNotFound,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil error case
			if tt.err == nil {
				got := tt.checker(fmt.Errorf("placeholder"))
				if got != false {
					t.Errorf("checker should return false for non-matching error")
				}
				return
			}
			got := tt.checker(tt.err)
			if got != tt.want {
				t.Errorf("checker(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCause bool
		causeMsg  string
	}{
		{
			name:      "error with cause",
			err:       ErrActionFailed("dev", "deploy", fmt.Errorf("connection timeout")),
			wantCause: true,
			causeMsg:  "connection timeout",
		},
		{
			name:      "error with cause via routing",
			err:       ErrRoutingFailed("msg-1", fmt.Errorf("no profiles found")),
			wantCause: true,
			causeMsg:  "no profiles found",
		},
		{
			name:      "error without cause",
			err:       ErrLinkNotFound("tg", "dev"),
			wantCause: false,
		},
		{
			name:      "error without cause mapping conflict",
			err:       ErrMappingConflict("c1", "p1", "a1"),
			wantCause: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			me, ok := tt.err.(*MessengerError)
			if !ok {
				t.Fatalf("expected *MessengerError, got %T", tt.err)
			}

			unwrapped := me.Unwrap()
			if tt.wantCause {
				if unwrapped == nil {
					t.Fatal("expected a cause, got nil")
				}
				if unwrapped.Error() != tt.causeMsg {
					t.Errorf("Unwrap().Error() = %q, want %q", unwrapped.Error(), tt.causeMsg)
				}
				// Verify errors.Unwrap also works
				if errors.Unwrap(me) == nil {
					t.Error("errors.Unwrap should return the cause")
				}
			} else {
				if unwrapped != nil {
					t.Errorf("expected nil cause, got %v", unwrapped)
				}
			}
		})
	}
}

func TestErrorFormat_WithAndWithoutCause(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "without cause includes name and message",
			err:  ErrLinkNotFound("telegram", "dev"),
			want: "telegram: no link found for profile 'dev'",
		},
		{
			name: "with cause includes name message and cause",
			err:  ErrActionFailed("dev", "deploy", fmt.Errorf("timeout")),
			want: "dev: action 'deploy' failed: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
