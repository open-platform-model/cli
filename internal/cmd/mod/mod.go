// Package mod provides the `opm mod` command group.
package mod

import (
	"github.com/spf13/cobra"
)

// NewModCmd creates the mod command group.
func NewModCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mod",
		Short: "Module operations",
		Long:  `Commands for creating, building, and managing OPM modules.`,
	}

	// Add subcommands
	cmd.AddCommand(
		NewInitCmd(),
		NewTemplateCmd(),
		NewVetCmd(),
		NewTidyCmd(),
		NewBuildCmd(),
		NewApplyCmd(),
		NewDiffCmd(),
		NewStatusCmd(),
		NewDeleteCmd(),
	)

	return cmd
}
