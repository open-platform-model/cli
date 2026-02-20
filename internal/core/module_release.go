package core

import "cuelang.org/go/cue"

// ModuleRelease represents the built module release after the build phase, before any transformations are applied.
// Contains the fully concrete components with all metadata extracted and values merged, ready for matching and transformation.
type ModuleRelease struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// Metadata contains release-level identity information for a deployed module.
	// This metadata is used for labeling resources, inventory tracking, and verbose output.
	Metadata *ReleaseMetadata `json:"metadata"`

	// The original module definition, preserved for reference and verbose output.
	Module Module `json:"module"`

	// Concrete components with all metadata extracted and values merged, ready for matching and transformation.
	// Must preserve the original order of the #ModuleRelease.components map for deterministic output and to support index-based inventory tracking.
	Components map[string]*Component `json:"components,omitempty"`

	// The values from the Module Release. End-user values.
	Values cue.Value `json:"values,omitempty"`
}

// ReleaseMetadata contains release-level identity information for a deployed module.
// This metadata is used for labeling resources, inventory tracking, and verbose output.
type ReleaseMetadata struct {
	// Name is the release name (resolved from RenderOptions.Name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	// From default config.cue or --namespace flag.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Deterministic UUID5 computed from ModuleMetadata.FQN+ReleaseMetadata.Name+ReleaseMetadata.Namespace.
	UUID string `json:"uuid"`

	// Labels from the module release.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module release.
	// TODO: not yet implemented. Populate this in extractReleaseMetadata (internal/build/release/metadata.go)
	// once CUE annotation extraction is added, then wire it into TransformerContext.ToMap
	// (internal/build/transform/context.go) alongside Labels.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Components lists the component names rendered in this release.
	// This is a CLI only field used for verbose output and inventory tracking; not part of the core render contract.
	Components []string `json:"components,omitempty"`
}
