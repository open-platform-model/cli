package modulecmd

import (
	"context"
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
	"github.com/open-platform-model/cli/internal/workflow/render"
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
			return runVet(c.Context(), cfg, args, &rf)
		},
	}

	rf.AddTo(c)

	return c
}

func runVet(ctx context.Context, cfg *config.GlobalConfig, args []string, rf *cmdutil.RenderFlags) error {
	modulePath := cmdutil.ResolveModulePath(args)
	return runVetModuleOnly(ctx, cfg, modulePath, rf)
}

// runVetModuleOnly validates a module directory without an instance.cue.
// It loads the module CUE package with the resolved registry, runs the
// identity/coordinate checks (0011:D16/D18/D21), then resolves the values (-f
// files, else the debugValues field) as kernel sources exactly as `opm
// module build` does and validates them against #config through the
// kernel's layered validation, so vet and build agree on a verdict. No
// instance wrapper, engine render, or cluster connection is required.
func runVetModuleOnly(ctx context.Context, cfg *config.GlobalConfig, modulePath string, rf *cmdutil.RenderFlags) error {
	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}

	k, identitySchema, err := identitySchemaForVet(cfg)
	if err != nil {
		return err
	}
	// Every load below builds in the runtime the schema lives in, so the
	// module's identity package unifies with #IdentityPackage exactly as it
	// did when the kernel handed out that context itself.
	cueCtx := identitySchema.Context() //nolint:staticcheck // SA1019: the deprecation's alternative (a fresh context, relying on cross-context unification) is a behavior change this migration deliberately avoids

	// Identity and coordinate checks run between module load and the values
	// stanza, so a module with no debugValues still reports coordinate drift.
	plan, modVal, err := publish.VetChecks(ctx, publish.Options{
		Dir:            modulePath,
		Kind:           publish.KindModule,
		Context:        cueCtx,
		Kernel:         k,
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

	// Resolve the values to validate against #config: -f files as
	// file-backed kernel sources, else debugValues as one source attributed
	// to the module's debugValues.
	sources, err := render.ResolveModuleValues(k, modVal, modulePath, rf.Values)
	if err != nil {
		// A -f file that cannot be read or parsed is an input error, not a
		// verdict on the module; a module without debugValues is.
		code := opmexit.ExitValidationError
		if len(rf.Values) > 0 {
			code = opmexit.ExitGeneralError
		}
		return &opmexit.ExitError{Code: code, Err: err}
	}

	if err := validateVetValues(k, modVal, modName, sources, len(rf.Values) > 0); err != nil {
		return err
	}

	moduleLog.Info(output.FormatVetCheck("Values satisfy #config", vetValuesDetail(rf.Values)))
	moduleLog.Info(output.FormatCheckmark("Module config valid"))

	return nil
}

// validateVetValues checks the resolved sources against the module's
// #config through the kernel. Values files need a #config to be checked
// against; build refuses the same input, so vet does too rather than
// reporting a vacuous pass. The kernel unifies the sources in stack order,
// walks disallowed fields, and asserts concreteness on the merged value, so
// a stack whose base leaves a field open for an override to fill passes
// here as it does in build; an incomplete merge is a #config violation at
// its position, printed as the grouped block and returned framed with the
// module name.
func validateVetValues(k *kernel.Kernel, modVal cue.Value, modName string, sources []kernel.Source, hasValuesFiles bool) error {
	configSchema := modVal.LookupPath(schema.Config)
	if hasValuesFiles && !configSchema.Exists() {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  fmt.Errorf("module does not define #config; values files cannot be validated"),
		}
	}
	if _, cfgErr := k.ValidateConfigDetailed(configSchema, sources); cfgErr != nil {
		err := fmt.Errorf("module %q: values do not satisfy #config: %w", modName, cfgErr)
		cmdutil.PrintValidationError("values do not satisfy #config", err)
		return &opmexit.ExitError{
			Code:    opmexit.ExitValidationError,
			Err:     err,
			Printed: true,
		}
	}
	return nil
}

// identitySchemaForVet builds the per-invocation kernel — the kernel-load
// gate acquires through it, and its schema cache carries the resolved
// registry — and resolves core's #IdentityPackage from that cache. The CUE
// context every load shares is the schema value's own
// (identitySchema.Context()).
func identitySchemaForVet(cfg *config.GlobalConfig) (*kernel.Kernel, cue.Value, error) {
	k := config.NewKernel(cfg.Registry)
	schemaVal, err := k.SchemaCache().Get()
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
	return k, identitySchema, nil
}

// vetValuesDetail names the values source for the "Values satisfy #config"
// line: the -f basenames joined by ", ", else debugValues.
func vetValuesDetail(valuesFiles []string) string {
	if len(valuesFiles) == 0 {
		return "debugValues"
	}
	basenames := make([]string, 0, len(valuesFiles))
	for _, valuesFile := range valuesFiles {
		basenames = append(basenames, filepath.Base(valuesFile))
	}
	return strings.Join(basenames, ", ")
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
