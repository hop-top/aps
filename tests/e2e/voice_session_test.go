package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVoiceSession_AppearsInUnifiedRegistry verifies the option-(b)
// collapse: a voice session started via 'aps voice start' shows up in
// 'aps session list' alongside standard sessions, and is reachable via
// the new '--type voice' filter.
func TestVoiceSession_AppearsInUnifiedRegistry(t *testing.T) {
	home := t.TempDir()

	// Start a voice session.
	stdout, stderr, err := runAPS(t, home, "voice", "start", "--profile", "test-profile", "--channel", "web")
	require.NoError(t, err, "voice start: stderr=%s", stderr)
	require.Contains(t, stdout, "Started voice session")

	// Extract session ID from "Started voice session <uuid> ..."
	parts := strings.Fields(stdout)
	require.GreaterOrEqual(t, len(parts), 4)
	sessID := parts[3]
	require.NotEmpty(t, sessID)

	// 'aps session list' lists the voice session.
	stdout, _, err = runAPS(t, home, "session", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, sessID, "voice session must appear in unified session list")
	assert.Contains(t, stdout, "test-profile")

	// '--type voice' filter includes the voice session.
	stdout, _, err = runAPS(t, home, "session", "list", "--type", "voice")
	require.NoError(t, err)
	assert.Contains(t, stdout, sessID, "voice filter must include voice session")

	// '--type standard' filter excludes the voice session.
	stdout, _, err = runAPS(t, home, "session", "list", "--type", "standard")
	require.NoError(t, err)
	assert.NotContains(t, stdout, sessID, "standard filter must exclude voice session")
}

// TestSessionList_TypeFilterRejectsBogus verifies the validator.
func TestSessionList_TypeFilterRejectsBogus(t *testing.T) {
	home := t.TempDir()
	_, stderr, err := runAPS(t, home, "session", "list", "--type", "bogus")
	require.Error(t, err)
	assert.Contains(t, stderr, "invalid --type")
}

// TestVoiceSessionListSubcommandRemoved verifies the voice session
// subtree was deleted (option-b collapse — there is exactly one place
// to list sessions: 'aps session list'). 'aps voice --help' must no
// longer advertise a 'session' subcommand.
func TestVoiceSessionListSubcommandRemoved(t *testing.T) {
	home := t.TempDir()
	stdout, _, err := runAPS(t, home, "voice", "--help")
	require.NoError(t, err)
	// 'service' and 'start' must remain; 'session' must not be a
	// subcommand of voice anymore.
	assert.Contains(t, stdout, "service")
	assert.Contains(t, stdout, "start")
	// Heuristic: voice subcommand listing must not include a 'session'
	// row. Match the leading whitespace cobra uses for command rows
	// to avoid false matches in flag/usage descriptions.
	assert.NotContains(t, stdout, "    session ", "'aps voice session ...' subtree must be removed")
}
