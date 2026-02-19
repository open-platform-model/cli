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
	Value           cue.Value                          // The concrete module value (with #config injected)
	Components      map[string]*module.LoadedComponent // Concrete components by name
	ReleaseMetadata ReleaseMetadata
	ModuleMetadata  module.ModuleMetadata
}

// ReleaseMetadata contains release-level identity information for a deployed module.
// This metadata is used for labeling resources, inventory tracking, and verbose output.
//
//nolint:revive // ReleaseMetadata is intentional: the type is re-exported as build.ReleaseMetadata without stutter.
type ReleaseMetadata struct {
	// Name is the release name (resolved from RenderOptions.Name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	// From default config.cue or --namespace flag.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Deterministic UUID5 computed from fqn+name+namespace.
	UUID string `json:"uuid"`

	// Labels from the module release.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module release.
	// Currently empty; populated when CUE annotation extraction is added.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Components lists the component names rendered in this release.
	Components []string `json:"components,omitempty"`
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
