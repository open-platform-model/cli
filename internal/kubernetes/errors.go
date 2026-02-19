package kubernetes

import (
	"errors"
	"fmt"
)

// errNoResourcesFound is returned when no resources match the selector.
var errNoResourcesFound = errors.New("no resources found")

// noResourcesFoundError contains details about a failed resource discovery.
type noResourcesFoundError struct {
	// ReleaseName is the release name that was searched (empty if using release-id).
	ReleaseName string
	// ReleaseID is the release-id that was searched (empty if using release-name).
	ReleaseID string
	// Namespace is the namespace that was searched.
	Namespace string
}

// Error implements the error interface.
func (e *noResourcesFoundError) Error() string {
	if e.ReleaseName != "" {
		return fmt.Sprintf("no resources found for release %s in namespace %s", e.ReleaseName, e.Namespace)
	}
	return fmt.Sprintf("no resources found for release-id %s in namespace %s", e.ReleaseID, e.Namespace)
}

// Is implements errors.Is for noResourcesFoundError.
func (e *noResourcesFoundError) Is(target error) bool {
	return target == errNoResourcesFound
}

// IsNoResourcesFound reports whether err (or any error in its chain)
// indicates that no resources matched the discovery selector, or that
// no inventory was found for the given release.
func IsNoResourcesFound(err error) bool {
	return errors.Is(err, errNoResourcesFound)
}

// ReleaseNotFoundError is returned when no inventory Secret exists for the
// given release name/namespace. It is used by commands that require an
// inventory to operate (status, delete).
type ReleaseNotFoundError struct {
	// Name is the release name or release-id that was searched.
	Name string
	// Namespace is the namespace that was searched.
	Namespace string
}

// Error implements the error interface.
func (e *ReleaseNotFoundError) Error() string {
	return fmt.Sprintf("release %q not found in namespace %q", e.Name, e.Namespace)
}

// Is implements errors.Is so that IsNoResourcesFound matches ReleaseNotFoundError,
// allowing --ignore-not-found to suppress both error types uniformly.
func (e *ReleaseNotFoundError) Is(target error) bool {
	return target == errNoResourcesFound
}
