package render

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
)

// ModuleRelease is the concrete deployment instance of a module as it moves
// through parse, process, and render stages.
type ModuleRelease struct {
	// Metadata is extracted for Go-side operations: inventory, K8s labeling, display.
	Metadata *ModuleReleaseMetadata

	// Module is the original module, preserved for reference.
	Module module.Module

	// RawCUE is the whole ModuleRelease CUE value.
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

// NewModuleRelease constructs a ModuleRelease.
func NewModuleRelease(metadata *ModuleReleaseMetadata, mod module.Module, rawCUE, dataComponents, config, values cue.Value) *ModuleRelease {
	return &ModuleRelease{
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
func (r *ModuleRelease) MatchComponents() cue.Value {
	return r.RawCUE.LookupPath(cue.ParsePath("components"))
}

// ExecuteComponents returns the finalized, constraint-free component view.
func (r *ModuleRelease) ExecuteComponents() cue.Value {
	return r.DataComponents
}

// ModuleReleaseMetadata contains release-level identity information.
// Used for K8s inventory tracking, resource labeling, and CLI output.
type ModuleReleaseMetadata struct {
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
