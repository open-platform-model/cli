// Package controllercmd provides CLI command implementations for the controller
// command group.
package controllercmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewControllerCmd creates the controller command group.
func NewControllerCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "controller",
		Short: "Install and manage the opm-controller on a cluster",
		Long: `Install, upgrade, and remove the opm-controller on a Kubernetes cluster.

opm controller prepares a cluster for OPM workflows that depend on the controller:
its CRDs (ModuleInstance, ModulePackage, Platform,
TransformerRegistration) and, for full deployments,
the running controller itself.`,
	}

	c.AddCommand(NewControllerInstallCmd(cfg))
	c.AddCommand(NewControllerUninstallCmd(cfg))

	return c
}
