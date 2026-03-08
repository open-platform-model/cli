package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/opmodel/cli/internal/output"
	pkgerrors "github.com/opmodel/cli/pkg/errors"
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
// For ConfigError (values schema failures), it prints a summary header followed by
// colored per-error blocks with file, position, path, and message.
//
// For generic errors, it falls back to the standard key-value log format.
func PrintValidationError(msg string, err error) {
	var configErr *pkgerrors.ConfigError
	if errors.As(err, &configErr) {
		fieldErrs := configErr.FieldErrors()
		n := len(fieldErrs)
		noun := "error"
		if n != 1 {
			noun = "errors"
		}
		output.Error(fmt.Sprintf("%s: %d %s", msg, n, noun))
		// Format each field error block similar to old ValuesValidationError output.
		for _, fe := range fieldErrs {
			output.Details(fmt.Sprintf("  %s: %s", fe.Path, fe.Message))
		}
		return
	}

	var valErr *pkgerrors.ValidationError
	if errors.As(err, &valErr) && valErr.Details != "" {
		output.Error(fmt.Sprintf("%s: %s", msg, valErr.Message))
		output.Details(valErr.Details)
		return
	}

	output.Error(msg, "error", err)
}

// WriteTransformerMatches writes compact transformer match output (always shown).
// Format: component <- provider - transformer
func WriteTransformerMatches(result *RenderResult) {
	if result.MatchPlan == nil {
		return
	}
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// One line per successful match.
	for _, pair := range result.MatchPlan.MatchedPairs() {
		releaseLog.Info(output.FormatTransformerMatch(pair.ComponentName, pair.TransformerFQN))
	}

	// Unmatched components.
	for _, comp := range result.MatchPlan.Unmatched {
		releaseLog.Warn(output.FormatTransformerUnmatched(comp))
	}
}

// WriteVerboseMatchLog writes detailed verbose output with release metadata,
// per-component properties, match reasons, and per-resource validation lines (--verbose only).
func WriteVerboseMatchLog(result *RenderResult) {
	if result.MatchPlan == nil {
		return
	}
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Release info — single line with key-value pairs.
	releaseLog.Info("release",
		"namespace", result.Release.Namespace,
		"version", result.Module.Version,
	)

	// Per-component summary — one line per component with its properties.
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

	// Transformer matching — one line per successful match.
	for _, pair := range result.MatchPlan.MatchedPairs() {
		releaseLog.Info(output.FormatTransformerMatch(pair.ComponentName, pair.TransformerFQN))
	}

	// Unmatched components.
	for _, comp := range result.MatchPlan.Unmatched {
		releaseLog.Warn(output.FormatTransformerUnmatched(comp))
	}

	// Generated resources — print each rendered resource line.
	for _, res := range result.Resources {
		kind := res.GetKind()
		ns := res.GetNamespace()
		name := res.GetName()
		releaseLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusValid))
	}
}
