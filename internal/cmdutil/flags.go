// Package cmdutil provides small CLI helpers shared across command packages.
// It holds reusable flag groups, annotations, instance targeting, and a few
// command-facing utility functions that are not full workflow orchestration.
package cmdutil

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// RenderFlags holds flags common to commands that render modules
// (apply, build, vet).
type RenderFlags struct {
	Values       []string
	Namespace    string
	InstanceName string
	Provider     string
	// Platform is the --platform local override file (0006 D21; highest
	// platform-source precedence). Supersedes --provider, which is retired
	// with kernel adoption (0006 C2 Phase C).
	Platform string
}

// AddTo registers the render flags on the given cobra command.
func (f *RenderFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&f.Namespace, "namespace", "n", "",
		"Target namespace")
	cmd.Flags().StringVar(&f.InstanceName, "instance-name", "",
		"Instance name (default: module name)") // Was: --release-name (enhancement 0002 D-X4.2)
	cmd.Flags().StringVar(&f.Provider, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().StringVar(&f.Platform, "platform", "",
		"Path to a local platform file (overrides the cluster Platform and ~/.opm/platform.cue)")
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

// InstanceSelectorFlags holds flags for identifying an instance on the cluster
// (delete, status). Was: ReleaseSelectorFlags (enhancement 0002 D10).
type InstanceSelectorFlags struct {
	InstanceName string
	InstanceID   string
	Namespace    string
}

// AddTo registers the instance selector flags on the given cobra command.
// Was: --release-name/--release-id (enhancement 0002 D-X4.2; hard rename, no alias).
func (f *InstanceSelectorFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Namespace, "namespace", "n", "",
		"Target namespace (default: from config)")
	cmd.Flags().StringVar(&f.InstanceName, "instance-name", "",
		"Instance name (mutually exclusive with --instance-id)")
	cmd.Flags().StringVar(&f.InstanceID, "instance-id", "",
		"Instance identity UUID (mutually exclusive with --instance-name)")
}

// Validate checks that exactly one of InstanceName or InstanceID is provided.
func (f *InstanceSelectorFlags) Validate() error {
	if f.InstanceName != "" && f.InstanceID != "" {
		return fmt.Errorf("--instance-name and --instance-id are mutually exclusive")
	}
	if f.InstanceName == "" && f.InstanceID == "" {
		return fmt.Errorf("either --instance-name or --instance-id is required")
	}
	return nil
}

// LogName returns a human-readable name for logging. It prefers InstanceName;
// if empty, it returns a truncated InstanceID prefix.
func (f *InstanceSelectorFlags) LogName() string {
	if f.InstanceName != "" {
		return f.InstanceName
	}
	if len(f.InstanceID) >= 8 {
		return fmt.Sprintf("instance:%s", f.InstanceID[:8])
	}
	return fmt.Sprintf("instance:%s", f.InstanceID)
}

// ResolveModulePath returns the module path from command args,
// defaulting to the current directory.
func ResolveModulePath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

// InstanceFileFlags holds flags specific to instance-file-based rendering.
type InstanceFileFlags struct {
	// Provider is the provider name to use (--provider flag).
	Provider string
	// Values are additional values CUE files (-f/--values flag).
	// When empty, values.cue next to the instance file is used if it exists.
	Values []string
	// Platform is the --platform local override file (0006 D21; highest
	// platform-source precedence). Supersedes --provider, which is retired
	// with kernel adoption (0006 C2 Phase C).
	Platform string
}

// AddTo registers the instance file flags on the given cobra command.
func (f *InstanceFileFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Additional values files (can be repeated; default: values.cue next to the instance file)")
	cmd.Flags().StringVar(&f.Platform, "platform", "",
		"Path to a local platform file (overrides the cluster Platform and ~/.opm/platform.cue)")
}

// uuidPattern matches a UUID v4/v5: 8-4-4-4-12 lowercase hex digits.
var uuidPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// ResolveInstanceIdentifier inspects a positional argument and returns either
// an instance name or an instance UUID. Detection is based on the UUID v4/v5
// pattern: 8-4-4-4-12 lowercase hex digits separated by dashes.
//
// Exactly one of the returned values will be non-empty.
func ResolveInstanceIdentifier(arg string) (name, uuid string) {
	if uuidPattern.MatchString(arg) {
		return "", arg
	}
	return arg, ""
}
