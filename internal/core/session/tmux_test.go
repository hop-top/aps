package session

import "testing"

func TestIsBenignTmuxError(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   bool
	}{
		{"can't find session", "can't find session: aps-foo", true},
		{"no server running", "no server running on /tmp/aps-tmux-foo-socket", true},
		{"no sessions", "no sessions", true},
		{"error connecting to", "error connecting to /tmp/aps-tmux-foo-socket (No such file or directory)", true},
		{"empty stderr", "", false},
		{"unrelated error", "tmux: invalid option -- 'Z'", false},
		{"permission denied", "permission denied", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBenignTmuxError(tc.stderr); got != tc.want {
				t.Fatalf("IsBenignTmuxError(%q) = %v, want %v", tc.stderr, got, tc.want)
			}
		})
	}
}
