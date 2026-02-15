package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/output"
)

// Vet command flags
var (
	vetValuesFlags   []string
	vetNamespaceFlag string
	vetNameFlag      string
	vetProviderFlag  string
	vetStrictFlag    bool
	vetVerboseFlag   bool
)

// NewModVetCmd creates the mod vet command.
func NewModVetCmd() *cobra.Command {
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
  opm mod vet ./my-module --verbose

  # Strict mode (error on unhandled traits)
  opm mod vet ./my-module --strict`,
		Args: cobra.MaximumNArgs(1),
		RunE: runVet,
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&vetValuesFlags, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&vetNamespaceFlag, "namespace", "n", "",
		"Target namespace (required if not in module)")
	cmd.Flags().StringVar(&vetNameFlag, "name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&vetProviderFlag, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().BoolVar(&vetStrictFlag, "strict", false,
		"Error on unhandled traits")
	cmd.Flags().BoolVarP(&vetVerboseFlag, "verbose", "v", false,
		"Show matching decisions and module metadata")

	return cmd
}

// runVet executes the vet command.
func runVet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine module path
	modulePath := "."
	if len(args) > 0 {
		modulePath = args[0]
	}

	// Get pre-loaded configuration
	opmConfig := GetOPMConfig()
	if opmConfig == nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// Build render options
	opts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     vetValuesFlags,
		Name:       vetNameFlag,
		Namespace:  vetNamespaceFlag,
		Provider:   vetProviderFlag,
		Strict:     vetStrictFlag,
		Registry:   GetRegistry(),
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Create pipeline
	pipeline := build.NewPipeline(opmConfig)

	// Execute render
	output.Debug("validating module",
		"module", modulePath,
		"namespace", opts.Namespace,
		"provider", opts.Provider,
	)

	result, err := pipeline.Render(ctx, opts)
	if err != nil {
		// Fatal error from Render() — CUE validation errors, missing provider, etc.
		printValidationError("validation failed", err)
		return &ExitError{Code: ExitValidationError, Err: err, Printed: true}
	}

	// Handle verbose output (before checking for errors)
	if vetVerboseFlag {
		writeVetVerboseOutput(result)
	}

	// Check for render errors
	if result.HasErrors() {
		printRenderErrors(result.Errors)
		return &ExitError{
			Code:    ExitValidationError,
			Err:     fmt.Errorf("%d render error(s)", len(result.Errors)),
			Printed: true,
		}
	}

	// Create scoped module logger for warnings
	modLog := output.ModuleLogger(result.Module.Name)

	// Print warnings
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			modLog.Warn(w)
		}
	}

	// Print values validation check line
	// If Render() succeeded, we know validateValuesAgainstConfig passed (Step 4b in release_builder.go)
	var valuesDetail string
	if len(vetValuesFlags) > 0 {
		// External values files — show comma-separated basenames
		basenames := make([]string, len(vetValuesFlags))
		for i, vf := range vetValuesFlags {
			basenames[i] = filepath.Base(vf)
		}
		valuesDetail = strings.Join(basenames, ", ")
	} else {
		// Module's own values.cue
		valuesDetail = "values.cue"
	}
	output.Println(output.FormatVetCheck("Values satisfy #config", valuesDetail))

	// Print per-resource validation lines
	for _, res := range result.Resources {
		line := output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), output.StatusValid)
		output.Println(line)
	}

	// Print final summary
	summary := fmt.Sprintf("Module valid (%d resources)", result.ResourceCount())
	output.Println(output.FormatCheckmark(summary))

	return nil
}

// writeVetVerboseOutput writes verbose output for mod vet.
// Reuses the same writeVerboseOutput function from mod build.
func writeVetVerboseOutput(result *build.RenderResult) {
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
		JSON:   false,
		Writer: os.Stderr,
	}

	if err := output.WriteVerboseResult(info, nil, verboseOpts); err != nil {
		output.Warn("writing verbose output", "error", err)
	}
}
