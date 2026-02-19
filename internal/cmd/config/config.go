// Package config provides CLI command implementations for the config command group.
package config

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
)

// NewConfigCmd creates the config command group.
func NewConfigCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Configuration management for the OPM CLI.`,
	}

	c.AddCommand(NewConfigInitCmd(cfg))
	c.AddCommand(NewConfigVetCmd(cfg))

	return c
}
