// Package instance provides CLI command implementations for the instance command group.
// Was: package release / "opm release" command group (renamed for enhancement 0002 D6).
package instance

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewInstanceCmd creates the instance command group.
// Was: NewReleaseCmd ("opm release", alias "rel"). The old verb/alias are dropped — no back-compat (enhancement 0002 D8).
func NewInstanceCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:     "instance",
		Aliases: []string{"inst"},
		Short:   "Work with instance files and deployed instances",
		Long: `Work with OPM instance files.

Use this command group when you are starting from an instance definition: render
it, validate it, diff it, apply it, or inspect an already deployed instance.

For operations that start from module source, use 'opm module'.`,
	}

	// Render commands (positional arg = instance file path)
	c.AddCommand(NewInstanceVetCmd(cfg))
	c.AddCommand(NewInstanceBuildCmd(cfg))
	c.AddCommand(NewInstanceApplyCmd(cfg))
	c.AddCommand(NewInstanceDiffCmd(cfg))

	// Cluster-query commands (positional arg = instance name or UUID)
	c.AddCommand(NewInstanceStatusCmd(cfg))
	c.AddCommand(NewInstanceTreeCmd(cfg))
	c.AddCommand(NewInstanceEventsCmd(cfg))
	c.AddCommand(NewInstanceDeleteCmd(cfg))
	c.AddCommand(NewInstanceListCmd(cfg))

	return c
}
