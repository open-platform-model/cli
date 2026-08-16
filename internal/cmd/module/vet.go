package modulecmd

import (
	"fmt"
	"path/filepath"
	"strings"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"github.com/spf13/cobra"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	"github.com/open-platform-model/cli/pkg/loader"
	"github.com/open-platform-model/cli/pkg/validate"
)

// NewModuleVetCmd creates the module vet command.
func NewModuleVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags

	c := &cobra.Command{
		Use:   "vet [path]",
		Short: "Validate module without generating manifests",
		Long: `Validate an OPM module's config inputs without generating manifests.

	This command first verifies the module's identity and coordinates — the
	identity package conforms to core's #IdentityPackage, metadata derives from
	it, and cue.mod agrees with the declared module path — then validates the
	module's #config contract using either the module's debugValues (default) or
	explicit values files passed with -f/--values.
	It does not render resources, resolve providers, or validate instance files.

	Arguments:
	  path    Path to module directory (default: current directory)

	Examples:
	  # Validate debugValues in current directory
	  opm module vet

	  # Validate module against explicit values
	  opm module vet ./my-module -f prod-values.cue

	  # Validate by merging multiple values files
	  opm module vet ./my-module -f base.cue -f prod.cue`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runVet(cfg, args, &rf)
		},
	}

	rf.AddTo(c)

	return c
}

func runVet(cfg *config.GlobalConfig, args []string, rf *cmdutil.RenderFlags) error {
	modulePath := cmdutil.ResolveModulePath(args)
	return runVetModuleOnly(cfg, modulePath, rf)
}

// runVetModuleOnly validates a module directory without an instance.cue.
// It loads the module CUE package with the resolved registry, runs the
// identity/coordinate checks (D16/D18/D21), validates the schema, and checks
// that the values (from -f flag or debugValues field) satisfy #config.
// No instance wrapper, engine render, or cluster connection is required.
func runVetModuleOnly(cfg *config.GlobalConfig, modulePath string, rf *cmdutil.RenderFlags) error {
	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}

	cueCtx, identitySchema, err := identitySchemaForVet(cfg)
	if err != nil {
		return err
	}

	// Identity and coordinate checks run between module load and the values
	// stanza, so a module with no debugValues still reports coordinate drift.
	plan, modVal, err := publish.VetChecks(publish.Options{
		Dir:            modulePath,
		Kind:           publish.KindModule,
		Context:        cueCtx,
		IdentitySchema: identitySchema,
		Registry:       cfg.Registry,
	})
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}
	if len(plan.Refusals) > 0 {
		cmdutil.PrintRefusals(plan.Refusals)
		return &opmexit.ExitError{
			Code:    opmexit.ExitValidationError,
			Err:     fmt.Errorf("module identity checks failed"),
			Printed: true,
		}
	}

	// Derive a display name for log output.
	modName := filepath.Base(modulePath)
	if nameVal := modVal.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
		if name, nameErr := nameVal.String(); nameErr == nil && name != "" {
			modName = name
		}
	}
	moduleLog := output.InstanceLogger(modName)

	moduleLog.Info(output.FormatVetCheck("Identity conforms to #IdentityPackage", "identity/identity.cue"))
	moduleLog.Info(output.FormatVetCheck("Coordinates agree", plan.DeclaredPath))
	if ver := versionState(plan); ver != "" {
		moduleLog.Info(output.FormatVetCheck("Version matches path major", ver))
	}

	// Resolve the values to validate against #config.
	valuesVals, valuesDetail, err := resolveVetValues(cueCtx, modVal, rf)
	if err != nil {
		return err
	}

	for _, valuesVal := range valuesVals {
		if err := valuesVal.Validate(cue.Concrete(true)); err != nil {
			cmdutil.PrintValidationError(valuesDetail+" not concrete", err)
			return &opmexit.ExitError{
				Code:    opmexit.ExitValidationError,
				Err:     fmt.Errorf("%s values are not fully concrete", valuesDetail),
				Printed: true,
			}
		}
	}

	configVal := modVal.LookupPath(cue.ParsePath("#config"))
	if configVal.Exists() {
		if _, cfgErr := validate.Config(configVal, valuesVals, "module", modName); cfgErr != nil {
			cmdutil.PrintValidationError("values do not satisfy #config", cfgErr)
			return &opmexit.ExitError{
				Code:    opmexit.ExitValidationError,
				Err:     cfgErr,
				Printed: true,
			}
		}
	}

	moduleLog.Info(output.FormatVetCheck("Values satisfy #config", valuesDetail))
	moduleLog.Info(output.FormatCheckmark("Module config valid"))

	return nil
}

// identitySchemaForVet builds the per-invocation kernel — the CUE context
// every load shares, and the schema cache the resolved registry threads into
// (vet's loads no longer read only the ambient process environment) — and
// resolves core's #IdentityPackage from it.
func identitySchemaForVet(cfg *config.GlobalConfig) (*cue.Context, cue.Value, error) {
	k := kernel.New(
		kernel.WithRegistry(cfg.Registry),
		kernel.WithSchemaLoader(schema.OCILoader{Registry: cfg.Registry}),
	)
	cueCtx := k.CueContext()
	schemaVal, err := k.SchemaCache().Get(cueCtx)
	if err != nil {
		// A registry round-trip, same failure class as publish's lookup and
		// push: connectivity (exit 3), not a verdict on the module.
		return nil, cue.Value{}, &opmexit.ExitError{
			Code: opmexit.ExitConnectivityError,
			Err:  fmt.Errorf("loading core schema: %w", err),
		}
	}
	identitySchema := schemaVal.LookupPath(cue.MakePath(cue.Def("IdentityPackage")))
	if !identitySchema.Exists() {
		return nil, cue.Value{}, &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("resolved core schema (%s) does not define #IdentityPackage; vet requires a core v2 schema", k.SchemaCache().ResolvedVersion()),
		}
	}
	return cueCtx, identitySchema, nil
}

// resolveVetValues resolves the values to validate against #config: explicit
// -f files, or the module's debugValues.
func resolveVetValues(cueCtx *cue.Context, modVal cue.Value, rf *cmdutil.RenderFlags) ([]cue.Value, string, error) {
	if len(rf.Values) > 0 {
		valuesVals := make([]cue.Value, 0, len(rf.Values))
		basenames := make([]string, 0, len(rf.Values))
		for _, valuesFile := range rf.Values {
			valuesVal, loadErr := loader.LoadValuesFile(cueCtx, valuesFile)
			if loadErr != nil {
				return nil, "", &opmexit.ExitError{
					Code: opmexit.ExitGeneralError,
					Err:  fmt.Errorf("loading values file %q: %w", valuesFile, loadErr),
				}
			}
			valuesVals = append(valuesVals, valuesVal)
			basenames = append(basenames, filepath.Base(valuesFile))
		}
		return valuesVals, strings.Join(basenames, ", "), nil
	}

	debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
	if !debugVal.Exists() {
		return nil, "", &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  fmt.Errorf("module does not define debugValues - add debugValues or provide values with -f"),
		}
	}
	return []cue.Value{debugVal}, "debugValues", nil
}

// versionState returns the concrete identity Version for the vet-check line,
// or "" when the field is open (a valid authoring state vet does not police).
func versionState(plan *publish.Plan) string {
	for _, f := range plan.Identity {
		if f.Name == "Version" && f.State == publish.StateConcrete {
			return f.Value
		}
	}
	return ""
}
