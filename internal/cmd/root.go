// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cmdconfig "github.com/opmodel/cli/internal/cmd/config"
	cmdmod "github.com/opmodel/cli/internal/cmd/mod"
	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

// NewRootCmd creates the root command for the OPM CLI.
func NewRootCmd() *cobra.Command {
	var cfg cmdtypes.GlobalConfig

	// Raw flag values — bound to cobra flags, then folded into cfg by PersistentPreRunE.
	var (
		configFlag     string
		registryFlag   string
		verboseFlag    bool
		timestampsFlag bool
	)

	rootCmd := &cobra.Command{
		Use:           "opm",
		Short:         "Open Platform Model CLI",
		Long:          `OPM CLI manages module lifecycle and configuration for the Open Platform Model.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd, &cfg, configFlag, registryFlag, verboseFlag, timestampsFlag)
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to config file (env: OPM_CONFIG)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&registryFlag, "registry", "", "CUE registry URL (env: OPM_REGISTRY)")
	rootCmd.PersistentFlags().BoolVar(&timestampsFlag, "timestamps", true, "Show timestamps in log output")

	// Add subcommands — sub-packages receive *cmdtypes.GlobalConfig for dependency injection.
	rootCmd.AddCommand(NewVersionCmd(&cfg))
	rootCmd.AddCommand(cmdmod.NewModCmd(&cfg))
	rootCmd.AddCommand(cmdconfig.NewConfigCmd(&cfg))

	return rootCmd
}

// initializeConfig sets up logging and loads configuration into cfg.
func initializeConfig(cmd *cobra.Command, cfg *cmdtypes.GlobalConfig, configFlag, registryFlag string, verboseFlag, timestampsFlag bool) error {
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

	cfg.OPMConfig = loadedConfig
	cfg.RegistryFlag = registryFlag
	cfg.Verbose = verboseFlag

	// Resolve base configuration values (config path, registry)
	var rawCfg *config.Config
	if loadedConfig != nil {
		rawCfg = loadedConfig.Config
	}

	resolved, err := config.ResolveBase(config.ResolveBaseOptions{
		ConfigFlag:   configFlag,
		RegistryFlag: registryFlag,
		Config:       rawCfg,
	})
	if err != nil {
		return err
	}

	cfg.ConfigPath = resolved.ConfigPath.Value
	cfg.Registry = resolved.Registry.Value

	// Build LogConfig with precedence: flag > config > default(true)
	logCfg := output.LogConfig{
		Verbose: verboseFlag,
	}

	// Resolve timestamps: flag (if explicitly set) > config > default (nil = true)
	if cmd.Flags().Changed("timestamps") {
		logCfg.Timestamps = output.BoolPtr(timestampsFlag)
	} else if loadedConfig != nil && loadedConfig.Config != nil && loadedConfig.Config.Log.Timestamps != nil {
		logCfg.Timestamps = loadedConfig.Config.Log.Timestamps
	}
	// else: nil means SetupLogging defaults to true

	output.SetupLogging(logCfg)

	// Log base config resolution at DEBUG level
	if verboseFlag {
		output.Debug("initializing CLI",
			"config", cfg.ConfigPath,
			"registry", cfg.Registry,
		)
	}

	return nil
}
