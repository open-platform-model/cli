// Package version provides version information for the CLI.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// cueModulePath is the module path of the CUE SDK dependency.
const cueModulePath = "cuelang.org/go"

// unknownCUESDKVersion is reported when the binary carries no build info or
// no record of the CUE SDK dependency.
const unknownCUESDKVersion = "unknown"

// These variables are set via ldflags at build time.
var (
	// Version is the CLI version.
	Version = "dev"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the build timestamp.
	BuildDate = "unknown"
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

	// CUESDKVersion is the CUE SDK version the binary is linked against,
	// read from the embedded build info ("unknown" when unavailable).
	CUESDKVersion string
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version:       Version,
		GitCommit:     GitCommit,
		BuildDate:     BuildDate,
		GoVersion:     runtime.Version(),
		CUESDKVersion: cueSDKVersion(),
	}
}

// cueSDKVersion reports the CUE SDK version linked into the running binary.
func cueSDKVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return unknownCUESDKVersion
	}
	return cueSDKVersionFrom(bi)
}

// cueSDKVersionFrom extracts the CUE SDK version from build info, honoring a
// go.mod replace directive. A path replacement (local checkout) carries no
// version, so the original version is reported with a "(replaced)" marker.
func cueSDKVersionFrom(bi *debug.BuildInfo) string {
	if bi == nil {
		return unknownCUESDKVersion
	}
	for _, d := range bi.Deps {
		if d == nil || d.Path != cueModulePath {
			continue
		}
		if d.Replace != nil {
			if d.Replace.Version != "" {
				return d.Replace.Version
			}
			if d.Version != "" {
				return d.Version + " (replaced)"
			}
			return unknownCUESDKVersion
		}
		if d.Version != "" {
			return d.Version
		}
		return unknownCUESDKVersion
	}
	return unknownCUESDKVersion
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("opm version %s (%s) built %s with %s\nCUE SDK: %s",
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.CUESDKVersion)
}
