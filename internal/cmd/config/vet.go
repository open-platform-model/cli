package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/config"
)

func newVetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vet",
		Short: "Validate the OPM configuration file",
		Long: `Validate the OPM configuration file against the internal schema.

The command validates the configuration file at ~/.opm/config.yaml by default.
Use --config flag to specify a different location.`,
		RunE: runVet,
	}
}

func runVet(command *cobra.Command, _ []string) error {
	// Get config file path
	configFile := command.Flag("config").Value.String()
	if configFile == "" {
		var err error
		configFile, err = config.GetConfigFile()
		if err != nil {
			return fmt.Errorf("getting config file path: %w", err)
		}
	}

	// Expand path
	expandedPath, err := config.ExpandPath(configFile)
	if err != nil {
		return fmt.Errorf("expanding config path: %w", err)
	}

	// Check if file exists
	exists, err := config.ConfigFileExists(expandedPath)
	if err != nil {
		return fmt.Errorf("checking config file: %w", err)
	}

	if !exists {
		return cmd.NewExitError(
			fmt.Errorf("config file not found: %s", expandedPath),
			cmd.ExitNotFound,
		)
	}

	// Create validator
	validator, err := config.NewValidator()
	if err != nil {
		return fmt.Errorf("creating validator: %w", err)
	}

	// Validate the file
	if err := validator.ValidateFile(expandedPath); err != nil {
		// Check if it's a validation error
		if validationErrs, ok := err.(config.ValidationErrors); ok {
			fmt.Fprintln(command.ErrOrStderr(), "Error: config validation failed")
			fmt.Fprintf(command.ErrOrStderr(), "  File: %s\n\n", expandedPath)
			for _, e := range validationErrs {
				fmt.Fprintf(command.ErrOrStderr(), "  %s: %s\n", e.Field, e.Message)
			}
			return cmd.NewExitError(err, cmd.ExitValidationError)
		}
		return fmt.Errorf("validating config: %w", err)
	}

	fmt.Fprintf(command.OutOrStdout(), "Config file is valid: %s\n", expandedPath)
	return nil
}
