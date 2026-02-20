// Package module provides module loading and AST inspection utilities.
package module

// Inspection contains metadata extracted from a module directory
// via AST inspection without CUE evaluation.
// Used internally by Load(); kept exported for use in tests.
type Inspection struct {
	Name             string // metadata.name (empty if not a string literal)
	DefaultNamespace string // metadata.defaultNamespace (empty if not a string literal)
	PkgName          string // CUE package name from inst.PkgName
}
