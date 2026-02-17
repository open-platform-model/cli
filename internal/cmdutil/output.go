package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/output"
)

// PrintValidationError prints a render/validation error in a user-friendly format.
// When the error is a ReleaseValidationError with CUE details, it prints a short
// summary line followed by the structured CUE error output (matching `cue vet` style).
// For other errors, it falls back to the standard key-value log format.
func PrintValidationError(msg string, err error) {
	var releaseErr *build.ReleaseValidationError
	if errors.As(err, &releaseErr) && releaseErr.Details != "" {
		output.Error(fmt.Sprintf("%s: %s", msg, releaseErr.Message))
		// Print CUE details as plain text to stderr for readable multi-line output
		output.Details(releaseErr.Details)
		return
	}
	output.Error(msg, "error", err)
}

// PrintRenderErrors prints render errors in a user-friendly format.
func PrintRenderErrors(errs []error) {
	output.Error("render completed with errors")
	for _, err := range errs {
		var unmatchedErr *build.UnmatchedComponentError
		var transformErr *build.TransformError

		switch {
		case errors.As(err, &unmatchedErr):
			output.Error(fmt.Sprintf("component %q: no matching transformer", unmatchedErr.ComponentName))
			if len(unmatchedErr.Available) > 0 {
				output.Info("Available transformers:")
				for _, t := range unmatchedErr.Available {
					output.Info(fmt.Sprintf("  %s", output.FormatFQN(t.FQN)))
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
				transformErr.ComponentName, output.FormatFQN(transformErr.TransformerFQN), transformErr.Cause))
		default:
			output.Error(err.Error())
		}
	}
}

// WriteTransformerMatches writes compact transformer match output (always shown).
// Format: component <- provider - transformer
func WriteTransformerMatches(result *build.RenderResult) {
	modLog := output.ModuleLogger(result.Module.Name)

	// Transformer matching — one line per match
	for compName, matches := range result.MatchPlan.Matches {
		for _, m := range matches {
			modLog.Info(output.FormatTransformerMatch(compName, m.TransformerFQN))
		}
	}

	// Unmatched components
	for _, comp := range result.MatchPlan.Unmatched {
		modLog.Warn(output.FormatTransformerUnmatched(comp))
	}
}

// WriteVerboseMatchLog writes detailed verbose output with module metadata,
// match reasons, and per-resource validation lines (--verbose only).
func WriteVerboseMatchLog(result *build.RenderResult) {
	modLog := output.ModuleLogger(result.Module.Name)

	// Module info — single line with key-value pairs
	modLog.Info("module",
		"namespace", result.Module.Namespace,
		"version", result.Module.Version,
		"components", strings.Join(result.Module.Components, ", "),
	)

	// Transformer matching — one line per match with reason
	for compName, matches := range result.MatchPlan.Matches {
		for _, m := range matches {
			modLog.Info(output.FormatTransformerMatchVerbose(compName, m.TransformerFQN, m.Reason))
		}
	}

	// Unmatched components
	for _, comp := range result.MatchPlan.Unmatched {
		modLog.Warn(output.FormatTransformerUnmatched(comp))
	}

	// Generated resources
	for _, res := range result.Resources {
		modLog.Info(output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), output.StatusValid))
	}
}
