package modulecmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	opmexit "github.com/opmodel/cli/internal/exit"
	"github.com/opmodel/cli/pkg/k8sgen"
	"github.com/opmodel/cli/pkg/loader"
)

// defaultXRDGroup mirrors defaultCRDGroup so `opm module crd` and
// `opm module xrd` produce artifacts under the same API group unless a
// caller explicitly overrides. Kept as a distinct constant so the two can
// diverge later without touching callers.
const defaultXRDGroup = defaultCRDGroup

// defaultXRDScope is the Crossplane v2 default — most XRDs should use
// Namespaced, which matches how OPM modules are meant to be consumed.
const defaultXRDScope = string(k8sgen.XRDScopeNamespaced)

// Composition flag defaults. Kept as CLI-layer constants so `--help` shows
// a stable value; the pkg/k8sgen layer owns the same defaults and applies
// them when these fields arrive empty.
const (
	defaultCompFunctionName = "function-opm"
	defaultCompStep         = "render-opm-module"
	defaultCompInputAPI     = "template.fn.crossplane.io/v1beta1"
)

// xrdFlags bundles the flag-backed config for the xrd command so runModuleXRD
// stays under the golangci gocyclo/arg-count thresholds as the feature grows.
type xrdFlags struct {
	group        string
	scope        string
	output       string
	compFunction string
	compStep     string
	compInputAPI string
}

// NewModuleXRDCmd creates the module xrd command.
func NewModuleXRDCmd(cfg *config.GlobalConfig) *cobra.Command {
	var f xrdFlags

	c := &cobra.Command{
		Use:   "xrd [path]",
		Short: "Generate a Crossplane v2 XRD and matching Composition from an OPM module",
		Long: `Generate a Crossplane v2 CompositeResourceDefinition (apiextensions.crossplane.io/v2)
and a matching Composition (apiextensions.crossplane.io/v1) from an OPM module. Both
manifests are emitted to stdout as a single multi-document stream.

The XRD's openAPIV3Schema is derived from the module's #config, wrapped under
spec.versions[0].schema.openAPIV3Schema.properties.spec so the XR contract matches
what Crossplane v2 requires. The Composition binds that XRD to function-opm and
tells the function which module to render. Group, kind, plural, singular, and
version are derived from the module's metadata:

  group     --group flag (default "module.opmodel.dev")
  scope     --scope flag (default "Namespaced"; also accepts "Cluster" or "LegacyCluster")
  kind      PascalCase of metadata.name (my-service -> MyService)
  plural    lowercase kind + "s"         (my-service -> myservices)
  singular  lowercase kind                (my-service -> myservice)
  version   v1alpha1 if module major is 0, otherwise v<major>

The Composition's pipeline runs a single step against function-opm. The step name,
function reference, and input apiVersion are configurable; the module path +
version embedded in the function input come from metadata.modulePath and
metadata.version (modulePath is required — without it the function has no module
to render).

Only Crossplane v2 is supported. Claims are removed in v2; the "X" prefix
convention that distinguished composites from claims is therefore not
applied — the kind is derived verbatim from metadata.name.

This is a POC; the derivation rules will change once we handle irregular
plurals and finer-grained version mapping. properties.status is not emitted
on the XRD — compositions own status in v2.

Known limitation: CUE's encoding/openapi cannot emit a schema for a #config
that (directly or transitively) references a definition which embeds another
definition inside a disjunction branch. The OPM catalog's schemas.#Secret
uses that shape, so any module referencing #Secret in its #config will fail
with an "openapi encoder cannot express" error that names the offending
definition. The XRD emitter shares this limitation with the CRD emitter; the
Composition emitter does not (it carries no schema).

A #config that declares a top-level field named "crossplane" is rejected
because Crossplane v2 reserves spec.crossplane.* for its own use.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Print the XRD + Composition for the module in the current directory
  opm module xrd

  # Generate for a module in a specific directory
  opm module xrd ./my-module

  # Use a custom API group
  opm module xrd ./my-module --group example.com

  # Produce a cluster-scoped XR
  opm module xrd ./my-module --scope Cluster

  # Point at a forked function
  opm module xrd ./my-module --comp-function function-opm-fork

  # Emit JSON instead of YAML
  opm module xrd ./my-module -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleXRD(args, cfg, f)
		},
	}

	c.Flags().StringVar(&f.group, "group", defaultXRDGroup, "API group for the generated XRD and Composition")
	c.Flags().StringVar(&f.scope, "scope", defaultXRDScope, "XR scope: Namespaced, Cluster, or LegacyCluster")
	c.Flags().StringVarP(&f.output, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().StringVar(&f.compFunction, "comp-function", defaultCompFunctionName, "Crossplane function referenced by the Composition pipeline step")
	c.Flags().StringVar(&f.compStep, "comp-step", defaultCompStep, "Name of the Composition pipeline step")
	c.Flags().StringVar(&f.compInputAPI, "comp-input-api", defaultCompInputAPI, "apiVersion of the Composition function input payload")

	return c
}

func runModuleXRD(args []string, cfg *config.GlobalConfig, f xrdFlags) error {
	modulePath := cmdutil.ResolveModulePath(args)

	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}

	outputFormat, err := cmdutil.ParseManifestOutputFormat(f.output)
	if err != nil {
		return err
	}

	modVal, err := loader.LoadModulePackage(cfg.CueContext, modulePath)
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("loading module: %w", err),
		}
	}

	xrdManifest, err := k8sgen.BuildXRD(modVal, k8sgen.XRDOptions{
		Group: f.group,
		Scope: k8sgen.XRDScope(f.scope),
	})
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("building XRD: %w", err),
		}
	}

	compManifest, err := k8sgen.BuildComposition(modVal, k8sgen.CompositionOptions{
		Group:           f.group,
		FunctionName:    f.compFunction,
		StepName:        f.compStep,
		InputAPIVersion: f.compInputAPI,
	})
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("building Composition: %w", err),
		}
	}

	return cmdutil.WriteManifestOutput(
		[]*unstructured.Unstructured{xrdManifest, compManifest},
		outputFormat,
		false,
		"",
		"",
	)
}
