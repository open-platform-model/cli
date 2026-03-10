// Package release provides CLI command implementations for the release command group.
package release

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
)

// NewReleaseCmd creates the release command group.
func NewReleaseCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:     "release",
		Aliases: []string{"rel"},
		Short:   "Work with release files and deployed releases",
		Long: `Work with OPM release files.

Use this command group when you are starting from a release definition: render
it, validate it, diff it, apply it, or inspect an already deployed release.

For operations that start from module source, use 'opm module'.`,
	}

	// Render commands (positional arg = release file path)
	c.AddCommand(NewReleaseVetCmd(cfg))
	c.AddCommand(NewReleaseBuildCmd(cfg))
	c.AddCommand(NewReleaseApplyCmd(cfg))
	c.AddCommand(NewReleaseDiffCmd(cfg))

	// Cluster-query commands (positional arg = release name or UUID)
	c.AddCommand(NewReleaseStatusCmd(cfg))
	c.AddCommand(NewReleaseTreeCmd(cfg))
	c.AddCommand(NewReleaseEventsCmd(cfg))
	c.AddCommand(NewReleaseDeleteCmd(cfg))
	c.AddCommand(NewReleaseListCmd(cfg))

	return c
}
