package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
)

// NewModVetCmd creates the mod vet command.
func NewModVetCmd() *cobra.Command {
	var rf cmdutil.RenderFlags

	cmd := &cobra.Command{
		Use:   "vet [path]",
		Short: "Validate module without generating manifests",
		Long: `Validate an OPM module via the render pipeline.

This command loads a module, matches components to transformers, and validates
the module can be rendered successfully. No manifests are output — purely a
pass/fail validation tool with per-resource feedback.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Validate module in current directory
  opm mod vet

  # Validate with custom values
  opm mod vet ./my-module -f prod-values.cue -n production

  # Validate with verbose output (show matching decisions)
  opm mod vet ./my-module --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVet(cmd, args, &rf)
		},
	}

	rf.AddTo(cmd)

	return cmd
}

// runVet executes the vet command.
func runVet(_ *cobra.Command, args []string, rf *cmdutil.RenderFlags) error {
	ctx := context.Background()

	// Render module via shared pipeline
	result, err := cmdutil.RenderModule(ctx, cmdutil.RenderModuleOpts{
		Args:      args,
		Render:    rf,
		OPMConfig: GetOPMConfig(),
		Registry:  GetRegistry(),
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: verboseFlag,
	}); err != nil {
		return err
	}

	// --- Vet-specific logic below ---

	// Create scoped module logger for vet output
	modLog := output.ModuleLogger(result.Module.Name)

	// Print values validation check line
	var valuesDetail string
	if len(rf.Values) > 0 {
		basenames := make([]string, len(rf.Values))
		for i, vf := range rf.Values {
			basenames[i] = filepath.Base(vf)
		}
		valuesDetail = strings.Join(basenames, ", ")
	} else {
		valuesDetail = "values.cue"
	}
	modLog.Info(output.FormatVetCheck("Values satisfy #config", valuesDetail))

	// Print per-resource validation lines (skip when --verbose already showed them)
	if !verboseFlag {
		for _, res := range result.Resources {
			line := output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), output.StatusValid)
			modLog.Info(line)
		}
	}

	// Print final summary
	summary := fmt.Sprintf("Module valid (%d resources)", result.ResourceCount())
	modLog.Info(output.FormatCheckmark(summary))

	return nil
}
