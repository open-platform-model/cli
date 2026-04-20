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

// defaultCRDGroup is the API group applied to generated CRDs when the caller
// does not pass --group. Aligned with the opmodel.dev registry used elsewhere
// in the CLI; override per-invocation for forks or private deployments.
const defaultCRDGroup = "module.opmodel.dev"

// NewModuleCRDCmd creates the module crd command.
func NewModuleCRDCmd(cfg *config.GlobalConfig) *cobra.Command {
	var (
		groupFlag  string
		outputFlag string
	)

	c := &cobra.Command{
		Use:   "crd [path]",
		Short: "Generate a CustomResourceDefinition from a module's #config",
		Long: `Generate a Kubernetes CustomResourceDefinition (apiextensions.k8s.io/v1) from an OPM module.

The module's #config definition is converted to an OpenAPI v3 schema and
embedded in spec.versions[0].schema.openAPIV3Schema. The CRD group, kind,
plural, singular, and version are derived from the module's metadata:

  group     --group flag (default "module.opmodel.dev")
  kind      PascalCase of metadata.name (my-service -> MyService)
  plural    lowercase kind + "s"         (my-service -> myservices)
  singular  lowercase kind                (my-service -> myservice)
  version   v1alpha1 if module major is 0, otherwise v<major>

This is a POC; the derivation rules will change once we handle irregular
plurals and finer-grained version mapping. Scope is always Namespaced and no
status subresource is emitted.

Known limitation: CUE's encoding/openapi cannot emit a schema for a #config
that (directly or transitively) references a definition which embeds another
definition inside a disjunction branch. The OPM catalog's schemas.#Secret
uses that shape, so any module referencing #Secret in its #config will fail
with an "openapi encoder cannot express" error that names the offending
definition.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Print the CRD for the module in the current directory
  opm module crd

  # Generate a CRD for a module in a specific directory
  opm module crd ./my-module

  # Use a custom API group
  opm module crd ./my-module --group example.com

  # Emit JSON instead of YAML
  opm module crd ./my-module -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleCRD(args, cfg, groupFlag, outputFlag)
		},
	}

	c.Flags().StringVar(&groupFlag, "group", defaultCRDGroup, "API group for the generated CRD")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")

	return c
}

func runModuleCRD(args []string, cfg *config.GlobalConfig, groupFlag, outputFmt string) error {
	modulePath := cmdutil.ResolveModulePath(args)

	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}

	outputFormat, err := cmdutil.ParseManifestOutputFormat(outputFmt)
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

	crdManifest, err := k8sgen.BuildCRD(modVal, k8sgen.Options{Group: groupFlag})
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("building CRD: %w", err),
		}
	}

	return cmdutil.WriteManifestOutput(
		[]*unstructured.Unstructured{crdManifest},
		outputFormat,
		false,
		"",
		"",
	)
}
