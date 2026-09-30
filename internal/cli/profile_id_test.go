package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

// seedDataDir points APS_DATA_PATH at a fresh dir holding a sentinel
// file and returns both paths.
func seedDataDir(t *testing.T) (dataDir, sentinel string) {
	t.Helper()
	dataDir = t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	sentinel = filepath.Join(dataDir, "sentinel")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0o600))
	return dataDir, sentinel
}

func requireNoProfileDirs(t *testing.T, dataDir string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dataDir, "profiles"))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	require.Empty(t, entries, "rejected id must not create a profile dir")
}

// `aps profile create <id> --force` removes an existing dir before
// creating; an invalid id must be rejected before that removal (".."
// would otherwise resolve to the data dir itself).
func TestProfileCreateCmd_RejectsInvalidIDBeforeTouchingFS(t *testing.T) {
	for _, id := range []string{"..", "nul", "Con.txt", "a:b", "x."} {
		t.Run(id, func(t *testing.T) {
			dataDir, sentinel := seedDataDir(t)

			require.NoError(t, profileCreateCmd.Flags().Set("force", "true"))
			t.Cleanup(func() { _ = profileCreateCmd.Flags().Set("force", "false") })

			// Asserted by message: once the root command is built, kit's
			// RunE middleware re-wraps errors as an output.Error envelope,
			// which does not carry the sentinel chain.
			err := profileCreateCmd.RunE(profileCreateCmd, []string{id})
			require.ErrorContains(t, err, fmt.Sprintf("invalid profile id %q", id))

			_, statErr := os.Stat(sentinel)
			require.NoError(t, statErr, "data dir must survive a rejected create")
			requireNoProfileDirs(t, dataDir)
		})
	}
}

const reservedSlugManifestDoc = `---
name: nul
slug: nul
---
`

func TestRunManifestImport_RejectsInvalidIDBeforeTouchingFS(t *testing.T) {
	cases := []struct {
		name       string
		idOverride string
		dryRun     bool
	}{
		{"manifest slug reserved", "", false},
		{"manifest slug reserved dry-run", "", true},
		{"override reserved with extension", "AUX.json", false},
		{"override invalid char", "a*b", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataDir, _ := seedDataDir(t)
			path := writeManifestFile(t, t.TempDir(), reservedSlugManifestDoc)

			var out, errOut strings.Builder
			err := runManifestImport(t.Context(), path, tc.idOverride, tc.dryRun, false, &out, &errOut)
			require.Error(t, err)
			require.True(t, errors.Is(err, core.ErrInvalidProfileID), "err = %v", err)
			require.Empty(t, out.String(), "no preview for a rejected id")
			requireNoProfileDirs(t, dataDir)
		})
	}
}
