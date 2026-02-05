// Package cmd provides CLI command implementations.
package cmd

import (
	"errors"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	oerrors "github.com/opmodel/cli/internal/errors"
)

// Exit codes per contracts/exit-codes.md.
const (
	ExitSuccess           = 0
	ExitGeneralError      = 1
	ExitValidationError   = 2
	ExitConnectivityError = 3
	ExitPermissionDenied  = 4
	ExitNotFound          = 5
)

// ExitCodeFromError maps an error to the appropriate exit code.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check OPM sentinel errors
	switch {
	case errors.Is(err, oerrors.ErrValidation):
		return ExitValidationError
	case errors.Is(err, oerrors.ErrConnectivity):
		return ExitConnectivityError
	case errors.Is(err, oerrors.ErrPermission):
		return ExitPermissionDenied
	case errors.Is(err, oerrors.ErrNotFound):
		return ExitNotFound
	}

	// Check Kubernetes API errors
	code := exitCodeFromK8sError(err)
	if code != ExitGeneralError {
		return code
	}

	return ExitGeneralError
}

// exitCodeFromK8sError maps Kubernetes API errors to exit codes.
func exitCodeFromK8sError(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return ExitNotFound
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return ExitPermissionDenied
	case apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err):
		return ExitConnectivityError
	default:
		return ExitGeneralError
	}
}

// Exit terminates the program with the appropriate exit code for the error.
func Exit(err error) {
	os.Exit(ExitCodeFromError(err))
}
