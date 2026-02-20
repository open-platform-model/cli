// Package module provides module loading and AST inspection utilities.
package module

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
