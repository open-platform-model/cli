// Package version provides version information for the CLI.
package version

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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

// CUEBinaryInfo contains CUE binary version information.
type CUEBinaryInfo struct {
	// Version is the CUE binary version.
	Version string

	// Path is the path to the CUE binary.
	Path string

	// Compatible indicates if version matches SDK.
	Compatible bool

	// Found indicates if CUE binary was found.
	Found bool
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

// GetCUEBinaryInfo detects the CUE binary and its version.
func GetCUEBinaryInfo() CUEBinaryInfo {
	path, err := exec.LookPath("cue")
	if err != nil {
		return CUEBinaryInfo{Found: false}
	}

	cmd := exec.Command(path, "version")
	output, err := cmd.Output()
	if err != nil {
		return CUEBinaryInfo{
			Path:  path,
			Found: true,
		}
	}

	version := parseCUEVersion(string(output))
	return CUEBinaryInfo{
		Version:    version,
		Path:       path,
		Found:      true,
		Compatible: CUEVersionCompatible(CUESDKVersion, version),
	}
}

// CUEVersionCompatible checks if binary version is compatible with SDK.
// Compatible means MAJOR.MINOR versions match.
func CUEVersionCompatible(sdkVersion, binaryVersion string) bool {
	sdkMajorMinor := extractMajorMinor(sdkVersion)
	binMajorMinor := extractMajorMinor(binaryVersion)

	if sdkMajorMinor == "" || binMajorMinor == "" {
		return false
	}

	return sdkMajorMinor == binMajorMinor
}

// extractMajorMinor extracts MAJOR.MINOR from a semver string.
func extractMajorMinor(version string) string {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}

	// Return MAJOR.MINOR
	return parts[0] + "." + parts[1]
}

// parseCUEVersion extracts the version from cue version output.
func parseCUEVersion(output string) string {
	// cue version output format:
	// cue version v0.15.0
	// ...
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cue version ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}
