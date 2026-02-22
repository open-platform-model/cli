package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	opmerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/pipeline"
)

// PrintValidationError prints a render/validation error in a user-friendly format.
// When the error is a ValidationError with CUE details, it prints a short
// summary line followed by the structured CUE error output (matching `cue vet` style).
// For other errors, it falls back to the standard key-value log format.
func PrintValidationError(msg string, err error) {
	var releaseErr *opmerrors.ValidationError
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
		var unmatchedErr *pipeline.UnmatchedComponentError
		var transformErr *opmerrors.TransformError

		switch {
		case errors.As(err, &unmatchedErr):
			output.Error(fmt.Sprintf("component %q: no matching transformer", unmatchedErr.ComponentName))
			if len(unmatchedErr.Available) > 0 {
				output.Info("Available transformers:")
				for _, t := range unmatchedErr.Available {
					output.Info(fmt.Sprintf("  %s", output.FormatFQN(t.GetFQN())))
					if len(t.GetRequiredLabels()) > 0 {
						output.Info(fmt.Sprintf("    requiredLabels: %v", t.GetRequiredLabels()))
					}
					if len(t.GetRequiredResources()) > 0 {
						output.Info(fmt.Sprintf("    requiredResources: %v", t.GetRequiredResources()))
					}
					if len(t.GetRequiredTraits()) > 0 {
						output.Info(fmt.Sprintf("    requiredTraits: %v", t.GetRequiredTraits()))
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
func WriteTransformerMatches(result *pipeline.RenderResult) {
	if result.MatchPlan == nil {
		return
	}
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Transformer matching — one line per successful match
	for _, m := range result.MatchPlan.Matches {
		if !m.Matched || m.Detail == nil {
			continue
		}
		releaseLog.Info(output.FormatTransformerMatch(m.Detail.ComponentName, m.Detail.TransformerFQN))
	}

	// Unmatched components
	for _, comp := range result.MatchPlan.Unmatched {
		releaseLog.Warn(output.FormatTransformerUnmatched(comp))
	}
}

// WriteVerboseMatchLog writes detailed verbose output with release metadata,
// match reasons, and per-resource validation lines (--verbose only).
func WriteVerboseMatchLog(result *pipeline.RenderResult) {
	if result.MatchPlan == nil {
		return
	}
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Release info — single line with key-value pairs
	releaseLog.Info("release",
		"namespace", result.Release.Namespace,
		"version", result.Module.Version,
		"components", strings.Join(result.Release.Components, ", "),
	)

	// Transformer matching — one line per successful match with reason
	for _, m := range result.MatchPlan.Matches {
		if !m.Matched || m.Detail == nil {
			continue
		}
		releaseLog.Info(output.FormatTransformerMatchVerbose(m.Detail.ComponentName, m.Detail.TransformerFQN, m.Detail.Reason))
	}

	// Unmatched components
	for _, comp := range result.MatchPlan.Unmatched {
		releaseLog.Warn(output.FormatTransformerUnmatched(comp))
	}

	// Generated resources
	for _, res := range result.Resources {
		releaseLog.Info(output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), output.StatusValid))
	}
}
