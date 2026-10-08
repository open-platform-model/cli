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
	// Platform is the --platform platform module directory (0006:D21;
	// highest platform-source precedence). Supersedes the retired --provider
	// flag.
	Platform string
	// SkipUnprovided is --skip-unprovided: the kernel's switch to skip
	// provider-fulfilled demands nothing on the platform provides.
	SkipUnprovided bool
}

// AddTo registers the render flags on the given cobra command.
func (f *RenderFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&f.Namespace, "namespace", "n", "",
		"Target namespace")
	cmd.Flags().StringVar(&f.InstanceName, "instance-name", "",
		"Instance name (default: <module name>-debug)")
	cmd.Flags().StringVar(&f.Platform, "platform", "",
		modulePlatformFlagHelp)
	cmd.Flags().BoolVar(&f.SkipUnprovided, "skip-unprovided", false,
		skipUnprovidedHelp)
}

// The --platform help text of the render-bearing commands: the flag names a
// platform module directory (0019:D5), never a data file, and outranks every
// other platform source. The module commands fall back to the module's own
// deps, the instance commands to the instance package's.
const (
	modulePlatformFlagHelp   = "Platform module directory (overrides the cluster Platform and the module's own deps)"
	instancePlatformFlagHelp = "Platform module directory (overrides the cluster Platform and the instance's own deps)"
)

// skipUnprovidedHelp is the --skip-unprovided help text of every
// render-bearing command. Which demands are skippable is the kernel's rule;
// the flag only sets its switch.
const skipUnprovidedHelp = "Render what the platform can: skip provider-fulfilled contracts nothing on the platform provides, and report each one"

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
// (delete, status). Was: ReleaseSelectorFlags (0002:D10).
type InstanceSelectorFlags struct {
	InstanceName string
	InstanceID   string
	Namespace    string
}

// AddTo registers the instance selector flags on the given cobra command.
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

// DeprecateFlag marks the flag old of c as a deprecated spelling of the flag
// replacement. Cobra then leaves old out of help and completion and prints
// one line on standard error when it is used; the flag keeps working. Both
// flags must be registered on c: a missing one is a programming error and
// panics when the command is built.
func DeprecateFlag(c *cobra.Command, old, replacement string) {
	if c.Flags().Lookup(replacement) == nil {
		panic(fmt.Sprintf("cmdutil.DeprecateFlag: %s has no --%s flag", c.CommandPath(), replacement))
	}
	if err := c.Flags().MarkDeprecated(old, "use --"+replacement); err != nil {
		panic(fmt.Sprintf("cmdutil.DeprecateFlag: %s: %v", c.CommandPath(), err))
	}
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
	// Values are additional values CUE files (-f/--values flag).
	// The instance package already carries its own values.cue; these files
	// are added on top of it as trailing values sources.
	Values []string
	// Platform is the --platform platform module directory (0006:D21;
	// highest platform-source precedence). Supersedes the retired --provider
	// flag.
	Platform string
	// SkipUnprovided is --skip-unprovided: the kernel's switch to skip
	// provider-fulfilled demands nothing on the platform provides.
	SkipUnprovided bool
}

// AddTo registers the instance file flags on the given cobra command.
func (f *InstanceFileFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Values, "values", "f", nil,
		"Values files added on top of the instance package, which already includes its values.cue (can be repeated)")
	cmd.Flags().StringVar(&f.Platform, "platform", "",
		instancePlatformFlagHelp)
	cmd.Flags().BoolVar(&f.SkipUnprovided, "skip-unprovided", false,
		skipUnprovidedHelp)
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

// DeleteDataPruneFlagHelp is the help of --delete-data on the apply commands.
const DeleteDataPruneFlagHelp = "Also prune stale PersistentVolumeClaims and the data on them (kept by default)"
