// Package mod provides CLI command implementations for the mod command group.
package mod

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
)

// NewModCmd creates the mod command group.
func NewModCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "mod",
		Short: "Module operations",
		Long:  `Module operations for OPM modules.`,
	}

	c.AddCommand(NewModInitCmd(cfg))
	c.AddCommand(NewModBuildCmd(cfg))
	c.AddCommand(NewModVetCmd(cfg))
	c.AddCommand(NewModApplyCmd(cfg))
	c.AddCommand(NewModDeleteCmd(cfg))
	c.AddCommand(NewModDiffCmd(cfg))
	c.AddCommand(NewModStatusCmd(cfg))

	return c
}
