// Package cmd provides CLI command implementations.
package cmd

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
