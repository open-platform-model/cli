// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/version"
)

// NewVersionCmd creates the version command.
func NewVersionCmd(_ *config.GlobalConfig) *cobra.Command {
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
	output.Println(info.String())
	return nil
}
