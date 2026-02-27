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
// Implements the pipeline.RenderError interface.
func (e *TransformError) Component() string {
	return e.ComponentName
}

// ValidationError indicates the release failed validation.
// This typically happens when values are incomplete or non-concrete.
type ValidationError struct {
	// Message describes what validation failed.
	Message string

	// Cause is the underlying error.
	Cause error

	// Details contains the formatted CUE error output with all individual
	// errors, their CUE paths, and source positions.
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
	// File is the values file name where the error occurred (e.g. "values-prod.cue").
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

// ValuesValidationError is returned by ValidateValues when one or more fields
// in a values file violate the module's #config schema. It collects all errors
// rather than stopping at the first.
type ValuesValidationError struct {
	Errors []FieldError
}

func (e *ValuesValidationError) Error() string {
	n := len(e.Errors)
	if n == 1 {
		return "values validation failed: 1 error"
	}
	return fmt.Sprintf("values validation failed: %d errors", n)
}

// Lines returns all errors formatted as plain text (no color), one per line.
// Useful for logging and test assertions.
func (e *ValuesValidationError) Lines() []string {
	lines := make([]string, 0, len(e.Errors)*2)
	for _, fe := range e.Errors {
		loc := fmt.Sprintf("%s:%d:%d", fe.File, fe.Line, fe.Column)
		lines = append(lines, loc, "  "+fe.Path+": "+fe.Message)
	}
	return lines
}

// PlainText returns all errors as a single plain-text string.
func (e *ValuesValidationError) PlainText() string {
	return strings.Join(e.Lines(), "\n")
}
