// Package version provides version information for the OPM CLI.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Build-time variables set via ldflags.
var (
	// Version is the CLI version (set via ldflags).
	Version = "v0.0.0-dev"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the build timestamp.
	BuildDate = "unknown"
)

// CUESDKVersion is the version of the CUE SDK this CLI was built with.
// This is determined at build time from the go.mod file.
const CUESDKVersion = "v0.15.3"

// Info contains version information.
type Info struct {
	// Version is the CLI version (set via ldflags).
	Version string `json:"version"`

	// GitCommit is the git commit hash.
	GitCommit string `json:"gitCommit"`

	// BuildDate is the build timestamp.
	BuildDate string `json:"buildDate"`

	// GoVersion is the Go version used to build.
	GoVersion string `json:"goVersion"`

	// CUESDKVersion is the CUE SDK version (embedded at build time).
	CUESDKVersion string `json:"cueSDKVersion"`
}

// CUEBinaryInfo contains CUE binary version information.
type CUEBinaryInfo struct {
	// Version is the CUE binary version.
	Version string `json:"version"`

	// Path is the path to the CUE binary.
	Path string `json:"path"`

	// Compatible indicates if version matches SDK.
	Compatible bool `json:"compatible"`

	// Found indicates if CUE binary was found.
	Found bool `json:"found"`

	// Message provides additional information about compatibility.
	Message string `json:"message,omitempty"`
}

// GetInfo returns the current version information.
func GetInfo() Info {
	return Info{
		Version:       Version,
		GitCommit:     GitCommit,
		BuildDate:     BuildDate,
		GoVersion:     runtime.Version(),
		CUESDKVersion: CUESDKVersion,
	}
}

// String returns a human-readable version string.
func (i Info) String() string {
	return fmt.Sprintf("OPM CLI:\n  Version:  %s\n  Build ID: %s/%s\n\nCUE:\n  SDK Version: %s",
		i.Version, i.BuildDate, i.GitCommit, i.CUESDKVersion)
}

// CUEVersionCompatible checks if binary version is compatible with SDK.
// Versions are compatible if MAJOR and MINOR components match.
func CUEVersionCompatible(sdkVersion, binaryVersion string) bool {
	// Strip "v" prefix if present
	sdkVersion = strings.TrimPrefix(sdkVersion, "v")
	binaryVersion = strings.TrimPrefix(binaryVersion, "v")

	sdkParts := strings.Split(sdkVersion, ".")
	binParts := strings.Split(binaryVersion, ".")

	if len(sdkParts) < 2 || len(binParts) < 2 {
		return false
	}

	// Compare MAJOR.MINOR only
	return sdkParts[0] == binParts[0] && sdkParts[1] == binParts[1]
}

// CompatibilityMessage returns a message explaining version compatibility.
func CompatibilityMessage(sdkVersion, binaryVersion string) string {
	if CUEVersionCompatible(sdkVersion, binaryVersion) {
		return "compatible"
	}

	sdkVersion = strings.TrimPrefix(sdkVersion, "v")
	binaryVersion = strings.TrimPrefix(binaryVersion, "v")

	sdkParts := strings.Split(sdkVersion, ".")
	binParts := strings.Split(binaryVersion, ".")

	if len(sdkParts) >= 2 && len(binParts) >= 2 {
		if sdkParts[0] != binParts[0] {
			return "incompatible - MAJOR version mismatch"
		}
		if sdkParts[1] != binParts[1] {
			return "incompatible - MINOR version mismatch"
		}
	}

	return "incompatible - invalid version format"
}

// String returns a human-readable CUE binary info string.
func (c CUEBinaryInfo) String() string {
	if !c.Found {
		return "  Binary Version: not found\n  Binary Path:    -"
	}

	compatStr := "compatible"
	if !c.Compatible {
		compatStr = c.Message
	}

	return fmt.Sprintf("  Binary Version: %s (%s)\n  Binary Path:    %s",
		c.Version, compatStr, c.Path)
}

// FullVersionString returns complete version information including CUE binary.
func FullVersionString(info Info, cueInfo CUEBinaryInfo) string {
	return fmt.Sprintf("OPM CLI:\n  Version:  %s\n  Build ID: %s/%s\n\nCUE:\n  SDK Version:    %s\n%s",
		info.Version, info.BuildDate, info.GitCommit, info.CUESDKVersion, cueInfo.String())
}
