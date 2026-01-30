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
  vet       Validate module schema
  tidy      Update module dependencies
  build     Render module to manifests (see 004-render-and-lifecycle-spec)
  apply     Apply manifests to cluster (see 004-render-and-lifecycle-spec)
  delete    Delete resources from cluster (see 004-render-and-lifecycle-spec)
  diff      Show differences with cluster (see 004-render-and-lifecycle-spec)
  status    Show resource status (see 004-render-and-lifecycle-spec)`,
	}

	// Add subcommands
	cmd.AddCommand(NewModInitCmd())
	cmd.AddCommand(NewModVetCmd())
	cmd.AddCommand(NewModTidyCmd())

	// Stub commands for 004-render-and-lifecycle-spec
	cmd.AddCommand(NewModBuildStubCmd())
	cmd.AddCommand(NewModApplyStubCmd())
	cmd.AddCommand(NewModDeleteStubCmd())
	cmd.AddCommand(NewModDiffStubCmd())
	cmd.AddCommand(NewModStatusStubCmd())

	return cmd
}
