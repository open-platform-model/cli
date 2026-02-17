// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

var (
	// Global flags
	configFlag     string
	verboseFlag    bool
	registryFlag   string
	timestampsFlag bool

	// Resolved configuration (loaded during PersistentPreRunE)
	opmConfig          *config.OPMConfig
	resolvedBaseConfig *config.ResolvedBaseConfig
)

// NewRootCmd creates the root command for the OPM CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "opm",
		Short:         "Open Platform Model CLI",
		Long:          `OPM CLI manages module lifecycle and configuration for the Open Platform Model.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeGlobals(cmd)
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to config file (env: OPM_CONFIG)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&registryFlag, "registry", "", "CUE registry URL (env: OPM_REGISTRY)")
	rootCmd.PersistentFlags().BoolVar(&timestampsFlag, "timestamps", true, "Show timestamps in log output")

	// Add subcommands
	rootCmd.AddCommand(NewModCmd())
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewVersionCmd())

	return rootCmd
}

// initializeGlobals sets up logging and loads configuration.
func initializeGlobals(cmd *cobra.Command) error {
	// Load configuration first so we can use config values for logging setup
	loadedConfig, err := config.LoadOPMConfig(config.LoaderOptions{
		RegistryFlag: registryFlag,
		ConfigFlag:   configFlag,
	})
	if err != nil {
		// Config file exists but is invalid - fail immediately
		// If config doesn't exist, LoadOPMConfig returns defaults (no error)
		return fmt.Errorf("configuration error: %w", err)
	}

	// Store loaded config in package-level var
	opmConfig = loadedConfig

	// Resolve base configuration values (config path, registry)
	var cfg *config.Config
	if opmConfig != nil {
		cfg = opmConfig.Config
	}

	resolved, err := config.ResolveBase(config.ResolveBaseOptions{
		ConfigFlag:   configFlag,
		RegistryFlag: registryFlag,
		Config:       cfg,
	})
	if err != nil {
		return err
	}

	// Store resolved base config in package-level var
	resolvedBaseConfig = resolved

	// Build LogConfig with precedence: flag > config > default(true)
	logCfg := output.LogConfig{
		Verbose: verboseFlag,
	}

	// Resolve timestamps: flag (if explicitly set) > config > default (nil = true)
	if cmd.Flags().Changed("timestamps") {
		// Flag was explicitly set by user
		logCfg.Timestamps = output.BoolPtr(timestampsFlag)
	} else if opmConfig != nil && opmConfig.Config != nil && opmConfig.Config.Log.Timestamps != nil {
		// Config has a value
		logCfg.Timestamps = opmConfig.Config.Log.Timestamps
	}
	// else: nil means SetupLogging defaults to true

	output.SetupLogging(logCfg)

	// Log base config resolution at DEBUG level
	if verboseFlag {
		output.Debug("initializing CLI",
			"config", resolvedBaseConfig.ConfigPath.Value,
			"registry", resolvedBaseConfig.Registry.Value,
		)
	}

	return nil
}

// GetOPMConfig returns the loaded OPM configuration.
func GetOPMConfig() *config.OPMConfig {
	return opmConfig
}

// GetConfigPath returns the resolved config path value.
func GetConfigPath() string {
	if resolvedBaseConfig != nil {
		return resolvedBaseConfig.ConfigPath.Value
	}
	return configFlag
}

// GetRegistry returns the resolved registry URL.
func GetRegistry() string {
	if resolvedBaseConfig != nil {
		return resolvedBaseConfig.Registry.Value
	}
	return ""
}

// GetRegistryFlag returns the raw --registry flag value.
func GetRegistryFlag() string {
	return registryFlag
}
