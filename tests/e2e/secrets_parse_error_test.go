package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const parseLeakSentinel = "SENTINEL-s3cr3t-value"

// Malformed secrets.env text must never reach stdout or stderr, whichever
// command trips over it and whatever output format is asked for.
func TestSecretsParseError_NeverEchoesContent(t *testing.T) {
	t.Parallel()
	shapes := map[string]string{
		"unterminated double": "GOOD=ok\nTOKEN=\"" + parseLeakSentinel + "\nNEXT=" + parseLeakSentinel + "2\n",
		"unterminated single": "GOOD=ok\nTOKEN='" + parseLeakSentinel + "\n",
		"bad key":             "GOOD=ok\nBAD!KEY" + parseLeakSentinel + "=v\n",
		"corrupt b64 payload": "GOOD=ok\nTOKEN=aps-b64:" + parseLeakSentinel + "\n",
		"missing equals":      "GOOD=ok\n" + parseLeakSentinel + "-no-equals\nAFTER=1\n",
	}
	commands := map[string][]string{
		"run":          {"run", "SHAPE_PROFILE", "--", "true"},
		"run json":     {"run", "SHAPE_PROFILE", "--format", "json", "--", "true"},
		"profile show": {"profile", "show", "SHAPE_PROFILE"},
		"profile list": {"profile", "list", "--has-secrets"},
	}
	for shape, content := range shapes {
		for cname, args := range commands {
			t.Run(shape+"/"+cname, func(t *testing.T) {
				t.Parallel()
				home := t.TempDir()
				_, _, err := runAPS(t, home, "profile", "create", "leakcheck")
				require.NoError(t, err)
				path := filepath.Join(home, ".local", "share", "aps", "profiles", "leakcheck", "secrets.env")
				require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

				argv := append([]string(nil), args...)
				for i, a := range argv {
					if a == "SHAPE_PROFILE" {
						argv[i] = "leakcheck"
					}
				}
				stdout, stderr, _ := runAPS(t, home, argv...)
				assert.NotContains(t, stdout, parseLeakSentinel)
				assert.NotContains(t, stderr, parseLeakSentinel)
				if cname == "run" {
					assert.Contains(t, stderr, "malformed secrets file")
				}
			})
		}
	}
}
