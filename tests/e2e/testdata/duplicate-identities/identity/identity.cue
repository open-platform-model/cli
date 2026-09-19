// Package identity is the single source of this module's path and version
// (core #IdentityPackage, 0010:D2, 0011:D12).
package identity

// ModulePath is the module's complete CUE module path, major suffix included
// — byte-identical to cue.mod's `module:` field.
ModulePath: "test.example.com/dupidentities@v0"

// Version is the module's bare SemVer. This fixture is never published; it
// exists only to be rendered by the e2e suite.
Version: "0.1.0"
