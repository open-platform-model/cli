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
		Short:   "Work with module source",
		Long: `Work with OPM modules.

		Use this command group when you are starting from module source: initialize a
		module, render it, or validate it.

		For all deployed release operations, use 'opm release'.`,
	}

	c.AddCommand(NewModuleInitCmd(cfg))
	c.AddCommand(NewModuleBuildCmd(cfg))
	c.AddCommand(NewModuleVetCmd(cfg))

	return c
}
