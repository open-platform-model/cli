package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	liberrors "github.com/open-platform-model/library/opm/errors"

	"github.com/open-platform-model/cli/internal/output"
	pkgerrors "github.com/open-platform-model/cli/pkg/errors"
)

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

	// Unresolved demands (kernel matcher, 0010 D28): render one line per
	// demand with its contract key and same-base alternatives instead of the
	// flat error blob the CUE-position heuristic would produce.
	var demandsErr *liberrors.UnresolvedDemandsError
	if errors.As(err, &demandsErr) {
		n := len(demandsErr.Demands)
		noun := "demand"
		if n != 1 {
			noun = "demands"
		}
		output.Error(fmt.Sprintf("%s: %d unresolved %s", msg, n, noun))
		output.Details(FormatUnresolvedDemands(demandsErr))
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

// FormatUnresolvedDemands renders an unresolved-demands aggregate as one
// block: per demand the component, kind, and contract key, followed by the
// same-base alternatives the platform does implement (the actionable half of
// the diagnostic) or an explicit nothing-implements line.
func FormatUnresolvedDemands(demandsErr *liberrors.UnresolvedDemandsError) string {
	var b strings.Builder
	for _, d := range demandsErr.Demands {
		fmt.Fprintf(&b, "component %q: unresolved %s demand %q\n", d.Component, d.Kind, d.FQN)
		if len(d.Alternatives) > 0 {
			fmt.Fprintf(&b, "  implemented at: %s\n", strings.Join(d.Alternatives, ", "))
		} else {
			b.WriteString("  nothing on this platform implements this contract\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
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
