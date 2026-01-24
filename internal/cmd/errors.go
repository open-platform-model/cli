package cmd

import (
	"errors"
	"fmt"
)

// Sentinel errors for known conditions.
var (
	// ErrValidation indicates a CUE schema validation failure.
	ErrValidation = errors.New("validation error")

	// ErrConnectivity indicates a Kubernetes cluster connectivity failure.
	ErrConnectivity = errors.New("connectivity error")

	// ErrPermission indicates insufficient RBAC permissions.
	ErrPermission = errors.New("permission denied")

	// ErrNotFound indicates a resource, module, or artifact was not found.
	ErrNotFound = errors.New("not found")

	// ErrVersion indicates a CUE binary version mismatch.
	ErrVersion = errors.New("version mismatch")
)

// ExitError wraps an error with an exit code.
type ExitError struct {
	Err  error
	Code int
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewExitError creates a new ExitError with the given error and exit code.
func NewExitError(err error, code int) *ExitError {
	return &ExitError{Err: err, Code: code}
}

// ExitCodeFromError determines the appropriate exit code for an error.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check for ExitError first
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	// Check for sentinel errors
	switch {
	case errors.Is(err, ErrValidation):
		return ExitValidationError
	case errors.Is(err, ErrConnectivity):
		return ExitConnectivityError
	case errors.Is(err, ErrPermission):
		return ExitPermissionDenied
	case errors.Is(err, ErrNotFound):
		return ExitNotFound
	case errors.Is(err, ErrVersion):
		return ExitVersionMismatch
	default:
		return ExitGeneralError
	}
}

// WrapValidation wraps an error with ErrValidation.
func WrapValidation(err error, msg string) error {
	return fmt.Errorf("%s: %w: %w", msg, ErrValidation, err)
}

// WrapConnectivity wraps an error with ErrConnectivity.
func WrapConnectivity(err error, msg string) error {
	return fmt.Errorf("%s: %w: %w", msg, ErrConnectivity, err)
}

// WrapPermission wraps an error with ErrPermission.
func WrapPermission(err error, msg string) error {
	return fmt.Errorf("%s: %w: %w", msg, ErrPermission, err)
}

// WrapNotFound wraps an error with ErrNotFound.
func WrapNotFound(err error, msg string) error {
	return fmt.Errorf("%s: %w: %w", msg, ErrNotFound, err)
}

// WrapVersion wraps an error with ErrVersion.
func WrapVersion(err error, msg string) error {
	return fmt.Errorf("%s: %w: %w", msg, ErrVersion, err)
}
