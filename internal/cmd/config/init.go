package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/config"
)

var initForce bool

func newInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new OPM configuration file",
		Long: `Create a new OPM configuration file with default values.

The configuration file is created at ~/.opm/config.yaml by default.
Use --config flag to specify a different location.`,
		RunE: runInit,
	}

	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config file")

	return initCmd
}

func runInit(command *cobra.Command, _ []string) error {
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

	if exists && !initForce {
		return cmd.NewExitError(
			fmt.Errorf("config file already exists at %s (use --force to overwrite)", expandedPath),
			cmd.ExitGeneralError,
		)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Add header comment
	header := []byte("# OPM CLI Configuration\n# See: https://opmodel.dev/docs/cli/config\n\n")
	data = append(header, data...)

	// Write file
	if err := os.WriteFile(expandedPath, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Fprintf(command.OutOrStdout(), "Config file created: %s\n", expandedPath)
	return nil
}
