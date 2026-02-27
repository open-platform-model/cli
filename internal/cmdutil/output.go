package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	opmerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/pipeline"
)

// formatFQNList formats a slice of FQN strings into a compact comma-separated
// list using output.FormatFQN for each entry.
func formatFQNList(fqns []string) string {
	if len(fqns) == 0 {
		return ""
	}
	formatted := make([]string, len(fqns))
	for i, fqn := range fqns {
		formatted[i] = output.FormatFQN(fqn)
	}
	return strings.Join(formatted, ", ")
}

// PrintValidationError prints a render/validation error in a user-friendly format.
//
// For ValuesValidationError (values schema failures), it prints a summary
// header followed by colored per-error blocks with file, position, path, and
// message — one block per field error.
//
// For generic ValidationError with CUE details, it prints a summary line
// followed by the structured CUE output.
//
// All other errors fall back to the standard key-value log format.
func PrintValidationError(msg string, err error) {
	var valuesErr *opmerrors.ValuesValidationError
	if errors.As(err, &valuesErr) {
		n := len(valuesErr.Errors)
		noun := "error"
		if n != 1 {
			noun = "errors"
		}
		output.Error(fmt.Sprintf("%s: %d %s", msg, n, noun))
		output.Details(output.FormatValuesValidationError(valuesErr))
		return
	}

	var releaseErr *opmerrors.ValidationError
	if errors.As(err, &releaseErr) && releaseErr.Details != "" {
		output.Error(fmt.Sprintf("%s: %s", msg, releaseErr.Message))
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
// per-component properties, match reasons, and per-resource validation lines (--verbose only).
func WriteVerboseMatchLog(result *pipeline.RenderResult) {
	if result.MatchPlan == nil {
		return
	}
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Release info — single line with key-value pairs
	releaseLog.Info("release",
		"namespace", result.Release.Namespace,
		"version", result.Module.Version,
	)

	// Per-component summary — one line per component with its properties
	for _, comp := range result.Components {
		attrs := []any{}
		if resources := formatFQNList(comp.ResourceFQNs); resources != "" {
			attrs = append(attrs, "resources", resources)
		}
		if traits := formatFQNList(comp.TraitFQNs); traits != "" {
			attrs = append(attrs, "traits", traits)
		}
		for k, v := range comp.Labels {
			attrs = append(attrs, k, v)
		}
		releaseLog.Info(fmt.Sprintf("component: %s", comp.Name), attrs...)
	}

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
