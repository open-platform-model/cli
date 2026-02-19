// Package cmdtypes provides shared types for the cmd package and its sub-packages.
// It is separate from internal/cmd to avoid import cycles between internal/cmd
// and its sub-packages (internal/cmd/mod, internal/cmd/config).
package cmdtypes

import (
	oerrors "github.com/opmodel/cli/internal/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/opmodel/cli/internal/config"
)

// GlobalConfig holds CLI-wide configuration resolved during PersistentPreRunE.
// It is populated once at startup and passed explicitly into every sub-command
// constructor, replacing the former package-level mutable globals.
type GlobalConfig struct {
	OPMConfig    *config.OPMConfig
	ConfigPath   string // resolved --config path
	Registry     string // resolved --registry URL
	RegistryFlag string // raw --registry flag value (needed by config vet)
	Verbose      bool
}

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

// ExitCodeFromK8sError maps Kubernetes API errors to exit codes.
func ExitCodeFromK8sError(err error) int {
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
