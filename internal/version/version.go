// Package version provides build-time version information.
// These variables are populated by ldflags during the build process.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

// devVersion is the placeholder used when no version was injected at build time.
const devVersion = "dev"

// Build-time variables (set via ldflags).
var (
	// Version is the raw injected version. Accepts a bare semver ("0.6.0"),
	// a v-prefixed one ("v0.6.0"), or a monorepo `git describe` string with a
	// tag prefix ("aps/v0.6.0-alpha.0-70-g7a18b94"). Read it through Short()
	// or Get(), which normalize; never format it directly.
	Version = devVersion
	// Commit is the git commit SHA.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
	// BuiltBy is the build system (e.g., "goreleaser").
	BuiltBy = "manual"
)

// Info represents complete version information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Normalize reduces an injected version string to a bare semver-ish value:
// it strips a monorepo tag prefix (everything up to and including the last
// "/") and a leading "v". Empty input yields "dev".
func Normalize(v string) string {
	if i := strings.LastIndex(v, "/"); i >= 0 {
		v = v[i+1:]
	}
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return devVersion
	}
	return v
}

// Get returns the complete version information.
func Get() Info {
	return Info{
		Version:   Short(),
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable version string.
func (i Info) String() string {
	commit := i.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	v := Normalize(i.Version)
	if v != devVersion {
		v = "v" + v
	}
	return fmt.Sprintf("aps %s (%s)", v, commit)
}

// Short returns just the normalized version number (no "v", no tag prefix).
func Short() string {
	return Normalize(Version)
}
