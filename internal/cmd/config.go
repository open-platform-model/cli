// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command group.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	Long: `Configuration management for the OPM CLI.`,
	}

	// Add subcommands
	cmd.AddCommand(NewConfigInitCmd())
	cmd.AddCommand(NewConfigVetCmd())

	return cmd
}
