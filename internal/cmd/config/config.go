// Package config provides the config command group for the OPM CLI.
package config

import (
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command group.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "CLI configuration management",
		Long:  `Manage OPM CLI configuration settings.`,
	}

	// Add subcommands
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newVetCmd())

	return cmd
}
