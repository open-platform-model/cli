// Package registrycmd provides the registry command group — authenticating
// to the OCI registries OPM publishes to and pulls from (0011 D11/D24).
package registrycmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
)

// NewRegistryCmd creates the registry command group. Login is its first
// member; the group names what is authenticated to (D24 — a bare `opm
// login` reads as logging in to an OPM service that does not exist) and
// gives logout and the credential-helper flow a home when they arrive.
func NewRegistryCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "registry",
		Short: "Authenticate to OCI registries",
		Long: `Work with the OCI registries OPM publishes to and pulls from.

	Use this command group to manage registry credentials: they are stored in
	the standard docker credential file, the store OPM's push and pull both
	read — the same file 'docker login' writes.`,
	}

	c.AddCommand(NewRegistryLoginCmd(cfg))

	return c
}
