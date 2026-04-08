package session

import "strings"

// IsBenignTmuxError reports whether a tmux stderr message indicates
// that the target session or server was already gone, which is a
// benign race during teardown. Callers should treat a benign error
// as success.
func IsBenignTmuxError(stderr string) bool {
	for _, needle := range []string{
		"can't find session",
		"no server running",
		"no sessions",
		"error connecting to",
	} {
		if strings.Contains(stderr, needle) {
			return true
		}
	}
	return false
}
