package module

import (
	"cuelang.org/go/cue"
)

// Instance is a fully prepared module instance ready for rendering.
// When an *Instance exists, all invariants hold: Spec is concrete and complete,
// Values is concrete and merged, Metadata is decoded.
//
// Was: Release (enhancement 0002 D8 hard-rename).
type Instance struct {
	// Metadata is the decoded instance identity from the concrete instance spec.
	Metadata *InstanceMetadata

	// Module is the original module used to prepare the instance.
	Module Module

	// Spec is the concrete, values-filled #ModuleInstance CUE value.
	// Concrete (all regular fields resolved) but NOT finalized — CUE definition
	// fields (#resources, #traits, #blueprints) are preserved. Required by
	// MatchComponents() for component-transformer matching.
	// MUST NOT be passed to finalizeValue or v.Syntax(cue.Final()).
	Spec cue.Value

	// Values is the concrete, merged values applied to the instance.
	Values cue.Value
}

// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints.
func (r *Instance) MatchComponents() cue.Value {
	return r.Spec.LookupPath(cue.ParsePath("components"))
}

// InstanceMetadata contains instance-level identity information.
// Used for K8s inventory tracking, resource labeling, and CLI output.
//
// Was: ReleaseMetadata (enhancement 0002 D8 hard-rename).
type InstanceMetadata struct {
	// Name is the instance name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// UUID is the instance identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged instance labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged instance annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
