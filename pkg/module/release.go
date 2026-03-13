package module

import (
	"cuelang.org/go/cue"
)

// Release is the concrete deployment instance of a module as it moves
// through parse, process, and render stages.
type Release struct {
	// Metadata is extracted for Go-side operations: inventory, K8s labeling, display.
	Metadata *ReleaseMetadata

	// Module is the original module, preserved for reference.
	Module Module

	// RawCUE is the whole Release CUE value.
	// It is initially the parse-only raw value, then replaced with the concrete
	// processed value after successful value filling.
	RawCUE cue.Value

	// DataComponents is the finalized, constraint-free components value.
	// Safe for FillPath injection into transformer #transform definitions.
	DataComponents cue.Value

	// Config is the #config schema extracted from the release's module view.
	Config cue.Value

	// Values is the merged, validated concrete values supplied by the caller.
	Values cue.Value
}

// NewRelease constructs a Release.
func NewRelease(metadata *ReleaseMetadata, mod Module, rawCUE, dataComponents, config, values cue.Value) *Release {
	return &Release{
		Metadata:       metadata,
		Module:         mod,
		RawCUE:         rawCUE,
		DataComponents: dataComponents,
		Config:         config,
		Values:         values,
	}
}

// MatchComponents returns the concrete component view used for matching.
// This must preserve definition fields such as #resources and #traits.
func (r *Release) MatchComponents() cue.Value {
	return r.RawCUE.LookupPath(cue.ParsePath("components"))
}

// ExecuteComponents returns the finalized, constraint-free component view.
func (r *Release) ExecuteComponents() cue.Value {
	return r.DataComponents
}

// ReleaseMetadata contains release-level identity information.
// Used for K8s inventory tracking, resource labeling, and CLI output.
type ReleaseMetadata struct {
	// Name is the release name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged release labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
