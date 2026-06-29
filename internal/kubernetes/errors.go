package kubernetes

import (
	"errors"
	"fmt"
)

// errNoResourcesFound is returned when no resources match the selector.
var errNoResourcesFound = errors.New("no resources found")

// noResourcesFoundError contains details about a failed resource discovery.
type noResourcesFoundError struct {
	// InstanceName is the instance name that was searched (empty if using instance-id).
	InstanceName string
	// InstanceID is the instance-id that was searched (empty if using instance-name).
	InstanceID string
	// Namespace is the namespace that was searched.
	Namespace string
}

// Error implements the error interface.
func (e *noResourcesFoundError) Error() string {
	if e.InstanceName != "" {
		return fmt.Sprintf("no resources found for instance %s in namespace %s", e.InstanceName, e.Namespace)
	}
	return fmt.Sprintf("no resources found for instance-id %s in namespace %s", e.InstanceID, e.Namespace)
}

// Is implements errors.Is for noResourcesFoundError.
func (e *noResourcesFoundError) Is(target error) bool {
	return target == errNoResourcesFound
}

// IsNoResourcesFound reports whether err (or any error in its chain)
// indicates that no resources matched the discovery selector, or that
// no inventory was found for the given instance.
func IsNoResourcesFound(err error) bool {
	return errors.Is(err, errNoResourcesFound)
}

// InstanceNotFoundError is returned when no inventory Secret exists for the
// given instance name/namespace. It is used by commands that require an
// inventory to operate (status, delete).
type InstanceNotFoundError struct {
	// Name is the instance name or instance-id that was searched.
	Name string
	// Namespace is the namespace that was searched.
	Namespace string
}

// Error implements the error interface.
func (e *InstanceNotFoundError) Error() string {
	return fmt.Sprintf("instance %q not found in namespace %q", e.Name, e.Namespace)
}

// Is implements errors.Is so that IsNoResourcesFound matches InstanceNotFoundError,
// allowing both error types to map to the same ExitNotFound exit code.
func (e *InstanceNotFoundError) Is(target error) bool {
	return target == errNoResourcesFound
}
