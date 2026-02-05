package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

// Build command flags
var (
	buildValuesFlags     []string
	buildNamespaceFlag   string
	buildNameFlag        string
	buildProviderFlag    string
	buildOutputFlag      string
	buildSplitFlag       bool
	buildOutDirFlag      string
	buildStrictFlag      bool
	buildVerboseFlag     bool
	buildVerboseJSONFlag bool
)

// ExitError wraps an error with an exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewModBuildCmd creates the mod build command.
func NewModBuildCmd() *cobra.Command {
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
		RunE: runBuild,
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&buildValuesFlags, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&buildNamespaceFlag, "namespace", "n", "",
		"Target namespace (required if not in module)")
	cmd.Flags().StringVar(&buildNameFlag, "name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&buildProviderFlag, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().StringVarP(&buildOutputFlag, "output", "o", "yaml",
		"Output format: yaml, json")
	cmd.Flags().BoolVar(&buildSplitFlag, "split", false,
		"Write separate files per resource")
	cmd.Flags().StringVar(&buildOutDirFlag, "out-dir", "./manifests",
		"Directory for split output")
	cmd.Flags().BoolVar(&buildStrictFlag, "strict", false,
		"Error on unhandled traits")
	cmd.Flags().BoolVarP(&buildVerboseFlag, "verbose", "v", false,
		"Show matching decisions")
	cmd.Flags().BoolVar(&buildVerboseJSONFlag, "verbose-json", false,
		"Structured JSON verbose output")

	return cmd
}

// runBuild executes the build command.
func runBuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine module path
	modulePath := "."
	if len(args) > 0 {
		modulePath = args[0]
	}

	// Validate output format
	outputFormat, valid := output.ParseFormat(buildOutputFlag)
	if !valid {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: yaml, json)", buildOutputFlag),
		}
	}

	// Load configuration
	opmConfig, err := config.LoadOPMConfig(config.LoaderOptions{
		RegistryFlag: GetRegistryFlag(),
		ConfigFlag:   GetConfigPath(),
	})
	if err != nil {
		output.Error("loading configuration", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Build render options
	opts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     buildValuesFlags,
		Name:       buildNameFlag,
		Namespace:  buildNamespaceFlag,
		Provider:   buildProviderFlag,
		Strict:     buildStrictFlag,
		Registry:   GetRegistry(),
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Create pipeline
	pipeline := build.NewPipeline(opmConfig)

	// Execute render
	output.Debug("starting render",
		"module", modulePath,
		"namespace", opts.Namespace,
		"provider", opts.Provider,
	)

	result, err := pipeline.Render(ctx, opts)
	if err != nil {
		output.Error("render failed", "error", err)
		return &ExitError{Code: ExitValidationError, Err: err}
	}

	// Handle verbose output
	if buildVerboseFlag || buildVerboseJSONFlag {
		writeVerboseOutput(result, buildVerboseJSONFlag)
	}

	// Check for render errors
	if result.HasErrors() {
		printRenderErrors(result.Errors)
		return &ExitError{
			Code: ExitValidationError,
			Err:  fmt.Errorf("%d render error(s)", len(result.Errors)),
		}
	}

	// Print warnings
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			output.Warn(w)
		}
	}

	// Convert resources to ResourceInfo interface
	resourceInfos := make([]output.ResourceInfo, len(result.Resources))
	for i, r := range result.Resources {
		resourceInfos[i] = r
	}

	// Output results
	if buildSplitFlag {
		// Split output to files
		splitOpts := output.SplitOptions{
			OutDir: buildOutDirFlag,
			Format: outputFormat,
		}
		if err := output.WriteSplitManifests(resourceInfos, splitOpts); err != nil {
			return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("writing split manifests: %w", err)}
		}
		output.Info(fmt.Sprintf("wrote %d resources to %s", len(result.Resources), buildOutDirFlag))
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

// writeVerboseOutput writes verbose output to stderr.
func writeVerboseOutput(result *build.RenderResult, jsonOutput bool) {
	// Convert to RenderResultInfo
	matches := make(map[string][]output.TransformerMatchInfo)
	for compName, matchList := range result.MatchPlan.Matches {
		for _, m := range matchList {
			matches[compName] = append(matches[compName], output.TransformerMatchInfo{
				TransformerFQN: m.TransformerFQN,
				Reason:         m.Reason,
			})
		}
	}

	// Convert resources to ResourceInfo
	resourceInfos := make([]output.ResourceInfo, len(result.Resources))
	for i, r := range result.Resources {
		resourceInfos[i] = r
	}

	info := &output.RenderResultInfo{
		ModuleName:       result.Module.Name,
		ModuleNamespace:  result.Module.Namespace,
		ModuleVersion:    result.Module.Version,
		ModuleComponents: result.Module.Components,
		ModuleLabels:     result.Module.Labels,
		Matches:          matches,
		Unmatched:        result.MatchPlan.Unmatched,
		Resources:        resourceInfos,
		Errors:           result.Errors,
		Warnings:         result.Warnings,
	}

	verboseOpts := output.VerboseOptions{
		JSON:   jsonOutput,
		Writer: os.Stderr,
	}

	if err := output.WriteVerboseResult(info, nil, verboseOpts); err != nil {
		output.Warn("writing verbose output", "error", err)
	}
}

// printRenderErrors prints render errors in a user-friendly format.
func printRenderErrors(errs []error) {
	output.Error("render completed with errors")
	for _, err := range errs {
		var unmatchedErr *build.UnmatchedComponentError
		var transformErr *build.TransformError
		var unhandledTraitErr *build.UnhandledTraitError

		switch {
		case errors.As(err, &unmatchedErr):
			output.Error(fmt.Sprintf("component %q: no matching transformer", unmatchedErr.ComponentName))
			if len(unmatchedErr.Available) > 0 {
				output.Info("Available transformers:")
				for _, t := range unmatchedErr.Available {
					output.Info(fmt.Sprintf("  %s", t.FQN))
					if len(t.RequiredLabels) > 0 {
						output.Info(fmt.Sprintf("    requiredLabels: %v", t.RequiredLabels))
					}
					if len(t.RequiredResources) > 0 {
						output.Info(fmt.Sprintf("    requiredResources: %v", t.RequiredResources))
					}
					if len(t.RequiredTraits) > 0 {
						output.Info(fmt.Sprintf("    requiredTraits: %v", t.RequiredTraits))
					}
				}
			}
		case errors.As(err, &transformErr):
			output.Error(fmt.Sprintf("component %q: transform failed with %s: %v",
				transformErr.ComponentName, transformErr.TransformerFQN, transformErr.Cause))
		case errors.As(err, &unhandledTraitErr):
			output.Error(fmt.Sprintf("component %q: unhandled trait %q", unhandledTraitErr.ComponentName, unhandledTraitErr.TraitFQN))
		default:
			output.Error(err.Error())
		}
	}
}
