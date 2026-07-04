// Package operatorcmd provides CLI command implementations for the operator
// command group.
package operatorcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewOperatorCmd creates the operator command group.
func NewOperatorCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "operator",
		Short: "Install and manage the opm-operator on a cluster",
		Long: `Install, upgrade, and remove the opm-operator on a Kubernetes cluster.

opm operator prepares a cluster for OPM workflows that depend on the operator:
its CRDs (ModuleInstance, ModulePackage, Platform) and, for full deployments,
the running controller itself.`,
	}

	c.AddCommand(NewOperatorInstallCmd(cfg))
	c.AddCommand(NewOperatorUninstallCmd(cfg))

	return c
}
