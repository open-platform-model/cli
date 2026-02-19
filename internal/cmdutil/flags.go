// Package cmdutil provides shared command utilities for mod subcommands.
// It centralizes flag group management, render pipeline orchestration,
// Kubernetes client creation, and output formatting helpers.
package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RenderFlags holds flags common to commands that render modules
// (apply, build, vet, diff).
type RenderFlags struct {
	Values      []string
	Namespace   string
	ReleaseName string
	Provider    string
}

// AddTo registers the render flags on the given cobra command.
func (f *RenderFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&f.Namespace, "namespace", "n", "",
		"Target namespace")
	cmd.Flags().StringVar(&f.ReleaseName, "release-name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&f.Provider, "provider", "",
		"Provider to use (default: from config)")
}

// K8sFlags holds flags for Kubernetes cluster connection
// (apply, diff, delete, status).
type K8sFlags struct {
	Kubeconfig string
	Context    string
}

// AddTo registers the Kubernetes connection flags on the given cobra command.
func (f *K8sFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Kubeconfig, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&f.Context, "context", "",
		"Kubernetes context to use")
}

// ReleaseSelectorFlags holds flags for identifying a release on the cluster
// (delete, status).
type ReleaseSelectorFlags struct {
	ReleaseName string
	ReleaseID   string
	Namespace   string
}

// AddTo registers the release selector flags on the given cobra command.
func (f *ReleaseSelectorFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Namespace, "namespace", "n", "",
		"Target namespace (default: from config)")
	cmd.Flags().StringVar(&f.ReleaseName, "release-name", "",
		"Release name (mutually exclusive with --release-id)")
	cmd.Flags().StringVar(&f.ReleaseID, "release-id", "",
		"Release identity UUID (mutually exclusive with --release-name)")
}

// Validate checks that exactly one of ReleaseName or ReleaseID is provided.
func (f *ReleaseSelectorFlags) Validate() error {
	if f.ReleaseName != "" && f.ReleaseID != "" {
		return fmt.Errorf("--release-name and --release-id are mutually exclusive")
	}
	if f.ReleaseName == "" && f.ReleaseID == "" {
		return fmt.Errorf("either --release-name or --release-id is required")
	}
	return nil
}

// LogName returns a human-readable name for logging. It prefers ReleaseName;
// if empty, it returns a truncated ReleaseID prefix.
func (f *ReleaseSelectorFlags) LogName() string {
	if f.ReleaseName != "" {
		return f.ReleaseName
	}
	if len(f.ReleaseID) >= 8 {
		return fmt.Sprintf("release:%s", f.ReleaseID[:8])
	}
	return fmt.Sprintf("release:%s", f.ReleaseID)
}

// ResolveModulePath returns the module path from command args,
// defaulting to the current directory.
func ResolveModulePath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}
