// Package cmdutil provides shared command utilities for mod subcommands.
// It centralizes flag group management, render pipeline orchestration,
// Kubernetes client creation, and output formatting helpers.
package cmdutil

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// RenderFlags holds flags common to commands that render modules
// (apply, build, vet).
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
// (apply, delete, status).
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

// ReleaseFileFlags holds flags specific to release-file-based rendering.
type ReleaseFileFlags struct {
	// Module is the path to a local module directory (--module flag).
	// Used to fill #module in the release file when not imported from a registry.
	Module string
	// Provider is the provider name to use (--provider flag).
	Provider string
	// Values are additional values CUE files (-f/--values flag).
	// When empty, values.cue next to the release file is used if it exists.
	Values []string
}

// AddTo registers the release file flags on the given cobra command.
func (f *ReleaseFileFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Module, "module", "",
		"Path to local module directory (fills #module in the release file)")
	cmd.Flags().StringVar(&f.Provider, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Additional values files (can be repeated; default: values.cue next to the release file)")
}

// uuidPattern matches a UUID v4/v5: 8-4-4-4-12 lowercase hex digits.
var uuidPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// ResolveReleaseIdentifier inspects a positional argument and returns either
// a release name or a release UUID. Detection is based on the UUID v4/v5
// pattern: 8-4-4-4-12 lowercase hex digits separated by dashes.
//
// Exactly one of the returned values will be non-empty.
func ResolveReleaseIdentifier(arg string) (name, uuid string) {
	if uuidPattern.MatchString(arg) {
		return "", arg
	}
	return arg, ""
}
