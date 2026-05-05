package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfilePresentation_AutoAvatarColor — story 063 scenario 1.
// `--auto-avatar --auto-color` produce non-empty deterministic values.
func TestProfilePresentation_AutoAvatarColor(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "alpha",
		"--display-name", "Alpha",
		"--auto-avatar", "--auto-color",
	)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "alpha")
	require.NoError(t, err)
	assert.Contains(t, stdout, "avatar:", "avatar field should be present")
	assert.Contains(t, stdout, "color:", "color field should be present")
	assert.Contains(t, stdout, "dicebear.com", "default provider URL should appear")
	// Color should be a hex literal #RRGGBB.
	assert.Regexp(t, `color:\s*['"]?#[0-9a-fA-F]{6}['"]?`, stdout)
}

// TestProfilePresentation_ExplicitFlagsBeatAuto — story 063 scenario 2.
// Explicit --avatar / --color values win over --auto-* flags.
func TestProfilePresentation_ExplicitFlagsBeatAuto(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "beta",
		"--avatar", "https://example.com/manual.png",
		"--color", "#abcdef",
		"--auto-avatar", "--auto-color",
	)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "beta")
	require.NoError(t, err)
	assert.Contains(t, stdout, "https://example.com/manual.png")
	assert.Contains(t, stdout, "#abcdef")
	assert.NotContains(t, stdout, "dicebear.com",
		"explicit --avatar should not be overwritten by --auto-avatar")
}

// TestProfilePresentation_ConfigDefaults — story 063 scenario 3.
// `profile.color: true` and `profile.avatar.enabled: true` in
// XDG_CONFIG_HOME apply to creates with no flags.
func TestProfilePresentation_ConfigDefaults(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")
	apsConfigDir := filepath.Join(configHome, "aps")
	require.NoError(t, os.MkdirAll(apsConfigDir, 0755))
	configContent := `prefix: APS
profile:
  color: true
  avatar:
    enabled: true
    provider: dicebear
    style: bottts
    size: 128
`
	require.NoError(t, os.WriteFile(filepath.Join(apsConfigDir, "config.yaml"),
		[]byte(configContent), 0644))

	env := map[string]string{"XDG_CONFIG_HOME": configHome}
	_, _, err := runAPSWithEnv(t, home, env, "profile", "create", "gamma",
		"--display-name", "Gamma")
	require.NoError(t, err)

	stdout, _, err := runAPSWithEnv(t, home, env, "profile", "show", "gamma")
	require.NoError(t, err)
	assert.Contains(t, stdout, "bottts", "configured style should be applied")
	assert.Contains(t, stdout, "size=128", "configured size should be applied")
	assert.Regexp(t, `color:\s*['"]?#[0-9a-fA-F]{6}['"]?`, stdout,
		"configured color: true should auto-generate")
}

// TestProfilePresentation_EditFields — story 063 scenario 4.
// `profile edit` updates only the named field.
func TestProfilePresentation_EditFields(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "delta",
		"--auto-avatar", "--auto-color")
	require.NoError(t, err)

	// Capture the auto-generated avatar.
	stdout, _, err := runAPS(t, home, "profile", "show", "delta")
	require.NoError(t, err)
	originalAvatar := extractField(stdout, "avatar")
	require.NotEmpty(t, originalAvatar, "auto-avatar should be set")

	// Edit color only — avatar must persist unchanged.
	_, _, err = runAPS(t, home, "profile", "edit", "delta", "--color", "#ff0000")
	require.NoError(t, err)

	stdout, _, err = runAPS(t, home, "profile", "show", "delta")
	require.NoError(t, err)
	assert.Contains(t, stdout, "#ff0000")
	assert.Contains(t, stdout, originalAvatar,
		"avatar should not change when editing only color")

	// Clear the avatar via empty string.
	_, _, err = runAPS(t, home, "profile", "edit", "delta", "--avatar", "")
	require.NoError(t, err)
	stdout, _, err = runAPS(t, home, "profile", "show", "delta")
	require.NoError(t, err)
	assert.NotContains(t, stdout, originalAvatar,
		"avatar should be cleared when set to empty string")
	assert.Contains(t, stdout, "#ff0000",
		"color should remain set after clearing avatar")
}

// TestProfilePresentation_DeterministicSeed — story 063 scenario 5.
// Same id on different temp homes produces identical avatar+color.
func TestProfilePresentation_DeterministicSeed(t *testing.T) {
	t.Parallel()
	avatar1, color1 := createAndExtract(t, "epsilon")
	avatar2, color2 := createAndExtract(t, "epsilon")

	assert.Equal(t, avatar1, avatar2, "avatar must be deterministic across runs")
	assert.Equal(t, color1, color2, "color must be deterministic across runs")
	assert.NotEmpty(t, avatar1)
	assert.NotEmpty(t, color1)
}

// TestProfilePresentation_UnknownProviderGraceful — story 063 scenario 6.
// Unknown provider yields empty avatar but doesn't fail the create.
func TestProfilePresentation_UnknownProviderGraceful(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	stdout, stderr, err := runAPS(t, home, "profile", "create", "zeta",
		"--auto-avatar",
		"--avatar-provider", "nonexistent-provider")
	require.NoError(t, err, "create should succeed; stderr=%s", stderr)
	assert.Contains(t, stdout, "created successfully")

	stdout, _, err = runAPS(t, home, "profile", "show", "zeta")
	require.NoError(t, err)
	// Avatar field either absent (omitempty) or empty string — both are fine.
	avatar := extractField(stdout, "avatar")
	assert.Empty(t, avatar,
		"unknown provider should yield empty avatar, not a malformed URL")
}

// TestProfilePresentation_StyleSizeFlags — story 063 scenario 7.
// --avatar-style and --avatar-size are forwarded to the provider URL.
func TestProfilePresentation_StyleSizeFlags(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "eta",
		"--auto-avatar",
		"--avatar-style", "bottts",
		"--avatar-size", "128")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "eta")
	require.NoError(t, err)
	avatar := extractField(stdout, "avatar")
	assert.Contains(t, avatar, "bottts", "style should be in URL: %s", avatar)
	assert.Contains(t, avatar, "size=128", "size should be in URL: %s", avatar)
}

// --- helpers ---

// createAndExtract creates a profile with --auto-* and returns
// (avatar, color) values from the resulting yaml. Uses an isolated
// temp home each call.
func createAndExtract(t *testing.T, id string) (string, string) {
	t.Helper()
	home := t.TempDir()
	_, _, err := runAPS(t, home, "profile", "create", id,
		"--auto-avatar", "--auto-color")
	require.NoError(t, err)
	stdout, _, err := runAPS(t, home, "profile", "show", id)
	require.NoError(t, err)
	return extractField(stdout, "avatar"), extractField(stdout, "color")
}

// extractField returns the value of a top-level yaml field from
// `aps profile show` output. Strips surrounding quotes if present.
// Returns "" if the field is missing or empty.
func extractField(out, field string) string {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		prefix := field + ":"
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		val = strings.Trim(val, `'"`)
		return val
	}
	return ""
}
