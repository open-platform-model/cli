package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
)

// NewModBuildCmd creates the mod build command.
func NewModBuildCmd() *cobra.Command {
	var rf cmdutil.RenderFlags

	// Build-specific flags (local to this command)
	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	cmd := &cobra.Command{
		Use:   "build [path]",
		Short: "Render module to manifests",
		Long: `Render an OPM module to Kubernetes manifests.

This command loads a module, matches components to transformers from the
configured provider, and outputs platform-specific resources.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Build module in current directory
  opm mod build

  # Build with custom values
  opm mod build ./my-module -f prod-values.cue -n production

  # Build with split output
  opm mod build ./my-module --split --out-dir ./manifests

  # Build with verbose output
  opm mod build ./my-module --verbose

  # Build as JSON
  opm mod build ./my-module -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd, args, &rf, outputFlag, splitFlag, outDirFlag)
		},
	}

	rf.AddTo(cmd)

	// Build-specific flags
	cmd.Flags().StringVarP(&outputFlag, "output", "o", "yaml",
		"Output format: yaml, json")
	cmd.Flags().BoolVar(&splitFlag, "split", false,
		"Write separate files per resource")
	cmd.Flags().StringVar(&outDirFlag, "out-dir", "./manifests",
		"Directory for split output")
	return cmd
}

// runBuild executes the build command.
func runBuild(_ *cobra.Command, args []string, rf *cmdutil.RenderFlags, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: yaml, json)", outputFmt),
		}
	}

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

	// --- Build-specific output logic below ---

	// Convert resources to ResourceInfo interface
	resourceInfos := make([]output.ResourceInfo, len(result.Resources))
	for i, r := range result.Resources {
		resourceInfos[i] = r
	}

	// Create scoped module logger
	modLog := output.ModuleLogger(result.Module.Name)

	// Output results
	if split {
		// Split output to files
		splitOpts := output.SplitOptions{
			OutDir: outDir,
			Format: outputFormat,
		}
		if err := output.WriteSplitManifests(resourceInfos, splitOpts); err != nil {
			return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("writing split manifests: %w", err)}
		}
		modLog.Info(fmt.Sprintf("wrote %d resources to %s", len(result.Resources), outDir))
	} else {
		// Output to stdout
		manifestOpts := output.ManifestOptions{
			Format: outputFormat,
			Writer: os.Stdout,
		}
		if err := output.WriteManifests(resourceInfos, manifestOpts); err != nil {
			return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("writing manifests: %w", err)}
		}
	}

	return nil
}
