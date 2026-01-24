package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmd/config"
	"github.com/opmodel/cli/internal/cmd/mod"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/version"
)

var (
	// Global flags
	flagKubeconfig string
	flagContext    string
	flagNamespace  string
	flagConfig     string
	flagVerbose    bool
)

// rootCmd is the base command for the OPM CLI.
var rootCmd = &cobra.Command{
	Use:   "opm",
	Short: "Open Platform Model CLI",
	Long: `OPM CLI manages OPM modules and bundles for Kubernetes deployments.

It provides commands to:
  - Initialize, validate, and build modules
  - Deploy and manage modules on Kubernetes clusters
  - Publish and fetch modules from OCI registries
  - Manage bundles (collections of modules)`,
	PersistentPreRunE: initializeGlobals,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&flagKubeconfig, "kubeconfig", "", "path to kubeconfig file (env: OPM_KUBECONFIG)")
	rootCmd.PersistentFlags().StringVar(&flagContext, "context", "", "kubernetes context to use (env: OPM_CONTEXT)")
	rootCmd.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "", "target namespace (env: OPM_NAMESPACE)")
	rootCmd.PersistentFlags().StringVarP(&flagConfig, "config", "c", "", "path to config file (env: OPM_CONFIG)")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "increase output verbosity")

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(mod.NewModCmd())
}

// initializeGlobals sets up logging and config based on global flags.
func initializeGlobals(cmd *cobra.Command, _ []string) error {
	// Set up logging
	output.SetupLogging(flagVerbose)

	// Log version info at debug level
	info := version.GetInfo()
	output.Debug("OPM CLI started",
		"version", info.Version,
		"cue_sdk", info.CUESDKVersion,
	)

	// Check CUE binary compatibility and warn if needed
	cueInfo := version.DetectCUEBinary()
	if cueInfo.Found && !cueInfo.Compatible {
		output.Warn("CUE binary version mismatch",
			"sdk", info.CUESDKVersion,
			"binary", cueInfo.Version,
			"message", cueInfo.Message,
		)
	}

	return nil
}

// GetKubeconfig returns the kubeconfig path from flags or environment.
func GetKubeconfig() string {
	if flagKubeconfig != "" {
		return flagKubeconfig
	}
	if env := os.Getenv("OPM_KUBECONFIG"); env != "" {
		return env
	}
	return ""
}

// GetContext returns the kubernetes context from flags or environment.
func GetContext() string {
	if flagContext != "" {
		return flagContext
	}
	if env := os.Getenv("OPM_CONTEXT"); env != "" {
		return env
	}
	return ""
}

// GetNamespace returns the namespace from flags or environment.
func GetNamespace() string {
	if flagNamespace != "" {
		return flagNamespace
	}
	if env := os.Getenv("OPM_NAMESPACE"); env != "" {
		return env
	}
	return ""
}

// GetConfigFile returns the config file path from flags or environment.
func GetConfigFile() string {
	if flagConfig != "" {
		return flagConfig
	}
	if env := os.Getenv("OPM_CONFIG"); env != "" {
		return env
	}
	return ""
}
