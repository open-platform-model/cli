// Package identity is the single source of this module's path and version
// (core #IdentityPackage, enhancements 0010 D38 / 0011 D12). It sits at the
// bottom of the module's import graph — no intra-module imports, no core
// import; validation is external (a publishing tool unifies this package
// against core's #IdentityPackage).
package identity

// ModulePath is the module's complete CUE module path, major suffix included
// — byte-identical to cue.mod's `module:` field.
ModulePath: "opmodel.dev/templates/standard@v1"

// Version is the module's bare SemVer; its major must agree with ModulePath's.
// A plain literal: the kernel's loader gate requires a concrete value, and a
// defaulted disjunction is not one. Written by opm module version set.
Version: "1.0.1"
