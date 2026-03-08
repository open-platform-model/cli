// Package bundle defines the Bundle and BundleMetadata types, mirroring the
// #Bundle definition in the CUE catalog (v1alpha1). A Bundle represents a
// collection of modules grouped for distribution.
//
// Go does not inspect #instances, #config, or #policies — those are
// definition fields that stay in CUE. This package carries only the identity
// metadata needed for display and the raw CUE value for the loader.
package bundle

import (
	"cuelang.org/go/cue"
)

// Bundle holds a loaded bundle definition.
type Bundle struct {
	// Metadata is extracted for display and bundle selection.
	Metadata *BundleMetadata

	// Data is the fully evaluated CUE value for the bundle.
	Data cue.Value
}

// BundleMetadata holds identity metadata for a bundle.
//
//nolint:revive // stutter intentional: bundle.BundleMetadata reads clearly at call sites
type BundleMetadata struct {
	// Name is the bundle name (kebab-case).
	Name string `json:"name"`

	// Description is a brief human-readable description.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path (e.g., "opmodel.dev/bundles").
	ModulePath string `json:"modulePath"`

	// Version is the bundle major version (e.g., "v1").
	Version string `json:"version"`

	// FQN is the fully qualified bundle name (modulePath/name:version).
	FQN string `json:"fqn"`

	// UUID is the bundle identity UUID, computed by CUE as SHA1(OPMNamespace, fqn).
	UUID string `json:"uuid"`

	// Labels for bundle categorization.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations for additional bundle metadata.
	Annotations map[string]string `json:"annotations,omitempty"`
}
