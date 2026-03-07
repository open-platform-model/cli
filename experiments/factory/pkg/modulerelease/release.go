// Package modulerelease defines the ModuleRelease and ReleaseMetadata types.
// A ModuleRelease is the concrete deployment instance of a module after values
// are merged. CUE handles all component resolution, matching, and transformation;
// Go only owns the release-level identity needed for inventory and K8s operations.
package modulerelease

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/experiments/factory/pkg/module"
)

// ModuleRelease is the built module release, ready for pipeline evaluation.
// Two CUE representations are kept:
//   - DataComponents: finalized, constraint-free components — safe for FillPath injection.
//   - Schema: original evaluated value — preserves definition fields (#resources, #traits,
//     #config, etc.) needed for matching and introspection.
type ModuleRelease struct {
	// Metadata is extracted for Go-side operations: inventory, K8s labeling, display.
	Metadata *ReleaseMetadata `json:"metadata"`

	// Module is the original module, preserved for reference.
	Module module.Module `json:"#module"`

	// DataComponents is the finalized, constraint-free components value.
	// Produced by finalizing only the `components` field of the release,
	// which avoids problematic fields (values, #config) that carry schema
	// validators like matchN. Safe for FillPath injection into transformers.
	DataComponents cue.Value `json:"-"`

	// Schema is the original CUE value as evaluated by BuildInstance.
	// Preserves definition fields (#resources, #traits, #config, etc.)
	// required for transformer matching and component introspection.
	Schema cue.Value `json:"schema"`
}

// ReleaseMetadata contains release-level identity information.
// Used for K8s inventory tracking, resource labeling, and CLI output.
// Most of this is also available inside Schema; it is extracted here
// so Go code does not need to reach into CUE for routine operations.
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
