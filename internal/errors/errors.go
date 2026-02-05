// Package errors provides sentinel errors for the OPM CLI.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for known conditions.
var (
	// ErrValidation indicates a CUE schema validation failure.
	ErrValidation = errors.New("validation error")

	// ErrConnectivity indicates a network connectivity issue.
	ErrConnectivity = errors.New("connectivity error")

	// ErrPermission indicates insufficient permissions.
	ErrPermission = errors.New("permission denied")

	// ErrNotFound indicates a resource, module, or file was not found.
	ErrNotFound = errors.New("not found")
)

// DetailError captures structured error information per contracts/error-format.md.
type DetailError struct {
	// Type is the error category (required).
	Type string

	// Message is the specific description (required).
	Message string

	// Location is the file path and line number (optional).
	Location string

	// Field is the field name for schema errors (optional).
	Field string

	// Context contains additional key-value context (optional).
	Context map[string]string

	// Hint provides actionable guidance (optional).
	Hint string

	// Cause is the underlying error (optional).
	Cause error
}

// Error implements the error interface.
func (e *DetailError) Error() string {
	var b strings.Builder

	b.WriteString("Error: ")
	b.WriteString(e.Type)
	b.WriteString("\n")

	if e.Location != "" {
		b.WriteString("  Location: ")
		b.WriteString(e.Location)
		b.WriteString("\n")
	}
	if e.Field != "" {
		b.WriteString("  Field: ")
		b.WriteString(e.Field)
		b.WriteString("\n")
	}
	for k, v := range e.Context {
		b.WriteString("  ")
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\n")
	}

	b.WriteString("\n  ")
	b.WriteString(e.Message)
	b.WriteString("\n")

	if e.Hint != "" {
		b.WriteString("\nHint: ")
		b.WriteString(e.Hint)
		b.WriteString("\n")
	}

	return b.String()
}

// Unwrap returns the underlying error.
func (e *DetailError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a validation error with details.
func NewValidationError(message, location, field, hint string) error {
	return &DetailError{
		Type:     "validation failed",
		Message:  message,
		Location: location,
		Field:    field,
		Hint:     hint,
		Cause:    ErrValidation,
	}
}

// NewConnectivityError creates a connectivity error with details.
func NewConnectivityError(message string, context map[string]string, hint string) error {
	return &DetailError{
		Type:    "connectivity failed",
		Message: message,
		Context: context,
		Hint:    hint,
		Cause:   ErrConnectivity,
	}
}

// NewNotFoundError creates a not found error with details.
func NewNotFoundError(message, location, hint string) error {
	return &DetailError{
		Type:     "not found",
		Message:  message,
		Location: location,
		Hint:     hint,
		Cause:    ErrNotFound,
	}
}

// NewPermissionError creates a permission denied error with details.
func NewPermissionError(message string, context map[string]string, hint string) error {
	return &DetailError{
		Type:    "permission denied",
		Message: message,
		Context: context,
		Hint:    hint,
		Cause:   ErrPermission,
	}
}

// Wrap wraps an error with a sentinel error type.
func Wrap(sentinel error, message string) error {
	return fmt.Errorf("%s: %w", message, sentinel)
}
