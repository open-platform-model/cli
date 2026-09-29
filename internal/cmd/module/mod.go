// Package modulecmd provides CLI command implementations for the module command group.
package modulecmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewModuleCmd creates the module command group.
func NewModuleCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:     "module",
		Aliases: []string{"mod"},
		Short:   "Work with module source",
		Long: `Work with OPM modules.

Use this command group when you are starting from a module: initialize,
tidy, validate, version and publish its source, or render and deploy it
through a synthetic instance with 'opm module build' and 'opm module apply'.
build and apply take a module directory or a published module path.

For an instance package you own, use 'opm instance build' or
'opm instance apply'.`,
	}

	c.AddCommand(NewModuleInitCmd(cfg))
	c.AddCommand(NewModuleTemplateCmd())
	c.AddCommand(NewModuleVetCmd(cfg))
	c.AddCommand(NewModuleBuildCmd(cfg))
	c.AddCommand(NewModuleApplyCmd(cfg))
	c.AddCommand(NewModulePublishCmd(cfg))
	c.AddCommand(NewModuleVersionCmd())
	c.AddCommand(NewModuleTidyCmd(cfg))

	return c
}
