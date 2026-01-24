// Package cue provides CUE module loading and rendering functionality.
package cue

import (
	"cuelang.org/go/cue"
)

// Module represents a loaded OPM module.
type Module struct {
	// Metadata from the module definition.
	Metadata ModuleMetadata

	// Root is the CUE value after loading and unification.
	Root cue.Value

	// Dir is the directory containing the module.
	Dir string

	// ValuesFiles are the values files that were unified.
	ValuesFiles []string
}

// ModuleMetadata contains module identification information.
type ModuleMetadata struct {
	// APIVersion is the module API version (e.g., "example.com/modules@v0").
	APIVersion string `json:"apiVersion"`

	// Name is the module name.
	Name string `json:"name"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`
}

// Bundle represents a loaded OPM bundle.
type Bundle struct {
	// Metadata from the bundle definition.
	Metadata BundleMetadata

	// Modules contained in the bundle.
	Modules map[string]*Module

	// Root is the CUE value.
	Root cue.Value

	// Dir is the directory containing the bundle.
	Dir string
}

// BundleMetadata contains bundle identification information.
type BundleMetadata struct {
	// APIVersion is the bundle API version.
	APIVersion string `json:"apiVersion"`

	// Name is the bundle name.
	Name string `json:"name"`

	// Version is the bundle version (optional).
	Version string `json:"version,omitempty"`
}
