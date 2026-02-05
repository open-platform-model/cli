// Package version provides version information for the CLI.
package version

import (
	"fmt"
	"runtime"
)

// These variables are set via ldflags at build time.
var (
	// Version is the CLI version.
	Version = "dev"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the build timestamp.
	BuildDate = "unknown"

	// CUESDKVersion is the CUE SDK version embedded at build time.
	CUESDKVersion = "v0.15.0"
)

// Info contains version information.
type Info struct {
	// Version is the CLI version (set via ldflags).
	Version string

	// GitCommit is the git commit hash.
	GitCommit string

	// BuildDate is the build timestamp.
	BuildDate string

	// GoVersion is the Go version used to build.
	GoVersion string

	// CUESDKVersion is the CUE SDK version (embedded at build time).
	CUESDKVersion string
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version:       Version,
		GitCommit:     GitCommit,
		BuildDate:     BuildDate,
		GoVersion:     runtime.Version(),
		CUESDKVersion: CUESDKVersion,
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("opm %s (%s) built %s with %s\nCUE SDK: %s",
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.CUESDKVersion)
}
