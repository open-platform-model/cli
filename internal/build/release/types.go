// Package release provides release building functionality for OPM modules.
package release

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/build/module"
)

// Options configures release building.
type Options struct {
	Name      string // Required: release name
	Namespace string // Required: target namespace
	PkgName   string // Internal: CUE package name (set by InspectModule, skip detectPackageName)
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
	Value      cue.Value                          // The concrete module value (with #config injected)
	Components map[string]*module.LoadedComponent // Concrete components by name
	Metadata   Metadata
}

// Metadata contains release-level metadata.
type Metadata struct {
	Name      string
	Namespace string
	Version   string
	FQN       string
	Labels    map[string]string
	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string
	// ReleaseIdentity is the release identity UUID.
	// Computed by the CUE overlay via uuid.SHA1(OPMNamespace, "fqn:name:namespace").
	ReleaseIdentity string
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
