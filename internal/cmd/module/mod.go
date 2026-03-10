// Package modulecmd provides CLI command implementations for the module command group.
package modulecmd

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
)

// NewModuleCmd creates the module command group.
func NewModuleCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:     "module",
		Aliases: []string{"mod"},
		Short:   "Work with module source and deployed module releases",
		Long: `Work with OPM modules.

Use this command group when you are starting from module source: initialize a
module, render it, validate it, or apply it directly to a cluster.

For operations that start from a release file, use 'opm release'.`,
	}

	c.AddCommand(NewModuleInitCmd(cfg))
	c.AddCommand(NewModuleBuildCmd(cfg))
	c.AddCommand(NewModuleVetCmd(cfg))
	c.AddCommand(NewModuleListCmd(cfg))
	c.AddCommand(NewModuleApplyCmd(cfg))
	c.AddCommand(NewModuleDeleteCmd(cfg))
	c.AddCommand(NewModuleStatusCmd(cfg))
	c.AddCommand(NewModuleTreeCmd(cfg))
	c.AddCommand(NewModuleEventsCmd(cfg))

	return c
}
