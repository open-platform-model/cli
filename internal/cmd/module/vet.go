package modulecmd

import (
	"fmt"
	"path/filepath"
	"strings"

	opmexit "github.com/opmodel/cli/internal/exit"

	"cuelang.org/go/cue"
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/loader"
	"github.com/opmodel/cli/pkg/validate"
)

// NewModuleVetCmd creates the module vet command.
func NewModuleVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags

	c := &cobra.Command{
		Use:   "vet [path]",
		Short: "Validate module without generating manifests",
		Long: `Validate an OPM module's config inputs without generating manifests.

	This command validates the module's #config contract using either the module's
	debugValues (default) or explicit values files passed with -f/--values.
	It does not render resources, resolve providers, or validate release files.

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
			return runVet(args, cfg, &rf)
		},
	}

	rf.AddTo(c)

	return c
}

func runVet(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags) error {
	modulePath := cmdutil.ResolveModulePath(args)
	return runVetModuleOnly(modulePath, cfg, rf)
}

// runVetModuleOnly validates a module directory without a release.cue.
// It loads the module CUE package, validates the schema, and checks that the
// values (from -f flag or debugValues field) satisfy #config.
// No release wrapper, engine render, or cluster connection is required.
func runVetModuleOnly(modulePath string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags) error {
	cueCtx := cfg.CueContext

	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  err,
		}
	}

	// Load and structurally validate the module CUE package.
	modVal, err := loader.LoadModulePackage(cueCtx, modulePath)
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("loading module: %w", err),
		}
	}

	// Derive a display name for log output.
	modName := filepath.Base(modulePath)
	if nameVal := modVal.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
		if name, nameErr := nameVal.String(); nameErr == nil && name != "" {
			modName = name
		}
	}
	moduleLog := output.ReleaseLogger(modName)

	// Resolve the values to validate against #config.
	valuesVals := make([]cue.Value, 0, len(rf.Values))
	var valuesDetail string

	if len(rf.Values) > 0 {
		basenames := make([]string, 0, len(rf.Values))
		for _, valuesFile := range rf.Values {
			valuesVal, loadErr := loader.LoadValuesFile(cueCtx, valuesFile)
			if loadErr != nil {
				return &opmexit.ExitError{
					Code: opmexit.ExitGeneralError,
					Err:  fmt.Errorf("loading values file %q: %w", valuesFile, loadErr),
				}
			}
			valuesVals = append(valuesVals, valuesVal)
			basenames = append(basenames, filepath.Base(valuesFile))
		}
		valuesDetail = strings.Join(basenames, ", ")
	} else {
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			return &opmexit.ExitError{
				Code: opmexit.ExitValidationError,
				Err:  fmt.Errorf("module does not define debugValues - add debugValues or provide values with -f"),
			}
		}
		valuesVals = append(valuesVals, debugVal)
		valuesDetail = "debugValues"
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
		if _, cfgErr := validate.ValidateConfig(configVal, valuesVals, "module", modName); cfgErr != nil {
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
