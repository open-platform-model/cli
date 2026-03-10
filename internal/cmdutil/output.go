package cmdutil

import (
	"errors"
	"fmt"
	"sort"
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
// For ConfigError (values schema failures), it prints a summary header with the
// total grouped issue count, then a grouped block where each distinct error message
// appears once followed by all source positions that report it. This naturally
// surfaces conflicts (same message, multiple files) as a single entry.
//
// For any error wrapping a CUE error (e.g. raw build errors), the same grouped
// format is applied via GroupedErrorsFromError.
//
// For generic errors, it falls back to the standard key-value log format.
func PrintValidationError(msg string, err error) {
	var configErr *pkgerrors.ConfigError
	if errors.As(err, &configErr) {
		printGrouped(msg, configErr.GroupedErrors())
		return
	}

	var valErr *pkgerrors.ValidationError
	if errors.As(err, &valErr) && valErr.Details != "" {
		output.Error(fmt.Sprintf("%s: %s", msg, valErr.Message))
		output.Details(valErr.Details)
		return
	}

	// Try to extract CUE errors from any wrapped error chain before falling back
	// to the raw key-value format. Only use grouped display when at least one
	// location has a valid source position — plain errors promoted by CUE have
	// no position and should not trigger this path.
	if groups := pkgerrors.GroupedErrorsFromError(err); hasPositions(groups) {
		printGrouped(msg, groups)
		return
	}

	output.Error(msg, "error", err)
}

// hasPositions reports whether any location in any group has a valid source
// position (Line > 0). Used to distinguish genuine CUE structural errors from
// plain errors promoted by cueerrors.Promote.
func hasPositions(groups []pkgerrors.GroupedError) bool {
	for _, g := range groups {
		for _, loc := range g.Locations {
			if loc.Line > 0 {
				return true
			}
		}
	}
	return false
}

// printGrouped prints a grouped-error summary header and formatted body.
func printGrouped(msg string, groups []pkgerrors.GroupedError) {
	n := len(groups)
	noun := "issue"
	if n != 1 {
		noun = "issues"
	}
	output.Error(fmt.Sprintf("%s: %d %s", msg, n, noun))
	output.Details(output.FormatGroupedErrors(groups))
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
//
//nolint:gocyclo // verbose formatting intentionally gathers several output sections in one place
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

	// Transformer matching — interleave matched (INFO) and non-matched (DEBUG)
	// lines sorted by component name then transformer FQN so that all results
	// for a given component appear together.
	type matchLine struct {
		compName string
		tfFQN    string
		matched  bool
		missing  struct{ labels, resources, traits []string }
	}
	var lines []matchLine
	for _, p := range result.MatchPlan.MatchedPairs() {
		lines = append(lines, matchLine{compName: p.ComponentName, tfFQN: p.TransformerFQN, matched: true})
	}
	for _, p := range result.MatchPlan.NonMatchedPairs() {
		l := matchLine{compName: p.ComponentName, tfFQN: p.TransformerFQN, matched: false}
		l.missing.labels = p.MissingLabels
		l.missing.resources = p.MissingResources
		l.missing.traits = p.MissingTraits
		lines = append(lines, l)
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].compName != lines[j].compName {
			return lines[i].compName < lines[j].compName
		}
		return lines[i].tfFQN < lines[j].tfFQN
	})
	for _, l := range lines {
		if l.matched {
			releaseLog.Info(output.FormatTransformerMatch(l.compName, l.tfFQN))
		} else {
			attrs := []any{}
			if len(l.missing.labels) > 0 {
				attrs = append(attrs, "missing-labels", strings.Join(l.missing.labels, ", "))
			}
			if len(l.missing.resources) > 0 {
				attrs = append(attrs, "missing-resources", strings.Join(l.missing.resources, ", "))
			}
			if len(l.missing.traits) > 0 {
				attrs = append(attrs, "missing-traits", strings.Join(l.missing.traits, ", "))
			}
			releaseLog.Debug(output.FormatTransformerSkipped(l.compName, l.tfFQN), attrs...)
		}
	}

	// Unmatched components (zero matching transformers — error condition surfaced as warning here).
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
