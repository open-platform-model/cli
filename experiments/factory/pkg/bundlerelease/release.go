// Package bundlerelease defines the BundleRelease and BundleReleaseMetadata types.
// A BundleRelease is the concrete deployment instance of a bundle after values
// are supplied. CUE evaluates the bundle into a map of ModuleReleases;
// Go iterates this map and calls ModuleRenderer.Render() for each entry.
package bundlerelease

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/experiments/factory/pkg/bundle"
	"github.com/opmodel/cli/experiments/factory/pkg/modulerelease"
)

// BundleRelease is the built bundle release, ready for orchestrated rendering.
// Each ModuleRelease carries its own DataComponents (finalized, constraint-free
// components) and Schema (original evaluated value with definition fields).
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

	// Schema is the original CUE value as evaluated by BuildInstance.
	// Preserves definition fields required for policy resolution and introspection.
	Schema cue.Value
}

// BundleReleaseMetadata contains release-level identity information.
// Structurally parallel to modulerelease.ReleaseMetadata but without Namespace —
// bundles do not have a single target namespace.
type BundleReleaseMetadata struct {
	// Name is the bundle release name.
	Name string `json:"name"`

	// UUID is the release identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, bundleUUID:name).
	UUID string `json:"uuid"`

	// Labels are the merged release labels.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
