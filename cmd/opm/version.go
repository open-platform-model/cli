package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show CLI version information",
		Long: `Display version information for the OPM CLI.

Shows the CLI version, build information, and CUE SDK/binary compatibility status.`,
		RunE: runVersion,
	}
}

func runVersion(cmd *cobra.Command, _ []string) error {
	info := version.GetInfo()
	cueInfo := version.DetectCUEBinary()

	fmt.Fprintln(cmd.OutOrStdout(), version.FullVersionString(info, cueInfo))
	return nil
}
