// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/version"
)

// versionOutput is the json and yaml form of 'opm version'. The field names
// are a contract for scripts.
type versionOutput struct {
	Version       string `json:"version" yaml:"version"`
	GitCommit     string `json:"gitCommit" yaml:"gitCommit"`
	BuildDate     string `json:"buildDate" yaml:"buildDate"`
	GoVersion     string `json:"goVersion" yaml:"goVersion"`
	CUESDKVersion string `json:"cueSDKVersion" yaml:"cueSDKVersion"`
}

// NewVersionCmd creates the version command.
func NewVersionCmd(_ *config.GlobalConfig) *cobra.Command {
	var outputFlag string

	c := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Show OPM CLI version information.

Displays:
  - OPM CLI version, commit, and build date
  - CUE SDK version (the one the binary is linked against)

Examples:
  # Print the version for a person to read
  opm version

  # Print it for a script
  opm version -o json`,
		RunE: func(c *cobra.Command, _ []string) error {
			return runVersion(c, outputFlag)
		},
		Annotations: map[string]string{
			cmdutil.SkipConfigLoadAnnotation: "true",
		},
	}

	c.Flags().StringVarP(&outputFlag, "output", "o", "text", "Output format (text, json, yaml)")

	return c
}

func runVersion(c *cobra.Command, outputFmt string) error {
	if err := cmdutil.CheckOutputFormat(outputFmt, "text", "json", "yaml"); err != nil {
		return err
	}

	info := version.Get()
	if outputFmt == "text" {
		_, err := fmt.Fprintln(c.OutOrStdout(), info.String())
		return err
	}
	return cmdutil.WriteStructured(c.OutOrStdout(), outputFmt, versionOutput{
		Version:       info.Version,
		GitCommit:     info.GitCommit,
		BuildDate:     info.BuildDate,
		GoVersion:     info.GoVersion,
		CUESDKVersion: info.CUESDKVersion,
	})
}
