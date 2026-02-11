// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

var (
	// Global flags
	kubeconfigFlag   string
	contextFlag      string
	namespaceFlag    string
	configFlag       string
	outputFormatFlag string
	verboseFlag      bool
	registryFlag     string
	providerFlag     string
	timestampsFlag   bool

	// Resolved configuration (loaded during PersistentPreRunE)
	resolvedRegistry string
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
	rootCmd.PersistentFlags().StringVar(&kubeconfigFlag, "kubeconfig", "", "Path to kubeconfig file (env: OPM_KUBECONFIG)")
	rootCmd.PersistentFlags().StringVar(&contextFlag, "context", "", "Kubernetes context to use (env: OPM_CONTEXT)")
	rootCmd.PersistentFlags().StringVarP(&namespaceFlag, "namespace", "n", "", "Kubernetes namespace (env: OPM_NAMESPACE)")
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to config file (env: OPM_CONFIG)")
	rootCmd.PersistentFlags().StringVarP(&outputFormatFlag, "output", "o", "yaml", "Output format: yaml, json, table, dir")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&registryFlag, "registry", "", "CUE registry URL (env: OPM_REGISTRY)")
	rootCmd.PersistentFlags().StringVar(&providerFlag, "provider", "", "Provider to use for operations")
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
	opmConfig, err := config.LoadOPMConfig(config.LoaderOptions{
		RegistryFlag: registryFlag,
		ConfigFlag:   configFlag,
	})
	if err != nil {
		output.Debug("config load error", "error", err)
		// Don't fail here - allow commands that don't need config to work
	}

	// Store resolved registry
	if opmConfig != nil {
		resolvedRegistry = opmConfig.Registry
	}

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

	// Log config resolution at DEBUG level
	if verboseFlag {
		output.Debug("initializing CLI",
			"kubeconfig", kubeconfigFlag,
			"context", contextFlag,
			"namespace", namespaceFlag,
			"config", configFlag,
			"output", outputFormatFlag,
			"registry_flag", registryFlag,
			"resolved_registry", resolvedRegistry,
		)
	}

	return nil
}

// GetKubeconfig returns the kubeconfig flag value.
func GetKubeconfig() string {
	return kubeconfigFlag
}

// GetContext returns the context flag value.
func GetContext() string {
	return contextFlag
}

// GetNamespace returns the namespace flag value.
func GetNamespace() string {
	return namespaceFlag
}

// GetConfigPath returns the config flag value.
func GetConfigPath() string {
	return configFlag
}

// GetRegistry returns the resolved registry URL (flag > env > config).
func GetRegistry() string {
	return resolvedRegistry
}

// GetRegistryFlag returns the raw --registry flag value.
func GetRegistryFlag() string {
	return registryFlag
}

// GetProvider returns the provider flag value.
func GetProvider() string {
	return providerFlag
}
