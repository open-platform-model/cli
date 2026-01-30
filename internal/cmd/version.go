// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/version"
)

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Show OPM CLI and CUE version information.

Displays:
  - OPM CLI version, commit, and build date
  - CUE SDK version (embedded in CLI)
  - CUE binary version and compatibility status`,
		RunE: runVersion,
	}
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := version.Get()

	// Print CLI version info
	output.Println(fmt.Sprintf("opm version %s", info.Version))
	output.Println(fmt.Sprintf("  Commit:    %s", info.GitCommit))
	output.Println(fmt.Sprintf("  Built:     %s", info.BuildDate))
	output.Println(fmt.Sprintf("  Go:        %s", info.GoVersion))
	output.Println(fmt.Sprintf("  CUE SDK:   %s", info.CUESDKVersion))

	// Check CUE binary
	cueInfo := version.GetCUEBinaryInfo()
	output.Println("")

	if !cueInfo.Found {
		output.Warn("CUE binary not found in PATH")
		output.Println("  Install CUE from https://cuelang.org/docs/install/")
		return nil
	}

	output.Println(fmt.Sprintf("cue binary %s", cueInfo.Version))
	output.Println(fmt.Sprintf("  Path:       %s", cueInfo.Path))

	if cueInfo.Compatible {
		output.Println("  Compatible: yes")
	} else {
		output.Warn("CUE binary version is not compatible with CLI")
		output.Println(fmt.Sprintf("  Compatible: no (expected %s.x)", extractMajorMinor(info.CUESDKVersion)))
	}

	return nil
}

// extractMajorMinor extracts MAJOR.MINOR from a version string.
func extractMajorMinor(v string) string {
	// Remove 'v' prefix
	if len(v) > 0 && v[0] == 'v' {
		v = v[1:]
	}

	// Find second dot
	dotCount := 0
	for i, c := range v {
		if c == '.' {
			dotCount++
			if dotCount == 2 {
				return v[:i]
			}
		}
	}

	return v
}
