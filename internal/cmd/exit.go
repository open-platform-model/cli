// Package cmd provides command implementations for the OPM CLI.
package cmd

// Exit codes per contracts/exit-codes.md.
const (
	// ExitSuccess indicates the command completed successfully.
	ExitSuccess = 0

	// ExitGeneralError indicates an unspecified error occurred.
	ExitGeneralError = 1

	// ExitValidationError indicates CUE schema validation failed.
	ExitValidationError = 2

	// ExitConnectivityError indicates cannot reach Kubernetes cluster.
	ExitConnectivityError = 3

	// ExitPermissionDenied indicates insufficient RBAC permissions.
	ExitPermissionDenied = 4

	// ExitNotFound indicates resource, module, or artifact not found.
	ExitNotFound = 5

	// ExitVersionMismatch indicates CUE binary version incompatible.
	ExitVersionMismatch = 6
)

// ExitCodeName returns the name of the exit code.
func ExitCodeName(code int) string {
	switch code {
	case ExitSuccess:
		return "Success"
	case ExitGeneralError:
		return "General Error"
	case ExitValidationError:
		return "Validation Error"
	case ExitConnectivityError:
		return "Connectivity Error"
	case ExitPermissionDenied:
		return "Permission Denied"
	case ExitNotFound:
		return "Not Found"
	case ExitVersionMismatch:
		return "Version Mismatch"
	default:
		return "Unknown"
	}
}
