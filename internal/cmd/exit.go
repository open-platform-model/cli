// Package cmd provides CLI command implementations.
package cmd

import (
	oerrors "github.com/opmodel/cli/internal/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Exit codes — type aliases to internal/errors constants.
const (
	ExitSuccess           = oerrors.ExitSuccess
	ExitGeneralError      = oerrors.ExitGeneralError
	ExitValidationError   = oerrors.ExitValidationError
	ExitConnectivityError = oerrors.ExitConnectivityError
	ExitPermissionDenied  = oerrors.ExitPermissionDenied
	ExitNotFound          = oerrors.ExitNotFound
)

// ExitError is a type alias to internal/errors.ExitError.
// This allows cmd package code to continue using cmd.ExitError
// while using the same underlying type across all packages.
type ExitError = oerrors.ExitError

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
