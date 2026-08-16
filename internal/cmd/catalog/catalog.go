// Package catalogcmd provides CLI command implementations for the catalog
// command group.
package catalogcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewCatalogCmd creates the catalog command group.
func NewCatalogCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "catalog",
		Short: "Work with catalog source",
		Long: `Work with OPM catalogs.

		Use this command group when you are starting from catalog source: publish a
		catalog release from its committed tree.`,
	}

	c.AddCommand(NewCatalogPublishCmd(cfg))
	c.AddCommand(NewCatalogRegistryCmd(cfg))
	c.AddCommand(NewCatalogVersionCmd())

	return c
}
