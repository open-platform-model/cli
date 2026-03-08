// Package bundlerelease defines the BundleRelease and BundleReleaseMetadata types.
// A BundleRelease is the concrete deployment instance of a bundle after values
// are supplied. CUE evaluates the bundle into a map of ModuleReleases;
// Go iterates this map and calls ModuleRenderer.Render() for each entry.
package bundlerelease

import (
	"github.com/opmodel/cli/pkg/bundle"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// BundleRelease is the built bundle release, ready for orchestrated rendering.
type BundleRelease struct {
	// Metadata is extracted for Go-side operations: display, inventory, UUID tracking.
	Metadata *BundleReleaseMetadata

	// Bundle is the original bundle definition, preserved for reference.
	Bundle bundle.Bundle

	// Releases is the map of ModuleReleases produced by CUE evaluation
	// of #BundleRelease.releases. Keys are the instance names from #bundle.#instances
	// (e.g., "server", "proxy"). The orchestrator iterates this map and calls
	// ModuleRenderer.Render() for each entry.
	Releases map[string]*modulerelease.ModuleRelease
}

// BundleReleaseMetadata contains release-level identity information.
// Parallel to modulerelease.ReleaseMetadata but without Namespace —
// bundles do not have a single target namespace.
type BundleReleaseMetadata struct {
	// Name is the bundle release name.
	Name string `json:"name"`

	// UUID is the release identity UUID.
	UUID string `json:"uuid"`

	// Labels are the merged release labels.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
