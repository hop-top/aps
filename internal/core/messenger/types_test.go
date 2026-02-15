package messenger

import (
	"testing"
	"time"
)

func TestNormalizedMessage_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		msg     NormalizedMessage
		wantErr string
	}{
		{
			name: "valid message",
			msg: NormalizedMessage{
				ID:        "msg-001",
				Timestamp: now,
				Platform:  "telegram",
				Sender:    Sender{ID: "user-1", Name: "Alice"},
				Channel:   Channel{ID: "chan-1", Name: "general"},
				Text:      "hello world",
			},
			wantErr: "",
		},
		{
			name: "missing ID",
			msg: NormalizedMessage{
				Timestamp: now,
				Platform:  "telegram",
				Sender:    Sender{ID: "user-1"},
				Channel:   Channel{ID: "chan-1"},
			},
			wantErr: "message ID is required",
		},
		{
			name: "missing platform",
			msg: NormalizedMessage{
				ID:        "msg-001",
				Timestamp: now,
				Sender:    Sender{ID: "user-1"},
				Channel:   Channel{ID: "chan-1"},
			},
			wantErr: "platform is required",
		},
		{
			name: "missing sender ID",
			msg: NormalizedMessage{
				ID:        "msg-001",
				Timestamp: now,
				Platform:  "slack",
				Sender:    Sender{Name: "Alice"},
				Channel:   Channel{ID: "chan-1"},
			},
			wantErr: "sender ID is required",
		},
		{
			name: "missing channel ID",
			msg: NormalizedMessage{
				ID:        "msg-001",
				Timestamp: now,
				Platform:  "slack",
				Sender:    Sender{ID: "user-1"},
				Channel:   Channel{Name: "general"},
			},
			wantErr: "channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNormalizedMessage_TextPreview(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "truncation at boundary",
			text:   "This is a long message that should be truncated at the boundary",
			maxLen: 10,
			want:   "This is a ...",
		},
		{
			name:   "short text no truncation",
			text:   "hello",
			maxLen: 100,
			want:   "hello",
		},
		{
			name:   "empty text",
			text:   "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "exact boundary",
			text:   "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "one over boundary",
			text:   "12345678901",
			maxLen: 10,
			want:   "1234567890...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &NormalizedMessage{Text: tt.text}
			got := msg.TextPreview(tt.maxLen)
			if got != tt.want {
				t.Errorf("TextPreview(%d) = %q, want %q", tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseTargetAction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantProfile string
		wantAction  string
		wantErr    bool
	}{
		{
			name:        "equals separator",
			input:       "dev-profile=run-tests",
			wantProfile: "dev-profile",
			wantAction:  "run-tests",
		},
		{
			name:        "colon separator legacy",
			input:       "dev-profile:run-tests",
			wantProfile: "dev-profile",
			wantAction:  "run-tests",
		},
		{
			name:        "equals takes priority when both present",
			input:       "profile=action:extra",
			wantProfile: "profile",
			wantAction:  "action:extra",
		},
		{
			name:    "no separator",
			input:   "noseparatorhere",
			wantErr: true,
		},
		{
			name:    "empty profile part",
			input:   "=action",
			wantErr: true,
		},
		{
			name:    "empty action part",
			input:   "profile=",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only separator",
			input:   "=",
			wantErr: true,
		},
		{
			name:    "only colon separator",
			input:   ":",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTargetAction(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ProfileID != tt.wantProfile {
				t.Errorf("ProfileID = %q, want %q", got.ProfileID, tt.wantProfile)
			}
			if got.ActionName != tt.wantAction {
				t.Errorf("ActionName = %q, want %q", got.ActionName, tt.wantAction)
			}
		})
	}
}

func TestTargetAction_String(t *testing.T) {
	tests := []struct {
		name   string
		target TargetAction
		want   string
	}{
		{
			name:   "canonical equals format",
			target: TargetAction{ProfileID: "my-profile", ActionName: "deploy"},
			want:   "my-profile=deploy",
		},
		{
			name:   "empty fields",
			target: TargetAction{},
			want:   "=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProfileMessengerLink_Validate(t *testing.T) {
	tests := []struct {
		name    string
		link    ProfileMessengerLink
		wantErr string
	}{
		{
			name: "valid link",
			link: ProfileMessengerLink{
				ProfileID:     "profile-1",
				MessengerName: "telegram-bot",
			},
			wantErr: "",
		},
		{
			name: "missing profile ID",
			link: ProfileMessengerLink{
				MessengerName: "telegram-bot",
			},
			wantErr: "profile ID is required",
		},
		{
			name: "missing messenger name",
			link: ProfileMessengerLink{
				ProfileID: "profile-1",
			},
			wantErr: "messenger name is required",
		},
		{
			name:    "both missing",
			link:    ProfileMessengerLink{},
			wantErr: "profile ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.link.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestProfileMessengerLink_GetActionForChannel(t *testing.T) {
	tests := []struct {
		name          string
		link          ProfileMessengerLink
		channelID     string
		wantAction    string
		wantFound     bool
	}{
		{
			name: "mapped channel",
			link: ProfileMessengerLink{
				Mappings: map[string]string{
					"chan-1": "profile-a=handle-msg",
					"chan-2": "profile-b=deploy",
				},
				DefaultAction: "profile-a=fallback",
			},
			channelID:  "chan-1",
			wantAction: "profile-a=handle-msg",
			wantFound:  true,
		},
		{
			name: "unmapped channel with default",
			link: ProfileMessengerLink{
				Mappings: map[string]string{
					"chan-1": "profile-a=handle-msg",
				},
				DefaultAction: "profile-a=fallback",
			},
			channelID:  "unknown-chan",
			wantAction: "profile-a=fallback",
			wantFound:  true,
		},
		{
			name: "unmapped channel without default",
			link: ProfileMessengerLink{
				Mappings: map[string]string{
					"chan-1": "profile-a=handle-msg",
				},
			},
			channelID:  "unknown-chan",
			wantAction: "",
			wantFound:  false,
		},
		{
			name: "nil mappings with default",
			link: ProfileMessengerLink{
				DefaultAction: "profile-a=catch-all",
			},
			channelID:  "any-chan",
			wantAction: "profile-a=catch-all",
			wantFound:  true,
		},
		{
			name:       "nil mappings without default",
			link:       ProfileMessengerLink{},
			channelID:  "any-chan",
			wantAction: "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, found := tt.link.GetActionForChannel(tt.channelID)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
		})
	}
}
