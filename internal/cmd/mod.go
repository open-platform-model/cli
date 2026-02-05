// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/spf13/cobra"
)

// NewModCmd creates the mod command group.
func NewModCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mod",
		Short: "Module operations",
		Long: `Module operations for OPM modules.

Commands:
  init      Create a new module from template
  build     Render module to manifests
  apply     Apply manifests to cluster (not yet implemented)
  delete    Delete resources from cluster (not yet implemented)
  diff      Show differences with cluster (not yet implemented)
  status    Show resource status (not yet implemented)`,
	}

	// Add subcommands
	cmd.AddCommand(NewModInitCmd())
	cmd.AddCommand(NewModBuildCmd())

	// Stub commands for future implementation
	cmd.AddCommand(NewModApplyStubCmd())
	cmd.AddCommand(NewModDeleteStubCmd())
	cmd.AddCommand(NewModDiffStubCmd())
	cmd.AddCommand(NewModStatusStubCmd())

	return cmd
}
