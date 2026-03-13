// A Release is the concrete deployment instance of a bundle after values
// are supplied. CUE evaluates the bundle into a map of ModuleReleases;
// Go iterates this map and calls ModuleRenderer.Render() for each entry.
package bundle

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
)

// Release is the concrete deployment instance of a bundle as it moves
// through parse, process, and render stages.
type Release struct {
	// Metadata is extracted for Go-side operations: display, inventory, UUID tracking.
	Metadata *ReleaseMetadata

	// Bundle is the original bundle definition, preserved for reference.
	Bundle Bundle

	// RawCUE is the whole Release CUE value.
	RawCUE cue.Value

	// Releases is the map of ModuleReleases produced during bundle processing.
	Releases map[string]*module.Release

	// Config is the #config schema extracted from the release's bundle view.
	Config cue.Value

	// Values is the merged, validated concrete values supplied by the caller.
	Values cue.Value
}

// ReleaseMetadata contains release-level identity information.
// Parallel to ModuleReleaseMetadata but without Namespace —
// bundles do not have a single target namespace.
type ReleaseMetadata struct {
	// Name is the bundle release name.
	Name string `json:"name"`

	// UUID is the release identity UUID.
	UUID string `json:"uuid"`

	// Labels are the merged release labels.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
