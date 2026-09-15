// Package platformcmd provides CLI command implementations for the platform
// command group.
package platformcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewPlatformCmd creates the platform command group.
func NewPlatformCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "platform",
		Short: "Inspect platform modules",
		Long: `Inspect the platform a render would run against.

opm platform reads a platform module offline: it applies nothing, renders
nothing and contacts no cluster.`,
	}

	c.AddCommand(NewPlatformCheckCmd(cfg))

	return c
}
