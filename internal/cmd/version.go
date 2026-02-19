// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/version"
)

// NewVersionCmd creates the version command.
func NewVersionCmd(_ *cmdtypes.GlobalConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Show OPM CLI version information.

Displays:
  - OPM CLI version, commit, and build date
  - CUE SDK version (embedded in CLI)`,
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

	return nil
}
