// Package core provides shared primitives for the factory pipeline.
package core

import (
	"cuelang.org/go/cue"
)

// Resource is a single rendered platform resource produced by the pipeline.
//
// Value holds the raw CUE output from the transformer, avoiding premature
// conversion to Go-native formats. Callers convert to their required format
// (YAML, JSON, unstructured.Unstructured, etc.) only when needed.
//
// Release, Component, and Transformer record provenance for inventory tracking
// and display. Release is especially important when rendering a BundleRelease,
// where resources from multiple ModuleReleases are collected together.
type Resource struct {
	// Value is the CUE value of the rendered resource (e.g. a Kubernetes manifest).
	// Concrete and fully evaluated — safe to encode directly to YAML or JSON.
	Value cue.Value

	// Release is the name of the ModuleRelease that produced this resource.
	// Set by the engine from rel.Metadata.Name.
	Release string

	// Component is the source component name within the release.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer string
}
