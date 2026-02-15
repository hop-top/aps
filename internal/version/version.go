// Package version provides build-time version information.
// These variables are populated by ldflags during the build process.
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables (set via ldflags)
var (
	// Version is the semantic version (e.g., "1.0.0-alpha.1")
	Version = "dev"
	// Commit is the git commit SHA
	Commit = "none"
	// Date is the build date
	Date = "unknown"
	// BuiltBy is the build system (e.g., "goreleaser")
	BuiltBy = "manual"
)

// Info represents complete version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the complete version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable version string
func (i Info) String() string {
	commit := i.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("aps v%s (%s)", i.Version, commit)
}

// Short returns just the version number
func Short() string {
	return Version
}
