// Package module provides module loading and AST inspection utilities.
package module

import "cuelang.org/go/cue"

// LoadedComponent is a component with extracted metadata.
// Components are extracted by the release builder during the build phase.
// This type is shared across build subpackages via type alias in the root package.
type LoadedComponent struct {
	Name        string
	Labels      map[string]string    // Effective labels (merged from resources/traits)
	Annotations map[string]string    // Annotations from metadata.annotations
	Resources   map[string]cue.Value // FQN -> resource value
	Traits      map[string]cue.Value // FQN -> trait value
	Value       cue.Value            // Full component value
}

// Inspection contains metadata extracted from a module directory
// via AST inspection without CUE evaluation.
type Inspection struct {
	Name             string // metadata.name (empty if not a string literal)
	DefaultNamespace string // metadata.defaultNamespace (empty if not a string literal)
	PkgName          string // CUE package name from inst.PkgName
}

// MetadataPreview contains lightweight module metadata extracted
// before the full overlay build. Used for name/namespace resolution.
type MetadataPreview struct {
	Name             string
	DefaultNamespace string
}
