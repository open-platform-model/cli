package cmdutil

import (
	"errors"
	"fmt"

	"github.com/opmodel/cli/internal/output"
	pkgerrors "github.com/opmodel/cli/pkg/errors"
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
