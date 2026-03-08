// Package modulerelease defines the ModuleRelease and ReleaseMetadata types.
// A ModuleRelease is the concrete deployment instance of a module after values
// are merged. CUE handles all component resolution, matching, and transformation;
// Go only owns the release-level identity needed for inventory and K8s operations.
package modulerelease

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
)

// ModuleRelease is the built module release, ready for pipeline evaluation.
//
// Two CUE representations are kept internally:
//   - schema: the original evaluated value — preserves definition fields
//     (#resources, #traits, etc.) needed for CUE-native #MatchPlan matching.
//   - dataComponents: finalized, constraint-free components — safe for FillPath
//     injection into transformer #transform definitions.
//
// Access them via MatchComponents() and ExecuteComponents() respectively.
type ModuleRelease struct {
	// Metadata is extracted for Go-side operations: inventory, K8s labeling, display.
	Metadata *ReleaseMetadata

	// Module is the original module, preserved for reference.
	Module module.Module

	// schema is the original CUE value as evaluated by BuildInstance.
	// Preserves definition fields (#resources, #traits, #config, etc.)
	// required for transformer matching and component introspection.
	schema cue.Value

	// dataComponents is the finalized, constraint-free components value.
	// Produced by finalizing only the `components` field of the release,
	// which avoids problematic fields (values, #config) that carry schema
	// validators like matchN. Safe for FillPath injection into transformers.
	dataComponents cue.Value
}

// NewModuleRelease constructs a ModuleRelease with both CUE representations.
// schema must preserve definition fields; dataComponents must be finalized.
func NewModuleRelease(metadata *ReleaseMetadata, mod module.Module, schema, dataComponents cue.Value) *ModuleRelease {
	return &ModuleRelease{
		Metadata:       metadata,
		Module:         mod,
		schema:         schema,
		dataComponents: dataComponents,
	}
}

// MatchComponents returns the schema CUE value that preserves #resources,
// #traits, and #blueprints definition fields needed for CUE-native #MatchPlan
// evaluation.
func (r *ModuleRelease) MatchComponents() cue.Value {
	return r.schema.LookupPath(cue.ParsePath("components"))
}

// Schema returns the full schema CUE value (the original evaluated release).
// Used by the engine to pass to buildMatchPlan.
func (r *ModuleRelease) Schema() cue.Value {
	return r.schema
}

// ExecuteComponents returns the finalized, constraint-free CUE value of the
// components. Safe for FillPath injection into transformer #transform definitions
// without schema constraint conflicts.
func (r *ModuleRelease) ExecuteComponents() cue.Value {
	return r.dataComponents
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
