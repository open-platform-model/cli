package errors

import (
	"fmt"
	"strings"
)

// TransformError indicates transformer execution failed.
type TransformError struct {
	ComponentName  string
	TransformerFQN string
	Cause          error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("component %q, transformer %q: %v",
		e.ComponentName, e.TransformerFQN, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}

// Component returns the component name where the error occurred.
func (e *TransformError) Component() string {
	return e.ComponentName
}

// ValidationError indicates the release failed validation.
type ValidationError struct {
	// Message describes what validation failed.
	Message string

	// Cause is the underlying error.
	Cause error

	// Details contains the formatted CUE error output.
	Details string
}

func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return "release validation failed: " + e.Message + ": " + e.Cause.Error()
	}
	return "release validation failed: " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// FieldError is a single validation error tied to a specific source location
// in a values file.
type FieldError struct {
	// File is the values file name where the error occurred.
	File string

	// Line is the 1-based line number in File.
	Line int

	// Column is the 1-based column number in File.
	Column int

	// Path is the dot-joined field path from the values root (e.g. "values.db.port").
	Path string

	// Message is the human-readable error description.
	Message string
}

// ConflictLocation is a source position for one side of a cross-file conflict.
type ConflictLocation struct {
	File   string
	Line   int
	Column int
}

// ConflictError represents a conflict between values in two or more files for
// the same field.
type ConflictError struct {
	// Path is the dot-joined field path.
	Path string

	// Message is the human-readable conflict description.
	Message string

	// Locations holds one entry per conflicting source position.
	Locations []ConflictLocation
}

// ValuesValidationError is returned when one or more fields in a values file
// violate the module's #config schema. It collects all errors rather than
// stopping at the first.
type ValuesValidationError struct {
	Errors    []FieldError
	Conflicts []ConflictError
}

func (e *ValuesValidationError) Error() string {
	n := len(e.Errors) + len(e.Conflicts)
	if n == 1 {
		return "values validation failed: 1 error"
	}
	return fmt.Sprintf("values validation failed: %d errors", n)
}

// Lines returns all errors and conflicts formatted as plain text.
func (e *ValuesValidationError) Lines() []string {
	lines := make([]string, 0, (len(e.Errors)+len(e.Conflicts))*2)
	for _, fe := range e.Errors {
		loc := fmt.Sprintf("%s:%d:%d", fe.File, fe.Line, fe.Column)
		lines = append(lines, loc, "  "+fe.Path+": "+fe.Message)
	}
	for _, ce := range e.Conflicts {
		locs := make([]string, len(ce.Locations))
		for i, l := range ce.Locations {
			locs[i] = fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
		}
		lines = append(lines, strings.Join(locs, " vs "), "  "+ce.Path+": "+ce.Message)
	}
	return lines
}

// PlainText returns all errors as a single plain-text string.
func (e *ValuesValidationError) PlainText() string {
	return strings.Join(e.Lines(), "\n")
}
